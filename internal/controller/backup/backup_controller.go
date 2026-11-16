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

// Package backup contains the BackupReconciler which manages Backup resources.
package backup

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	"github.com/openeverest/openeverest/v2/pkg/common"
)

// BackupReconciler reconciles Backup resources. The provider that executes a
// Backup's class reconciles the backup itself; this controller only does what
// no provider can: it labels the Backup with its Instance, reports a Backup
// whose class does not exist yet, and marks a Backup being deleted.
type BackupReconciler struct {
	Client client.Client
	Scheme *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (r *BackupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("Backup").
		For(&backupv1alpha1.Backup{}).
		Watches(&backupv1alpha1.BackupClass{},
			operationsFor[backupv1alpha1.BackupList](mgr.GetClient(), false, backupNamesClass),
			builder.WithPredicates(referentAppearedOrGone()),
		).
		Complete(r)
}

//+kubebuilder:rbac:groups=backup.openeverest.io,resources=backups,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=backup.openeverest.io,resources=backups/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=backup.openeverest.io,resources=backupclasses,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *BackupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).
		WithName("BackupReconciler").
		WithValues(
			"name", req.Name,
			"namespace", req.Namespace,
		)
	logger.Info("Reconciling")
	defer func() {
		logger.Info("Reconciled")
	}()

	backup := &backupv1alpha1.Backup{}
	if err := r.Client.Get(ctx, req.NamespacedName, backup); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !backup.GetDeletionTimestamp().IsZero() {
		if backup.Status.State == backupv1alpha1.BackupStateDeleting {
			return ctrl.Result{}, nil
		}
		backup.Status.State = backupv1alpha1.BackupStateDeleting
		if err := r.Client.Status().Update(ctx, backup); err != nil {
			logger.Error(err, "Failed to update backup status to Deleting")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Ensure the instance name label is set on the Backup resource.
	if err := r.ensureInstanceNameLabel(ctx, backup); err != nil {
		return ctrl.Result{}, err
	}

	if backup.Status.State == backupv1alpha1.BackupStateSucceeded ||
		backup.Status.State == backupv1alpha1.BackupStateFailed {
		// Already complete, no need to reconcile again.
		return ctrl.Result{}, nil
	}

	if err := r.Client.Get(ctx, client.ObjectKey{
		Name: backup.Spec.ClassRef.Name,
	}, &backupv1alpha1.BackupClass{}); err != nil {
		// Nobody else knows which controller owns the Backup until its class
		// exists, so it waits here.
		if apierrors.IsNotFound(err) {
			backup.Status.State = backupv1alpha1.BackupStatePending
			backup.Status.Message = fmt.Sprintf("backup class %s not found", backup.Spec.ClassRef.Name)
			backup.Status.LastObservedGeneration = backup.GetGeneration()
			return ctrl.Result{}, r.Client.Status().Update(ctx, backup)
		}
		return ctrl.Result{}, fmt.Errorf("failed to get backup class: %w", err)
	}

	// The class exists: the provider that executes it takes over, and this
	// controller leaves the Backup's status alone.
	return ctrl.Result{}, nil
}

// ensureInstanceNameLabel ensures that the Backup resource has the instance name
// label, used for filtering backups by instance using label selectors.
func (r *BackupReconciler) ensureInstanceNameLabel(ctx context.Context, backup *backupv1alpha1.Backup) error {
	// External backups have no Instance to label.
	if backup.Spec.Origin.Type == backupv1alpha1.BackupOriginTypeExternal {
		return nil
	}

	labels := backup.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}

	if labels[common.InstanceNameLabel] == backup.Spec.Origin.InstanceRef.Name {
		return nil
	}

	labels[common.InstanceNameLabel] = backup.Spec.Origin.InstanceRef.Name
	backup.SetLabels(labels)
	if err := r.Client.Update(ctx, backup); err != nil {
		return fmt.Errorf("failed to update instance name label: %w", err)
	}

	return nil
}
