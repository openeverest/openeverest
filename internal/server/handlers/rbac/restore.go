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

package rbac

import (
	"context"
	"fmt"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

// GetRestore returns a specific restore by namespace and name, gated by RBAC
// on the instance it targets. A restore not found and a restore the caller
// isn't authorized for both come back as ErrInsufficientPermissions, so a
// name cannot be probed for existence.
func (h *rbacHandler) GetRestore(ctx context.Context, cluster, namespace, name string) (*backupv1alpha1.Restore, error) {
	restore, err := h.next.GetRestore(ctx, cluster, namespace, name)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, ErrInsufficientPermissions
		}
		return nil, fmt.Errorf("GetRestore failed: %w", err)
	}
	object := rbac.ClusterNamespacedObjectName(cluster, namespace, restore.Spec.InstanceRef.Name)
	if err := h.enforce(ctx, rbac.ResourceRestores, rbac.ActionRead, object); err != nil {
		return nil, err
	}
	return restore, nil
}

// CreateRestore creates a new restore, gated by RBAC on the instance it targets.
// InstanceRef is a value, not a pointer, so a request body with no instanceRef decodes to an empty name.
// Fail closed on that rather than build an object like "cluster/ns/" that a namespace-wide wildcard grant
// would match despite naming no real instance.
func (h *rbacHandler) CreateRestore(ctx context.Context, cluster string, restore *backupv1alpha1.Restore) (*backupv1alpha1.Restore, error) {
	if restore.Spec.InstanceRef.Name == "" {
		return nil, ErrInsufficientPermissions
	}
	object := rbac.ClusterNamespacedObjectName(cluster, restore.GetNamespace(), restore.Spec.InstanceRef.Name)
	if err := h.enforce(ctx, rbac.ResourceRestores, rbac.ActionCreate, object); err != nil {
		return nil, err
	}
	if err := h.ensureDataSourceReadable(ctx, cluster, restore.GetNamespace(), restore.Spec.DataSource); err != nil {
		return nil, err
	}
	return h.next.CreateRestore(ctx, cluster, restore)
}

// DeleteRestore deletes a restore by namespace and name, gated by RBAC on the
// instance it targets. Same not-found/denied collapsing as GetRestore.
func (h *rbacHandler) DeleteRestore(ctx context.Context, cluster, namespace, name string) error {
	restore, err := h.next.GetRestore(ctx, cluster, namespace, name)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return ErrInsufficientPermissions
		}
		return fmt.Errorf("GetRestore failed: %w", err)
	}
	object := rbac.ClusterNamespacedObjectName(cluster, namespace, restore.Spec.InstanceRef.Name)
	if err := h.enforce(ctx, rbac.ResourceRestores, rbac.ActionDelete, object); err != nil {
		return err
	}
	return h.next.DeleteRestore(ctx, cluster, namespace, name)
}

// ensureDataSourceReadable checks the caller may read the backups a restore or
// a new instance is seeded from. Backups are keyed by the instance they belong
// to, so a backup reference is resolved through GetBackup, which also collapses
// a missing backup into ErrInsufficientPermissions.
func (h *rbacHandler) ensureDataSourceReadable(ctx context.Context, cluster, namespace string, dataSource backupv1alpha1.DataSource) error {
	if dataSource.Backup != nil {
		if _, err := h.GetBackup(ctx, cluster, namespace, dataSource.Backup.BackupRef.Name); err != nil {
			return err
		}
	}
	if pitr := dataSource.PointInTime; pitr != nil {
		storageObject := rbac.ClusterNamespacedObjectName(cluster, namespace, pitr.Source.StorageRef.Name)
		if err := h.enforce(ctx, rbac.ResourceBackupStorages, rbac.ActionRead, storageObject); err != nil {
			return err
		}
		var sourceInstance string
		if pitr.Source.InstanceRef != nil {
			sourceInstance = pitr.Source.InstanceRef.Name
		}
		backupsObject := rbac.ClusterNamespacedObjectName(cluster, namespace, sourceInstance)
		if err := h.enforce(ctx, rbac.ResourceBackups, rbac.ActionRead, backupsObject); err != nil {
			return err
		}
	}
	return nil
}
