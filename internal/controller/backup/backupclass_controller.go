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

package backup

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	rtmonitoring "github.com/openeverest/openeverest/v2/provider-runtime/monitoring"
)

const (
	// backupClassFieldOwner is the field manager of the status core writes on
	// BackupClasses: providers. The class's executor owns Accepted.
	backupClassFieldOwner = "openeverest-backup-class"

	// backupClassProviderField indexes BackupClasses by the Provider that
	// executes them.
	backupClassProviderField = ".spec.providerManaged.providerRef.name"
)

// BackupClassReconciler publishes BackupClass.status.providers, the
// compatibility of each class with the Provider that executes it.
type BackupClassReconciler struct { //nolint:revive // named like the other reconcilers of the package
	client.Client

	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=backup.openeverest.io,resources=backupclasses,verbs=get;list;watch
// +kubebuilder:rbac:groups=backup.openeverest.io,resources=backupclasses/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=core.openeverest.io,resources=providers,verbs=get;list;watch

// Reconcile publishes the class's compatibility with its executor Provider.
func (r *BackupClassReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	class := &backupv1alpha1.BackupClass{}
	if err := r.Get(ctx, req.NamespacedName, class); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	providerFound := false
	if name := class.Spec.ExecutorProvider(); name != "" {
		err := r.Get(ctx, types.NamespacedName{Name: name}, &corev1alpha1.Provider{})
		switch {
		case err == nil:
			providerFound = true
		case !apierrors.IsNotFound(err):
			return ctrl.Result{}, fmt.Errorf("get Provider %q: %w", name, err)
		}
	}
	entries := providerCompatibilities(class, providerFound)
	providers := make([]any, 0, len(entries))
	for i := range entries {
		raw, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&entries[i])
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("convert provider compatibility: %w", err)
		}
		providers = append(providers, raw)
	}
	return ctrl.Result{}, rtmonitoring.ApplyStatus(ctx, r.Client, class, backupClassFieldOwner, map[string]any{
		"providers": providers,
	})
}

// SetupWithManager sets up the controller with the Manager. Status changes
// of a class are watched too: its executor's Accepted condition is an input.
func (r *BackupClassReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &backupv1alpha1.BackupClass{}, backupClassProviderField,
		func(obj client.Object) []string {
			bc, ok := obj.(*backupv1alpha1.BackupClass)
			if !ok || bc.Spec.ExecutorProvider() == "" {
				return nil
			}
			return []string{bc.Spec.ExecutorProvider()}
		},
	); err != nil {
		return fmt.Errorf("index BackupClasses by provider: %w", err)
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&backupv1alpha1.BackupClass{}).
		Named("backup-backupclass").
		Watches(&corev1alpha1.Provider{}, handler.EnqueueRequestsFromMapFunc(r.classesOfProvider)).
		Complete(r)
}

// classesOfProvider maps a Provider to the BackupClasses it executes.
func (r *BackupClassReconciler) classesOfProvider(ctx context.Context, obj client.Object) []reconcile.Request {
	list := &backupv1alpha1.BackupClassList{}
	if err := r.List(ctx, list, client.MatchingFieldsSelector{Selector: fields.OneTermEqualSelector(backupClassProviderField, obj.GetName())}); err != nil {
		logf.FromContext(ctx).Error(err, "List BackupClasses to enqueue", "provider", obj.GetName())
		return nil
	}
	reqs := make([]reconcile.Request, 0, len(list.Items))
	for i := range list.Items {
		reqs = append(reqs, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(&list.Items[i])})
	}
	return reqs
}
