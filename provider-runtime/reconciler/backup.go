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
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

const (
	backupRuntimeFinalizer = "everest.percona.com/backup-runtime-finalizer"
	// defaultBackupRequeue is used when the operator backup is still in progress
	// and the provider did not return an explicit WaitFor duration.
	defaultBackupRequeue = 10 * time.Second
)

// backupRuntimeReconciler dispatches Backup CRs whose resolved BackupClass uses
// executionMode "ProviderManaged" to the provider's BackupProvider implementation.
// Backups whose class uses executionMode "Job" or whose Instance is owned by a
// different provider are skipped (returning nil) so multiple provider runtimes
// can coexist in the same namespace without trampling each other.
type backupRuntimeReconciler struct {
	client       client.Client
	provider     controller.BackupProvider
	providerName string
}

func setupBackupReconciler(mgr ctrl.Manager, bp controller.BackupProvider, providerName string) error {
	r := &backupRuntimeReconciler{
		client:       mgr.GetClient(),
		provider:     bp,
		providerName: providerName,
	}
	b := ctrl.NewControllerManagedBy(mgr).
		For(&backupv1alpha1.Backup{}).
		Named(providerName + "-backup-controller")

	if bw, ok := bp.(controller.BackupWatcher); ok {
		applyWatchConfigs(b, bw.BackupWatches())
	}
	return b.Complete(r)
}

// applyWatchConfigs wires a list of WatchConfig entries onto the supplied
// builder using the same semantics as the Instance reconciler: Owned=true
// uses Owns() (owner-reference based enqueue), Owned=false uses Watches()
// with the supplied handler.
func applyWatchConfigs(b *builder.Builder, configs []controller.WatchConfig) {
	for _, wc := range configs {
		if wc.Owned {
			if len(wc.Predicates) > 0 {
				b.Owns(wc.Object, builder.WithPredicates(wc.Predicates...))
			} else {
				b.Owns(wc.Object)
			}
			continue
		}
		h := wc.Handler
		if h == nil {
			h = &handler.EnqueueRequestForObject{}
		}
		opts := wc.WatchOptions
		if len(wc.Predicates) > 0 {
			opts = append(opts, builder.WithPredicates(wc.Predicates...))
		}
		b.Watches(wc.Object, h, opts...)
	}
}

func (r *backupRuntimeReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger := log.FromContext(ctx).WithValues("provider", r.providerName, "backup", req.NamespacedName)

	backup := &backupv1alpha1.Backup{}
	if err := r.client.Get(ctx, req.NamespacedName, backup); err != nil {
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	instance, bc, ours, err := resolveBackupOwnership(ctx, r.client, backup, r.providerName)
	if err != nil {
		return reconcile.Result{}, err
	}
	if !ours {
		return reconcile.Result{}, nil
	}

	inCtx := controller.NewContext(ctx, r.client, instance, r.providerName)

	if !backup.DeletionTimestamp.IsZero() {
		if !controllerutil.ContainsFinalizer(backup, backupRuntimeFinalizer) {
			return reconcile.Result{}, nil
		}
		done, cerr := r.provider.CleanupBackup(inCtx, backup)
		if cerr != nil {
			if controller.IsWaitError(cerr) {
				return reconcile.Result{RequeueAfter: controller.GetWaitDuration(cerr)}, nil
			}
			return reconcile.Result{}, cerr
		}
		if !done {
			return reconcile.Result{RequeueAfter: defaultBackupRequeue}, nil
		}
		controllerutil.RemoveFinalizer(backup, backupRuntimeFinalizer)
		if err := r.client.Update(ctx, backup); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	if controllerutil.AddFinalizer(backup, backupRuntimeFinalizer) {
		if err := r.client.Update(ctx, backup); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{Requeue: true}, nil
	}

	exec, err := r.provider.SyncBackup(inCtx, backup)
	if err != nil {
		if controller.IsWaitError(err) {
			logger.Info("SyncBackup waiting", "reason", err.Error())
			return reconcile.Result{RequeueAfter: controller.GetWaitDuration(err)}, nil
		}
		backup.Status.State = backupv1alpha1.BackupStateError
		backup.Status.Message = err.Error()
		_ = r.updateStatus(ctx, backup, bc)
		return reconcile.Result{}, err
	}

	applyBackupExecutionStatus(backup, bc, exec)
	if err := r.updateStatus(ctx, backup, bc); err != nil {
		return reconcile.Result{}, err
	}

	if backup.Status.State == backupv1alpha1.BackupStateRunning ||
		backup.Status.State == backupv1alpha1.BackupStatePending {
		return reconcile.Result{RequeueAfter: defaultBackupRequeue}, nil
	}
	return reconcile.Result{}, nil
}

func resolveBackupOwnership(
	ctx context.Context,
	c client.Client,
	backup *backupv1alpha1.Backup,
	providerName string,
) (*corev1alpha1.Instance, *backupv1alpha1.BackupClass, bool, error) {
	bc := &backupv1alpha1.BackupClass{}
	if err := c.Get(ctx, client.ObjectKey{Name: backup.Spec.BackupClassName}, bc); err != nil {
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
		Namespace: backup.Namespace,
		Name:      backup.Spec.InstanceName,
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

func (r *backupRuntimeReconciler) updateStatus(
	ctx context.Context,
	backup *backupv1alpha1.Backup,
	bc *backupv1alpha1.BackupClass,
) error {
	if bc != nil {
		backup.Status.ExecutionMode = bc.Spec.ExecutionMode
	}
	backup.Status.LastObservedGeneration = backup.Generation
	return r.client.Status().Update(ctx, backup)
}

func applyBackupExecutionStatus(backup *backupv1alpha1.Backup, bc *backupv1alpha1.BackupClass, exec controller.BackupExecutionStatus) {
	if bc != nil {
		backup.Status.ExecutionMode = bc.Spec.ExecutionMode
	}
	if exec.State != "" {
		backup.Status.State = exec.State
	}
	if exec.Message != "" {
		backup.Status.Message = exec.Message
	}
	if exec.OperatorBackupRef != nil {
		backup.Status.OperatorBackupRef = exec.OperatorBackupRef
	}
	if exec.StartedAt != nil && backup.Status.StartedAt == nil {
		backup.Status.StartedAt = exec.StartedAt
	}
	if exec.CompletedAt != nil {
		backup.Status.CompletedAt = exec.CompletedAt
	}
	if exec.Size != nil {
		backup.Status.Size = exec.Size
	}
}
