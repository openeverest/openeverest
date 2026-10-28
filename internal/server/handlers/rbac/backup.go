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

// Package rbac provides the RBAC handler.
package rbac

import (
	"context"
	"errors"
	"fmt"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	api "github.com/openeverest/openeverest/v2/internal/server/api"
	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

// ListBackups returns only the backups the caller may read:
//   - Instance origin: backups read on the Instance.
//   - External origin: namespace-wide backups read and read on its BackupStorage.
func (h *rbacHandler) ListBackups(ctx context.Context, cluster, namespace string, params *api.ListBackupsParams) (*backupv1alpha1.BackupList, error) {
	list, err := h.next.ListBackups(ctx, cluster, namespace, params)
	if err != nil {
		return nil, fmt.Errorf("ListBackups failed: %w", err)
	}

	filtered := make([]backupv1alpha1.Backup, 0, len(list.Items))
	for _, b := range list.Items {
		if err := h.enforceReadBackup(ctx, cluster, &b); errors.Is(err, ErrInsufficientPermissions) {
			continue
		} else if err != nil {
			return nil, fmt.Errorf("enforce failed: %w", err)
		}

		filtered = append(filtered, b)
	}

	list.Items = filtered
	return list, nil
}

// GetBackup returns the backup if the caller may read it:
//   - Instance origin: backups read on the Instance.
//   - External origin: namespace-wide backups read and read on its BackupStorage.
//
// Not found and denied both return ErrInsufficientPermissions to prevent name
// probing.
func (h *rbacHandler) GetBackup(ctx context.Context, cluster, namespace, name string) (*backupv1alpha1.Backup, error) {
	backup, err := h.next.GetBackup(ctx, cluster, namespace, name)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, ErrInsufficientPermissions
		}
		return nil, fmt.Errorf("GetBackup failed: %w", err)
	}
	if err := h.enforceReadBackup(ctx, cluster, backup); err != nil {
		return nil, err
	}
	return backup, nil
}

// CreateBackup creates a backup if the caller may create it:
//   - Instance origin: backups create on the Instance.
//   - External origin: namespace-wide backups create.
//
// Either way the caller must also read the BackupStorage it writes to and the
// BackupClass it runs, since the class's job permissions become the
// ServiceAccount the backup job runs as.
func (h *rbacHandler) CreateBackup(ctx context.Context, cluster string, backup *backupv1alpha1.Backup) (*backupv1alpha1.Backup, error) {
	namespace := backup.GetNamespace()
	// External backups have no Instance to grant on, so they need a
	// namespace-wide backups grant.
	object := rbac.ClusterNamespacedObjectName(cluster, namespace, "*")
	if ref := backup.Spec.Origin.InstanceRef; ref != nil {
		object = rbac.ClusterNamespacedObjectName(cluster, namespace, ref.Name)
	}
	if err := h.enforce(ctx, rbac.ResourceBackups, rbac.ActionCreate, object); err != nil {
		return nil, err
	}
	storageObject := rbac.ClusterNamespacedObjectName(cluster, namespace, backup.Spec.StorageRef.Name)
	if err := h.enforce(ctx, rbac.ResourceBackupStorages, rbac.ActionRead, storageObject); err != nil {
		return nil, err
	}
	classObject := rbac.ClusterObjectName(cluster, backup.Spec.ClassRef.Name)
	if err := h.enforce(ctx, rbac.ResourceBackupClasses, rbac.ActionRead, classObject); err != nil {
		return nil, err
	}
	return h.next.CreateBackup(ctx, cluster, backup)
}

// DeleteBackup deletes a backup if the caller may delete it:
//   - Instance origin: backups delete on the Instance.
//   - External origin: namespace-wide backups delete. No storage check, since
//     external backups are always Retain and deletion leaves the data intact.
//
// Not found and denied both return ErrInsufficientPermissions to prevent name
// probing.
func (h *rbacHandler) DeleteBackup(ctx context.Context, cluster, namespace, name string, params *api.DeleteBackupParams) error {
	backup, err := h.next.GetBackup(ctx, cluster, namespace, name)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return ErrInsufficientPermissions
		}
		return fmt.Errorf("GetBackup failed: %w", err)
	}
	object := rbac.ClusterNamespacedObjectName(cluster, namespace, "*")
	if ref := backup.Spec.Origin.InstanceRef; ref != nil {
		object = rbac.ClusterNamespacedObjectName(cluster, namespace, ref.Name)
	}
	if err := h.enforce(ctx, rbac.ResourceBackups, rbac.ActionDelete, object); err != nil {
		return err
	}
	return h.next.DeleteBackup(ctx, cluster, namespace, name, params)
}

// enforceReadBackup returns error if the caller may not read the backup:
//   - Instance origin: backups read on the Instance. Backups are keyed by
//     their Instance because their own names are usually generated and
//     cannot be granted in advance.
//   - External origin: namespace-wide backups read and its BackupStorage read.
func (h *rbacHandler) enforceReadBackup(ctx context.Context, cluster string, backup *backupv1alpha1.Backup) error {
	namespace := backup.GetNamespace()
	// Instance backups hold the instance's own data, so the instance grant is
	// enough; requiring storage access would hide them once it is revoked.
	if ref := backup.Spec.Origin.InstanceRef; ref != nil {
		object := rbac.ClusterNamespacedObjectName(cluster, namespace, ref.Name)
		return h.enforce(ctx, rbac.ResourceBackups, rbac.ActionRead, object)
	}
	// External backups have no Instance to grant on, so they need a
	// namespace-wide backups grant. Storage read hides backup paths in the
	// storage from callers without access to it.
	storageObject := rbac.ClusterNamespacedObjectName(cluster, namespace, backup.Spec.StorageRef.Name)
	if err := h.enforce(ctx, rbac.ResourceBackupStorages, rbac.ActionRead, storageObject); err != nil {
		return err
	}
	namespaceWide := rbac.ClusterNamespacedObjectName(cluster, namespace, "*")
	return h.enforce(ctx, rbac.ResourceBackups, rbac.ActionRead, namespaceWide)
}
