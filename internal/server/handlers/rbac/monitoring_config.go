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

	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
	api "github.com/openeverest/openeverest/v2/internal/server/api"
)

// ListMonitoringConfigs proxies the request to the next handler.
func (h *rbacHandler) ListMonitoringConfigs(ctx context.Context, namespace string) (*monitoringv1alpha1.MonitoringConfigList, error) {
	// Add RBAC checks
	return h.next.ListMonitoringConfigs(ctx, namespace)
}

// CreateMonitoringConfig proxies the request to the next handler.
func (h *rbacHandler) CreateMonitoringConfig(ctx context.Context, namespace string, req *api.MonitoringConfigCreateParams) (*monitoringv1alpha1.MonitoringConfig, error) {
	// Add RBAC checks
	return h.next.CreateMonitoringConfig(ctx, namespace, req)
}

// DeleteMonitoringConfig proxies the request to the next handler.
func (h *rbacHandler) DeleteMonitoringConfig(ctx context.Context, namespace, name string) error {
	// Add RBAC checks
	return h.next.DeleteMonitoringConfig(ctx, namespace, name)
}

// GetMonitoringConfig proxies the request to the next handler.
func (h *rbacHandler) GetMonitoringConfig(ctx context.Context, namespace, name string) (*monitoringv1alpha1.MonitoringConfig, error) {
	// Add RBAC checks
	return h.next.GetMonitoringConfig(ctx, namespace, name)
}

// UpdateMonitoringConfig proxies the request to the next handler.
func (h *rbacHandler) UpdateMonitoringConfig(ctx context.Context, namespace, name string, req *api.MonitoringConfigUpdateParams) (*monitoringv1alpha1.MonitoringConfig, error) {
	// Add RBAC checks
	return h.next.UpdateMonitoringConfig(ctx, namespace, name, req)
}
