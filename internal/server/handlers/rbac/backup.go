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
	"fmt"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	api "github.com/openeverest/openeverest/v2/internal/server/api"
	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

// backupInstanceName returns the name of the Instance a Backup belongs to, or
// "" if it doesn't reference one. Backups are keyed by this rather than their
// own (usually generateName-produced) name, which nobody can write a grant for.
func backupInstanceName(backup *backupv1alpha1.Backup) string {
	if backup.Spec.Origin.InstanceRef == nil {
		return ""
	}
	return backup.Spec.Origin.InstanceRef.Name
}

// GetBackup returns a backup, gated by RBAC on the instance it belongs to. A
// backup not found and one the caller isn't authorized for both come back as
// ErrInsufficientPermissions, so a name cannot be probed for existence.
func (h *rbacHandler) GetBackup(ctx context.Context, cluster, namespace, name string) (*backupv1alpha1.Backup, error) {
	backup, err := h.next.GetBackup(ctx, cluster, namespace, name)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, ErrInsufficientPermissions
		}
		return nil, fmt.Errorf("GetBackup failed: %w", err)
	}
	object := rbac.ClusterNamespacedObjectName(cluster, namespace, backupInstanceName(backup))
	if err := h.enforce(ctx, rbac.ResourceBackups, rbac.ActionRead, object); err != nil {
		return nil, err
	}
	return backup, nil
}

// CreateBackup creates a backup, gated by RBAC on the instance it belongs to.
func (h *rbacHandler) CreateBackup(ctx context.Context, cluster string, backup *backupv1alpha1.Backup) (*backupv1alpha1.Backup, error) {
	object := rbac.ClusterNamespacedObjectName(cluster, backup.GetNamespace(), backupInstanceName(backup))
	if err := h.enforce(ctx, rbac.ResourceBackups, rbac.ActionCreate, object); err != nil {
		return nil, err
	}
	return h.next.CreateBackup(ctx, cluster, backup)
}

// DeleteBackup deletes a backup, gated by RBAC on the instance it belongs to.
// Same not-found/denied collapsing as GetBackup.
func (h *rbacHandler) DeleteBackup(ctx context.Context, cluster, namespace, name string, params *api.DeleteBackupParams) error {
	backup, err := h.next.GetBackup(ctx, cluster, namespace, name)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return ErrInsufficientPermissions
		}
		return fmt.Errorf("GetBackup failed: %w", err)
	}
	object := rbac.ClusterNamespacedObjectName(cluster, namespace, backupInstanceName(backup))
	if err := h.enforce(ctx, rbac.ResourceBackups, rbac.ActionDelete, object); err != nil {
		return err
	}
	return h.next.DeleteBackup(ctx, cluster, namespace, name, params)
}
