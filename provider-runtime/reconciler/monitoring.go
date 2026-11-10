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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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

// maxMetricsEndpoints is the maxItems of MonitoringSources.metrics.
const maxMetricsEndpoints = 32

// indexDestinationSecrets indexes MonitoringDestinations by the name of
// every Secret they reference.
const indexDestinationSecrets = "spec.secretNames"

func destinationSecrets(obj client.Object) []string {
	d, ok := obj.(*monitoringv1alpha1.MonitoringDestination)
	if !ok {
		return nil
	}
	return d.Spec.SecretNames()
}

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
// renderings are built from both, or a destination changes readiness, which
// the entries' Ready condition quotes. A provider that renders integrations
// is also re-enqueued on any other destination status change and on any
// change of a Secret a destination references, so a rotated or late Secret
// is rendered.
func (r *ProviderReconciler) watchMonitoringBindings(b *builder.Builder) {
	b.Watches(&monitoringv1alpha1.MonitoringBinding{},
		handler.EnqueueRequestsFromMapFunc(r.instancesForBinding),
		builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
	)
	b.Watches(&monitoringv1alpha1.MonitoringDestination{},
		handler.EnqueueRequestsFromMapFunc(r.instancesForDestination),
		builder.WithPredicates(r.destinationChanged()),
	)
	if _, ok := r.provider.(controller.MonitoringIntegrationsProvider); ok {
		b.WatchesMetadata(&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.instancesForDestinationSecret),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		)
	}
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

// destinationChanged passes every destination change to a provider that
// renders integrations, and only spec and readiness changes to any other.
func (r *ProviderReconciler) destinationChanged() predicate.Predicate { //nolint:ireturn // builder.WithPredicates takes the interface
	if _, ok := r.provider.(controller.MonitoringIntegrationsProvider); ok {
		return predicate.ResourceVersionChangedPredicate{}
	}
	return predicate.Or(predicate.GenerationChangedPredicate{}, monitoring.DestinationReadinessChanged())
}

func (r *ProviderReconciler) instancesForDestination(ctx context.Context, obj client.Object) []reconcile.Request {
	return r.instancesBoundTo(ctx, obj.GetNamespace(), []string{obj.GetName()})
}

// instancesForDestinationSecret enqueues the Instances bound to a destination
// that references the Secret.
func (r *ProviderReconciler) instancesForDestinationSecret(ctx context.Context, obj client.Object) []reconcile.Request {
	destinations := &monitoringv1alpha1.MonitoringDestinationList{}
	if err := r.List(ctx, destinations, client.InNamespace(obj.GetNamespace()),
		client.MatchingFields{indexDestinationSecrets: obj.GetName()},
	); err != nil {
		log.FromContext(ctx).Error(err, "Listing monitoring destinations to enqueue failed")
		return nil
	}
	names := make([]string, 0, len(destinations.Items))
	for _, d := range destinations.Items {
		names = append(names, d.Name)
	}
	return r.instancesBoundTo(ctx, obj.GetNamespace(), names)
}

// instancesBoundTo enqueues this provider's Instances in namespace that have
// a binding to one of the destinations.
func (r *ProviderReconciler) instancesBoundTo(ctx context.Context, namespace string, destinations []string) []reconcile.Request {
	if len(destinations) == 0 {
		return nil
	}
	bindings := &monitoringv1alpha1.MonitoringBindingList{}
	if err := r.List(ctx, bindings, client.InNamespace(namespace)); err != nil {
		log.FromContext(ctx).Error(err, "Listing monitoring bindings to enqueue failed")
		return nil
	}
	bound := make(map[string]struct{})
	for _, b := range bindings.Items {
		if slices.Contains(destinations, b.Labels[monitoringv1alpha1.BindingDestinationLabel]) {
			bound[b.Spec.InstanceRef.Name] = struct{}{}
		}
	}
	if len(bound) == 0 {
		return nil
	}
	return r.instancesMatching(ctx, namespace, func(in *v1alpha1.Instance) bool {
		_, ok := bound[in.Name]
		return ok
	})
}

func (r *ProviderReconciler) instancesForClass(ctx context.Context, obj client.Object) []reconcile.Request {
	bindings := &monitoringv1alpha1.MonitoringBindingList{}
	if err := r.List(ctx, bindings, client.MatchingLabels{monitoringv1alpha1.BindingClassLabel: obj.GetName()}); err != nil {
		log.FromContext(ctx).Error(err, "Listing monitoring bindings to enqueue failed")
		return nil
	}
	bound := make(map[client.ObjectKey]struct{})
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
// MonitoringConfigured and MonitoringReady conditions after a successful
// Sync, and writes the Configured condition of the ProviderManaged bindings.
// Failures are logged and reported on the conditions; they never fail the
// reconcile, since the engine itself is healthy.
func (r *ProviderReconciler) reconcileMonitoring(ctx context.Context, syncCtx *controller.Context, in *v1alpha1.Instance) {
	logger := log.FromContext(ctx).WithName("monitoring")

	var entries []v1alpha1.InstanceMonitoringDestination
	if in.Spec.Monitoring != nil {
		entries = in.Spec.Monitoring.Destinations
	}
	previous := previousEntryConditions(in)

	if !monitoringEnabled(r.provider) {
		r.reportMonitoringUnsupported(in, entries, previous)
		return
	}

	sources, note := r.monitoringSources(ctx, syncCtx, in, logger)
	in.Status.Monitoring = nil
	if sources != nil || len(entries) > 0 {
		in.Status.Monitoring = &v1alpha1.InstanceMonitoringStatus{Sources: sources}
	}

	if len(entries) == 0 {
		removeMonitoringConditions(in)
		return
	}
	bindings, err := syncCtx.MonitoringBindings()
	if err != nil {
		logger.Error(err, "Listing monitoring bindings failed")
		msg := err.Error()
		setCondition(in, v1alpha1.ConditionMonitoringReady, metav1.ConditionUnknown,
			v1alpha1.ReasonMonitoringPending, msg, metav1.Now(),
		)
		if note != "" {
			msg += "; " + note
		}
		setCondition(in, v1alpha1.ConditionMonitoringConfigured, metav1.ConditionUnknown,
			v1alpha1.ReasonMonitoringPending, msg, metav1.Now(),
		)
		return
	}
	byEntry := make(map[string]controller.ResolvedBinding, len(bindings))
	for _, b := range bindings {
		byEntry[b.Binding.Labels[monitoringv1alpha1.BindingEntryLabel]] = b
	}

	now := metav1.Now()
	for _, entry := range entries {
		var mode v1alpha1.MonitoringExecutionMode
		configured := entryCondition(v1alpha1.MonitoringEntryConditionConfigured, metav1.ConditionUnknown,
			string(monitoringv1alpha1.ReasonPending), "waiting for the binding to be created",
		)
		ready := entryCondition(v1alpha1.MonitoringEntryConditionReady, metav1.ConditionUnknown,
			string(monitoringv1alpha1.ReasonBindingReadyPending), "Waiting for the binding to be created",
		)
		if rb, ok := byEntry[entry.Name]; ok {
			mode = rb.Mode()
			configured = r.entryConfigured(ctx, syncCtx, rb, logger)
			ready = entryReady(rb, configured)
		}
		in.Status.Monitoring.Destinations = append(in.Status.Monitoring.Destinations,
			entryStatus(in, entry.Name, mode, previous[entry.Name], now, configured, ready),
		)
	}
	rollUpMonitoring(in, "", note)
}

// monitoringRollUp is an Instance condition aggregating one condition type
// of status.monitoring.destinations[].
type monitoringRollUp struct {
	condType      string
	entryCondType string
	trueReason    string
	trueMessage   string
}

// rollUpMonitoring sets MonitoringConfigured and MonitoringReady from the
// entries; falseReason is the reason of either when False, defaulting to
// its own NotConfigured or NotReady reason. note is appended to the
// MonitoringConfigured message.
func rollUpMonitoring(in *v1alpha1.Instance, falseReason, note string) {
	configuredFalse, readyFalse := v1alpha1.ReasonMonitoringNotConfigured, v1alpha1.ReasonMonitoringNotReady
	if falseReason != "" {
		configuredFalse, readyFalse = falseReason, falseReason
	}
	configured := monitoringRollUp{
		condType:      v1alpha1.ConditionMonitoringConfigured,
		entryCondType: v1alpha1.MonitoringEntryConditionConfigured,
		trueReason:    v1alpha1.ReasonMonitoringConfigured,
		trueMessage:   "All monitoring destinations are configured",
	}
	ready := monitoringRollUp{
		condType:      v1alpha1.ConditionMonitoringReady,
		entryCondType: v1alpha1.MonitoringEntryConditionReady,
		trueReason:    v1alpha1.ReasonMonitoringReady,
		trueMessage:   "All monitoring destinations are ready",
	}
	rollUp(in, configured, configuredFalse, note)
	rollUp(in, ready, readyFalse, "")
}

// rollUp sets ru's Instance condition from ru's condition on every entry:
// False with falseReason when any entry is False, else Unknown while any is
// not True, else True. The message lists "entry: reason" for every entry
// that is not True, followed by note.
func rollUp(in *v1alpha1.Instance, ru monitoringRollUp, falseReason, note string) {
	var notTrue []string
	anyFalse := false
	for _, d := range in.Status.Monitoring.Destinations {
		c := meta.FindStatusCondition(d.Conditions, ru.entryCondType)
		if c == nil {
			notTrue = append(notTrue, d.Name+": "+string(monitoringv1alpha1.ReasonPending))
			continue
		}
		if c.Status == metav1.ConditionTrue {
			continue
		}
		notTrue = append(notTrue, d.Name+": "+c.Reason)
		anyFalse = anyFalse || c.Status == metav1.ConditionFalse
	}
	status, reason, msg := metav1.ConditionUnknown, v1alpha1.ReasonMonitoringPending, strings.Join(notTrue, "; ")
	switch {
	case len(notTrue) == 0:
		status, reason, msg = metav1.ConditionTrue, ru.trueReason, ru.trueMessage
	case anyFalse:
		status, reason = metav1.ConditionFalse, falseReason
	}
	if note != "" {
		msg += "; " + note
	}
	setCondition(in, ru.condType, status, reason, msg, metav1.Now())
}

// removeMonitoringConditions drops the roll-ups of an Instance without
// destination entries.
func removeMonitoringConditions(in *v1alpha1.Instance) {
	meta.RemoveStatusCondition(&in.Status.Conditions, v1alpha1.ConditionMonitoringConfigured)
	meta.RemoveStatusCondition(&in.Status.Conditions, v1alpha1.ConditionMonitoringReady)
}

// reportMonitoringUnsupported fills the status for a provider that
// implements no monitoring interface: no class can be fulfilled, so every
// entry says so instead of staying silently empty.
func (r *ProviderReconciler) reportMonitoringUnsupported(
	in *v1alpha1.Instance,
	entries []v1alpha1.InstanceMonitoringDestination,
	previous map[string][]metav1.Condition,
) {
	in.Status.Monitoring = nil
	if len(entries) == 0 {
		removeMonitoringConditions(in)
		return
	}
	msg := fmt.Sprintf("provider %s does not implement the monitoring contract", r.provider.Name())
	in.Status.Monitoring = &v1alpha1.InstanceMonitoringStatus{}
	now := metav1.Now()
	for _, e := range entries {
		in.Status.Monitoring.Destinations = append(in.Status.Monitoring.Destinations, entryStatus(in, e.Name, "", previous[e.Name], now,
			entryCondition(v1alpha1.MonitoringEntryConditionConfigured, metav1.ConditionFalse, monitoringv1alpha1.ReasonIntegrationUnsupported, msg),
			entryCondition(v1alpha1.MonitoringEntryConditionReady, metav1.ConditionFalse, string(monitoringv1alpha1.ReasonBindingNotConfigured), msg),
		))
	}
	rollUpMonitoring(in, v1alpha1.ReasonMonitoringIntegrationUnsupported, "")
}

// previousEntryConditions returns the conditions of the entries currently in
// the status, by entry name.
func previousEntryConditions(in *v1alpha1.Instance) map[string][]metav1.Condition {
	if in.Status.Monitoring == nil {
		return nil
	}
	previous := make(map[string][]metav1.Condition, len(in.Status.Monitoring.Destinations))
	for _, d := range in.Status.Monitoring.Destinations {
		previous[d.Name] = d.Conditions
	}
	return previous
}

func entryCondition(condType string, status metav1.ConditionStatus, reason, message string) metav1.Condition {
	return metav1.Condition{Type: condType, Status: status, Reason: reason, Message: message}
}

// entryStatus builds an entry from its conditions, observed at the
// Instance's generation. A condition keeps its previous lastTransitionTime
// while its status does not change.
func entryStatus(
	in *v1alpha1.Instance,
	name string,
	mode v1alpha1.MonitoringExecutionMode,
	previous []metav1.Condition,
	now metav1.Time,
	conditions ...metav1.Condition,
) v1alpha1.InstanceMonitoringDestinationStatus {
	st := v1alpha1.InstanceMonitoringDestinationStatus{Name: name, Mode: mode}
	for _, c := range conditions {
		c.ObservedGeneration = in.Generation
		c.LastTransitionTime = now
		if prev := meta.FindStatusCondition(previous, c.Type); prev != nil && prev.Status == c.Status {
			c.LastTransitionTime = prev.LastTransitionTime
		}
		st.Conditions = append(st.Conditions, c)
	}
	return st
}

// monitoringSources asks the provider for its current sources and validates
// them; a failure keeps the previously published ones. The note, when not
// empty, reports the failure or the endpoints validation dropped.
func (r *ProviderReconciler) monitoringSources(ctx context.Context, syncCtx *controller.Context, in *v1alpha1.Instance, logger logr.Logger) (*v1alpha1.MonitoringSources, string) {
	sp, ok := r.provider.(controller.MonitoringSourcesProvider)
	if !ok {
		return nil, ""
	}
	var previous *v1alpha1.MonitoringSources
	if in.Status.Monitoring != nil {
		previous = in.Status.Monitoring.Sources
	}
	computed, err := sp.MonitoringSources(syncCtx)
	if err != nil {
		logger.Error(err, "MonitoringSources failed; keeping the published sources")
		return previous, "monitoring sources not updated: " + err.Error()
	}
	if computed == nil {
		return nil, ""
	}
	validated, err := r.validateSources(ctx, syncCtx, computed, logger)
	if err != nil {
		logger.Error(err, "Validating monitoring sources failed; keeping the published sources")
		return previous, "monitoring sources not updated: " + err.Error()
	}
	if dropped := len(computed.Metrics) - len(validated.Metrics); dropped > 0 {
		return validated, fmt.Sprintf("%d of %d metrics endpoints dropped", dropped, len(computed.Metrics))
	}
	if dropped := len(computed.Connections) - len(validated.Connections); dropped > 0 {
		return validated, fmt.Sprintf("%d of %d connection endpoints dropped", dropped, len(computed.Connections))
	}
	return validated, ""
}

// entryConfigured summarises one binding as its entry's Configured
// condition and, for a ProviderManaged binding, writes the binding's
// Configured condition.
func (r *ProviderReconciler) entryConfigured(ctx context.Context, syncCtx *controller.Context, rb controller.ResolvedBinding, logger logr.Logger) metav1.Condition {
	cond := entryCondition(v1alpha1.MonitoringEntryConditionConfigured, metav1.ConditionUnknown, string(monitoringv1alpha1.ReasonPending), "")
	accepted := meta.FindStatusCondition(rb.Binding.Status.Conditions, monitoringv1alpha1.MonitoringBindingConditionAccepted)
	switch {
	case accepted != nil && accepted.Status == metav1.ConditionFalse:
		cond.Status, cond.Reason, cond.Message = metav1.ConditionFalse, accepted.Reason, accepted.Message
		if rb.Mode() == v1alpha1.MonitoringExecutionModeProviderManaged {
			r.retainBinding(ctx, rb.Binding, accepted, logger)
		}
	case !monitoring.IsAccepted(rb.Binding):
		cond.Message = "waiting for the binding to be accepted"
	case rb.Mode() == v1alpha1.MonitoringExecutionModeProviderManaged && !rb.Apply:
		// The provider held the live rendering; core is about to revoke Accepted.
		cond.Message = "waiting for the destination and class to be resolved"
	case rb.Mode() == v1alpha1.MonitoringExecutionModeProviderManaged:
		written := r.configureBinding(ctx, syncCtx, rb, logger)
		cond.Status, cond.Reason, cond.Message = written.Status, written.Reason, written.Message
	default:
		cond.Message = "waiting for the class controller"
		configured := meta.FindStatusCondition(rb.Binding.Status.Conditions, monitoringv1alpha1.MonitoringBindingConditionConfigured)
		if configured != nil && configured.ObservedGeneration == rb.Binding.Generation {
			cond.Status, cond.Reason, cond.Message = configured.Status, configured.Reason, configured.Message
		}
	}
	return cond
}

// entryReady derives the entry's Ready condition the way core derives the
// binding's, but from the Configured condition this reconcile just
// determined, so the two conditions of an entry never disagree.
func entryReady(rb controller.ResolvedBinding, configured metav1.Condition) metav1.Condition {
	var accepted metav1.Condition
	if c := meta.FindStatusCondition(rb.Binding.Status.Conditions, monitoringv1alpha1.MonitoringBindingConditionAccepted); c != nil &&
		(c.Status == metav1.ConditionFalse || c.ObservedGeneration == rb.Binding.Generation) {
		accepted = *c
	}
	ready := monitoring.BindingReady(rb.Binding, accepted, configured, rb.Destination)
	return entryCondition(v1alpha1.MonitoringEntryConditionReady, ready.Status, ready.Reason, ready.Message)
}

// configureBinding writes the Configured condition of an Accepted
// ProviderManaged binding from the Sync outcome and any staged result. It
// returns the condition written, or a Pending one carrying the write error.
func (r *ProviderReconciler) configureBinding(ctx context.Context, syncCtx *controller.Context, rb controller.ResolvedBinding, logger logr.Logger) metav1.Condition {
	b := rb.Binding
	ip, renders := r.provider.(controller.MonitoringIntegrationsProvider)
	renders = renders && slices.Contains(ip.MonitoringIntegrations(), rb.Integration())
	staged, stagedMsg, isStaged := syncCtx.MonitoringBindingResult(b.Name)
	reason, msg := monitoringv1alpha1.ReasonConfigured, "Rendered by provider "+r.provider.Name()
	switch {
	case !renders:
		reason, msg = monitoringv1alpha1.ReasonRenderFailed, fmt.Sprintf("provider %s does not render integration %s", r.provider.Name(), rb.Integration())
	case isStaged:
		reason, msg = staged, stagedMsg
	}
	cond := monitoring.NewCondition(b, monitoringv1alpha1.MonitoringBindingConditionConfigured, reason.Status(), string(reason), msg)
	if err := monitoring.ApplyCondition(ctx, r.Client, b, b.Status.Conditions, r.configuredFieldOwner(), cond); err != nil {
		logger.Error(err, "Failed to write the Configured condition", "binding", b.Name)
		return metav1.Condition{
			Status:  metav1.ConditionUnknown,
			Reason:  string(monitoringv1alpha1.ReasonPending),
			Message: "failed to write the Configured condition: " + err.Error(),
		}
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
// sources, so they always satisfy the Instance status schema, and stamps the
// identity.
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

	out := computed.DeepCopy()
	out.Credentials, err = r.controlledCredentials(ctx, in, out.Credentials, logger)
	if err != nil {
		return nil, err
	}
	out.Metrics = validMetrics(in, spec, out.Credentials, out.Metrics, logger)
	out.Connections = validConnections(in, spec, out.Credentials, out.Connections, logger)

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

// controlledCredentials drops duplicate profiles and credentials whose
// Secret is not controlled by the Instance.
func (r *ProviderReconciler) controlledCredentials(ctx context.Context, in *v1alpha1.Instance, credentials []v1alpha1.MonitoringCredential, logger logr.Logger) ([]v1alpha1.MonitoringCredential, error) {
	var out []v1alpha1.MonitoringCredential
	seen := make(map[string]bool)
	for _, cred := range credentials {
		if seen[cred.Profile] {
			logger.Info("Dropping duplicate monitoring credential profile", "profile", cred.Profile)
			continue
		}
		seen[cred.Profile] = true
		secret := &metav1.PartialObjectMetadata{}
		secret.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Secret"))
		getErr := r.Get(ctx, client.ObjectKey{Namespace: in.Namespace, Name: cred.SecretRef.Name}, secret)
		if getErr != nil && !apierrors.IsNotFound(getErr) {
			return nil, fmt.Errorf("get credential Secret %s: %w", cred.SecretRef.Name, getErr)
		}
		if getErr != nil || !metav1.IsControlledBy(secret, in) {
			logger.Info("Dropping monitoring credential whose Secret is not controlled by the Instance",
				"profile", cred.Profile, "secret", cred.SecretRef.Name,
			)
			continue
		}
		out = append(out, cred)
	}
	return out, nil
}

// validMetrics drops the endpoints the Provider declaration or the Instance
// status schema rejects, sets each kind to its component type's declared
// kind (see endpointType) and keeps at most maxMetricsEndpoints.
func validMetrics(in *v1alpha1.Instance, spec *v1alpha1.ProviderSpec, credentials []v1alpha1.MonitoringCredential, metrics []v1alpha1.MetricsEndpoint, logger logr.Logger) []v1alpha1.MetricsEndpoint {
	var declared map[string]v1alpha1.MonitoringComponentType
	if spec.Monitoring != nil {
		declared = spec.Monitoring.ComponentTypes
	}
	profiles := make(map[string]bool, len(credentials))
	for _, cred := range credentials {
		profiles[cred.Profile] = true
	}

	var out []v1alpha1.MetricsEndpoint
	for _, ep := range metrics {
		componentType, kind, kindValid := endpointType(in, spec, ep.Component, ep.Kind)
		decl, isDeclared := declared[componentType]
		switch {
		case !isDeclared:
			logger.Info("Dropping metrics endpoint of an undeclared component type", "component", ep.Component, "type", componentType)
		case !kindValid:
			logger.Info("Dropping metrics endpoint whose kind is not the declared kind of its component type",
				"component", ep.Component, "type", componentType, "kind", ep.Kind, "declaredKind", kind,
			)
		case decl.Metrics == v1alpha1.MetricsSourceNone:
			logger.Info("Dropping metrics endpoint of a component type declared without metrics", "component", ep.Component, "type", componentType)
		case ep.PodSelector[controller.InstanceLabel] != in.Name:
			logger.Info("Dropping metrics endpoint whose pod selector lacks the Instance label", "component", ep.Component)
		case ep.Scheme != "" && ep.Scheme != "http" && ep.Scheme != "https":
			logger.Info("Dropping metrics endpoint with an unsupported scheme", "component", ep.Component, "scheme", ep.Scheme)
		case ep.Auth != nil && !profiles[ep.Auth.CredentialProfile]:
			logger.Info("Dropping metrics endpoint naming an unpublished credential profile", "component", ep.Component, "profile", ep.Auth.CredentialProfile)
		default:
			ep.Kind = kind
			out = append(out, ep)
		}
	}
	if len(out) > maxMetricsEndpoints {
		logger.Info("Dropping metrics endpoints beyond the limit", "limit", maxMetricsEndpoints, "dropped", len(out)-maxMetricsEndpoints)
		out = out[:maxMetricsEndpoints]
	}
	return out
}

// validConnections drops the connection endpoints the Provider declaration
// or the Instance status schema rejects, sets each kind to its component
// type's declared kind (see endpointType) and keeps at most
// maxMetricsEndpoints.
func validConnections(in *v1alpha1.Instance, spec *v1alpha1.ProviderSpec, credentials []v1alpha1.MonitoringCredential, connections []v1alpha1.ConnectionEndpoint, logger logr.Logger) []v1alpha1.ConnectionEndpoint {
	var declared map[string]v1alpha1.MonitoringComponentType
	if spec.Monitoring != nil {
		declared = spec.Monitoring.ComponentTypes
	}
	profiles := make(map[string]bool, len(credentials))
	for _, cred := range credentials {
		profiles[cred.Profile] = true
	}

	var out []v1alpha1.ConnectionEndpoint
	for _, ep := range connections {
		componentType, kind, kindValid := endpointType(in, spec, ep.Component, ep.Kind)
		decl, isDeclared := declared[componentType]
		switch {
		case !isDeclared:
			logger.Info("Dropping connection endpoint of an undeclared component type", "component", ep.Component, "type", componentType)
		case !kindValid:
			logger.Info("Dropping connection endpoint whose kind is not the declared kind of its component type",
				"component", ep.Component, "type", componentType, "kind", ep.Kind, "declaredKind", kind,
			)
		case decl.Connection != v1alpha1.ConnectionSupported:
			logger.Info("Dropping connection endpoint of a component type declared without connections", "component", ep.Component, "type", componentType)
		case ep.PodSelector[controller.InstanceLabel] != in.Name:
			logger.Info("Dropping connection endpoint whose pod selector lacks the Instance label", "component", ep.Component)
		case ep.Auth != nil && !profiles[ep.Auth.CredentialProfile]:
			logger.Info("Dropping connection endpoint naming an unpublished credential profile", "component", ep.Component, "profile", ep.Auth.CredentialProfile)
		default:
			ep.Kind = kind
			out = append(out, ep)
		}
	}
	if len(out) > maxMetricsEndpoints {
		logger.Info("Dropping connection endpoints beyond the limit", "limit", maxMetricsEndpoints, "dropped", len(out)-maxMetricsEndpoints)
		out = out[:maxMetricsEndpoints]
	}
	return out
}

// endpointType returns the type of an endpoint's component (from the
// Instance, else the Provider) and the kind the type declares in
// Provider.spec.componentTypes, which every published endpoint of the type
// carries, and whether an endpoint the provider computed with kind may be
// published: the type must declare a kind and the provider may leave kind
// empty, but an endpoint whose kind contradicts the declaration is dropped
// rather than relabelled, since consumers pick their client by kind.
func endpointType(in *v1alpha1.Instance, spec *v1alpha1.ProviderSpec, component, kind string) (string, string, bool) {
	componentType := in.Spec.Components[component].Type
	if componentType == "" {
		componentType = spec.Components[component].Type
	}
	declared := spec.ComponentTypes[componentType].Kind
	return componentType, declared, declared != "" && (kind == "" || kind == declared)
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
