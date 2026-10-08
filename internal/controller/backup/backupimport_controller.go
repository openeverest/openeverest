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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

// BackupImportReconciler reconciles BackupImport resources.
type BackupImportReconciler struct { //nolint:revive // use full CRD name for clarity
	Client client.Client
	Scheme *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (r *BackupImportReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("BackupImport").
		For(&backupv1alpha1.BackupImport{}).
		Complete(r)
}

// +kubebuilder:rbac:groups=backup.openeverest.io,resources=backupimports,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=backup.openeverest.io,resources=backupimports/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=backup.openeverest.io,resources=backupclasses,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *BackupImportReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logf.FromContext(ctx).
		WithName("BackupImportReconciler").
		WithValues("name", req.Name, "namespace", req.Namespace)
	logger.Info("Reconciling")

	defer func() {
		logger.Info("Reconciled")
	}()

	imp := &backupv1alpha1.BackupImport{}
	if err := r.Client.Get(ctx, req.NamespacedName, imp); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !imp.GetDeletionTimestamp().IsZero() {
		return ctrl.Result{}, nil
	}

	// Succeeded and Failed are terminal; nothing left to decide.
	if imp.Status.State == backupv1alpha1.BackupImportStateSucceeded ||
		imp.Status.State == backupv1alpha1.BackupImportStateFailed {
		return ctrl.Result{}, nil
	}

	bc := &backupv1alpha1.BackupClass{}
	if err := r.Client.Get(ctx, client.ObjectKey{
		Name: imp.Spec.ClassRef.Name,
	}, bc,
	); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to get backup class: %w", err)
	}

	// The class does not support import, record failure.
	if err := controller.ValidateBackupImportSupported(bc); err != nil {
		imp.Status.State = backupv1alpha1.BackupImportStateFailed
		imp.Status.Message = err.Error()
		imp.Status.LastObservedGeneration = imp.GetGeneration()
		if err := r.Client.Status().Update(ctx, imp); err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// ProviderManaged classes are reconciled by the provider-runtime.
	// Let the provider's runtime take over.
	if bc.Spec.ExecutionMode != backupv1alpha1.BackupExecutionModeJob {
		return ctrl.Result{}, nil
	}

	// TODO: implement Job-based import.
	imp.Status.State = backupv1alpha1.BackupImportStateFailed
	imp.Status.Message = "Job execution mode is not yet supported for BackupImport"
	imp.Status.LastObservedGeneration = imp.GetGeneration()
	if err := r.Client.Status().Update(ctx, imp); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
