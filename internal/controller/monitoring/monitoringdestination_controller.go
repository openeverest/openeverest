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
	"fmt"
	"slices"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
	"github.com/openeverest/openeverest/v2/internal/controller/inuse"
	rtmonitoring "github.com/openeverest/openeverest/v2/provider-runtime/monitoring"
)

// destinationSecretsField indexes MonitoringDestinations by the name of every
// Secret they reference.
const destinationSecretsField = ".spec.secretNames"

func destinationSecretNames(obj client.Object) []string {
	dst, ok := obj.(*monitoringv1alpha1.MonitoringDestination)
	if !ok {
		return nil
	}
	return dst.Spec.SecretNames()
}

// DestinationReconciler owns the Accepted condition, status.attachedInstances
// and the in-use protection finalizer of MonitoringDestinations. Ready is
// written by the class controller.
type DestinationReconciler struct {
	client.Client

	// APIReader lists Instances from the API server before the finalizer of
	// a destination being deleted is removed: the cache may not have seen an
	// Instance created just before the deletion.
	APIReader client.Reader
	Recorder  events.EventRecorder
}

// SetupWithManager registers the MonitoringDestination index on the Secrets
// it references and sets up the controller. The Instance index it counts
// Instances by is registered by SetupIndexes.
func (r *DestinationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &monitoringv1alpha1.MonitoringDestination{},
		destinationSecretsField, destinationSecretNames,
	); err != nil {
		return fmt.Errorf("indexing monitoring destination by secret: %w", err)
	}
	return ctrl.NewControllerManagedBy(mgr).
		Named("MonitoringDestination").
		// The API server bumps metadata.generation when it marks an object for
		// deletion, so deletions pass the predicate.
		For(&monitoringv1alpha1.MonitoringDestination{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Watches(&monitoringv1alpha1.MonitoringClass{},
			handler.EnqueueRequestsFromMapFunc(r.enqueueDestinationsForClass),
			builder.WithPredicates(predicate.GenerationChangedPredicate{}),
		).
		Watches(&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.enqueueDestinationsForSecret),
		).
		Watches(&corev1alpha1.Instance{}, enqueueDestinationsOfInstance()).
		Complete(r)
}

// +kubebuilder:rbac:groups=monitoring.openeverest.io,resources=monitoringdestinations,verbs=get;list;watch;patch
// +kubebuilder:rbac:groups=monitoring.openeverest.io,resources=monitoringdestinations/status,verbs=patch
// +kubebuilder:rbac:groups=monitoring.openeverest.io,resources=monitoringdestinations/finalizers,verbs=update
// +kubebuilder:rbac:groups=monitoring.openeverest.io,resources=monitoringclasses,verbs=get;list;watch
// +kubebuilder:rbac:groups=core.openeverest.io,resources=instances,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=events.k8s.io,resources=events,verbs=create;patch

// Reconcile validates the destination against its class, counts the
// Instances that list it and holds its deletion while any does.
func (r *DestinationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	dst := &monitoringv1alpha1.MonitoringDestination{}
	if err := r.Get(ctx, req.NamespacedName, dst); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	guard := inuse.Guard{
		Client:       r.Client,
		Recorder:     r.Recorder,
		Finalizer:    monitoringv1alpha1.InUseProtectionFinalizer,
		ReferrerKind: "Instance",
	}
	if err := guard.Protect(ctx, dst); err != nil {
		return ctrl.Result{}, err
	}
	attached, err := r.attachedInstances(ctx, dst)
	if err != nil {
		return ctrl.Result{}, err
	}
	cond, err := r.acceptedCondition(ctx, dst)
	if err != nil {
		return ctrl.Result{}, err
	}
	if !dst.DeletionTimestamp.IsZero() {
		// Before the status apply, which changes the resourceVersion the
		// finalizer patch is locked on.
		if _, err := guard.Release(ctx, dst, attached.Referrers); err != nil {
			return ctrl.Result{}, err
		}
		if len(dst.Finalizers) == 0 {
			// The release let the deletion finish.
			return ctrl.Result{}, nil
		}
	}
	// One apply for every status field this owner writes: a second apply
	// would prune the first one's fields.
	if err := rtmonitoring.ApplyCondition(ctx, r.Client, dst, dst.Status.Conditions, rtmonitoring.MaterializerFieldOwner, cond,
		map[string]any{"attachedInstances": int64(attached.Count)},
	); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	return ctrl.Result{}, nil
}

// attachedInstances finds the Instances whose spec lists dst and counts them
// by inuse.Count. While dst is not being deleted, they are read from the
// cache by the Instance index and those being deleted are left out of the
// count; once it is, they are read from the API server, which the cache may
// lag, and all of them count.
func (r *DestinationReconciler) attachedInstances(ctx context.Context, dst *monitoringv1alpha1.MonitoringDestination) (inuse.Usage, error) {
	return inuse.Measure(ctx, dst, func(ctx context.Context, live bool) ([]*corev1alpha1.Instance, error) {
		instances := &corev1alpha1.InstanceList{}
		var err error
		if live {
			// The API server cannot select custom resources by this field.
			err = r.APIReader.List(ctx, instances, client.InNamespace(dst.Namespace))
		} else {
			err = r.List(ctx, instances,
				client.InNamespace(dst.Namespace),
				client.MatchingFieldsSelector{Selector: fields.OneTermEqualSelector(instanceMonitoringDestinationField, dst.Name)},
			)
		}
		if err != nil {
			return nil, fmt.Errorf("list Instances using MonitoringDestination %q: %w", dst.Name, err)
		}
		var listing []*corev1alpha1.Instance
		for i := range instances.Items {
			if slices.Contains(destinationNames(&instances.Items[i]), dst.Name) {
				listing = append(listing, &instances.Items[i])
			}
		}
		return listing, nil
	})
}

// acceptedCondition: class found, parameters valid against its schema,
// authentication mode defined by the class with every credentials key
// present, and the CA bundle parseable.
func (r *DestinationReconciler) acceptedCondition(ctx context.Context, dst *monitoringv1alpha1.MonitoringDestination) (metav1.Condition, error) {
	reject := func(reason, msg string) (metav1.Condition, error) {
		return rtmonitoring.NewCondition(dst, monitoringv1alpha1.MonitoringDestinationConditionAccepted, metav1.ConditionFalse, reason, msg), nil
	}

	class := &monitoringv1alpha1.MonitoringClass{}
	if err := r.Get(ctx, types.NamespacedName{Name: dst.Spec.ClassRef.Name}, class); err != nil {
		if !apierrors.IsNotFound(err) {
			return metav1.Condition{}, fmt.Errorf("get MonitoringClass %q: %w", dst.Spec.ClassRef.Name, err)
		}
		return reject(monitoringv1alpha1.ReasonClassNotFound, fmt.Sprintf("MonitoringClass %q not found", dst.Spec.ClassRef.Name))
	}
	schema := class.Spec.DestinationParametersSchema
	params := dst.Spec.Parameters
	// An absent payload must still satisfy the schema's required properties.
	if params == nil && schema != nil && schema.OpenAPIV3Schema != nil {
		params = &runtime.RawExtension{Raw: []byte("{}")}
	}
	if err := schema.Validate(params); err != nil {
		return reject(monitoringv1alpha1.ReasonInvalidParameters,
			fmt.Sprintf("parameters rejected by the destinationParametersSchema of MonitoringClass %q: %v", class.Name, err),
		)
	}
	mode := class.Spec.FindAuthMode(dst.Spec.Authentication.Mode)
	if mode == nil {
		return reject(monitoringv1alpha1.ReasonAuthModeUnsupported, fmt.Sprintf("MonitoringClass %q has no authentication mode %q; it supports %s",
			class.Name, dst.Spec.Authentication.Mode, strings.Join(class.Spec.AuthModeNames(), ", "),
		))
	}
	msg, err := r.credentialsProblem(ctx, dst, mode)
	if err != nil {
		return metav1.Condition{}, err
	}
	if msg != "" {
		return reject(monitoringv1alpha1.ReasonCredentialsNotFound, msg)
	}
	if msg, err = r.caBundleProblem(ctx, dst); err != nil {
		return metav1.Condition{}, err
	}
	if msg != "" {
		return reject(monitoringv1alpha1.ReasonInvalidCABundle, msg)
	}
	return rtmonitoring.NewCondition(dst, monitoringv1alpha1.MonitoringDestinationConditionAccepted, metav1.ConditionTrue,
		monitoringv1alpha1.ReasonAccepted, fmt.Sprintf("valid for MonitoringClass %q", class.Name),
	), nil
}

// credentialsProblem describes what is missing from the destination's
// credentials Secret for mode, or returns "" when nothing is.
func (r *DestinationReconciler) credentialsProblem(ctx context.Context, dst *monitoringv1alpha1.MonitoringDestination, mode *common.AuthMode) (string, error) {
	ref := dst.Spec.Authentication.CredentialsSecretRef
	if ref == nil {
		return "", nil
	}
	secret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: dst.Namespace, Name: ref.Name}, secret); err != nil {
		if !apierrors.IsNotFound(err) {
			return "", fmt.Errorf("get credentials Secret %q: %w", ref.Name, err)
		}
		return fmt.Sprintf("credentials Secret %q not found", ref.Name), nil
	}
	if err := mode.ValidateCredentials(secret.Data); err != nil {
		return fmt.Sprintf("credentials Secret %q: %v", ref.Name, err), nil
	}
	return "", nil
}

// caBundleProblem describes what is wrong with the destination's CA bundle,
// or returns "" when nothing is.
func (r *DestinationReconciler) caBundleProblem(ctx context.Context, dst *monitoringv1alpha1.MonitoringDestination) (string, error) {
	if dst.Spec.TLS == nil || dst.Spec.TLS.CA == nil || dst.Spec.TLS.CA.SecretKeyRef == nil {
		return "", nil
	}
	ref := dst.Spec.TLS.CA.SecretKeyRef
	secret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Namespace: dst.Namespace, Name: ref.Name}, secret); err != nil {
		if !apierrors.IsNotFound(err) {
			return "", fmt.Errorf("get CA Secret %q: %w", ref.Name, err)
		}
		return fmt.Sprintf("CA Secret %q not found", ref.Name), nil
	}
	if _, err := common.ParseCABundle(secret.Data[ref.Key]); err != nil {
		return fmt.Sprintf("CA Secret %q key %q: %v", ref.Name, ref.Key, err), nil
	}
	return "", nil
}

func (r *DestinationReconciler) enqueueDestinationsForClass(ctx context.Context, obj client.Object) []reconcile.Request {
	list := &monitoringv1alpha1.MonitoringDestinationList{}
	if err := r.List(ctx, list); err != nil {
		log.FromContext(ctx).Error(err, "List MonitoringDestinations for MonitoringClass", "class", obj.GetName())
		return nil
	}
	var reqs []reconcile.Request
	for i := range list.Items {
		if list.Items[i].Spec.ClassRef.Name == obj.GetName() {
			reqs = append(reqs, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(&list.Items[i])})
		}
	}
	return reqs
}

func (r *DestinationReconciler) enqueueDestinationsForSecret(ctx context.Context, obj client.Object) []reconcile.Request {
	list := &monitoringv1alpha1.MonitoringDestinationList{}
	if err := r.List(ctx, list,
		client.InNamespace(obj.GetNamespace()),
		client.MatchingFieldsSelector{Selector: fields.OneTermEqualSelector(destinationSecretsField, obj.GetName())},
	); err != nil {
		log.FromContext(ctx).Error(err, "List MonitoringDestinations for Secret", "secret", obj.GetName())
		return nil
	}
	reqs := make([]reconcile.Request, 0, len(list.Items))
	for i := range list.Items {
		reqs = append(reqs, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(&list.Items[i])})
	}
	return reqs
}

// enqueueDestinationsOfInstance enqueues the MonitoringDestinations whose
// attachedInstances an Instance event can change: every destination a
// created or deleted Instance lists and, on update, those it started or
// stopped listing, plus every destination it lists once its deletion
// starts, since an Instance being deleted stops counting. Mapping only the
// new object would miss the destination an update stopped listing.
func enqueueDestinationsOfInstance() handler.Funcs {
	enqueue := func(q workqueue.TypedRateLimitingInterface[reconcile.Request], namespace string, names []string) {
		for _, name := range names {
			q.Add(reconcile.Request{NamespacedName: types.NamespacedName{Namespace: namespace, Name: name}})
		}
	}
	return handler.Funcs{
		CreateFunc: func(_ context.Context, e event.CreateEvent, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
			enqueue(q, e.Object.GetNamespace(), instanceMonitoringDestinationNames(e.Object))
		},
		UpdateFunc: func(_ context.Context, e event.UpdateEvent, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
			newNames := instanceMonitoringDestinationNames(e.ObjectNew)
			names := symmetricDifference(instanceMonitoringDestinationNames(e.ObjectOld), newNames)
			if e.ObjectOld.GetDeletionTimestamp().IsZero() && !e.ObjectNew.GetDeletionTimestamp().IsZero() {
				names = append(names, newNames...)
			}
			enqueue(q, e.ObjectNew.GetNamespace(), names)
		},
		DeleteFunc: func(_ context.Context, e event.DeleteEvent, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
			enqueue(q, e.Object.GetNamespace(), instanceMonitoringDestinationNames(e.Object))
		},
		GenericFunc: func(_ context.Context, e event.GenericEvent, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
			enqueue(q, e.Object.GetNamespace(), instanceMonitoringDestinationNames(e.Object))
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
