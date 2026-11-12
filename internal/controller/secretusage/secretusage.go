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

// Package secretusage counts the objects that use each Secret core tracks
// and protects the Secret from deletion while any does.
//
// Core does not own Secrets, so it marks them only with metadata:
//
//   - one usage count annotation per referrer kind, such as
//     monitoring.openeverest.io/attached-destinations, written on every
//     tracked Secret, "0" included, so a missing key means "not counted
//     yet". The secret-in-use.openeverest.io admission policy refuses
//     deleting a Secret while any count is not "0";
//   - the openeverest.io/in-use-protection finalizer, held only while an
//     object uses the Secret, those being deleted included, so deleting an
//     unused Secret is immediate.
//
// The counts follow inuse.Count: while the Secret is not being deleted,
// referrers being deleted do not count, so a referrer and its Secret can
// be deleted together; the finalizer holds the Secret until they are gone.
//
// One Reconciler serves every referrer kind, each an entry of its
// registry. The monitoring subsystem registers MonitoringDestination
// (monitoring.openeverest.io/attached-destinations); the backup subsystem
// registers BackupStorage (backup.openeverest.io/attached-storages) the
// same way, and provider-declared Secrets will later join as another entry.
package secretusage

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/openeverest/openeverest/v2/internal/controller/inuse"
	"github.com/openeverest/openeverest/v2/internal/secretdefinition"
	"github.com/openeverest/openeverest/v2/pkg/common"
)

// finalizer is held on a Secret while an object uses it.
const finalizer = common.OpenEverestInUseProtectionFinalizer

// Referrer is a kind of object that references Secrets in its namespace
// from typed fields, and the annotation that counts them on each Secret.
type Referrer struct {
	// Kind is the kind of the objects, e.g. "MonitoringDestination". It names
	// them in events.
	Kind string
	// CountKey is the annotation that counts, on a Secret, the objects of the
	// kind that use it. The secret-in-use.openeverest.io policy reads it.
	CountKey string
	// Object is an empty object of the kind, for the watch.
	Object client.Object
	// NewList returns an empty list of the kind.
	NewList func() client.ObjectList
	// IndexField is the cache field index from an object of the kind to the
	// names Names returns. The subsystem that owns the kind registers it
	// before the manager starts.
	IndexField string
	// Names returns the names of the Secrets obj references.
	Names func(obj client.Object) []string
}

// validate reports what is missing from r.
func (r Referrer) validate() error {
	if r.Kind == "" || r.CountKey == "" || r.Object == nil || r.NewList == nil || r.IndexField == "" || r.Names == nil {
		return fmt.Errorf("referrer %q: Kind, CountKey, Object, NewList, IndexField and Names are required", r.Kind)
	}
	return nil
}

// Reconciler keeps the usage counts and the in-use protection finalizer of
// Secrets up to date.
//
// It writes to tracked Secrets (secretdefinition.Tracked), whoever wrote
// them, and to Secrets that still carry the finalizer after they stopped
// being tracked: from those it removes the counts and, once nothing uses
// them, the finalizer. It never writes to any other Secret, such as an
// unlabelled CA bundle a destination references.
type Reconciler struct {
	client.Client

	// APIReader lists the referrers from the API server before the
	// finalizer is removed, since the cache may not have seen a referrer
	// created just before. It also reads a Secret the cache does not hold.
	APIReader client.Reader
	Recorder  events.EventRecorder
	// Referrers is the registry of the kinds that use Secrets.
	Referrers []Referrer
}

// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;patch
// +kubebuilder:rbac:groups=monitoring.openeverest.io,resources=monitoringdestinations,verbs=get;list;watch
// +kubebuilder:rbac:groups=events.k8s.io,resources=events,verbs=create;patch

// SetupWithManager sets up the controller: it reconciles the Secrets that
// are tracked or carry the finalizer, and requeues the Secrets a referrer
// names whenever the referrer can change their counts.
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := r.validate(); err != nil {
		return err
	}
	b := ctrl.NewControllerManagedBy(mgr).
		Named("SecretUsage").
		For(&corev1.Secret{}, builder.WithPredicates(secretPredicate()))
	for _, ref := range r.Referrers {
		b = b.Watches(ref.Object, enqueueSecretsOf(ref))
	}
	return b.Complete(r)
}

// Reconcile writes the Secret's counts and holds or releases its finalizer,
// in one patch, and only when either changes.
func (r *Reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	secret, err := r.secret(ctx, req.NamespacedName)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	tracked := secretdefinition.Tracked(secret.Labels)
	if !tracked && !controllerutil.ContainsFinalizer(secret, finalizer) {
		return ctrl.Result{}, nil
	}
	usages, err := r.usages(ctx, secret, tracked)
	if err != nil {
		return ctrl.Result{}, err
	}
	desired := r.desired(secret, tracked, usages)
	if maps.Equal(secret.Annotations, desired.Annotations) && slices.Equal(secret.Finalizers, desired.Finalizers) {
		return ctrl.Result{}, nil
	}
	if err := r.Patch(ctx, desired, client.MergeFromWithOptions(secret, client.MergeFromWithOptimisticLock{})); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(fmt.Errorf("patch Secret %s: %w", req.NamespacedName, err))
	}
	return ctrl.Result{}, nil
}

// validate checks the registry: each kind and count key once, and every
// field of each entry set.
func (r *Reconciler) validate() error {
	if len(r.Referrers) == 0 {
		return errors.New("no referrer kind registered")
	}
	seen := make(map[string]bool)
	for _, ref := range r.Referrers {
		if err := ref.validate(); err != nil {
			return err
		}
		if seen[ref.Kind] || seen[ref.CountKey] {
			return fmt.Errorf("referrer %q: kind or count key %q registered twice", ref.Kind, ref.CountKey)
		}
		seen[ref.Kind], seen[ref.CountKey] = true, true
	}
	return nil
}

// usages returns how every registered kind uses secret. Whether anything
// still uses an untracked Secret only the API server can tell. When the
// cache finds no referrer of a tracked Secret that holds the finalizer,
// the API server confirms it before the release, since the cache may not
// have seen a referrer yet.
func (r *Reconciler) usages(ctx context.Context, secret *corev1.Secret, tracked bool) ([]inuse.Usage, error) {
	usages, err := r.measure(ctx, secret, !tracked)
	if err != nil || !tracked || !secret.DeletionTimestamp.IsZero() || inUse(usages) ||
		!controllerutil.ContainsFinalizer(secret, finalizer) {
		return usages, err
	}
	return r.measure(ctx, secret, true)
}

// desired returns secret with the counts and the finalizer usages call
// for: a tracked Secret carries every count, an untracked one none, and
// the finalizer is added while a tracked Secret not being deleted is in
// use, and released by the guard once nothing uses the Secret.
func (r *Reconciler) desired(secret *corev1.Secret, tracked bool, usages []inuse.Usage) *corev1.Secret {
	desired := secret.DeepCopy()
	for i, ref := range r.Referrers {
		if tracked {
			metav1.SetMetaDataAnnotation(&desired.ObjectMeta, ref.CountKey, strconv.Itoa(usages[i].Count))
		} else {
			delete(desired.Annotations, ref.CountKey)
		}
	}
	if tracked && desired.DeletionTimestamp.IsZero() && inUse(usages) {
		controllerutil.AddFinalizer(desired, finalizer)
	}
	// Every usage a release can follow was read from the API server: the
	// Secret is being deleted or is untracked, or the API server confirmed
	// what the cache found.
	r.guard().Unhold(desired, referrersOf(r.Referrers, usages)...)
	return desired
}

// guard is the in-use protection of Secrets: the finalizer is held only
// while a Secret is in use.
func (r *Reconciler) guard() inuse.Guard {
	return inuse.Guard{
		Client:     r.Client,
		Recorder:   r.Recorder,
		Finalizer:  finalizer,
		WhileInUse: true,
	}
}

// secret reads the Secret from the cache, and from the API server when the
// cache does not hold it: a cache filtered by label drops a Secret whose
// labels were removed, which may still carry the finalizer.
func (r *Reconciler) secret(ctx context.Context, key types.NamespacedName) (*corev1.Secret, error) {
	secret := &corev1.Secret{}
	err := r.Get(ctx, key, secret)
	if apierrors.IsNotFound(err) {
		err = r.APIReader.Get(ctx, key, secret)
	}
	return secret, err
}

// measure returns how every registered kind uses secret, in registry
// order, by inuse.Measure: from the cache while secret is not being
// deleted, and from the API server once it is or with live.
func (r *Reconciler) measure(ctx context.Context, secret *corev1.Secret, live bool) ([]inuse.Usage, error) {
	usages := make([]inuse.Usage, 0, len(r.Referrers))
	for _, ref := range r.Referrers {
		usage, err := inuse.Measure(ctx, secret, func(ctx context.Context, fromAPIServer bool) ([]client.Object, error) {
			return r.list(ctx, ref, secret, fromAPIServer || live)
		})
		if err != nil {
			return nil, err
		}
		usages = append(usages, usage)
	}
	return usages, nil
}

// list returns the objects of ref's kind in secret's namespace that name
// secret, those being deleted included.
func (r *Reconciler) list(ctx context.Context, ref Referrer, secret *corev1.Secret, live bool) ([]client.Object, error) {
	list := ref.NewList()
	var err error
	if live {
		// The API server cannot select the objects by the Secrets they name,
		// so the namespace is listed and filtered.
		err = r.APIReader.List(ctx, list, client.InNamespace(secret.Namespace))
	} else {
		err = r.List(ctx, list, client.InNamespace(secret.Namespace), client.MatchingFields{ref.IndexField: secret.Name})
	}
	if err != nil {
		return nil, fmt.Errorf("list %ss using Secret %q: %w", ref.Kind, secret.Name, err)
	}
	items, err := meta.ExtractList(list)
	if err != nil {
		return nil, fmt.Errorf("read the %s list: %w", ref.Kind, err)
	}
	var referrers []client.Object
	for _, item := range items {
		if obj, ok := item.(client.Object); ok && slices.Contains(ref.Names(obj), secret.Name) {
			referrers = append(referrers, obj)
		}
	}
	return referrers, nil
}

// inUse reports whether any object uses the Secret, those being deleted
// included.
func inUse(usages []inuse.Usage) bool {
	return slices.ContainsFunc(usages, func(u inuse.Usage) bool { return len(u.Referrers) > 0 })
}

// referrersOf names the referrers of every kind.
func referrersOf(refs []Referrer, usages []inuse.Usage) []inuse.Referrers {
	out := make([]inuse.Referrers, 0, len(refs))
	for i, ref := range refs {
		out = append(out, inuse.Referrers{Kind: ref.Kind, Names: usages[i].Referrers})
	}
	return out
}

// secretPredicate passes the events of Secrets that are tracked or carry
// the finalizer, before or after the change: a Secret that starts being
// tracked must be counted, and one that stops must lose its marks.
func secretPredicate() predicate.Funcs {
	relevant := func(obj client.Object) bool {
		return secretdefinition.Tracked(obj.GetLabels()) || controllerutil.ContainsFinalizer(obj, finalizer)
	}
	return predicate.Funcs{
		CreateFunc:  func(e event.CreateEvent) bool { return relevant(e.Object) },
		UpdateFunc:  func(e event.UpdateEvent) bool { return relevant(e.ObjectOld) || relevant(e.ObjectNew) },
		DeleteFunc:  func(e event.DeleteEvent) bool { return relevant(e.Object) },
		GenericFunc: func(e event.GenericEvent) bool { return relevant(e.Object) },
	}
}

// enqueueSecretsOf enqueues the Secrets whose counts an event of one of
// ref's objects can change: every Secret a created or deleted object
// names; on update, those it started or stopped naming, plus every Secret
// it names once its deletion starts, since an object being deleted stops
// counting. Mapping only the new object would miss the Secret an update
// stopped naming.
func enqueueSecretsOf(ref Referrer) handler.Funcs {
	enqueue := func(q workqueue.TypedRateLimitingInterface[reconcile.Request], namespace string, names []string) {
		for _, name := range names {
			q.Add(reconcile.Request{NamespacedName: types.NamespacedName{Namespace: namespace, Name: name}})
		}
	}
	return handler.Funcs{
		CreateFunc: func(_ context.Context, e event.CreateEvent, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
			enqueue(q, e.Object.GetNamespace(), ref.Names(e.Object))
		},
		UpdateFunc: func(_ context.Context, e event.UpdateEvent, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
			newNames := ref.Names(e.ObjectNew)
			names := symmetricDifference(ref.Names(e.ObjectOld), newNames)
			if e.ObjectOld.GetDeletionTimestamp().IsZero() && !e.ObjectNew.GetDeletionTimestamp().IsZero() {
				names = append(names, newNames...)
			}
			enqueue(q, e.ObjectNew.GetNamespace(), names)
		},
		DeleteFunc: func(_ context.Context, e event.DeleteEvent, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
			enqueue(q, e.Object.GetNamespace(), ref.Names(e.Object))
		},
		GenericFunc: func(_ context.Context, e event.GenericEvent, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
			enqueue(q, e.Object.GetNamespace(), ref.Names(e.Object))
		},
	}
}

// symmetricDifference returns the names in exactly one of a and b.
func symmetricDifference(a, b []string) []string {
	var out []string
	for _, n := range a {
		if !slices.Contains(b, n) {
			out = append(out, n)
		}
	}
	for _, n := range b {
		if !slices.Contains(a, n) {
			out = append(out, n)
		}
	}
	return out
}

// ContentChanged passes the Secret events that can change what a Secret
// means to an object that reads it: creates, deletes, and updates of its
// data, type, labels or deletionTimestamp. The counts and the finalizer
// the Reconciler writes change none of these, so a watch with this
// predicate does not reconcile, nor re-render anything, for them.
func ContentChanged() predicate.Funcs {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldSecret, okOld := e.ObjectOld.(*corev1.Secret)
			newSecret, okNew := e.ObjectNew.(*corev1.Secret)
			if !okOld || !okNew {
				return true
			}
			return oldSecret.Type != newSecret.Type ||
				!maps.Equal(oldSecret.Labels, newSecret.Labels) ||
				!oldSecret.DeletionTimestamp.Equal(newSecret.DeletionTimestamp) ||
				!maps.EqualFunc(oldSecret.Data, newSecret.Data, slices.Equal[[]byte])
		},
	}
}
