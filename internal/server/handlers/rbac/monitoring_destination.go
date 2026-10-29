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

	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

// ListMonitoringDestinations returns monitoring destinations filtered by RBAC permissions.
func (h *rbacHandler) ListMonitoringDestinations(ctx context.Context, cluster, namespace, class string) (*monitoringv1alpha1.MonitoringDestinationList, error) {
	list, err := h.next.ListMonitoringDestinations(ctx, cluster, namespace, class)
	if err != nil {
		return nil, fmt.Errorf("ListMonitoringDestinations failed: %w", err)
	}
	filtered := make([]monitoringv1alpha1.MonitoringDestination, 0, len(list.Items))
	for _, destination := range list.Items {
		object := rbac.ClusterNamespacedObjectName(cluster, destination.GetNamespace(), destination.GetName())
		if err := h.enforce(ctx, rbac.ResourceMonitoringDestinations, rbac.ActionRead, object); errors.Is(err, ErrInsufficientPermissions) {
			continue
		} else if err != nil {
			return nil, fmt.Errorf("enforce failed: %w", err)
		}
		filtered = append(filtered, destination)
	}
	list.Items = filtered
	return list, nil
}

// GetMonitoringDestination returns a monitoring destination, gated by RBAC.
func (h *rbacHandler) GetMonitoringDestination(ctx context.Context, cluster, namespace, name string) (*monitoringv1alpha1.MonitoringDestination, error) {
	object := rbac.ClusterNamespacedObjectName(cluster, namespace, name)
	if err := h.enforce(ctx, rbac.ResourceMonitoringDestinations, rbac.ActionRead, object); err != nil {
		return nil, err
	}
	return h.next.GetMonitoringDestination(ctx, cluster, namespace, name)
}

// CreateMonitoringDestination creates a monitoring destination, gated by RBAC.
// It also requires read on the class it references and on every Secret it
// references, whose contents are sent to or trusted for the destination.
func (h *rbacHandler) CreateMonitoringDestination(ctx context.Context, cluster, namespace string, destination *monitoringv1alpha1.MonitoringDestination) (*monitoringv1alpha1.MonitoringDestination, error) {
	object := rbac.ClusterNamespacedObjectName(cluster, namespace, destination.GetName())
	if err := h.enforce(ctx, rbac.ResourceMonitoringDestinations, rbac.ActionCreate, object); err != nil {
		return nil, err
	}
	class := rbac.ClusterObjectName(cluster, destination.Spec.ClassRef.Name)
	if err := h.enforce(ctx, rbac.ResourceMonitoringClasses, rbac.ActionRead, class); err != nil {
		return nil, err
	}
	for _, name := range destination.Spec.SecretNames() {
		secret := rbac.ClusterNamespacedObjectName(cluster, namespace, name)
		if err := h.enforce(ctx, rbac.ResourceSecrets, rbac.ActionRead, secret); err != nil {
			return nil, err
		}
	}
	return h.next.CreateMonitoringDestination(ctx, cluster, namespace, destination)
}

// UpdateMonitoringDestination updates a monitoring destination, gated by RBAC
// like CreateMonitoringDestination.
func (h *rbacHandler) UpdateMonitoringDestination(ctx context.Context, cluster, namespace, name string, destination *monitoringv1alpha1.MonitoringDestination) (*monitoringv1alpha1.MonitoringDestination, error) {
	object := rbac.ClusterNamespacedObjectName(cluster, namespace, name)
	if err := h.enforce(ctx, rbac.ResourceMonitoringDestinations, rbac.ActionUpdate, object); err != nil {
		return nil, err
	}
	class := rbac.ClusterObjectName(cluster, destination.Spec.ClassRef.Name)
	if err := h.enforce(ctx, rbac.ResourceMonitoringClasses, rbac.ActionRead, class); err != nil {
		return nil, err
	}
	for _, name := range destination.Spec.SecretNames() {
		secret := rbac.ClusterNamespacedObjectName(cluster, namespace, name)
		if err := h.enforce(ctx, rbac.ResourceSecrets, rbac.ActionRead, secret); err != nil {
			return nil, err
		}
	}
	return h.next.UpdateMonitoringDestination(ctx, cluster, namespace, name, destination)
}

// DeleteMonitoringDestination deletes a monitoring destination, gated by RBAC.
func (h *rbacHandler) DeleteMonitoringDestination(ctx context.Context, cluster, namespace, name string) error {
	object := rbac.ClusterNamespacedObjectName(cluster, namespace, name)
	if err := h.enforce(ctx, rbac.ResourceMonitoringDestinations, rbac.ActionDelete, object); err != nil {
		return err
	}
	return h.next.DeleteMonitoringDestination(ctx, cluster, namespace, name)
}
