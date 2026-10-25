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
	"bytes"
	"context"
	"errors"
	"fmt"

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

const destinationKind = "MonitoringDestination"

// instanceMonitoringDestinationNames is the extractor for
// InstanceMonitoringDestinationField.
func instanceMonitoringDestinationNames(obj client.Object) []string {
	in, ok := obj.(*corev1alpha1.Instance)
	if !ok {
		return nil
	}
	entries := instanceMonitoringDestinations(in)
	names := make([]string, 0, len(entries))
	for _, e := range entries {
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

// BindingReconciler materialises one MonitoringBinding per
// Instance.spec.monitoring.destinations[] entry and owns the bindings' spec
// and Accepted condition. It never touches Instance status.
type BindingReconciler struct {
	client.Client

	Scheme *runtime.Scheme
}

// SetupWithManager registers the Instance index on
// spec.monitoring.destinations[].destinationRef.name and sets up the
// controller.
func (r *BindingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1alpha1.Instance{},
		InstanceMonitoringDestinationField, instanceMonitoringDestinationNames); err != nil {
		return fmt.Errorf("indexing instance by monitoring destination: %w", err)
	}
	return ctrl.NewControllerManagedBy(mgr).
		Named("MonitoringBinding").
		For(&corev1alpha1.Instance{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Owns(&monitoringv1alpha1.MonitoringBinding{}).
		Watches(&monitoringv1alpha1.MonitoringDestination{},
			handler.EnqueueRequestsFromMapFunc(r.enqueueInstancesForDestination),
			builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Watches(&monitoringv1alpha1.MonitoringClass{},
			handler.EnqueueRequestsFromMapFunc(r.enqueueInstancesForClass)).
		Watches(&corev1alpha1.Provider{},
			handler.EnqueueRequestsFromMapFunc(r.enqueueInstancesForProvider),
			builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Complete(r)
}

// +kubebuilder:rbac:groups=monitoring.openeverest.io,resources=monitoringbindings,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups=monitoring.openeverest.io,resources=monitoringbindings/status,verbs=patch
// +kubebuilder:rbac:groups=monitoring.openeverest.io,resources=monitoringdestinations,verbs=get;list;watch
// +kubebuilder:rbac:groups=monitoring.openeverest.io,resources=monitoringclasses,verbs=get;list;watch
// +kubebuilder:rbac:groups=core.openeverest.io,resources=instances,verbs=get;list;watch
// +kubebuilder:rbac:groups=core.openeverest.io,resources=providers,verbs=get;list;watch

// Reconcile materialises the bindings of one Instance.
func (r *BindingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithName("MonitoringBindingReconciler")

	in := &corev1alpha1.Instance{}
	if err := r.Get(ctx, req.NamespacedName, in); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	if !in.DeletionTimestamp.IsZero() {
		// Bindings are owned by the Instance and garbage-collected with it.
		return ctrl.Result{}, nil
	}

	existing, err := rtmonitoring.ListBindingsForInstance(ctx, r.Client, in)
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
	for _, entry := range entriesByPrecedence(instanceMonitoringDestinations(in), byEntry) {
		b := byEntry[entry.Name]
		delete(byEntry, entry.Name)
		if err := r.reconcileEntry(ctx, in, entry, b, usedIntegrations); err != nil {
			errs = append(errs, fmt.Errorf("entry %q: %w", entry.Name, err))
		}
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

// entriesByPrecedence puts entries already bound ProviderManaged first, so a
// new entry for the same integration cannot take it from them.
func entriesByPrecedence(
	entries []corev1alpha1.InstanceMonitoringDestination,
	byEntry map[string]*monitoringv1alpha1.MonitoringBinding,
) []corev1alpha1.InstanceMonitoringDestination {
	ordered := make([]corev1alpha1.InstanceMonitoringDestination, 0, len(entries))
	var rest []corev1alpha1.InstanceMonitoringDestination
	for _, e := range entries {
		if b := byEntry[e.Name]; b != nil && b.Spec.ExecutionMode == corev1alpha1.MonitoringExecutionModeProviderManaged {
			ordered = append(ordered, e)
		} else {
			rest = append(rest, e)
		}
	}
	return append(ordered, rest...)
}

func (r *BindingReconciler) reconcileEntry(
	ctx context.Context,
	in *corev1alpha1.Instance,
	entry corev1alpha1.InstanceMonitoringDestination,
	existing *monitoringv1alpha1.MonitoringBinding,
	usedIntegrations map[string]string,
) error {
	if existing != nil && !matchesEntry(existing, entry) {
		if err := r.Delete(ctx, existing); client.IgnoreNotFound(err) != nil {
			return fmt.Errorf("delete binding to recreate it: %w", err)
		}
		log.FromContext(ctx).Info("Deleted binding to recreate it for the changed entry", "binding", existing.Name, "entry", entry.Name)
		existing = nil
	}
	res, err := r.resolve(ctx, in, entry, existing, usedIntegrations)
	if err != nil {
		return err
	}
	return r.materialise(ctx, in, entry, existing, res)
}

// matchesEntry reports whether b still serves entry. A binding's
// destinationRef and executionMode are immutable, so an entry that points
// elsewhere or pins another mode gets a new binding.
func matchesEntry(b *monitoringv1alpha1.MonitoringBinding, entry corev1alpha1.InstanceMonitoringDestination) bool {
	if b.Spec.DestinationRef != destinationRefOf(entry) {
		return false
	}
	return entry.ExecutionMode == "" || b.Spec.ExecutionMode == "" || entry.ExecutionMode == b.Spec.ExecutionMode
}

func destinationRefOf(entry corev1alpha1.InstanceMonitoringDestination) common.MonitoringDestinationRef {
	ref := entry.DestinationRef
	if ref.Kind == "" {
		ref.Kind = destinationKind
	}
	return ref
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
func (r *BindingReconciler) resolve(
	ctx context.Context,
	in *corev1alpha1.Instance,
	entry corev1alpha1.InstanceMonitoringDestination,
	existing *monitoringv1alpha1.MonitoringBinding,
	usedIntegrations map[string]string,
) (resolution, error) {
	class, rejected, err := r.resolveClass(ctx, in, entry, existing)
	if err != nil || class == nil {
		return rejected, err
	}
	classRef := &common.ObjectRef{Name: class.Name}

	provider := &corev1alpha1.Provider{}
	if err := r.Get(ctx, types.NamespacedName{Name: in.Spec.ProviderRef.Name}, provider); err != nil {
		if !apierrors.IsNotFound(err) {
			return resolution{}, fmt.Errorf("get Provider %q: %w", in.Spec.ProviderRef.Name, err)
		}
		return rejection(existing, classRef, monitoringv1alpha1.ReasonProviderNotFound,
			fmt.Sprintf("Provider %q not found", in.Spec.ProviderRef.Name)), nil
	}

	if err := class.Spec.InstanceParametersSchema.Validate(entry.Parameters); err != nil {
		return rejection(existing, classRef, monitoringv1alpha1.ReasonInvalidParameters,
			fmt.Sprintf("parameters rejected by the instanceParametersSchema of MonitoringClass %q: %v", class.Name, err)), nil
	}

	f := assess(provider, class, instanceComponentTypes(in, provider))
	if f.providerManaged.ok {
		if by, used := usedIntegrations[class.Spec.ProviderManaged.Integration]; used {
			f.providerManaged = verdict{
				reason:  monitoringv1alpha1.ReasonIntegrationInUse,
				message: fmt.Sprintf("integration %q is already fulfilled by entry %q", class.Spec.ProviderManaged.Integration, by),
			}
		}
	}
	mode, reason, msg := resolveMode(f, entry, existing)
	if reason != "" {
		return rejection(existing, classRef, reason, msg), nil
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
	}, nil
}

// resolveClass follows entry -> destination -> class and returns the
// claimed class, or nil and the rejection naming the missing or unusable
// link.
func (r *BindingReconciler) resolveClass(
	ctx context.Context,
	in *corev1alpha1.Instance,
	entry corev1alpha1.InstanceMonitoringDestination,
	existing *monitoringv1alpha1.MonitoringBinding,
) (*monitoringv1alpha1.MonitoringClass, resolution, error) {
	dst := &monitoringv1alpha1.MonitoringDestination{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: in.Namespace, Name: entry.DestinationRef.Name}, dst); err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, resolution{}, fmt.Errorf("get MonitoringDestination %q: %w", entry.DestinationRef.Name, err)
		}
		return nil, rejection(existing, nil, monitoringv1alpha1.ReasonDestinationNotFound,
			fmt.Sprintf("MonitoringDestination %q not found", entry.DestinationRef.Name)), nil
	}
	if existing != nil && existing.Spec.ClassRef != nil && existing.Spec.ClassRef.Name != dst.Spec.ClassRef.Name {
		return nil, rejection(existing, nil, monitoringv1alpha1.ReasonIncompatible,
			fmt.Sprintf("binding was created for MonitoringClass %q but MonitoringDestination %q now references %q",
				existing.Spec.ClassRef.Name, dst.Name, dst.Spec.ClassRef.Name)), nil
	}

	class := &monitoringv1alpha1.MonitoringClass{}
	if err := r.Get(ctx, types.NamespacedName{Name: dst.Spec.ClassRef.Name}, class); err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, resolution{}, fmt.Errorf("get MonitoringClass %q: %w", dst.Spec.ClassRef.Name, err)
		}
		return nil, rejection(existing, nil, monitoringv1alpha1.ReasonClassNotFound,
			fmt.Sprintf("MonitoringClass %q not found", dst.Spec.ClassRef.Name)), nil
	}
	if !rtmonitoring.ClassIsAccepted(class) {
		return nil, rejection(existing, &common.ObjectRef{Name: class.Name}, monitoringv1alpha1.ReasonClassNotAccepted,
			fmt.Sprintf("no controller has claimed MonitoringClass %q (controllerName %q)", class.Name, class.Spec.ControllerName)), nil
	}
	return class, resolution{}, nil
}

// rejection is an Accepted=False resolution. classRef, when known, is
// recorded so the binding points at its class; the class and mode the
// existing binding already carries are always kept.
func rejection(existing *monitoringv1alpha1.MonitoringBinding, classRef *common.ObjectRef, reason, msg string) resolution {
	res := resolution{
		classRef: classRef,
		accepted: metav1.Condition{
			Type: monitoringv1alpha1.MonitoringBindingConditionAccepted, Status: metav1.ConditionFalse, Reason: reason, Message: msg,
		},
	}
	if existing != nil {
		res.executionMode = existing.Spec.ExecutionMode
		if existing.Spec.ClassRef != nil {
			res.classRef = existing.Spec.ClassRef
		}
	}
	return res
}

// resolveMode picks the execution mode: the mode recorded on the binding or
// pinned on the entry wins and is re-checked for feasibility; otherwise
// ProviderManaged is preferred over ExtensionManaged.
func resolveMode(
	f fulfilment,
	entry corev1alpha1.InstanceMonitoringDestination,
	existing *monitoringv1alpha1.MonitoringBinding,
) (corev1alpha1.MonitoringExecutionMode, string, string) {
	want := entry.ExecutionMode
	if existing != nil && existing.Spec.ExecutionMode != "" {
		want = existing.Spec.ExecutionMode
	}
	switch want {
	case corev1alpha1.MonitoringExecutionModeProviderManaged:
		if !f.providerManaged.ok {
			return "", f.providerManaged.reason, f.providerManaged.message
		}
		return want, "", ""
	case corev1alpha1.MonitoringExecutionModeExtensionManaged:
		if !f.extensionManaged.ok {
			return "", f.extensionManaged.reason, f.extensionManaged.message
		}
		return want, "", ""
	}
	return f.preferred()
}

// materialise creates or updates the binding for an entry and applies the
// Accepted condition.
func (r *BindingReconciler) materialise(
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
	labels := rtmonitoring.BindingLabels(in.Name, entry.Name, entry.DestinationRef.Name, className)

	b := existing
	if b == nil {
		created, err := r.createBinding(ctx, in, entry, res, labels)
		if err != nil {
			return err
		}
		b = created
	} else if err := r.updateBinding(ctx, b, entry, res, labels); err != nil {
		return err
	}

	cond := res.accepted
	cond.ObservedGeneration = b.Generation
	return rtmonitoring.ApplyCondition(ctx, r.Client, b, b.Status.Conditions, rtmonitoring.MaterializerFieldOwner, cond)
}

func (r *BindingReconciler) createBinding(
	ctx context.Context,
	in *corev1alpha1.Instance,
	entry corev1alpha1.InstanceMonitoringDestination,
	res resolution,
	labels map[string]string,
) (*monitoringv1alpha1.MonitoringBinding, error) {
	b := &monitoringv1alpha1.MonitoringBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rtmonitoring.BindingName(in.Name, entry.Name),
			Namespace: in.Namespace,
			Labels:    labels,
		},
		Spec: monitoringv1alpha1.MonitoringBindingSpec{
			InstanceRef:    common.ObjectRef{Name: in.Name},
			DestinationRef: destinationRefOf(entry),
			ClassRef:       res.classRef,
			ExecutionMode:  res.executionMode,
			Parameters:     entry.Parameters,
		},
	}
	if err := controllerutil.SetControllerReference(in, b, r.Scheme); err != nil {
		return nil, fmt.Errorf("set owner: %w", err)
	}
	if err := r.Create(ctx, b, client.FieldOwner(rtmonitoring.MaterializerFieldOwner)); err != nil {
		return nil, fmt.Errorf("create binding: %w", err)
	}
	return b, nil
}

// updateBinding fills in classRef and executionMode once they resolve, and
// follows the entry's parameters and the binding labels.
func (r *BindingReconciler) updateBinding(
	ctx context.Context,
	b *monitoringv1alpha1.MonitoringBinding,
	entry corev1alpha1.InstanceMonitoringDestination,
	res resolution,
	labels map[string]string,
) error {
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
	for k, v := range labels {
		if b.Labels[k] != v {
			if b.Labels == nil {
				b.Labels = map[string]string{}
			}
			b.Labels[k] = v
			changed = true
		}
	}
	if !changed {
		return nil
	}
	if err := r.Update(ctx, b, client.FieldOwner(rtmonitoring.MaterializerFieldOwner)); err != nil {
		return fmt.Errorf("update binding: %w", err)
	}
	return nil
}

func rawEqual(a, b *runtime.RawExtension) bool {
	switch {
	case a == nil && b == nil:
		return true
	case a == nil || b == nil:
		return false
	}
	return bytes.Equal(a.Raw, b.Raw)
}

func (r *BindingReconciler) enqueueInstancesForDestination(ctx context.Context, obj client.Object) []reconcile.Request {
	instances := &corev1alpha1.InstanceList{}
	if err := r.List(ctx, instances,
		client.InNamespace(obj.GetNamespace()),
		client.MatchingFieldsSelector{Selector: fields.OneTermEqualSelector(InstanceMonitoringDestinationField, obj.GetName())},
	); err != nil {
		log.FromContext(ctx).Error(err, "List Instances for MonitoringDestination", "destination", obj.GetName())
		return nil
	}
	reqs := make([]reconcile.Request, 0, len(instances.Items))
	for i := range instances.Items {
		reqs = append(reqs, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(&instances.Items[i])})
	}
	return reqs
}

func (r *BindingReconciler) enqueueInstancesForClass(ctx context.Context, obj client.Object) []reconcile.Request {
	destinations := &monitoringv1alpha1.MonitoringDestinationList{}
	if err := r.List(ctx, destinations); err != nil {
		log.FromContext(ctx).Error(err, "List MonitoringDestinations for MonitoringClass", "class", obj.GetName())
		return nil
	}
	var reqs []reconcile.Request
	for i := range destinations.Items {
		if dst := &destinations.Items[i]; dst.Spec.ClassRef.Name == obj.GetName() {
			reqs = append(reqs, r.enqueueInstancesForDestination(ctx, dst)...)
		}
	}
	return reqs
}

func (r *BindingReconciler) enqueueInstancesForProvider(ctx context.Context, obj client.Object) []reconcile.Request {
	instances := &corev1alpha1.InstanceList{}
	if err := r.List(ctx, instances); err != nil {
		log.FromContext(ctx).Error(err, "List Instances for Provider", "provider", obj.GetName())
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
