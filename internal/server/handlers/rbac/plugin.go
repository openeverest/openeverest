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

// ListPlugins returns the enabled plugins filtered by RBAC permissions.
func (h *rbacHandler) ListPlugins(ctx context.Context, cluster string) (api.PluginDescriptorList, error) {
	list, err := h.next.ListPlugins(ctx, cluster)
	if err != nil {
		return nil, fmt.Errorf("ListPlugins failed: %w", err)
	}
	filtered := make(api.PluginDescriptorList, 0, len(list))
	for _, p := range list {
		object := rbac.ClusterObjectName(cluster, p.Name)
		if err := h.enforce(ctx, rbac.ResourcePlugins, rbac.ActionRead, object); errors.Is(err, ErrInsufficientPermissions) {
			continue
		} else if err != nil {
			return nil, fmt.Errorf("enforce failed: %w", err)
		}
		filtered = append(filtered, p)
	}
	return filtered, nil
}

// GetPluginContext returns the plugin context. It carries only the caller's own
// identity and namespace list, so it is not gated on a per-plugin object — any
// authenticated user may fetch it, as before the move into the handler chain.
func (h *rbacHandler) GetPluginContext(ctx context.Context, cluster string) (*api.PluginContext, error) {
	return h.next.GetPluginContext(ctx, cluster)
}
