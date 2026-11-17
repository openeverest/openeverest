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
	"errors"
	"fmt"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

// ListBackupImports returns a list of backup imports, gated by RBAC.
// Only items with read on the backup import and read on its referenced
// backup storage are returned; others are filtered out.
func (h *rbacHandler) ListBackupImports(ctx context.Context, cluster, namespace string) (*backupv1alpha1.BackupImportList, error) {
	list, err := h.next.ListBackupImports(ctx, cluster, namespace)
	if err != nil {
		return nil, err
	}
	filtered := []backupv1alpha1.BackupImport{}
	for _, bi := range list.Items {
		if err := h.enforceBackupImportRead(ctx, cluster, &bi); errors.Is(err, ErrInsufficientPermissions) {
			continue
		} else if err != nil {
			return nil, fmt.Errorf("enforce failed: %w", err)
		}
		filtered = append(filtered, bi)
	}
	list.Items = filtered
	return list, nil
}

// GetBackupImport returns a backup import, gated by RBAC.
// Requires read on the backup import and read on its referenced backup storage.
func (h *rbacHandler) GetBackupImport(ctx context.Context, cluster, namespace, name string) (*backupv1alpha1.BackupImport, error) {
	object := rbac.ClusterNamespacedObjectName(cluster, namespace, name)
	if err := h.enforce(ctx, rbac.ResourceBackupImports, rbac.ActionRead, object); err != nil {
		return nil, err
	}
	bi, err := h.next.GetBackupImport(ctx, cluster, namespace, name)
	if err != nil {
		return nil, err
	}
	storage := rbac.ClusterNamespacedObjectName(cluster, namespace, bi.Spec.StorageRef.Name)
	if err := h.enforce(ctx, rbac.ResourceBackupStorages, rbac.ActionRead, storage); err != nil {
		return nil, err
	}
	return bi, nil
}

// CreateBackupImport creates a backup import, gated by RBAC.
// Requires namespace-wide create on backup imports and read on the referenced
// backup storage, so users cannot interfere with storages they cannot access.
func (h *rbacHandler) CreateBackupImport(ctx context.Context, cluster string, backupImport *backupv1alpha1.BackupImport) (*backupv1alpha1.BackupImport, error) {
	namespace := backupImport.GetNamespace()
	namespaceWide := rbac.ClusterNamespacedObjectName(cluster, namespace, "*")
	if err := h.enforce(ctx, rbac.ResourceBackupImports, rbac.ActionCreate, namespaceWide); err != nil {
		return nil, err
	}
	storage := rbac.ClusterNamespacedObjectName(cluster, namespace, backupImport.Spec.StorageRef.Name)
	if err := h.enforce(ctx, rbac.ResourceBackupStorages, rbac.ActionRead, storage); err != nil {
		return nil, err
	}
	return h.next.CreateBackupImport(ctx, cluster, backupImport)
}

// DeleteBackupImport deletes a backup import, gated by RBAC.
func (h *rbacHandler) DeleteBackupImport(ctx context.Context, cluster, namespace, name string) error {
	object := rbac.ClusterNamespacedObjectName(cluster, namespace, name)
	if err := h.enforce(ctx, rbac.ResourceBackupImports, rbac.ActionDelete, object); err != nil {
		return err
	}
	return h.next.DeleteBackupImport(ctx, cluster, namespace, name)
}

// enforceBackupImportRead requires read on the backup import and its referenced storage.
// Storage read is checked, as status errors may leak storage details.
func (h *rbacHandler) enforceBackupImportRead(ctx context.Context, cluster string, bi *backupv1alpha1.BackupImport) error {
	object := rbac.ClusterNamespacedObjectName(cluster, bi.GetNamespace(), bi.GetName())
	if err := h.enforce(ctx, rbac.ResourceBackupImports, rbac.ActionRead, object); err != nil {
		return err
	}
	storage := rbac.ClusterNamespacedObjectName(cluster, bi.GetNamespace(), bi.Spec.StorageRef.Name)
	return h.enforce(ctx, rbac.ResourceBackupStorages, rbac.ActionRead, storage)
}
