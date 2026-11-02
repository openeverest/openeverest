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
	"encoding/json"
	"errors"
	"fmt"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	valhandler "github.com/openeverest/openeverest/v2/internal/server/handlers/validation"
	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

// ListBackupStorages returns backup storages filtered by RBAC permissions.
func (h *rbacHandler) ListBackupStorages(ctx context.Context, cluster, namespace, backupClass, backupStorageClass string) (*backupv1alpha1.BackupStorageList, error) {
	list, err := h.next.ListBackupStorages(ctx, cluster, namespace, backupClass, backupStorageClass)
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

// CreateBackupStorage creates a backup storage, gated by RBAC. It also
// requires read on the class it references and on every Secret it names, so
// the caller cannot hand executors credentials they may not read.
func (h *rbacHandler) CreateBackupStorage(ctx context.Context, cluster string, bs *backupv1alpha1.BackupStorage) (*backupv1alpha1.BackupStorage, error) {
	object := rbac.ClusterNamespacedObjectName(cluster, bs.GetNamespace(), bs.GetName())
	if err := h.enforce(ctx, rbac.ResourceBackupStorages, rbac.ActionCreate, object); err != nil {
		return nil, err
	}
	if err := h.enforceStorageReferences(ctx, cluster, bs.GetNamespace(), bs.Spec.BackupStorageClassRef.Name, bs.Spec.SecretNames()); err != nil {
		return nil, err
	}
	return h.next.CreateBackupStorage(ctx, cluster, bs)
}

// UpdateBackupStorage updates a backup storage, gated by RBAC like
// CreateBackupStorage.
func (h *rbacHandler) UpdateBackupStorage(ctx context.Context, cluster string, bs *backupv1alpha1.BackupStorage) (*backupv1alpha1.BackupStorage, error) {
	object := rbac.ClusterNamespacedObjectName(cluster, bs.GetNamespace(), bs.GetName())
	if err := h.enforce(ctx, rbac.ResourceBackupStorages, rbac.ActionUpdate, object); err != nil {
		return nil, err
	}
	if err := h.enforceStorageReferences(ctx, cluster, bs.GetNamespace(), bs.Spec.BackupStorageClassRef.Name, bs.Spec.SecretNames()); err != nil {
		return nil, err
	}
	return h.next.UpdateBackupStorage(ctx, cluster, bs)
}

// PatchBackupStorage patches a backup storage, gated by RBAC. A patch that
// names a Secret also requires read on it.
func (h *rbacHandler) PatchBackupStorage(ctx context.Context, cluster, namespace, name string, patch []byte) (*backupv1alpha1.BackupStorage, error) {
	object := rbac.ClusterNamespacedObjectName(cluster, namespace, name)
	if err := h.enforce(ctx, rbac.ResourceBackupStorages, rbac.ActionUpdate, object); err != nil {
		return nil, err
	}
	var doc struct {
		Spec backupv1alpha1.BackupStorageSpec `json:"spec"`
	}
	if err := json.Unmarshal(patch, &doc); err != nil {
		return nil, errors.Join(valhandler.ErrInvalidRequest, fmt.Errorf("invalid patch: %w", err))
	}
	if err := h.enforceStorageReferences(ctx, cluster, namespace, "", doc.Spec.SecretNames()); err != nil {
		return nil, err
	}
	return h.next.PatchBackupStorage(ctx, cluster, namespace, name, patch)
}

// enforceStorageReferences requires read on the BackupStorageClass, unless
// className is empty, and on every Secret in secrets.
func (h *rbacHandler) enforceStorageReferences(ctx context.Context, cluster, namespace, className string, secrets []string) error {
	if className != "" {
		if err := h.enforce(ctx, rbac.ResourceBackupStorageClasses, rbac.ActionRead, rbac.ClusterObjectName(cluster, className)); err != nil {
			return err
		}
	}
	for _, name := range secrets {
		if err := h.enforce(ctx, rbac.ResourceSecrets, rbac.ActionRead, rbac.ClusterNamespacedObjectName(cluster, namespace, name)); err != nil {
			return err
		}
	}
	return nil
}

// ListBackupStorageClasses returns backup storage classes filtered by RBAC
// permissions.
func (h *rbacHandler) ListBackupStorageClasses(ctx context.Context, cluster string) (*backupv1alpha1.BackupStorageClassList, error) {
	list, err := h.next.ListBackupStorageClasses(ctx, cluster)
	if err != nil {
		return nil, fmt.Errorf("ListBackupStorageClasses failed: %w", err)
	}
	filtered := make([]backupv1alpha1.BackupStorageClass, 0, len(list.Items))
	for _, class := range list.Items {
		object := rbac.ClusterObjectName(cluster, class.GetName())
		if err := h.enforce(ctx, rbac.ResourceBackupStorageClasses, rbac.ActionRead, object); errors.Is(err, ErrInsufficientPermissions) {
			continue
		} else if err != nil {
			return nil, fmt.Errorf("enforce failed: %w", err)
		}
		filtered = append(filtered, class)
	}
	list.Items = filtered
	return list, nil
}

// GetBackupStorageClass returns a backup storage class, gated by RBAC.
func (h *rbacHandler) GetBackupStorageClass(ctx context.Context, cluster, name string) (*backupv1alpha1.BackupStorageClass, error) {
	if err := h.enforce(ctx, rbac.ResourceBackupStorageClasses, rbac.ActionRead, rbac.ClusterObjectName(cluster, name)); err != nil {
		return nil, err
	}
	return h.next.GetBackupStorageClass(ctx, cluster, name)
}

// DeleteBackupStorage deletes a backup storage, gated by RBAC.
func (h *rbacHandler) DeleteBackupStorage(ctx context.Context, cluster, namespace, name string) error {
	object := rbac.ClusterNamespacedObjectName(cluster, namespace, name)
	if err := h.enforce(ctx, rbac.ResourceBackupStorages, rbac.ActionDelete, object); err != nil {
		return err
	}
	return h.next.DeleteBackupStorage(ctx, cluster, namespace, name)
}
