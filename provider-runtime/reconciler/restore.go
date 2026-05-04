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

package reconciler

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

const restoreRuntimeFinalizer = "everest.percona.com/restore-runtime-finalizer"

// restoreRuntimeReconciler dispatches Restore CRs to the provider's
// BackupProvider implementation. Restores whose resolved BackupClass uses
// executionMode "Job" or whose Instance belongs to a different provider are
// skipped, mirroring the backup reconciler's behavior.
type restoreRuntimeReconciler struct {
	client       client.Client
	provider     controller.BackupProvider
	providerName string
}

func setupRestoreReconciler(mgr ctrl.Manager, bp controller.BackupProvider, providerName string) error {
	r := &restoreRuntimeReconciler{
		client:       mgr.GetClient(),
		provider:     bp,
		providerName: providerName,
	}
	b := ctrl.NewControllerManagedBy(mgr).
		For(&backupv1alpha1.Restore{}).
		Named(providerName + "-restore-controller")

	if rw, ok := bp.(controller.RestoreWatcher); ok {
		applyWatchConfigs(b, rw.RestoreWatches())
	}
	return b.Complete(r)
}

func (r *restoreRuntimeReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger := log.FromContext(ctx).WithValues("provider", r.providerName, "restore", req.NamespacedName)

	restore := &backupv1alpha1.Restore{}
	if err := r.client.Get(ctx, req.NamespacedName, restore); err != nil {
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	instance, bc, ours, err := resolveRestoreOwnership(ctx, r.client, restore, r.providerName)
	if err != nil {
		return reconcile.Result{}, err
	}
	if !ours {
		return reconcile.Result{}, nil
	}

	inCtx := controller.NewContext(ctx, r.client, instance, r.providerName)

	if !restore.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(restore, restoreRuntimeFinalizer) {
			return reconcile.Result{}, nil
		}
		done, cerr := r.provider.CleanupRestore(inCtx, restore)
		if cerr != nil {
			if controller.IsWaitError(cerr) {
				return reconcile.Result{RequeueAfter: controller.GetWaitDuration(cerr)}, nil
			}
			return reconcile.Result{}, cerr
		}
		if !done {
			return reconcile.Result{RequeueAfter: defaultBackupRequeue}, nil
		}
		controllerutil.RemoveFinalizer(restore, restoreRuntimeFinalizer)
		if err := r.client.Update(ctx, restore); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	if controllerutil.AddFinalizer(restore, restoreRuntimeFinalizer) {
		if err := r.client.Update(ctx, restore); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{Requeue: true}, nil
	}

	exec, err := r.provider.SyncRestore(inCtx, restore)
	if err != nil {
		if controller.IsWaitError(err) {
			logger.Info("SyncRestore waiting", "reason", err.Error())
			return reconcile.Result{RequeueAfter: controller.GetWaitDuration(err)}, nil
		}
		restore.Status.State = backupv1alpha1.RestoreStateError
		restore.Status.Message = err.Error()
		_ = r.updateStatus(ctx, restore, bc)
		return reconcile.Result{}, err
	}

	applyRestoreExecutionStatus(restore, bc, exec)
	if err := r.updateStatus(ctx, restore, bc); err != nil {
		return reconcile.Result{}, err
	}

	if restore.Status.State == backupv1alpha1.RestoreStateRunning ||
		restore.Status.State == backupv1alpha1.RestoreStatePending {
		return reconcile.Result{RequeueAfter: defaultBackupRequeue}, nil
	}
	return reconcile.Result{}, nil
}

// resolveRestoreOwnership resolves the BackupClass and Instance for a Restore
// and reports whether this provider should handle it. The BackupClass is
// resolved from either spec.dataSource.external.backupClassName or via the
// referenced Backup CR.
func resolveRestoreOwnership(
	ctx context.Context,
	c client.Client,
	restore *backupv1alpha1.Restore,
	providerName string,
) (*corev1alpha1.Instance, *backupv1alpha1.BackupClass, bool, error) {
	bcName, err := backupClassNameForRestore(ctx, c, restore)
	if err != nil {
		return nil, nil, false, err
	}
	if bcName == "" {
		return nil, nil, false, nil
	}
	bc := &backupv1alpha1.BackupClass{}
	if err := c.Get(ctx, client.ObjectKey{Name: bcName}, bc); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil, false, nil
		}
		return nil, nil, false, fmt.Errorf("failed to get BackupClass: %w", err)
	}
	if bc.Spec.ExecutionMode != backupv1alpha1.BackupExecutionModeProviderManaged {
		return nil, bc, false, nil
	}
	instance := &corev1alpha1.Instance{}
	if err := c.Get(ctx, client.ObjectKey{
		Namespace: restore.Namespace,
		Name:      restore.Spec.InstanceName,
	}, instance); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, bc, false, nil
		}
		return nil, bc, false, fmt.Errorf("failed to get Instance: %w", err)
	}
	if instance.Spec.Provider != providerName {
		return instance, bc, false, nil
	}
	return instance, bc, true, nil
}

func backupClassNameForRestore(ctx context.Context, c client.Client, restore *backupv1alpha1.Restore) (string, error) {
	if restore.Spec.DataSource.External != nil && restore.Spec.DataSource.External.BackupClassName != "" {
		return restore.Spec.DataSource.External.BackupClassName, nil
	}
	if restore.Spec.DataSource.BackupName != "" {
		backup := &backupv1alpha1.Backup{}
		if err := c.Get(ctx, client.ObjectKey{
			Namespace: restore.Namespace,
			Name:      restore.Spec.DataSource.BackupName,
		}, backup); err != nil {
			if apierrors.IsNotFound(err) {
				return "", nil
			}
			return "", fmt.Errorf("failed to get referenced Backup: %w", err)
		}
		return backup.Spec.BackupClassName, nil
	}
	return "", nil
}

func (r *restoreRuntimeReconciler) updateStatus(
	ctx context.Context,
	restore *backupv1alpha1.Restore,
	bc *backupv1alpha1.BackupClass,
) error {
	if bc != nil {
		restore.Status.ExecutionMode = bc.Spec.ExecutionMode
	}
	restore.Status.LastObservedGeneration = restore.Generation
	return r.client.Status().Update(ctx, restore)
}

func applyRestoreExecutionStatus(restore *backupv1alpha1.Restore, bc *backupv1alpha1.BackupClass, exec controller.RestoreExecutionStatus) {
	if bc != nil {
		restore.Status.ExecutionMode = bc.Spec.ExecutionMode
	}
	if exec.State != "" {
		restore.Status.State = exec.State
	}
	if exec.Message != "" {
		restore.Status.Message = exec.Message
	}
	if exec.OperatorRestoreRef != nil {
		restore.Status.OperatorRestoreRef = exec.OperatorRestoreRef
	}
	if exec.StartedAt != nil && restore.Status.StartedAt == nil {
		restore.Status.StartedAt = exec.StartedAt
	}
	if exec.CompletedAt != nil {
		restore.Status.CompletedAt = exec.CompletedAt
	}
}
