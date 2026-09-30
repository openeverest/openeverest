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

package monitoring

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/blang/semver/v4"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
	rtmonitoring "github.com/openeverest/openeverest/v2/provider-runtime/monitoring"
)

// InstanceMonitoringDestinationField indexes Instances by every
// spec.monitoring.destinations[].destinationRef.name.
const InstanceMonitoringDestinationField = ".spec.monitoring.destinations.destinationRef.name"

// instanceMonitoringDestinationNames is the extractor for
// InstanceMonitoringDestinationField.
func instanceMonitoringDestinationNames(obj client.Object) []string {
	in, ok := obj.(*corev1alpha1.Instance)
	if !ok || in.Spec.Monitoring == nil {
		return nil
	}
	names := make([]string, 0, len(in.Spec.Monitoring.Destinations))
	for _, e := range in.Spec.Monitoring.Destinations {
		names = append(names, e.DestinationRef.Name)
	}
	return names
}

// instanceMonitoringDestinations returns the Instance's destination entries.
func instanceMonitoringDestinations(in *corev1alpha1.Instance) []corev1alpha1.InstanceMonitoringDestination {
	if in.Spec.Monitoring == nil {
		return nil
	}
	return in.Spec.Monitoring.Destinations
}

// MonitoringBindingReconciler materialises one MonitoringBinding per
// Instance.spec.monitoring.destinations[] entry and owns the bindings' spec and Accepted
// condition. It never touches Instance status.
type MonitoringBindingReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager. The Instance index
// on spec.monitoring.destinations[].destinationRef.name is registered by
// MonitoringDestinationReconciler, which must be set up first.
func (r *MonitoringBindingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("MonitoringBinding").
		For(&corev1alpha1.Instance{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&monitoringv1alpha1.MonitoringBinding{}).
		Watches(&monitoringv1alpha1.MonitoringDestination{},
			handler.EnqueueRequestsFromMapFunc(r.enqueueInstancesForDestination)).
		Watches(&monitoringv1alpha1.MonitoringClass{},
			handler.EnqueueRequestsFromMapFunc(r.enqueueInstancesForClass)).
		Watches(&corev1alpha1.Provider{},
			handler.EnqueueRequestsFromMapFunc(r.enqueueInstancesForProvider),
			builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Complete(r)
}

// +kubebuilder:rbac:groups=monitoring.openeverest.io,resources=monitoringbindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=monitoring.openeverest.io,resources=monitoringbindings/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=monitoring.openeverest.io,resources=monitoringclasses,verbs=get;list;watch
// +kubebuilder:rbac:groups=core.openeverest.io,resources=providers,verbs=get;list;watch

// Reconcile materialises the bindings of one Instance.
func (r *MonitoringBindingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithName("MonitoringBindingReconciler")

	in := &corev1alpha1.Instance{}
	if err := r.Get(ctx, req.NamespacedName, in); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if !in.DeletionTimestamp.IsZero() {
		// Bindings are owned by the Instance and garbage-collected with it.
		return ctrl.Result{}, nil
	}

	existing, err := rtmonitoring.ListBindingsForInstance(ctx, r.Client, in.Namespace, in.Name)
	if err != nil {
		return ctrl.Result{}, err
	}
	byEntry := make(map[string]*monitoringv1alpha1.MonitoringBinding, len(existing))
	for i := range existing {
		byEntry[existing[i].Labels[monitoringv1alpha1.BindingEntryLabel]] = &existing[i]
	}

	// ProviderManaged integrations already claimed by an earlier entry.
	usedIntegrations := map[string]string{}
	var errs []error
	for _, entry := range instanceMonitoringDestinations(in) {
		res := r.resolve(ctx, in, entry, byEntry[entry.Name], usedIntegrations)
		if err := r.materialise(ctx, in, entry, byEntry[entry.Name], res); err != nil {
			errs = append(errs, fmt.Errorf("entry %q: %w", entry.Name, err))
		}
		delete(byEntry, entry.Name)
	}

	// Whatever is left no longer has a spec entry.
	for entry, b := range byEntry {
		if err := r.Delete(ctx, b); client.IgnoreNotFound(err) != nil {
			errs = append(errs, fmt.Errorf("delete stale binding for entry %q: %w", entry, err))
		} else {
			logger.Info("Deleted stale binding", "binding", b.Name, "entry", entry)
		}
	}

	return ctrl.Result{}, errors.Join(errs...)
}

// resolution is the outcome of resolving one entry.
type resolution struct {
	classRef      *common.ObjectRef
	executionMode corev1alpha1.MonitoringExecutionMode
	accepted      metav1.Condition
}

// resolve computes the class, execution mode and Accepted condition for an
// entry without writing anything. The mode already recorded on an existing
// binding is authoritative: it never changes on its own.
func (r *MonitoringBindingReconciler) resolve(
	ctx context.Context,
	in *corev1alpha1.Instance,
	entry corev1alpha1.InstanceMonitoringDestination,
	existing *monitoringv1alpha1.MonitoringBinding,
	usedIntegrations map[string]string,
) resolution {
	reject := func(reason, msg string) resolution {
		res := resolution{accepted: metav1.Condition{
			Type: monitoringv1alpha1.MonitoringBindingConditionAccepted, Status: metav1.ConditionFalse, Reason: reason, Message: msg,
		}}
		if existing != nil {
			res.classRef = existing.Spec.ClassRef
			res.executionMode = existing.Spec.ExecutionMode
		}
		return res
	}

	dst := &monitoringv1alpha1.MonitoringDestination{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: in.Namespace, Name: entry.DestinationRef.Name}, dst); err != nil {
		if apierrors.IsNotFound(err) {
			return reject(monitoringv1alpha1.ReasonDestinationNotFound, fmt.Sprintf("MonitoringDestination %q not found", entry.DestinationRef.Name))
		}
		return reject(monitoringv1alpha1.ReasonDestinationNotFound, err.Error())
	}
	if existing != nil && existing.Spec.ClassRef != nil && existing.Spec.ClassRef.Name != dst.Spec.ClassRef.Name {
		return reject(monitoringv1alpha1.ReasonIncompatible, fmt.Sprintf("binding was created for class %q but the destination now references %q", existing.Spec.ClassRef.Name, dst.Spec.ClassRef.Name))
	}

	class := &monitoringv1alpha1.MonitoringClass{}
	if err := r.Get(ctx, types.NamespacedName{Name: dst.Spec.ClassRef.Name}, class); err != nil {
		if apierrors.IsNotFound(err) {
			return reject(monitoringv1alpha1.ReasonClassNotFound, fmt.Sprintf("MonitoringClass %q not found", dst.Spec.ClassRef.Name))
		}
		return reject(monitoringv1alpha1.ReasonClassNotFound, err.Error())
	}
	classRef := &common.ObjectRef{Name: class.Name}
	if !rtmonitoring.ClassIsAccepted(class) {
		res := reject(monitoringv1alpha1.ReasonClassNotAccepted, fmt.Sprintf("no controller has claimed MonitoringClass %q (controllerName %q)", class.Name, class.Spec.ControllerName))
		res.classRef = classRef
		return res
	}

	provider := &corev1alpha1.Provider{}
	if err := r.Get(ctx, types.NamespacedName{Name: in.Spec.ProviderRef.Name}, provider); err != nil {
		res := reject(monitoringv1alpha1.ReasonProviderNotFound, fmt.Sprintf("Provider %q not found", in.Spec.ProviderRef.Name))
		res.classRef = classRef
		return res
	}

	if class.Spec.InstanceParametersSchema != nil || entry.Parameters != nil {
		if err := class.Spec.InstanceParametersSchema.Validate(entry.Parameters); err != nil {
			res := reject(monitoringv1alpha1.ReasonInvalidParameters, fmt.Sprintf("parameters rejected by MonitoringClass %q instanceParametersSchema: %v", class.Name, err))
			res.classRef = classRef
			return res
		}
	}

	mode, reason, msg := resolveMode(in, entry, existing, provider, class, usedIntegrations)
	if reason != "" {
		res := reject(reason, msg)
		res.classRef = classRef
		return res
	}
	if mode == corev1alpha1.MonitoringExecutionModeProviderManaged {
		usedIntegrations[class.Spec.ProviderManaged.Integration] = entry.Name
	}

	return resolution{
		classRef:      classRef,
		executionMode: mode,
		accepted: metav1.Condition{
			Type: monitoringv1alpha1.MonitoringBindingConditionAccepted, Status: metav1.ConditionTrue,
			Reason: monitoringv1alpha1.ReasonAccepted, Message: fmt.Sprintf("resolved to %s", mode),
		},
	}
}

// resolveMode picks the execution mode: the mode recorded on the binding or
// pinned on the entry wins and is re-checked for feasibility; otherwise
// ProviderManaged is preferred over ExtensionManaged.
func resolveMode(
	in *corev1alpha1.Instance,
	entry corev1alpha1.InstanceMonitoringDestination,
	existing *monitoringv1alpha1.MonitoringBinding,
	provider *corev1alpha1.Provider,
	class *monitoringv1alpha1.MonitoringClass,
	usedIntegrations map[string]string,
) (corev1alpha1.MonitoringExecutionMode, string, string) {
	pmOK, pmReason, pmMsg := providerManagedFeasible(provider, class, usedIntegrations)
	genOK, genMsg := extensionManagedFeasible(in, provider, class)

	want := entry.ExecutionMode
	if existing != nil && existing.Spec.ExecutionMode != "" {
		want = existing.Spec.ExecutionMode
	}
	switch want {
	case corev1alpha1.MonitoringExecutionModeProviderManaged:
		if !pmOK {
			return "", pmReason, pmMsg
		}
		return want, "", ""
	case corev1alpha1.MonitoringExecutionModeExtensionManaged:
		if !genOK {
			return "", monitoringv1alpha1.ReasonIncompatible, genMsg
		}
		return want, "", ""
	}
	if pmOK {
		return corev1alpha1.MonitoringExecutionModeProviderManaged, "", ""
	}
	if genOK {
		return corev1alpha1.MonitoringExecutionModeExtensionManaged, "", ""
	}
	reason := monitoringv1alpha1.ReasonIncompatible
	if pmReason == monitoringv1alpha1.ReasonIntegrationInUse {
		reason = pmReason
	}
	return "", reason, fmt.Sprintf("providerManaged: %s; extensionManaged: %s", pmMsg, genMsg)
}

func providerManagedFeasible(provider *corev1alpha1.Provider, class *monitoringv1alpha1.MonitoringClass, usedIntegrations map[string]string) (bool, string, string) {
	pm := class.Spec.ProviderManaged
	if pm == nil {
		return false, monitoringv1alpha1.ReasonIncompatible, "class declares no providerManaged block"
	}
	if provider.Spec.Monitoring == nil {
		return false, monitoringv1alpha1.ReasonIntegrationUnsupported, fmt.Sprintf("Provider %q declares no monitoring contract", provider.Name)
	}
	integration, ok := provider.Spec.Monitoring.Integrations[pm.Integration]
	if !ok {
		return false, monitoringv1alpha1.ReasonIntegrationUnsupported, fmt.Sprintf("Provider %q does not declare integration %q", provider.Name, pm.Integration)
	}
	if by, used := usedIntegrations[pm.Integration]; used {
		return false, monitoringv1alpha1.ReasonIntegrationInUse, fmt.Sprintf("integration %q is already fulfilled by entry %q", pm.Integration, by)
	}
	if integration.AgentVersions != "" && pm.AgentVersion != "" {
		rng, err := semver.ParseRange(integration.AgentVersions)
		if err != nil {
			return false, monitoringv1alpha1.ReasonIncompatible, fmt.Sprintf("Provider %q agentVersions %q: %v", provider.Name, integration.AgentVersions, err)
		}
		v, err := semver.ParseTolerant(pm.AgentVersion)
		if err != nil {
			return false, monitoringv1alpha1.ReasonIncompatible, fmt.Sprintf("class agentVersion %q: %v", pm.AgentVersion, err)
		}
		if !rng(v) {
			return false, monitoringv1alpha1.ReasonIncompatible, fmt.Sprintf("class agentVersion %s is outside the Provider's supported range %q", pm.AgentVersion, integration.AgentVersions)
		}
	}
	return true, "", ""
}

// extensionManagedFeasible reports whether at least one Instance component is
// eligible for the class's extensionManaged block, using only static data.
func extensionManagedFeasible(in *corev1alpha1.Instance, provider *corev1alpha1.Provider, class *monitoringv1alpha1.MonitoringClass) (bool, string) {
	gen := class.Spec.ExtensionManaged
	if gen == nil {
		return false, "class declares no extensionManaged block"
	}
	if provider.Spec.Monitoring == nil {
		return false, fmt.Sprintf("Provider %q declares no monitoring contract", provider.Name)
	}
	for name, comp := range in.Spec.Components {
		ctype := comp.Type
		if ctype == "" {
			ctype = provider.Spec.Components[name].Type
		}
		tt, ok := provider.Spec.Monitoring.ComponentTypes[ctype]
		if !ok {
			continue
		}
		if componentEligible(tt, gen.Requires) {
			return true, ""
		}
	}
	return false, "no component satisfies the class requirements"
}

func componentEligible(tt corev1alpha1.MonitoringComponentType, req monitoringv1alpha1.SourceRequirements) bool {
	if len(req.Kinds) > 0 && !slices.Contains(req.Kinds, tt.Kind) {
		return false
	}
	if req.Metrics && (tt.Metrics == "" || tt.Metrics == corev1alpha1.MetricsSourceNone) {
		return false
	}
	for _, p := range req.CredentialProfiles {
		if !slices.Contains(tt.CredentialProfiles, p) {
			return false
		}
	}
	for _, f := range req.Features {
		if !slices.Contains(tt.Features, f) {
			return false
		}
	}
	return true
}

// materialise creates or updates the binding for an entry and applies the
// Accepted condition.
func (r *MonitoringBindingReconciler) materialise(
	ctx context.Context,
	in *corev1alpha1.Instance,
	entry corev1alpha1.InstanceMonitoringDestination,
	existing *monitoringv1alpha1.MonitoringBinding,
	res resolution,
) error {
	className := ""
	if res.classRef != nil {
		className = res.classRef.Name
	}
	desiredLabels := rtmonitoring.BindingLabels(in.Name, entry.Name, className)
	destinationRef := entry.DestinationRef
	if destinationRef.Kind == "" {
		destinationRef.Kind = "MonitoringDestination"
	}

	b := existing
	if b == nil {
		b = &monitoringv1alpha1.MonitoringBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      rtmonitoring.BindingName(in.Name, entry.Name),
				Namespace: in.Namespace,
				Labels:    desiredLabels,
			},
			Spec: monitoringv1alpha1.MonitoringBindingSpec{
				InstanceRef:    common.ObjectRef{Name: in.Name},
				DestinationRef: destinationRef,
				ClassRef:       res.classRef,
				ExecutionMode:  res.executionMode,
				Parameters:     entry.Parameters,
			},
		}
		if err := controllerutil.SetControllerReference(in, b, r.Scheme); err != nil {
			return fmt.Errorf("set owner: %w", err)
		}
		if err := r.Create(ctx, b); err != nil {
			return fmt.Errorf("create binding: %w", err)
		}
	} else {
		changed := false
		if b.Spec.ClassRef == nil && res.classRef != nil {
			b.Spec.ClassRef = res.classRef
			changed = true
		}
		if b.Spec.ExecutionMode == "" && res.executionMode != "" {
			b.Spec.ExecutionMode = res.executionMode
			changed = true
		}
		if !rawEqual(b.Spec.Parameters, entry.Parameters) {
			b.Spec.Parameters = entry.Parameters
			changed = true
		}
		if !maps.Equal(b.Labels, desiredLabels) && className != "" {
			b.Labels = desiredLabels
			changed = true
		}
		if changed {
			if err := r.Update(ctx, b); err != nil {
				return fmt.Errorf("update binding: %w", err)
			}
		}
	}

	cond := res.accepted
	cond.ObservedGeneration = b.Generation
	return rtmonitoring.ApplyCondition(ctx, r.Client, b, b.Status.Conditions, rtmonitoring.MaterializerFieldOwner, cond)
}

func rawEqual(a, b *runtime.RawExtension) bool {
	switch {
	case a == nil && b == nil:
		return true
	case a == nil || b == nil:
		return false
	}
	return string(a.Raw) == string(b.Raw)
}

func (r *MonitoringBindingReconciler) enqueueInstancesForDestination(ctx context.Context, obj client.Object) []reconcile.Request {
	instances := &corev1alpha1.InstanceList{}
	if err := r.List(ctx, instances,
		client.InNamespace(obj.GetNamespace()),
		client.MatchingFieldsSelector{Selector: fields.OneTermEqualSelector(InstanceMonitoringDestinationField, obj.GetName())},
	); err != nil {
		return nil
	}
	return requestsFor(instances)
}

func (r *MonitoringBindingReconciler) enqueueInstancesForClass(ctx context.Context, obj client.Object) []reconcile.Request {
	destinations := &monitoringv1alpha1.MonitoringDestinationList{}
	if err := r.List(ctx, destinations); err != nil {
		return nil
	}
	var reqs []reconcile.Request
	for i := range destinations.Items {
		dst := &destinations.Items[i]
		if dst.Spec.ClassRef.Name != obj.GetName() {
			continue
		}
		reqs = append(reqs, r.enqueueInstancesForDestination(ctx, dst)...)
	}
	return reqs
}

func (r *MonitoringBindingReconciler) enqueueInstancesForProvider(ctx context.Context, obj client.Object) []reconcile.Request {
	instances := &corev1alpha1.InstanceList{}
	if err := r.List(ctx, instances); err != nil {
		return nil
	}
	var reqs []reconcile.Request
	for i := range instances.Items {
		in := &instances.Items[i]
		if in.Spec.ProviderRef.Name == obj.GetName() && len(instanceMonitoringDestinations(in)) > 0 {
			reqs = append(reqs, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(in)})
		}
	}
	return reqs
}

func requestsFor(list *corev1alpha1.InstanceList) []reconcile.Request {
	reqs := make([]reconcile.Request, 0, len(list.Items))
	for i := range list.Items {
		reqs = append(reqs, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(&list.Items[i])})
	}
	return reqs
}
