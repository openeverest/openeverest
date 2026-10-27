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

package validation

import (
	"context"

	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
)

// ListMonitoringDestinations proxies the request to the next handler.
func (h *validateHandler) ListMonitoringDestinations(ctx context.Context, cluster, namespace, class string) (*monitoringv1alpha1.MonitoringDestinationList, error) {
	return h.next.ListMonitoringDestinations(ctx, cluster, namespace, class)
}

// GetMonitoringDestination proxies the request to the next handler.
func (h *validateHandler) GetMonitoringDestination(ctx context.Context, cluster, namespace, name string) (*monitoringv1alpha1.MonitoringDestination, error) {
	return h.next.GetMonitoringDestination(ctx, cluster, namespace, name)
}

// CreateMonitoringDestination proxies the request to the next handler; parameters are checked against the class schema at reconcile time.
func (h *validateHandler) CreateMonitoringDestination(ctx context.Context, cluster, namespace string, destination *monitoringv1alpha1.MonitoringDestination) (*monitoringv1alpha1.MonitoringDestination, error) {
	return h.next.CreateMonitoringDestination(ctx, cluster, namespace, destination)
}

// UpdateMonitoringDestination proxies the request to the next handler; parameters are checked against the class schema at reconcile time.
func (h *validateHandler) UpdateMonitoringDestination(ctx context.Context, cluster, namespace, name string, destination *monitoringv1alpha1.MonitoringDestination) (*monitoringv1alpha1.MonitoringDestination, error) {
	return h.next.UpdateMonitoringDestination(ctx, cluster, namespace, name, destination)
}

// DeleteMonitoringDestination proxies the request to the next handler.
func (h *validateHandler) DeleteMonitoringDestination(ctx context.Context, cluster, namespace, name string) error {
	return h.next.DeleteMonitoringDestination(ctx, cluster, namespace, name)
}
