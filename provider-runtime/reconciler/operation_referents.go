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
	"slices"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

// reconcileWithoutInstance handles a Backup whose Instance does not exist. A
// finished Backup is left alone, a running one fails once an uncached read
// confirms the Instance is gone, and any other waits in Pending until the
// Instance appears.
func (r *backupRuntimeReconciler) reconcileWithoutInstance(
	ctx context.Context,
	backup *backupv1alpha1.Backup,
	bc *backupv1alpha1.BackupClass,
) (reconcile.Result, error) {
	if !backup.DeletionTimestamp.IsZero() {
		return reconcile.Result{}, nil
	}
	name := backup.Spec.Origin.InstanceRef.Name
	switch {
	case !backupUnfinished(backup):
		return reconcile.Result{}, nil
	case backup.Status.State == backupv1alpha1.BackupStateRunning:
		gone, err := instanceGone(ctx, r.reader, backup.Namespace, name)
		if err != nil || !gone {
			return reconcile.Result{RequeueAfter: defaultBackupRequeue}, err
		}
		backup.Status.State = backupv1alpha1.BackupStateFailed
		backup.Status.Message = fmt.Sprintf("instance %s was deleted while the backup was running", name)
	default:
		backup.Status.State = backupv1alpha1.BackupStatePending
		backup.Status.Message = fmt.Sprintf("instance %s not found in %s", name, backup.Namespace)
	}
	return reconcile.Result{}, r.updateStatus(ctx, backup, bc)
}

// reconcileWithoutInstance handles a Restore whose target Instance does not
// exist, like the Backup counterpart.
func (r *restoreRuntimeReconciler) reconcileWithoutInstance(
	ctx context.Context,
	restore *backupv1alpha1.Restore,
	bc *backupv1alpha1.BackupClass,
) (reconcile.Result, error) {
	name := restore.Spec.InstanceRef.Name
	switch {
	case !restoreUnfinished(restore):
		return reconcile.Result{}, nil
	case restore.Status.State == backupv1alpha1.RestoreStateRunning:
		gone, err := instanceGone(ctx, r.reader, restore.Namespace, name)
		if err != nil || !gone {
			return reconcile.Result{RequeueAfter: defaultBackupRequeue}, err
		}
		restore.Status.State = backupv1alpha1.RestoreStateFailed
		restore.Status.Message = fmt.Sprintf("instance %s was deleted while the restore was running", name)
	default:
		restore.Status.State = backupv1alpha1.RestoreStatePending
		restore.Status.Message = fmt.Sprintf("instance %s not found in %s", name, restore.Namespace)
	}
	return reconcile.Result{}, r.updateStatus(ctx, restore, bc)
}

// instanceGone confirms with a read that bypasses the cache that the Instance
// does not exist, so a lagging cache never fails a running operation.
func instanceGone(ctx context.Context, reader client.Reader, namespace, name string) (bool, error) {
	err := reader.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, &corev1alpha1.Instance{})
	if apierrors.IsNotFound(err) {
		return true, nil
	}
	return false, err
}

func backupUnfinished(b *backupv1alpha1.Backup) bool {
	return b.Status.State != backupv1alpha1.BackupStateSucceeded && b.Status.State != backupv1alpha1.BackupStateFailed
}

// backupsForInstance enqueues the unfinished Backups of the Instance.
func (r *backupRuntimeReconciler) backupsForInstance(ctx context.Context, obj client.Object) []reconcile.Request {
	return r.backupsMatching(ctx, obj.GetNamespace(), func(b *backupv1alpha1.Backup) bool {
		return b.Spec.Origin.InstanceRef != nil && b.Spec.Origin.InstanceRef.Name == obj.GetName()
	})
}

// backupsForClass enqueues the unfinished Backups that name the BackupClass,
// so the runtime takes over a Backup that waited for its class.
func (r *backupRuntimeReconciler) backupsForClass(ctx context.Context, obj client.Object) []reconcile.Request {
	return r.backupsMatching(ctx, "", func(b *backupv1alpha1.Backup) bool {
		return b.Spec.ClassRef.Name == obj.GetName()
	})
}

// backupsForStorage enqueues the unfinished Backups that write to the
// BackupStorage.
func (r *backupRuntimeReconciler) backupsForStorage(ctx context.Context, obj client.Object) []reconcile.Request {
	return r.backupsMatching(ctx, obj.GetNamespace(), func(b *backupv1alpha1.Backup) bool {
		return b.Spec.StorageRef.Name == obj.GetName()
	})
}

// backupsForSecret enqueues the unfinished Backups whose storage keeps its
// credentials in the Secret.
func (r *backupRuntimeReconciler) backupsForSecret(ctx context.Context, obj client.Object) []reconcile.Request {
	storages := storagesUsingSecret(ctx, r.client, obj.GetNamespace(), obj.GetName())
	if len(storages) == 0 {
		return nil
	}
	return r.backupsMatching(ctx, obj.GetNamespace(), func(b *backupv1alpha1.Backup) bool {
		return slices.Contains(storages, b.Spec.StorageRef.Name)
	})
}

// backupsMatching lists the unfinished Backups in namespace (every namespace
// when empty) that keep selects.
func (r *backupRuntimeReconciler) backupsMatching(
	ctx context.Context,
	namespace string,
	keep func(*backupv1alpha1.Backup) bool,
) []reconcile.Request {
	list := &backupv1alpha1.BackupList{}
	if err := r.client.List(ctx, list, client.InNamespace(namespace)); err != nil {
		log.FromContext(ctx).Error(err, "Listing backups to enqueue failed")
		return nil
	}
	var reqs []reconcile.Request
	for i := range list.Items {
		b := &list.Items[i]
		if backupUnfinished(b) && keep(b) {
			reqs = append(reqs, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(b)})
		}
	}
	return reqs
}

// restoresForInstance enqueues the unfinished Restores into the Instance.
func (r *restoreRuntimeReconciler) restoresForInstance(ctx context.Context, obj client.Object) []reconcile.Request {
	list := &backupv1alpha1.RestoreList{}
	if err := r.client.List(ctx, list,
		client.InNamespace(obj.GetNamespace()),
		client.MatchingFields{controller.IndexRestoreInstanceName: obj.GetName()},
	); err != nil {
		log.FromContext(ctx).Error(err, "Listing restores to enqueue failed")
		return nil
	}
	var reqs []reconcile.Request
	for i := range list.Items {
		rs := &list.Items[i]
		if restoreUnfinished(rs) {
			reqs = append(reqs, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(rs)})
		}
	}
	return reqs
}

// restoresForStorage enqueues the unfinished point-in-time Restores that read
// their stream from the BackupStorage.
func (r *restoreRuntimeReconciler) restoresForStorage(ctx context.Context, obj client.Object) []reconcile.Request {
	return r.restoresMatching(ctx, obj.GetNamespace(), func(rs *backupv1alpha1.Restore) bool {
		pitr := rs.Spec.DataSource.PointInTime
		return pitr != nil && pitr.Source.StorageRef.Name == obj.GetName()
	})
}

// restoresForSource enqueues the unfinished Restores of the Backup, whose
// class decides which runtime owns them.
func (r *restoreRuntimeReconciler) restoresForSource(ctx context.Context, obj client.Object) []reconcile.Request {
	return r.restoresMatching(ctx, obj.GetNamespace(), func(rs *backupv1alpha1.Restore) bool {
		return rs.Spec.DataSource.Backup != nil && rs.Spec.DataSource.Backup.BackupRef.Name == obj.GetName()
	})
}

// restoresForClass enqueues the Restores still waiting for a referent; the
// class is reached through the source Backup or Instance, so it cannot be
// matched by name.
func (r *restoreRuntimeReconciler) restoresForClass(ctx context.Context, _ client.Object) []reconcile.Request {
	return r.restoresMatching(ctx, "", func(rs *backupv1alpha1.Restore) bool {
		return rs.Status.State == "" || rs.Status.State == backupv1alpha1.RestoreStatePending
	})
}

// restoresMatching lists the unfinished Restores in namespace (every
// namespace when empty) that keep selects.
func (r *restoreRuntimeReconciler) restoresMatching(
	ctx context.Context,
	namespace string,
	keep func(*backupv1alpha1.Restore) bool,
) []reconcile.Request {
	list := &backupv1alpha1.RestoreList{}
	if err := r.client.List(ctx, list, client.InNamespace(namespace)); err != nil {
		log.FromContext(ctx).Error(err, "Listing restores to enqueue failed")
		return nil
	}
	var reqs []reconcile.Request
	for i := range list.Items {
		rs := &list.Items[i]
		if restoreUnfinished(rs) && keep(rs) {
			reqs = append(reqs, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(rs)})
		}
	}
	return reqs
}

func restoreUnfinished(rs *backupv1alpha1.Restore) bool {
	return rs.Status.State != backupv1alpha1.RestoreStateSucceeded && rs.Status.State != backupv1alpha1.RestoreStateFailed
}

// importsForClass enqueues the unfinished BackupImports that name the
// BackupClass, so the runtime takes over an import that waited for its class.
func (r *backupImportReconciler) importsForClass(ctx context.Context, obj client.Object) []reconcile.Request {
	return r.importsMatching(ctx, "", func(imp *backupv1alpha1.BackupImport) bool {
		return imp.Spec.ClassRef.Name == obj.GetName()
	})
}

// importsForStorage enqueues the unfinished BackupImports that read the
// BackupStorage.
func (r *backupImportReconciler) importsForStorage(ctx context.Context, obj client.Object) []reconcile.Request {
	return r.importsMatching(ctx, obj.GetNamespace(), func(imp *backupv1alpha1.BackupImport) bool {
		return imp.Spec.StorageRef.Name == obj.GetName()
	})
}

// importsForSecret enqueues the unfinished BackupImports whose storage keeps
// its credentials in the Secret.
func (r *backupImportReconciler) importsForSecret(ctx context.Context, obj client.Object) []reconcile.Request {
	storages := storagesUsingSecret(ctx, r.client, obj.GetNamespace(), obj.GetName())
	if len(storages) == 0 {
		return nil
	}
	return r.importsMatching(ctx, obj.GetNamespace(), func(imp *backupv1alpha1.BackupImport) bool {
		return slices.Contains(storages, imp.Spec.StorageRef.Name)
	})
}

// importsMatching lists the unfinished BackupImports in namespace (every
// namespace when empty) that keep selects.
func (r *backupImportReconciler) importsMatching(
	ctx context.Context,
	namespace string,
	keep func(*backupv1alpha1.BackupImport) bool,
) []reconcile.Request {
	list := &backupv1alpha1.BackupImportList{}
	if err := r.client.List(ctx, list, client.InNamespace(namespace)); err != nil {
		log.FromContext(ctx).Error(err, "Listing backup imports to enqueue failed")
		return nil
	}
	var reqs []reconcile.Request
	for i := range list.Items {
		imp := &list.Items[i]
		if imp.Status.State == backupv1alpha1.BackupImportStateSucceeded || imp.Status.State == backupv1alpha1.BackupImportStateFailed {
			continue
		}
		if keep(imp) {
			reqs = append(reqs, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(imp)})
		}
	}
	return reqs
}
