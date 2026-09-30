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
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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

// Identity labels every monitoring class must attach to the series it ships.
const (
	identityLabelInstance  = "openeverest_instance"
	identityLabelNamespace = "openeverest_namespace"
	identityLabelProvider  = "openeverest_provider"
	identityLabelCluster   = "k8s_cluster_id"
)

// monitoringEnabled reports whether the provider opted into the monitoring
// contract through either optional interface.
func monitoringEnabled(p providerAdapter) bool {
	_, tp := p.(controller.MonitoringSourcesProvider)
	_, mp := p.(controller.MonitoringIntegrationsProvider)
	return tp || mp
}

// configuredFieldOwner is the SSA field manager under which the runtime
// writes the Configured condition of ProviderManaged bindings.
func (r *ProviderReconciler) configuredFieldOwner() string {
	return "provider-" + r.provider.Name()
}

// watchMonitoringBindings re-enqueues the owning Instance of this provider
// whenever one of its bindings changes (spec or status: Accepted flips matter).
func (r *ProviderReconciler) watchMonitoringBindings(b *builder.Builder) {
	b.Watches(&monitoringv1alpha1.MonitoringBinding{},
		handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
			instanceName := obj.GetLabels()[monitoringv1alpha1.BindingInstanceLabel]
			if instanceName == "" {
				return nil
			}
			in := &v1alpha1.Instance{}
			key := types.NamespacedName{Namespace: obj.GetNamespace(), Name: instanceName}
			if err := r.Client.Get(ctx, key, in); err != nil || in.Spec.ProviderRef.Name != r.provider.Name() {
				return nil
			}
			return []reconcile.Request{{NamespacedName: key}}
		}),
		builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
	)
}

// reconcileMonitoring fills Instance.status.monitoring (sources and destinations) and
// the MonitoringConfigured condition after a successful Sync, and writes the
// Configured condition of the ProviderManaged bindings this provider rendered.
// Failures are logged and reported on the condition; they never fail the
// reconcile: the engine itself is healthy.
func (r *ProviderReconciler) reconcileMonitoring(ctx context.Context, syncCtx *controller.Context, in *v1alpha1.Instance) {
	logger := log.FromContext(ctx).WithName("monitoring")

	var entries []v1alpha1.InstanceMonitoringDestination
	if in.Spec.Monitoring != nil {
		entries = in.Spec.Monitoring.Destinations
	}

	if !monitoringEnabled(r.provider) {
		in.Status.Monitoring = nil
		if len(entries) == 0 {
			meta.RemoveStatusCondition(&in.Status.Conditions, v1alpha1.ConditionMonitoringConfigured)
			return
		}
		// The provider implements no monitoring interface, so no class can be
		// fulfilled; say so instead of leaving the entries silently empty.
		in.Status.Monitoring = &v1alpha1.InstanceMonitoringStatus{}
		for _, e := range entries {
			in.Status.Monitoring.Destinations = append(in.Status.Monitoring.Destinations, v1alpha1.InstanceMonitoringDestinationStatus{
				Name:       e.Name,
				Configured: metav1.ConditionFalse,
				Reason:     v1alpha1.ReasonMonitoringIntegrationUnsupported,
				Message:    fmt.Sprintf("provider %q does not implement the monitoring contract", r.provider.Name()),
			})
		}
		setCondition(in, v1alpha1.ConditionMonitoringConfigured, metav1.ConditionFalse,
			v1alpha1.ReasonMonitoringIntegrationUnsupported,
			fmt.Sprintf("provider %q does not implement the monitoring contract", r.provider.Name()), metav1.Now())
		return
	}

	var sources *v1alpha1.MonitoringSources
	if in.Status.Monitoring != nil {
		sources = in.Status.Monitoring.Sources
	}
	if sp, ok := r.provider.(controller.MonitoringSourcesProvider); ok {
		computed, err := sp.MonitoringSources(syncCtx)
		if err != nil {
			logger.Error(err, "MonitoringSources failed; keeping the previous status.monitoring.sources")
		} else {
			sources = r.validateSources(ctx, syncCtx, in, computed)
		}
	}
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

	var notConfigured []string
	anyFalse := false
	for _, entry := range entries {
		st := v1alpha1.InstanceMonitoringDestinationStatus{Name: entry.Name}
		rb, ok := byEntry[entry.Name]
		switch {
		case !ok:
			st.Configured, st.Reason, st.Message = metav1.ConditionUnknown, string(monitoringv1alpha1.ReasonPending), "binding not yet materialised"
		default:
			st.Mode = rb.Mode()
			accepted := meta.FindStatusCondition(rb.Binding.Status.Conditions, monitoringv1alpha1.MonitoringBindingConditionAccepted)
			switch {
			case accepted == nil || accepted.Status == metav1.ConditionUnknown:
				st.Configured, st.Reason, st.Message = metav1.ConditionUnknown, string(monitoringv1alpha1.ReasonPending), "waiting for core to accept the binding"
			case accepted.Status == metav1.ConditionFalse:
				st.Configured, st.Reason, st.Message = metav1.ConditionFalse, accepted.Reason, accepted.Message
				if rb.Mode() == v1alpha1.MonitoringExecutionModeProviderManaged {
					r.retainBinding(ctx, rb.Binding, accepted, logger)
				}
			case rb.Mode() == v1alpha1.MonitoringExecutionModeProviderManaged:
				cond := r.configureBinding(ctx, syncCtx, rb.Binding, logger)
				st.Configured, st.Reason, st.Message = cond.Status, cond.Reason, cond.Message
			default:
				configured := meta.FindStatusCondition(rb.Binding.Status.Conditions, monitoringv1alpha1.MonitoringBindingConditionConfigured)
				st.Configured, st.Reason, st.Message = metav1.ConditionUnknown, string(monitoringv1alpha1.ReasonPending), "waiting for the class controller"
				if configured != nil {
					st.Configured, st.Reason, st.Message = configured.Status, configured.Reason, configured.Message
				}
			}
		}
		if st.Configured != metav1.ConditionTrue {
			notConfigured = append(notConfigured, entry.Name+": "+st.Reason)
		}
		if st.Configured == metav1.ConditionFalse {
			anyFalse = true
		}
		in.Status.Monitoring.Destinations = append(in.Status.Monitoring.Destinations, st)
	}

	switch {
	case len(notConfigured) == 0:
		setCondition(in, v1alpha1.ConditionMonitoringConfigured, metav1.ConditionTrue,
			v1alpha1.ReasonMonitoringConfigured, "all monitoring destinations are configured", metav1.Now())
	case anyFalse:
		setCondition(in, v1alpha1.ConditionMonitoringConfigured, metav1.ConditionFalse,
			v1alpha1.ReasonMonitoringNotConfigured, strings.Join(notConfigured, "; "), metav1.Now())
	default:
		setCondition(in, v1alpha1.ConditionMonitoringConfigured, metav1.ConditionUnknown,
			v1alpha1.ReasonMonitoringPending, strings.Join(notConfigured, "; "), metav1.Now())
	}
}

// configureBinding writes the Configured condition of an Accepted
// ProviderManaged binding from the Sync outcome and any staged result.
func (r *ProviderReconciler) configureBinding(ctx context.Context, syncCtx *controller.Context, b *monitoringv1alpha1.MonitoringBinding, logger interface{ Error(error, string, ...any) }) metav1.Condition {
	reason, msg := monitoringv1alpha1.ReasonConfigured, "rendered by provider "+r.provider.Name()
	if staged, stagedMsg, ok := syncCtx.MonitoringBindingResult(b.Name); ok {
		reason, msg = staged, stagedMsg
	}
	cond := monitoring.NewCondition(b, monitoringv1alpha1.MonitoringBindingConditionConfigured, reason.Status(), string(reason), msg)
	if err := monitoring.ApplyCondition(ctx, r.Client, b, b.Status.Conditions, r.configuredFieldOwner(), cond); err != nil {
		logger.Error(err, "Failed to write Configured condition", "binding", b.Name)
	}
	return cond
}

// retainBinding reports a ProviderManaged binding that lost Accepted:
// Retained when a rendering is live and kept, NotAccepted when nothing was
// ever rendered. Both quote the Accepted reason.
func (r *ProviderReconciler) retainBinding(ctx context.Context, b *monitoringv1alpha1.MonitoringBinding, accepted *metav1.Condition, logger interface{ Error(error, string, ...any) }) {
	prev := meta.FindStatusCondition(b.Status.Conditions, monitoringv1alpha1.MonitoringBindingConditionConfigured)
	reason := monitoringv1alpha1.ReasonNotAccepted
	msg := "binding is not Accepted (" + accepted.Reason + "); nothing rendered"
	if prev != nil && (prev.Status == metav1.ConditionTrue || prev.Reason == string(monitoringv1alpha1.ReasonRetained)) {
		reason = monitoringv1alpha1.ReasonRetained
		msg = "binding is no longer Accepted (" + accepted.Reason + "); keeping the last-applied rendering"
	}
	cond := monitoring.NewCondition(b, monitoringv1alpha1.MonitoringBindingConditionConfigured, reason.Status(), string(reason), msg)
	if err := monitoring.ApplyCondition(ctx, r.Client, b, b.Status.Conditions, r.configuredFieldOwner(), cond); err != nil {
		logger.Error(err, "Failed to write Configured condition", "binding", b.Name)
	}
}

// validateSources enforces the runtime invariants on the provider-computed
// sources: only declared component types, pod selectors carrying the
// Instance label, identity labels stamped by the runtime.
func (r *ProviderReconciler) validateSources(ctx context.Context, syncCtx *controller.Context, in *v1alpha1.Instance, t *v1alpha1.MonitoringSources) *v1alpha1.MonitoringSources {
	if t == nil {
		return nil
	}
	logger := log.FromContext(ctx).WithName("monitoring")
	out := t.DeepCopy()

	var declared map[string]v1alpha1.MonitoringComponentType
	var componentTypes map[string]v1alpha1.Component
	if spec, err := syncCtx.ProviderSpec(); err == nil && spec.Monitoring != nil {
		declared = spec.Monitoring.ComponentTypes
		componentTypes = spec.Components
	}
	out.Metrics = out.Metrics[:0]
	for _, ep := range t.Metrics {
		ctype := in.Spec.Components[ep.Component].Type
		if ctype == "" {
			ctype = componentTypes[ep.Component].Type
		}
		if _, ok := declared[ctype]; !ok {
			logger.Info("Dropping metrics endpoint for undeclared component type", "component", ep.Component, "type", ctype)
			continue
		}
		if len(ep.PodSelector) == 0 {
			logger.Info("Dropping metrics endpoint without pod selector", "component", ep.Component)
			continue
		}
		out.Metrics = append(out.Metrics, ep)
	}

	if out.Identity.Labels == nil {
		out.Identity.Labels = map[string]string{}
	}
	out.Identity.Labels[identityLabelInstance] = in.Name
	out.Identity.Labels[identityLabelNamespace] = in.Namespace
	out.Identity.Labels[identityLabelProvider] = r.provider.Name()
	if out.Identity.ClusterID == "" {
		// Uncached read: a single get needs no Namespace informer (and no
		// list/watch RBAC) in the provider.
		ns := &corev1.Namespace{}
		if err := r.manager.GetAPIReader().Get(ctx, types.NamespacedName{Name: "kube-system"}, ns); err == nil {
			out.Identity.ClusterID = string(ns.UID)
		}
	}
	if out.Identity.ClusterID != "" {
		out.Identity.Labels[identityLabelCluster] = out.Identity.ClusterID
	}
	return out
}
