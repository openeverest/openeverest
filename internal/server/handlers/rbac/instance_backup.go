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

// ListInstanceBackups returns instance backups filtered by RBAC permissions.
func (h *rbacHandler) ListInstanceBackups(ctx context.Context, cluster, namespace, instance string) (*backupv1alpha1.BackupList, error) {
	list, err := h.next.ListInstanceBackups(ctx, cluster, namespace, instance)
	if err != nil {
		return nil, fmt.Errorf("ListInstanceBackups failed: %w", err)
	}
	filtered := make([]backupv1alpha1.Backup, 0, len(list.Items))
	for _, b := range list.Items {
		object := rbac.ClusterNamespacedObjectName(cluster, b.GetNamespace(), b.GetName())
		if err := h.enforce(ctx, rbac.ResourceBackups, rbac.ActionRead, object); errors.Is(err, ErrInsufficientPermissions) {
			continue
		} else if err != nil {
			return nil, fmt.Errorf("enforce failed: %w", err)
		}
		filtered = append(filtered, b)
	}
	list.Items = filtered
	return list, nil
}
