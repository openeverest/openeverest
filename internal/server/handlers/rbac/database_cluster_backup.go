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

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"

	api "github.com/openeverest/openeverest/v2/internal/server/api"
	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

func (h *rbacHandler) ListDatabaseClusterBackups(ctx context.Context, namespace, clusterName string) (*everestv1alpha1.DatabaseClusterBackupList, error) {
	list, err := h.next.ListDatabaseClusterBackups(ctx, namespace, clusterName)
	if err != nil {
		return nil, err
	}
	filtered := []everestv1alpha1.DatabaseClusterBackup{}
	for _, dbbackup := range list.Items {
		if err := h.enforceDBBackupRead(ctx, &dbbackup); errors.Is(err, ErrInsufficientPermissions) {
			continue
		} else if err != nil {
			return nil, err
		}
		filtered = append(filtered, dbbackup)
	}
	list.Items = filtered
	return list, nil
}

func (h *rbacHandler) CreateDatabaseClusterBackup(ctx context.Context, req *everestv1alpha1.DatabaseClusterBackup) (*everestv1alpha1.DatabaseClusterBackup, error) {
	clusterName := req.Spec.DBClusterName
	namespace := req.GetNamespace()
	if err := h.enforce(ctx, rbac.ResourceDatabaseClusterBackups, rbac.ActionCreate, rbac.ObjectName(namespace, clusterName)); err != nil {
		return nil, err
	}
	return h.next.CreateDatabaseClusterBackup(ctx, req)
}

func (h *rbacHandler) DeleteDatabaseClusterBackup(ctx context.Context, namespace, name string, req *api.DeleteDatabaseClusterBackupParams) error {
	backup, err := h.next.GetDatabaseClusterBackup(ctx, namespace, name)
	if err != nil {
		return fmt.Errorf("GetDatabaseClusterBackup failed: %w", err)
	}
	clusterName := backup.Spec.DBClusterName
	if err := h.enforce(ctx, rbac.ResourceDatabaseClusterBackups, rbac.ActionDelete, rbac.ObjectName(namespace, clusterName)); err != nil {
		return err
	}
	return h.next.DeleteDatabaseClusterBackup(ctx, namespace, name, req)
}

func (h *rbacHandler) GetDatabaseClusterBackup(ctx context.Context, namespace, name string) (*everestv1alpha1.DatabaseClusterBackup, error) {
	dbbackup, err := h.next.GetDatabaseClusterBackup(ctx, namespace, name)
	if err != nil {
		return nil, err
	}
	if err := h.enforceDBBackupRead(ctx, dbbackup); err != nil {
		return nil, err
	}
	return h.next.GetDatabaseClusterBackup(ctx, namespace, name)
}

func (h *rbacHandler) enforceDBBackupRead(ctx context.Context, dbbackup *everestv1alpha1.DatabaseClusterBackup) error {
	clusterName := dbbackup.Spec.DBClusterName
	namespace := dbbackup.GetNamespace()
	if err := h.enforce(ctx, rbac.ResourceDatabaseClusterBackups, rbac.ActionRead, rbac.ObjectName(namespace, clusterName)); err != nil {
		return err
	}
	return nil
}
