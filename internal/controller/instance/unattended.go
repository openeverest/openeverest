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

// Package instance holds what core writes on Instances. Providers write the
// rest of an Instance's status.
package instance

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/contract"
)

const (
	// UnattendedFieldOwner is the field manager core writes the Unattended
	// condition with. No other writer applies that condition.
	UnattendedFieldOwner = "openeverest-instance-unattended"

	// DefaultProviderGrace is how long an Instance may name a Provider that
	// does not exist before that is reported. A GitOps tool often applies an
	// Instance before its Provider.
	DefaultProviderGrace = time.Minute

	// DefaultResponseDeadline is how long a Provider that exists may leave
	// an Instance's current spec unobserved before that is reported.
	DefaultResponseDeadline = 5 * time.Minute

	// conflictRequeue gives the cache time to catch up with the write that
	// won the optimistic lock.
	conflictRequeue = time.Second

	// listedProviders is how many installed Providers a message names.
	listedProviders = 5

	instanceKind = "Instance"
)

// UnattendedReconciler writes the Unattended condition on Instances whose
// current spec no provider has observed: True with reason ProviderNotFound
// while the Provider spec.providerRef names does not exist, and Unknown with
// reason NotResponding while it exists but has not reported. The condition
// is absent once a provider has caught up, and until a deadline has passed.
// The verdict rests on observed facts and the time only, never on what a
// provider says.
type UnattendedReconciler struct {
	client.Client

	// ProviderGrace is how long a missing Provider is waited for. Zero means
	// DefaultProviderGrace.
	ProviderGrace time.Duration
	// ResponseDeadline is how long a Provider that exists is waited for.
	// Zero means DefaultResponseDeadline.
	ResponseDeadline time.Duration
	// Clock returns the current time. Nil means time.Now.
	Clock func() time.Time

	seen sightings
}

// +kubebuilder:rbac:groups=core.openeverest.io,resources=instances,verbs=get;list;watch
// +kubebuilder:rbac:groups=core.openeverest.io,resources=instances/status,verbs=patch
// +kubebuilder:rbac:groups=core.openeverest.io,resources=providers,verbs=get;list;watch

// SetupWithManager registers the controller. The manager runs controllers on
// the leader only, so a single clock decides.
func (r *UnattendedReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("InstanceUnattended").
		For(&corev1alpha1.Instance{}, builder.WithPredicates(unattendedEvents())).
		Watches(&corev1alpha1.Provider{},
			handler.EnqueueRequestsFromMapFunc(r.instancesOfProvider),
			builder.WithPredicates(createdOrDeleted()),
		).
		Complete(r)
}

// Reconcile brings the Unattended condition of one Instance in line with
// what is observed now.
func (r *UnattendedReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	in := &corev1alpha1.Instance{}
	if err := r.Get(ctx, req.NamespacedName, in); err != nil {
		if apierrors.IsNotFound(err) {
			r.seen.forget(req.NamespacedName)
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	stored := meta.FindStatusCondition(in.Status.Conditions, corev1alpha1.ConditionUnattended)
	want, recheck, err := r.verdict(ctx, in, stored, r.now())
	if err != nil {
		return ctrl.Result{}, err
	}
	if !differs(stored, want) {
		return ctrl.Result{RequeueAfter: recheck}, nil
	}
	if err := r.apply(ctx, in, want); err != nil {
		switch {
		case apierrors.IsConflict(err):
			// The Instance changed since it was read: decide again on what
			// it is now.
			return ctrl.Result{RequeueAfter: conflictRequeue}, nil
		case apierrors.IsNotFound(err):
			return ctrl.Result{}, nil
		default:
			return ctrl.Result{}, err
		}
	}
	if want == nil {
		log.FromContext(ctx).Info("Withdrew the Unattended condition", "reason", stored.Reason)
	} else {
		log.FromContext(ctx).Info("Set the Unattended condition", "reason", want.Reason, "message", want.Message)
	}
	return ctrl.Result{RequeueAfter: recheck}, nil
}

// verdict returns the Unattended condition the Instance should carry, nil
// for none, and how soon the verdict can change with time alone.
//
// Before a deadline has passed, a stored condition the facts still support
// is kept: a restart of core, or an edit while nobody attends to the
// Instance, starts the clock again, which only delays a new verdict.
func (r *UnattendedReconciler) verdict(
	ctx context.Context, in *corev1alpha1.Instance, stored *metav1.Condition, now time.Time,
) (*metav1.Condition, time.Duration, error) {
	if !behind(in) {
		r.seen.forget(client.ObjectKeyFromObject(in))
		return nil, 0, nil
	}
	anchor := latest(in.CreationTimestamp.Time, r.seen.firstSeen(in, now))
	name := in.Spec.ProviderRef.Name

	provider := &corev1alpha1.Provider{}
	err := r.Get(ctx, client.ObjectKey{Name: name}, provider)
	switch {
	case apierrors.IsNotFound(err):
		if due := anchor.Add(r.providerGrace()); now.Before(due) {
			return holding(stored, corev1alpha1.ReasonProviderNotFound), due.Sub(now), nil
		}
		installed, err := r.installedProviders(ctx)
		if err != nil {
			return nil, 0, err
		}
		return newCondition(in, stored, metav1.ConditionTrue, corev1alpha1.ReasonProviderNotFound,
			providerNotFoundMessage(name, installed), now,
		), 0, nil
	case err != nil:
		return nil, 0, fmt.Errorf("get provider %s: %w", name, err)
	}

	if due := latest(anchor, provider.CreationTimestamp.Time).Add(r.responseDeadline()); now.Before(due) {
		return holding(stored, corev1alpha1.ReasonNotResponding), due.Sub(now), nil
	}
	return newCondition(in, stored, metav1.ConditionUnknown, corev1alpha1.ReasonNotResponding,
		notRespondingMessage(in, r.responseDeadline()), now,
	), 0, nil
}

// installedProviders returns the names of the Providers that exist, sorted.
func (r *UnattendedReconciler) installedProviders(ctx context.Context) ([]string, error) {
	providers := &corev1alpha1.ProviderList{}
	if err := r.List(ctx, providers); err != nil {
		return nil, fmt.Errorf("list providers: %w", err)
	}
	names := make([]string, 0, len(providers.Items))
	for _, p := range providers.Items {
		names = append(names, p.Name)
	}
	slices.Sort(names)
	return names, nil
}

// apply server-side applies cond as the only condition core's field manager
// owns on the Instance, or no condition when cond is nil, which removes the
// one it wrote before. The resourceVersion read is a precondition, so a
// verdict reached on a stale read is refused. Ownership is never forced: no
// other writer applies this condition.
func (r *UnattendedReconciler) apply(ctx context.Context, in *corev1alpha1.Instance, cond *metav1.Condition) error {
	conditions := []any{}
	if cond != nil {
		raw, err := runtime.DefaultUnstructuredConverter.ToUnstructured(cond)
		if err != nil {
			return fmt.Errorf("convert the %s condition: %w", cond.Type, err)
		}
		conditions = append(conditions, raw)
	}
	u := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": corev1alpha1.GroupVersion.String(),
		"kind":       instanceKind,
		"metadata": map[string]any{
			"name":            in.Name,
			"namespace":       in.Namespace,
			"resourceVersion": in.ResourceVersion,
		},
		"status": map[string]any{"conditions": conditions},
	}}
	if err := r.Status().Apply(ctx, client.ApplyConfigurationFromUnstructured(u), client.FieldOwner(UnattendedFieldOwner)); err != nil {
		return fmt.Errorf("apply the %s condition of instance %s/%s: %w",
			corev1alpha1.ConditionUnattended, in.Namespace, in.Name, err,
		)
	}
	return nil
}

// instancesOfProvider enqueues the Instances that name the Provider and
// that a verdict concerns, and every Instance whose Provider was not found:
// their message lists the installed Providers, so it changes whenever one
// is installed or removed.
func (r *UnattendedReconciler) instancesOfProvider(ctx context.Context, obj client.Object) []reconcile.Request {
	instances := &corev1alpha1.InstanceList{}
	if err := r.List(ctx, instances); err != nil {
		log.FromContext(ctx).Error(err, "List Instances for Provider", "provider", obj.GetName())
		return nil
	}
	var reqs []reconcile.Request
	for i := range instances.Items {
		in := &instances.Items[i]
		stored := meta.FindStatusCondition(in.Status.Conditions, corev1alpha1.ConditionUnattended)
		named := in.Spec.ProviderRef.Name == obj.GetName() && concerns(in)
		if named || (stored != nil && stored.Reason == corev1alpha1.ReasonProviderNotFound) {
			reqs = append(reqs, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(in)})
		}
	}
	return reqs
}

func (r *UnattendedReconciler) now() time.Time {
	if r.Clock != nil {
		return r.Clock()
	}
	return time.Now()
}

func (r *UnattendedReconciler) providerGrace() time.Duration {
	if r.ProviderGrace > 0 {
		return r.ProviderGrace
	}
	return DefaultProviderGrace
}

func (r *UnattendedReconciler) responseDeadline() time.Duration {
	if r.ResponseDeadline > 0 {
		return r.ResponseDeadline
	}
	return DefaultResponseDeadline
}

// sightings remembers when this process first saw each Instance's current
// generation.
type sightings struct {
	mu        sync.Mutex
	instances map[types.NamespacedName]sighting
}

type sighting struct {
	uid        types.UID
	generation int64
	at         time.Time
}

// firstSeen returns when the Instance's current generation was first seen:
// now, if it is new to this process.
func (s *sightings) firstSeen(in *corev1alpha1.Instance, now time.Time) time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := client.ObjectKeyFromObject(in)
	if seen, ok := s.instances[key]; ok && seen.uid == in.UID && seen.generation == in.Generation {
		return seen.at
	}
	if s.instances == nil {
		s.instances = make(map[types.NamespacedName]sighting)
	}
	s.instances[key] = sighting{uid: in.UID, generation: in.Generation, at: now}
	return now
}

func (s *sightings) forget(key types.NamespacedName) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.instances, key)
}

// behind reports whether no provider has observed the Instance's current
// spec. A status no provider has written has observedGeneration -1.
func behind(in *corev1alpha1.Instance) bool {
	return in.Status.ObservedGeneration < in.Generation
}

// concerns reports whether the Instance may need the Unattended condition
// set or removed.
func concerns(in *corev1alpha1.Instance) bool {
	return behind(in) || meta.FindStatusCondition(in.Status.Conditions, corev1alpha1.ConditionUnattended) != nil
}

// unattendedEvents passes creations, deletions, and the updates where the
// old or the new Instance is behind or carries the condition. The old one
// matters: the update in which a provider catches up is the one that has
// the condition withdrawn.
func unattendedEvents() predicate.Predicate { //nolint:ireturn // the builder takes the interface
	return predicate.Funcs{
		CreateFunc: func(event.CreateEvent) bool { return true },
		DeleteFunc: func(event.DeleteEvent) bool { return true },
		UpdateFunc: func(e event.UpdateEvent) bool {
			return concernsObject(e.ObjectOld) || concernsObject(e.ObjectNew)
		},
		GenericFunc: func(event.GenericEvent) bool { return false },
	}
}

func concernsObject(obj client.Object) bool {
	in, ok := obj.(*corev1alpha1.Instance)
	return ok && concerns(in)
}

// createdOrDeleted passes the Provider events that can change a verdict: a
// Provider appears or goes away.
func createdOrDeleted() predicate.Predicate { //nolint:ireturn // the builder takes the interface
	return predicate.Funcs{
		CreateFunc:  func(event.CreateEvent) bool { return true },
		DeleteFunc:  func(event.DeleteEvent) bool { return true },
		UpdateFunc:  func(event.UpdateEvent) bool { return false },
		GenericFunc: func(event.GenericEvent) bool { return false },
	}
}

// holding returns the stored condition if it has the reason the facts
// support, and nil otherwise.
func holding(stored *metav1.Condition, reason string) *metav1.Condition {
	if stored != nil && stored.Reason == reason {
		return stored
	}
	return nil
}

// newCondition builds the condition for the Instance's current generation.
// It keeps the stored transition time while the status stays the same.
func newCondition(
	in *corev1alpha1.Instance, stored *metav1.Condition,
	status metav1.ConditionStatus, reason, message string, now time.Time,
) *metav1.Condition {
	since := metav1.NewTime(now)
	if stored != nil && stored.Status == status {
		since = stored.LastTransitionTime
	}
	return &metav1.Condition{
		Type:               corev1alpha1.ConditionUnattended,
		Status:             status,
		ObservedGeneration: in.Generation,
		LastTransitionTime: since,
		Reason:             reason,
		Message:            message,
	}
}

// differs reports whether writing want would change what is stored.
func differs(stored, want *metav1.Condition) bool {
	if stored == nil || want == nil {
		return stored != want
	}
	return stored.Status != want.Status ||
		stored.Reason != want.Reason ||
		stored.Message != want.Message ||
		stored.ObservedGeneration != want.ObservedGeneration
}

// providerNotFoundMessage says that the Provider is not installed, which
// ones are, and how to use another one.
func providerNotFoundMessage(name string, installed []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Provider %q is not installed. ", name)
	if len(installed) == 0 {
		b.WriteString("No Provider is installed. ")
	} else {
		listed := installed[:min(len(installed), listedProviders)]
		b.WriteString("Installed Providers: " + strings.Join(listed, ", "))
		if more := len(installed) - len(listed); more > 0 {
			fmt.Fprintf(&b, " and %d more", more)
		}
		b.WriteString(". ")
	}
	b.WriteString("spec.providerRef cannot be changed: delete the Instance and create it again with the right provider.")
	return b.String()
}

// notRespondingMessage says where to look when the Provider exists but has
// not reported on the Instance's current spec. The runtime's finalizer or
// label shows that the provider picked the Instance up.
func notRespondingMessage(in *corev1alpha1.Instance, waited time.Duration) string {
	name := in.Spec.ProviderRef.Name
	if controllerutil.ContainsFinalizer(in, contract.ProviderFinalizer) || in.Labels[contract.ProviderLabel] != "" {
		return fmt.Sprintf("Provider %q picked this Instance up but has not reported on generation %d for %s; check its logs.",
			name, in.Generation, spellDuration(waited),
		)
	}
	return fmt.Sprintf("Provider %q has not picked this Instance up for %s; check that it is running.",
		name, spellDuration(waited),
	)
}

// spellDuration writes whole minutes in words, and any other duration as Go
// does.
func spellDuration(d time.Duration) string {
	if d < time.Minute || d%time.Minute != 0 {
		return d.String()
	}
	if minutes := int64(d / time.Minute); minutes != 1 {
		return fmt.Sprintf("%d minutes", minutes)
	}
	return "1 minute"
}

func latest(a, b time.Time) time.Time {
	if b.After(a) {
		return b
	}
	return a
}
