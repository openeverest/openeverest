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
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
	rtmonitoring "github.com/openeverest/openeverest/v2/provider-runtime/monitoring"
)

// MonitoringDestinationReconciler owns the Accepted condition and the in-use
// finalizer of MonitoringDestinations. Ready is written by the class
// controller.
type MonitoringDestinationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// SetupWithManager registers the Instance index on
// spec.monitoring.destinations[].destinationRef.name (shared with the binding
// controller) and sets up the controller.
func (r *MonitoringDestinationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1alpha1.Instance{},
		InstanceMonitoringDestinationField, instanceMonitoringDestinationNames); err != nil {
		return fmt.Errorf("indexing instance by monitoring destination: %w", err)
	}
	return ctrl.NewControllerManagedBy(mgr).
		Named("MonitoringDestination").
		For(&monitoringv1alpha1.MonitoringDestination{}).
		Watches(&monitoringv1alpha1.MonitoringClass{},
			handler.EnqueueRequestsFromMapFunc(r.enqueueDestinationsForClass),
			builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Watches(&corev1alpha1.Instance{},
			handler.EnqueueRequestsFromMapFunc(r.enqueueDestinationsForInstance),
			builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Complete(r)
}

// +kubebuilder:rbac:groups=monitoring.openeverest.io,resources=monitoringdestinations,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=monitoring.openeverest.io,resources=monitoringdestinations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=monitoring.openeverest.io,resources=monitoringdestinations/finalizers,verbs=update

// Reconcile validates the destination against its class and keeps the in-use
// finalizer in step with the Instances that send to it.
func (r *MonitoringDestinationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	dst := &monitoringv1alpha1.MonitoringDestination{}
	if err := r.Get(ctx, req.NamespacedName, dst); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	inUse, err := r.isInUse(ctx, dst)
	if err != nil {
		return ctrl.Result{}, err
	}
	if inUse != controllerutil.ContainsFinalizer(dst, inUseFinalizer) {
		base := dst.DeepCopy()
		if inUse {
			controllerutil.AddFinalizer(dst, inUseFinalizer)
		} else {
			controllerutil.RemoveFinalizer(dst, inUseFinalizer)
		}
		if err := r.Patch(ctx, dst, client.MergeFromWithOptions(base, client.MergeFromWithOptimisticLock{})); err != nil {
			return ctrl.Result{}, fmt.Errorf("update in-use finalizer: %w", err)
		}
	}
	if !dst.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, nil
	}

	cond := r.acceptedCondition(ctx, dst)
	if err := rtmonitoring.ApplyCondition(ctx, r.Client, dst, dst.Status.Conditions, rtmonitoring.MaterializerFieldOwner, cond); err != nil {
		return ctrl.Result{}, err
	}
	// Secrets are not watched; re-check a rejected destination periodically.
	if cond.Status != metav1.ConditionTrue {
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}
	return ctrl.Result{}, nil
}

func (r *MonitoringDestinationReconciler) isInUse(ctx context.Context, dst *monitoringv1alpha1.MonitoringDestination) (bool, error) {
	instances := &corev1alpha1.InstanceList{}
	if err := r.List(ctx, instances,
		client.InNamespace(dst.Namespace),
		client.MatchingFieldsSelector{Selector: fields.OneTermEqualSelector(InstanceMonitoringDestinationField, dst.Name)},
	); err != nil {
		return false, fmt.Errorf("list instances: %w", err)
	}
	return len(instances.Items) > 0, nil
}

// acceptedCondition: class found, parameters valid against its schema, and
// the credentials Secret carrying every key the class requires.
func (r *MonitoringDestinationReconciler) acceptedCondition(ctx context.Context, dst *monitoringv1alpha1.MonitoringDestination) metav1.Condition {
	cond := rtmonitoring.NewCondition(dst, monitoringv1alpha1.MonitoringDestinationConditionAccepted, metav1.ConditionFalse, "", "")
	class := &monitoringv1alpha1.MonitoringClass{}
	if err := r.Get(ctx, types.NamespacedName{Name: dst.Spec.ClassRef.Name}, class); err != nil {
		cond.Reason = monitoringv1alpha1.ReasonClassNotFound
		cond.Message = fmt.Sprintf("MonitoringClass %q: %v", dst.Spec.ClassRef.Name, err)
		return cond
	}
	if class.Spec.DestinationParametersSchema != nil || dst.Spec.Parameters != nil {
		if err := class.Spec.DestinationParametersSchema.Validate(dst.Spec.Parameters); err != nil {
			cond.Reason = monitoringv1alpha1.ReasonInvalidParameters
			cond.Message = err.Error()
			return cond
		}
	}
	if cs := class.Spec.CredentialsSchema; cs != nil && len(cs.Required) > 0 {
		if dst.Spec.CredentialsSecretRef == nil {
			cond.Reason = monitoringv1alpha1.ReasonInvalidParameters
			cond.Message = fmt.Sprintf("MonitoringClass %q requires credentialsSecretRef with keys %v", class.Name, cs.Required)
			return cond
		}
		secret := &corev1.Secret{}
		if err := r.Get(ctx, types.NamespacedName{Namespace: dst.Namespace, Name: dst.Spec.CredentialsSecretRef.Name}, secret); err != nil {
			cond.Reason = monitoringv1alpha1.ReasonInvalidParameters
			cond.Message = fmt.Sprintf("credentials Secret %q: %v", dst.Spec.CredentialsSecretRef.Name, err)
			return cond
		}
		for _, key := range cs.Required {
			if len(secret.Data[key]) == 0 {
				cond.Reason = monitoringv1alpha1.ReasonInvalidParameters
				cond.Message = fmt.Sprintf("credentials Secret %q is missing key %q", secret.Name, key)
				return cond
			}
		}
	}
	cond.Status = metav1.ConditionTrue
	cond.Reason = monitoringv1alpha1.ReasonAccepted
	cond.Message = fmt.Sprintf("valid for MonitoringClass %q", class.Name)
	return cond
}

func (r *MonitoringDestinationReconciler) enqueueDestinationsForClass(ctx context.Context, obj client.Object) []reconcile.Request {
	list := &monitoringv1alpha1.MonitoringDestinationList{}
	if err := r.List(ctx, list); err != nil {
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

func (r *MonitoringDestinationReconciler) enqueueDestinationsForInstance(_ context.Context, obj client.Object) []reconcile.Request {
	names := instanceMonitoringDestinationNames(obj)
	reqs := make([]reconcile.Request, 0, len(names))
	for _, n := range names {
		reqs = append(reqs, reconcile.Request{NamespacedName: types.NamespacedName{Namespace: obj.GetNamespace(), Name: n}})
	}
	return reqs
}
