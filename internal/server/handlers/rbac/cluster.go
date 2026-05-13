// everest
// Copyright (C) 2023 Percona LLC
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

	api "github.com/openeverest/openeverest/v2/internal/server/api"
	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

// ListClusters returns a list of clusters, filtered by RBAC permissions.
func (h *rbacHandler) ListClusters(ctx context.Context) (*api.ClusterList, error) {
	list, err := h.next.ListClusters(ctx)
	if err != nil {
		return nil, fmt.Errorf("ListClusters failed: %w", err)
	}
	filtered := make([]api.Cluster, 0, len(list.Items))
	for _, c := range list.Items {
		if err := h.enforce(ctx, rbac.ResourceClusters, rbac.ActionRead, c.Name); errors.Is(err, ErrInsufficientPermissions) {
			continue
		} else if err != nil {
			return nil, fmt.Errorf("enforce failed: %w", err)
		}
		filtered = append(filtered, c)
	}
	list.Items = filtered
	return list, nil
}

// GetCluster returns a cluster by name, gated by RBAC.
func (h *rbacHandler) GetCluster(ctx context.Context, name string) (*api.Cluster, error) {
	if err := h.enforce(ctx, rbac.ResourceClusters, rbac.ActionRead, name); err != nil {
		return nil, err
	}
	return h.next.GetCluster(ctx, name)
}
