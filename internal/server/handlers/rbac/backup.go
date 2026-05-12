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

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

// GetBackup returns a backup, gated by RBAC.
func (h *rbacHandler) GetBackup(ctx context.Context, cluster, namespace, name string) (*backupv1alpha1.Backup, error) {
	object := rbac.ClusterNamespacedObjectName(cluster, namespace, name)
	if err := h.enforce(ctx, rbac.ResourceBackups, rbac.ActionRead, object); err != nil {
		return nil, err
	}
	return h.next.GetBackup(ctx, cluster, namespace, name)
}

// CreateBackup creates a backup, gated by RBAC.
func (h *rbacHandler) CreateBackup(ctx context.Context, cluster string, backup *backupv1alpha1.Backup) (*backupv1alpha1.Backup, error) {
	object := rbac.ClusterNamespacedObjectName(cluster, backup.GetNamespace(), backup.GetName())
	if err := h.enforce(ctx, rbac.ResourceBackups, rbac.ActionCreate, object); err != nil {
		return nil, err
	}
	return h.next.CreateBackup(ctx, cluster, backup)
}

// DeleteBackup deletes a backup, gated by RBAC.
func (h *rbacHandler) DeleteBackup(ctx context.Context, cluster, namespace, name string) error {
	object := rbac.ClusterNamespacedObjectName(cluster, namespace, name)
	if err := h.enforce(ctx, rbac.ResourceBackups, rbac.ActionDelete, object); err != nil {
		return err
	}
	return h.next.DeleteBackup(ctx, cluster, namespace, name)
}
