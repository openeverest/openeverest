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

// ListBackupStorages returns backup storages filtered by RBAC permissions.
func (h *rbacHandler) ListBackupStorages(ctx context.Context, cluster, namespace string) (*backupv1alpha1.BackupStorageList, error) {
	list, err := h.next.ListBackupStorages(ctx, cluster, namespace)
	if err != nil {
		return nil, fmt.Errorf("ListBackupStorages failed: %w", err)
	}
	filtered := make([]backupv1alpha1.BackupStorage, 0, len(list.Items))
	for _, bs := range list.Items {
		object := rbac.ClusterNamespacedObjectName(cluster, bs.GetNamespace(), bs.GetName())
		if err := h.enforce(ctx, rbac.ResourceBackupStorages, rbac.ActionRead, object); errors.Is(err, ErrInsufficientPermissions) {
			continue
		} else if err != nil {
			return nil, fmt.Errorf("enforce failed: %w", err)
		}
		filtered = append(filtered, bs)
	}
	list.Items = filtered
	return list, nil
}

// GetBackupStorage returns a backup storage, gated by RBAC.
func (h *rbacHandler) GetBackupStorage(ctx context.Context, cluster, namespace, name string) (*backupv1alpha1.BackupStorage, error) {
	object := rbac.ClusterNamespacedObjectName(cluster, namespace, name)
	if err := h.enforce(ctx, rbac.ResourceBackupStorages, rbac.ActionRead, object); err != nil {
		return nil, err
	}
	return h.next.GetBackupStorage(ctx, cluster, namespace, name)
}

// CreateBackupStorage creates a backup storage, gated by RBAC.
func (h *rbacHandler) CreateBackupStorage(ctx context.Context, cluster string, bs *backupv1alpha1.BackupStorage) (*backupv1alpha1.BackupStorage, error) {
	object := rbac.ClusterNamespacedObjectName(cluster, bs.GetNamespace(), bs.GetName())
	if err := h.enforce(ctx, rbac.ResourceBackupStorages, rbac.ActionCreate, object); err != nil {
		return nil, err
	}
	return h.next.CreateBackupStorage(ctx, cluster, bs)
}

// UpdateBackupStorage updates a backup storage, gated by RBAC.
func (h *rbacHandler) UpdateBackupStorage(ctx context.Context, cluster string, bs *backupv1alpha1.BackupStorage) (*backupv1alpha1.BackupStorage, error) {
	object := rbac.ClusterNamespacedObjectName(cluster, bs.GetNamespace(), bs.GetName())
	if err := h.enforce(ctx, rbac.ResourceBackupStorages, rbac.ActionUpdate, object); err != nil {
		return nil, err
	}
	return h.next.UpdateBackupStorage(ctx, cluster, bs)
}

// DeleteBackupStorage deletes a backup storage, gated by RBAC.
func (h *rbacHandler) DeleteBackupStorage(ctx context.Context, cluster, namespace, name string) error {
	object := rbac.ClusterNamespacedObjectName(cluster, namespace, name)
	if err := h.enforce(ctx, rbac.ResourceBackupStorages, rbac.ActionDelete, object); err != nil {
		return err
	}
	return h.next.DeleteBackupStorage(ctx, cluster, namespace, name)
}
