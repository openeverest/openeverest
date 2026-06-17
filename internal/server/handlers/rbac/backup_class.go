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

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

// ListBackupClasses returns backup classes filtered by RBAC permissions.
func (h *rbacHandler) ListBackupClasses(ctx context.Context, cluster string) (*backupv1alpha1.BackupClassList, error) {
	list, err := h.next.ListBackupClasses(ctx, cluster)
	if err != nil {
		return nil, fmt.Errorf("ListBackupClasses failed: %w", err)
	}
	filtered := make([]backupv1alpha1.BackupClass, 0, len(list.Items))
	for _, bc := range list.Items {
		object := rbac.ClusterObjectName(cluster, bc.GetName())
		if err := h.enforce(ctx, rbac.ResourceBackupClasses, rbac.ActionRead, object); errors.Is(err, ErrInsufficientPermissions) {
			continue
		} else if err != nil {
			return nil, fmt.Errorf("enforce failed: %w", err)
		}
		filtered = append(filtered, bc)
	}
	list.Items = filtered
	return list, nil
}

// GetBackupClass returns a backup class, gated by RBAC.
func (h *rbacHandler) GetBackupClass(ctx context.Context, cluster, name string) (*backupv1alpha1.BackupClass, error) {
	object := rbac.ClusterObjectName(cluster, name)
	if err := h.enforce(ctx, rbac.ResourceBackupClasses, rbac.ActionRead, object); err != nil {
		return nil, err
	}
	return h.next.GetBackupClass(ctx, cluster, name)
}
