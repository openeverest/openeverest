// Copyright (C) 2026 The OpenEverest Contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package reconciler

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
	"github.com/openeverest/openeverest/v2/provider-runtime/monitoring"
)

// Identity labels every monitoring class attaches to the series it ships.
const (
	identityLabelInstance  = "openeverest_instance"
	identityLabelNamespace = "openeverest_namespace"
	identityLabelProvider  = "openeverest_provider"
	identityLabelCluster   = "k8s_cluster_id"
)

// monitoringEnabled reports whether the provider opted into the monitoring
// contract through either optional interface.
func monitoringEnabled(p providerAdapter) bool {
	_, sources := p.(controller.MonitoringSourcesProvider)
	_, integrations := p.(controller.MonitoringIntegrationsProvider)
	return sources || integrations
}

// configuredFieldOwner is the field manager under which the runtime writes
// the Configured condition of ProviderManaged bindings.
func (r *ProviderReconciler) configuredFieldOwner() string {
	return "provider-" + r.provider.Name()
}

// watchMonitoringBindings re-enqueues this provider's Instances when one of
// their bindings changes (spec or status: Accepted flips matter), and when a
// destination or class they are bound to changes spec, since ProviderManaged
// renderings are built from both.
func (r *ProviderReconciler) watchMonitoringBindings(b *builder.Builder) {
	b.Watches(&monitoringv1alpha1.MonitoringBinding{},
		handler.EnqueueRequestsFromMapFunc(r.instancesForBinding),
		builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
	)
	b.Watches(&monitoringv1alpha1.MonitoringDestination{},
		handler.EnqueueRequestsFromMapFunc(r.instancesForDestination),
		builder.WithPredicates(predicate.GenerationChangedPredicate{}),
	)
	b.Watches(&monitoringv1alpha1.MonitoringClass{},
		handler.EnqueueRequestsFromMapFunc(r.instancesForClass),
		builder.WithPredicates(predicate.GenerationChangedPredicate{}),
	)
}

func (r *ProviderReconciler) instancesForBinding(ctx context.Context, obj client.Object) []reconcile.Request {
	instance := obj.GetLabels()[monitoringv1alpha1.BindingInstanceLabel]
	if instance == "" {
		return nil
	}
	return r.instancesMatching(ctx, obj.GetNamespace(), func(in *v1alpha1.Instance) bool {
		return in.Name == instance
	})
}

func (r *ProviderReconciler) instancesForDestination(ctx context.Context, obj client.Object) []reconcile.Request {
	return r.instancesMatching(ctx, obj.GetNamespace(), func(in *v1alpha1.Instance) bool {
		return in.Spec.Monitoring != nil && slices.ContainsFunc(in.Spec.Monitoring.Destinations,
			func(d v1alpha1.InstanceMonitoringDestination) bool { return d.DestinationRef.Name == obj.GetName() })
	})
}

func (r *ProviderReconciler) instancesForClass(ctx context.Context, obj client.Object) []reconcile.Request {
	bindings := &monitoringv1alpha1.MonitoringBindingList{}
	if err := r.List(ctx, bindings, client.MatchingLabels{monitoringv1alpha1.BindingClassLabel: obj.GetName()}); err != nil {
		log.FromContext(ctx).Error(err, "Listing monitoring bindings to enqueue failed")
		return nil
	}
	bound := map[client.ObjectKey]struct{}{}
	for _, b := range bindings.Items {
		bound[client.ObjectKey{Namespace: b.Namespace, Name: b.Labels[monitoringv1alpha1.BindingInstanceLabel]}] = struct{}{}
	}
	if len(bound) == 0 {
		return nil
	}
	return r.instancesMatching(ctx, "", func(in *v1alpha1.Instance) bool {
		_, ok := bound[client.ObjectKeyFromObject(in)]
		return ok
	})
}

// reconcileMonitoring fills Instance.status.monitoring and the
// MonitoringConfigured condition after a successful Sync, and writes the
// Configured condition of the ProviderManaged bindings. Failures are logged
// and reported on the condition; they never fail the reconcile, since the
// engine itself is healthy.
func (r *ProviderReconciler) reconcileMonitoring(ctx context.Context, syncCtx *controller.Context, in *v1alpha1.Instance) {
	logger := log.FromContext(ctx).WithName("monitoring")

	var entries []v1alpha1.InstanceMonitoringDestination
	if in.Spec.Monitoring != nil {
		entries = in.Spec.Monitoring.Destinations
	}

	if !monitoringEnabled(r.provider) {
		r.reportMonitoringUnsupported(in, entries)
		return
	}

	sources := r.monitoringSources(ctx, syncCtx, in, logger)
	in.Status.Monitoring = nil
	if sources != nil || len(entries) > 0 {
		in.Status.Monitoring = &v1alpha1.InstanceMonitoringStatus{Sources: sources}
	}

	if len(entries) == 0 {
		meta.RemoveStatusCondition(&in.Status.Conditions, v1alpha1.ConditionMonitoringConfigured)
		return
	}
	bindings, err := syncCtx.MonitoringBindings()
	if err != nil {
		logger.Error(err, "Listing monitoring bindings failed")
		setCondition(in, v1alpha1.ConditionMonitoringConfigured, metav1.ConditionUnknown,
			v1alpha1.ReasonMonitoringPending, err.Error(), metav1.Now())
		return
	}
	byEntry := make(map[string]controller.ResolvedBinding, len(bindings))
	for _, b := range bindings {
		byEntry[b.Binding.Labels[monitoringv1alpha1.BindingEntryLabel]] = b
	}

	for _, entry := range entries {
		st := v1alpha1.InstanceMonitoringDestinationStatus{
			Name:       entry.Name,
			Configured: metav1.ConditionUnknown,
			Reason:     string(monitoringv1alpha1.ReasonPending),
			Message:    "waiting for the binding to be created",
		}
		if rb, ok := byEntry[entry.Name]; ok {
			st = r.destinationStatus(ctx, syncCtx, entry.Name, rb, logger)
		}
		in.Status.Monitoring.Destinations = append(in.Status.Monitoring.Destinations, st)
	}
	setMonitoringConfigured(in, v1alpha1.ReasonMonitoringNotConfigured)
}

// setMonitoringConfigured rolls status.monitoring.destinations up into the
// MonitoringConfigured condition: False with falseReason when any entry is
// False, else Unknown while any is pending, else True. The message lists
// "entry: reason" for every entry that is not True.
func setMonitoringConfigured(in *v1alpha1.Instance, falseReason string) {
	var notConfigured []string
	anyFalse := false
	for _, d := range in.Status.Monitoring.Destinations {
		if d.Configured == metav1.ConditionTrue {
			continue
		}
		notConfigured = append(notConfigured, d.Name+": "+d.Reason)
		anyFalse = anyFalse || d.Configured == metav1.ConditionFalse
	}
	switch {
	case len(notConfigured) == 0:
		setCondition(in, v1alpha1.ConditionMonitoringConfigured, metav1.ConditionTrue,
			v1alpha1.ReasonMonitoringConfigured, "All monitoring destinations are configured", metav1.Now())
	case anyFalse:
		setCondition(in, v1alpha1.ConditionMonitoringConfigured, metav1.ConditionFalse,
			falseReason, strings.Join(notConfigured, "; "), metav1.Now())
	default:
		setCondition(in, v1alpha1.ConditionMonitoringConfigured, metav1.ConditionUnknown,
			v1alpha1.ReasonMonitoringPending, strings.Join(notConfigured, "; "), metav1.Now())
	}
}

// reportMonitoringUnsupported fills the status for a provider that
// implements no monitoring interface: no class can be fulfilled, so every
// entry says so instead of staying silently empty.
func (r *ProviderReconciler) reportMonitoringUnsupported(in *v1alpha1.Instance, entries []v1alpha1.InstanceMonitoringDestination) {
	in.Status.Monitoring = nil
	if len(entries) == 0 {
		meta.RemoveStatusCondition(&in.Status.Conditions, v1alpha1.ConditionMonitoringConfigured)
		return
	}
	msg := fmt.Sprintf("provider %s does not implement the monitoring contract", r.provider.Name())
	in.Status.Monitoring = &v1alpha1.InstanceMonitoringStatus{}
	for _, e := range entries {
		in.Status.Monitoring.Destinations = append(in.Status.Monitoring.Destinations, v1alpha1.InstanceMonitoringDestinationStatus{
			Name:       e.Name,
			Configured: metav1.ConditionFalse,
			Reason:     monitoringv1alpha1.ReasonIntegrationUnsupported,
			Message:    msg,
		})
	}
	setMonitoringConfigured(in, v1alpha1.ReasonMonitoringIntegrationUnsupported)
}

// monitoringSources asks the provider for its current sources and validates
// them; a failure keeps the previously published ones.
func (r *ProviderReconciler) monitoringSources(ctx context.Context, syncCtx *controller.Context, in *v1alpha1.Instance, logger logr.Logger) *v1alpha1.MonitoringSources {
	sp, ok := r.provider.(controller.MonitoringSourcesProvider)
	if !ok {
		return nil
	}
	var previous *v1alpha1.MonitoringSources
	if in.Status.Monitoring != nil {
		previous = in.Status.Monitoring.Sources
	}
	computed, err := sp.MonitoringSources(syncCtx)
	if err != nil {
		logger.Error(err, "MonitoringSources failed; keeping the published sources")
		return previous
	}
	if computed == nil {
		return nil
	}
	validated, err := r.validateSources(ctx, syncCtx, computed, logger)
	if err != nil {
		logger.Error(err, "Validating monitoring sources failed; keeping the published sources")
		return previous
	}
	return validated
}

// destinationStatus summarises one binding for
// Instance.status.monitoring.destinations[] and, for a ProviderManaged one,
// writes its Configured condition.
func (r *ProviderReconciler) destinationStatus(ctx context.Context, syncCtx *controller.Context, entryName string, rb controller.ResolvedBinding, logger logr.Logger) v1alpha1.InstanceMonitoringDestinationStatus {
	st := v1alpha1.InstanceMonitoringDestinationStatus{Name: entryName, Mode: rb.Mode()}
	accepted := meta.FindStatusCondition(rb.Binding.Status.Conditions, monitoringv1alpha1.MonitoringBindingConditionAccepted)
	switch {
	case accepted != nil && accepted.Status == metav1.ConditionFalse:
		st.Configured, st.Reason, st.Message = metav1.ConditionFalse, accepted.Reason, accepted.Message
		if rb.Mode() == v1alpha1.MonitoringExecutionModeProviderManaged {
			r.retainBinding(ctx, rb.Binding, accepted, logger)
		}
	case !monitoring.IsAccepted(rb.Binding):
		st.Configured, st.Reason, st.Message = metav1.ConditionUnknown, string(monitoringv1alpha1.ReasonPending), "waiting for the binding to be accepted"
	case rb.Mode() == v1alpha1.MonitoringExecutionModeProviderManaged && !rb.Apply:
		// The provider held the live rendering; core is about to revoke Accepted.
		st.Configured, st.Reason, st.Message = metav1.ConditionUnknown, string(monitoringv1alpha1.ReasonPending), "waiting for the destination and class to be resolved"
	case rb.Mode() == v1alpha1.MonitoringExecutionModeProviderManaged:
		cond := r.configureBinding(ctx, syncCtx, rb.Binding, logger)
		st.Configured, st.Reason, st.Message = cond.Status, cond.Reason, cond.Message
	default:
		st.Configured, st.Reason, st.Message = metav1.ConditionUnknown, string(monitoringv1alpha1.ReasonPending), "waiting for the class controller"
		if configured := meta.FindStatusCondition(rb.Binding.Status.Conditions, monitoringv1alpha1.MonitoringBindingConditionConfigured); configured != nil {
			st.Configured, st.Reason, st.Message = configured.Status, configured.Reason, configured.Message
		}
	}
	return st
}

// configureBinding writes the Configured condition of an Accepted
// ProviderManaged binding from the Sync outcome and any staged result.
func (r *ProviderReconciler) configureBinding(ctx context.Context, syncCtx *controller.Context, b *monitoringv1alpha1.MonitoringBinding, logger logr.Logger) metav1.Condition {
	reason, msg := monitoringv1alpha1.ReasonConfigured, "Rendered by provider "+r.provider.Name()
	if staged, stagedMsg, ok := syncCtx.MonitoringBindingResult(b.Name); ok {
		reason, msg = staged, stagedMsg
	}
	cond := monitoring.NewCondition(b, monitoringv1alpha1.MonitoringBindingConditionConfigured, reason.Status(), string(reason), msg)
	if err := monitoring.ApplyCondition(ctx, r.Client, b, b.Status.Conditions, r.configuredFieldOwner(), cond); err != nil {
		logger.Error(err, "Failed to write the Configured condition", "binding", b.Name)
	}
	return cond
}

// retainBinding reports a ProviderManaged binding that is not Accepted:
// Retained when a rendering was configured and is kept, NotAccepted when
// nothing was ever rendered. Both quote the Accepted reason.
func (r *ProviderReconciler) retainBinding(ctx context.Context, b *monitoringv1alpha1.MonitoringBinding, accepted *metav1.Condition, logger logr.Logger) {
	prev := meta.FindStatusCondition(b.Status.Conditions, monitoringv1alpha1.MonitoringBindingConditionConfigured)
	reason := monitoringv1alpha1.ReasonNotAccepted
	msg := "Binding is not Accepted (" + accepted.Reason + "); nothing is rendered"
	if prev != nil && (prev.Status == metav1.ConditionTrue || prev.Reason == string(monitoringv1alpha1.ReasonRetained)) {
		reason = monitoringv1alpha1.ReasonRetained
		msg = "Binding is no longer Accepted (" + accepted.Reason + "); keeping the live rendering"
	}
	cond := monitoring.NewCondition(b, monitoringv1alpha1.MonitoringBindingConditionConfigured, reason.Status(), string(reason), msg)
	if err := monitoring.ApplyCondition(ctx, r.Client, b, b.Status.Conditions, r.configuredFieldOwner(), cond); err != nil {
		logger.Error(err, "Failed to write the Configured condition", "binding", b.Name)
	}
}

// validateSources enforces the runtime invariants on the provider-computed
// sources and stamps the identity.
func (r *ProviderReconciler) validateSources(ctx context.Context, syncCtx *controller.Context, computed *v1alpha1.MonitoringSources, logger logr.Logger) (*v1alpha1.MonitoringSources, error) {
	in := syncCtx.Instance()
	spec, err := syncCtx.ProviderSpec()
	if err != nil {
		return nil, err
	}
	clusterID, err := r.clusterID(ctx)
	if err != nil {
		return nil, err
	}
	var declared map[string]v1alpha1.MonitoringComponentType
	if spec.Monitoring != nil {
		declared = spec.Monitoring.ComponentTypes
	}
	profiles := map[string]bool{}
	for _, cred := range computed.Credentials {
		profiles[cred.Profile] = true
	}

	out := computed.DeepCopy()
	metrics := out.Metrics
	out.Metrics = nil
	for _, ep := range metrics {
		componentType := in.Spec.Components[ep.Component].Type
		if componentType == "" {
			componentType = spec.Components[ep.Component].Type
		}
		_, isDeclared := declared[componentType]
		switch {
		case !isDeclared:
			logger.Info("Dropping metrics endpoint of an undeclared component type", "component", ep.Component, "type", componentType)
		case ep.PodSelector[controller.InstanceLabel] != in.Name:
			logger.Info("Dropping metrics endpoint whose pod selector lacks the Instance label", "component", ep.Component)
		case ep.Auth != nil && !profiles[ep.Auth.CredentialProfile]:
			logger.Info("Dropping metrics endpoint naming an unpublished credential profile", "component", ep.Component, "profile", ep.Auth.CredentialProfile)
		default:
			out.Metrics = append(out.Metrics, ep)
		}
	}

	out.Identity = v1alpha1.MonitoringIdentity{
		ClusterID: clusterID,
		Labels: map[string]string{
			identityLabelInstance:  in.Name,
			identityLabelNamespace: in.Namespace,
			identityLabelProvider:  r.provider.Name(),
			identityLabelCluster:   clusterID,
		},
	}
	return out, nil
}

// clusterID returns the UID of the kube-system namespace. It is read once,
// through the uncached reader so the provider needs only get on namespaces.
func (r *ProviderReconciler) clusterID(ctx context.Context) (string, error) {
	if id, ok := r.kubeSystemUID.Load().(string); ok {
		return id, nil
	}
	ns := &corev1.Namespace{}
	if err := r.apiReader.Get(ctx, client.ObjectKey{Name: metav1.NamespaceSystem}, ns); err != nil {
		return "", fmt.Errorf("get namespace %s: %w", metav1.NamespaceSystem, err)
	}
	r.kubeSystemUID.Store(string(ns.UID))
	return string(ns.UID), nil
}
