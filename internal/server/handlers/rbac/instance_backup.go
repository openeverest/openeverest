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

// ListInstanceBackups returns the backups belonging to an instance, gated by RBAC on that instance.
// Every item in the list belongs to the same instance, so one check against it is sufficient.
func (h *rbacHandler) ListInstanceBackups(ctx context.Context, cluster, namespace, instance string) (*backupv1alpha1.BackupList, error) {
	object := rbac.ClusterNamespacedObjectName(cluster, namespace, instance)
	if err := h.enforce(ctx, rbac.ResourceBackups, rbac.ActionRead, object); err != nil {
		return nil, err
	}
	return h.next.ListInstanceBackups(ctx, cluster, namespace, instance)
}
