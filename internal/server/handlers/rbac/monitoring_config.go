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

	monitoringv1alpha2 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha2"
	api "github.com/openeverest/openeverest/v2/internal/server/api"
	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

// ListMonitoringConfigs returns monitoring configs filtered by RBAC permissions.
func (h *rbacHandler) ListMonitoringConfigs(ctx context.Context, cluster, namespace string) (*monitoringv1alpha2.MonitoringConfigList, error) {
	list, err := h.next.ListMonitoringConfigs(ctx, cluster, namespace)
	if err != nil {
		return nil, fmt.Errorf("ListMonitoringConfigs failed: %w", err)
	}
	filtered := make([]monitoringv1alpha2.MonitoringConfig, 0, len(list.Items))
	for _, mc := range list.Items {
		object := rbac.ClusterNamespacedObjectName(cluster, mc.GetNamespace(), mc.GetName())
		if err := h.enforce(ctx, rbac.ResourceMonitoringConfigs, rbac.ActionRead, object); errors.Is(err, ErrInsufficientPermissions) {
			continue
		} else if err != nil {
			return nil, fmt.Errorf("enforce failed: %w", err)
		}
		filtered = append(filtered, mc)
	}
	list.Items = filtered
	return list, nil
}

// CreateMonitoringConfig creates a monitoring config, gated by RBAC.
func (h *rbacHandler) CreateMonitoringConfig(ctx context.Context, cluster, namespace string, req *api.MonitoringConfigCreateParams) (*monitoringv1alpha2.MonitoringConfig, error) {
	object := rbac.ClusterNamespacedObjectName(cluster, namespace, req.Name)
	if err := h.enforce(ctx, rbac.ResourceMonitoringConfigs, rbac.ActionCreate, object); err != nil {
		return nil, err
	}
	return h.next.CreateMonitoringConfig(ctx, cluster, namespace, req)
}

// DeleteMonitoringConfig deletes a monitoring config, gated by RBAC.
func (h *rbacHandler) DeleteMonitoringConfig(ctx context.Context, cluster, namespace, name string) error {
	object := rbac.ClusterNamespacedObjectName(cluster, namespace, name)
	if err := h.enforce(ctx, rbac.ResourceMonitoringConfigs, rbac.ActionDelete, object); err != nil {
		return err
	}
	return h.next.DeleteMonitoringConfig(ctx, cluster, namespace, name)
}

// GetMonitoringConfig returns a monitoring config, gated by RBAC.
func (h *rbacHandler) GetMonitoringConfig(ctx context.Context, cluster, namespace, name string) (*monitoringv1alpha2.MonitoringConfig, error) {
	object := rbac.ClusterNamespacedObjectName(cluster, namespace, name)
	if err := h.enforce(ctx, rbac.ResourceMonitoringConfigs, rbac.ActionRead, object); err != nil {
		return nil, err
	}
	return h.next.GetMonitoringConfig(ctx, cluster, namespace, name)
}

// UpdateMonitoringConfig updates a monitoring config, gated by RBAC.
func (h *rbacHandler) UpdateMonitoringConfig(ctx context.Context, cluster, namespace, name string, req *api.MonitoringConfigUpdateParams) (*monitoringv1alpha2.MonitoringConfig, error) {
	object := rbac.ClusterNamespacedObjectName(cluster, namespace, name)
	if err := h.enforce(ctx, rbac.ResourceMonitoringConfigs, rbac.ActionUpdate, object); err != nil {
		return nil, err
	}
	return h.next.UpdateMonitoringConfig(ctx, cluster, namespace, name, req)
}
