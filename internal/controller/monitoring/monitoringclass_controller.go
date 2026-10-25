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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
	rtmonitoring "github.com/openeverest/openeverest/v2/provider-runtime/monitoring"
)

// ClassCompatibilityReconciler publishes MonitoringClass.status.providers,
// the compatibility of the class with every installed Provider. The
// Accepted condition belongs to the class controller and is never written
// here.
type ClassCompatibilityReconciler struct {
	client.Client
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClassCompatibilityReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("MonitoringClassCompatibility").
		For(&monitoringv1alpha1.MonitoringClass{}, builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Watches(&corev1alpha1.Provider{},
			handler.EnqueueRequestsFromMapFunc(r.enqueueAllClasses),
			builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Complete(r)
}

// +kubebuilder:rbac:groups=monitoring.openeverest.io,resources=monitoringclasses,verbs=get;list;watch
// +kubebuilder:rbac:groups=monitoring.openeverest.io,resources=monitoringclasses/status,verbs=patch
// +kubebuilder:rbac:groups=core.openeverest.io,resources=providers,verbs=get;list;watch

// Reconcile recomputes the compatibility matrix of one class.
func (r *ClassCompatibilityReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	class := &monitoringv1alpha1.MonitoringClass{}
	if err := r.Get(ctx, req.NamespacedName, class); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	providers := &corev1alpha1.ProviderList{}
	if err := r.List(ctx, providers); err != nil {
		return ctrl.Result{}, fmt.Errorf("list Providers: %w", err)
	}
	slices.SortFunc(providers.Items, func(a, b corev1alpha1.Provider) int { return strings.Compare(a.Name, b.Name) })

	entries := make([]any, 0, len(providers.Items))
	for i := range providers.Items {
		entry, err := runtime.DefaultUnstructuredConverter.ToUnstructured(new(providerCompatibility(&providers.Items[i], class)))
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("convert compatibility with Provider %q: %w", providers.Items[i].Name, err)
		}
		entries = append(entries, entry)
	}
	return ctrl.Result{}, rtmonitoring.ApplyStatus(ctx, r.Client, class, rtmonitoring.CompatibilityFieldOwner,
		map[string]any{"providers": entries})
}

func (r *ClassCompatibilityReconciler) enqueueAllClasses(ctx context.Context, obj client.Object) []reconcile.Request {
	classes := &monitoringv1alpha1.MonitoringClassList{}
	if err := r.List(ctx, classes); err != nil {
		log.FromContext(ctx).Error(err, "List MonitoringClasses for Provider", "provider", obj.GetName())
		return nil
	}
	reqs := make([]reconcile.Request, 0, len(classes.Items))
	for i := range classes.Items {
		reqs = append(reqs, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(&classes.Items[i])})
	}
	return reqs
}
