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
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

// RestoreReconciler reconciles Restore resources. The provider that executes
// the Restore's class reconciles the restore itself; this controller only
// labels the Restore with its Instance and reports a Restore whose source or
// class does not exist yet.
type RestoreReconciler struct {
	Client client.Client
	Scheme *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (r *RestoreReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("Restore").
		For(&backupv1alpha1.Restore{}).
		Watches(&backupv1alpha1.BackupClass{},
			operationsFor[backupv1alpha1.RestoreList](mgr.GetClient(), false, restoreWaitsForClass),
			builder.WithPredicates(referentAppearedOrGone()),
		).
		Watches(&backupv1alpha1.Backup{},
			operationsFor[backupv1alpha1.RestoreList](mgr.GetClient(), true, restoreNamesBackup),
			builder.WithPredicates(referentAppearedOrGone()),
		).
		Watches(&corev1alpha1.Instance{},
			operationsFor[backupv1alpha1.RestoreList](mgr.GetClient(), true, restoreNamesInstance),
			builder.WithPredicates(referentAppearedOrGone()),
		).
		Complete(r)
}

//+kubebuilder:rbac:groups=backup.openeverest.io,resources=restores,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=backup.openeverest.io,resources=restores/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=backup.openeverest.io,resources=backupclasses,verbs=get;list;watch
//+kubebuilder:rbac:groups=backup.openeverest.io,resources=backups,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *RestoreReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).
		WithName("RestoreReconciler").
		WithValues(
			"name", req.Name,
			"namespace", req.Namespace,
		)
	logger.Info("Reconciling")
	defer func() {
		logger.Info("Reconciled")
	}()

	restore := &backupv1alpha1.Restore{}
	if err := r.Client.Get(ctx, req.NamespacedName, restore); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !restore.GetDeletionTimestamp().IsZero() {
		return ctrl.Result{}, nil
	}

	// Ensure the instance name label is set on the Restore resource.
	if err := r.ensureInstanceNameLabel(ctx, restore); err != nil {
		return ctrl.Result{}, err
	}

	if restore.Status.State == backupv1alpha1.RestoreStateSucceeded ||
		restore.Status.State == backupv1alpha1.RestoreStateFailed {
		return ctrl.Result{}, nil
	}

	missing, err := r.resolveBackupClass(ctx, restore)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to resolve backup class: %w", err)
	}
	if missing == "" {
		// The class resolves: the provider that executes it takes over, and
		// this controller leaves the Restore's status alone.
		return ctrl.Result{}, nil
	}

	// Nobody else knows which controller owns the Restore until its class
	// resolves, so a Restore that has not started waits here.
	if restore.Status.State != "" && restore.Status.State != backupv1alpha1.RestoreStatePending {
		return ctrl.Result{}, nil
	}
	restore.Status.State = backupv1alpha1.RestoreStatePending
	restore.Status.Message = missing
	restore.Status.LastObservedGeneration = restore.GetGeneration()
	return ctrl.Result{}, r.Client.Status().Update(ctx, restore)
}

// ensureInstanceNameLabel ensures that the Restore resource has the instance name
// label, used for filtering restores by instance using label selectors.
func (r *RestoreReconciler) ensureInstanceNameLabel(ctx context.Context, restore *backupv1alpha1.Restore) error {
	labels := restore.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}

	if labels[common.InstanceNameLabel] == restore.Spec.InstanceRef.Name {
		return nil
	}

	labels[common.InstanceNameLabel] = restore.Spec.InstanceRef.Name
	restore.SetLabels(labels)
	if err := r.Client.Update(ctx, restore); err != nil {
		return fmt.Errorf("failed to update instance name label: %w", err)
	}

	return nil
}

// resolveBackupClass resolves the BackupClass to use for a Restore and returns
// what is missing for it to resolve, or "" when the class exists.
//
// This is the *read* class: it describes how the data was written and therefore
// how it must be read, and its executor reconciles the Restore. It may
// legitimately differ from the target Instance's .spec.backup.classRef, which
// governs how that Instance writes new backups -- an Instance that changed
// classes must still be able to restore its older backups.
//
//   - type=Backup      -> the source Backup's classRef.
//   - type=PointInTime -> the classRef of the Instance owning the stream, since
//     a stream has no Backup CR of its own.
func (r *RestoreReconciler) resolveBackupClass(
	ctx context.Context,
	restore *backupv1alpha1.Restore,
) (string, error) {
	ds := restore.Spec.DataSource

	var backupClassName string
	switch ds.Type {
	case backupv1alpha1.DataSourceTypeBackup:
		if ds.Backup == nil {
			return "", fmt.Errorf("dataSource.backup is required when type is %q", ds.Type)
		}
		backup := &backupv1alpha1.Backup{}
		if err := r.Client.Get(ctx, client.ObjectKey{
			Name:      ds.Backup.BackupRef.Name,
			Namespace: restore.GetNamespace(),
		}, backup); err != nil {
			if apierrors.IsNotFound(err) {
				return fmt.Sprintf("backup %s not found in %s", ds.Backup.BackupRef.Name, restore.GetNamespace()), nil
			}
			return "", fmt.Errorf("failed to get backup %q: %w", ds.Backup.BackupRef.Name, err)
		}
		backupClassName = backup.Spec.ClassRef.Name

	case backupv1alpha1.DataSourceTypePointInTime:
		name := controller.RestoreStreamInstanceName(restore)
		instance := &corev1alpha1.Instance{}
		if err := r.Client.Get(ctx, client.ObjectKey{
			Name:      name,
			Namespace: restore.GetNamespace(),
		}, instance); err != nil {
			if apierrors.IsNotFound(err) {
				return fmt.Sprintf("instance %s not found in %s", name, restore.GetNamespace()), nil
			}
			return "", fmt.Errorf("failed to get instance %q: %w", name, err)
		}
		if instance.Spec.Backup == nil || instance.Spec.Backup.ClassRef.Name == "" {
			return "", fmt.Errorf(
				"instance %q has no backup class configured; cannot resolve how to read its point-in-time stream",
				name,
			)
		}
		backupClassName = instance.Spec.Backup.ClassRef.Name

	default:
		return "", fmt.Errorf("unsupported dataSource type %q", ds.Type)
	}

	if err := r.Client.Get(ctx, client.ObjectKey{
		Name: backupClassName,
	}, &backupv1alpha1.BackupClass{}); err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Sprintf("backup class %s not found", backupClassName), nil
		}
		return "", fmt.Errorf("failed to get backup class %q: %w", backupClassName, err)
	}

	return "", nil
}
