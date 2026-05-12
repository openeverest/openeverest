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

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	api "github.com/openeverest/openeverest/v2/internal/server/api"
	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

// ListInstances returns instances filtered by RBAC permissions.
func (h *rbacHandler) ListInstances(ctx context.Context, cluster, namespace string) (*corev1alpha1.InstanceList, error) {
	list, err := h.next.ListInstances(ctx, cluster, namespace)
	if err != nil {
		return nil, fmt.Errorf("ListInstances failed: %w", err)
	}
	filtered := make([]corev1alpha1.Instance, 0, len(list.Items))
	for _, inst := range list.Items {
		object := rbac.ClusterNamespacedObjectName(cluster, inst.GetNamespace(), inst.GetName())
		if err := h.enforce(ctx, rbac.ResourceInstances, rbac.ActionRead, object); errors.Is(err, ErrInsufficientPermissions) {
			continue
		} else if err != nil {
			return nil, fmt.Errorf("enforce failed: %w", err)
		}
		filtered = append(filtered, inst)
	}
	list.Items = filtered
	return list, nil
}

// GetInstance returns an instance, gated by RBAC.
func (h *rbacHandler) GetInstance(ctx context.Context, cluster, namespace, name string) (*corev1alpha1.Instance, error) {
	object := rbac.ClusterNamespacedObjectName(cluster, namespace, name)
	if err := h.enforce(ctx, rbac.ResourceInstances, rbac.ActionRead, object); err != nil {
		return nil, err
	}
	return h.next.GetInstance(ctx, cluster, namespace, name)
}

// CreateInstance creates an instance, gated by RBAC.
func (h *rbacHandler) CreateInstance(ctx context.Context, cluster string, instance *corev1alpha1.Instance) (*corev1alpha1.Instance, error) {
	object := rbac.ClusterNamespacedObjectName(cluster, instance.GetNamespace(), instance.GetName())
	if err := h.enforce(ctx, rbac.ResourceInstances, rbac.ActionCreate, object); err != nil {
		return nil, err
	}
	return h.next.CreateInstance(ctx, cluster, instance)
}

// UpdateInstance updates an instance, gated by RBAC.
func (h *rbacHandler) UpdateInstance(ctx context.Context, cluster string, instance *corev1alpha1.Instance) (*corev1alpha1.Instance, error) {
	object := rbac.ClusterNamespacedObjectName(cluster, instance.GetNamespace(), instance.GetName())
	if err := h.enforce(ctx, rbac.ResourceInstances, rbac.ActionUpdate, object); err != nil {
		return nil, err
	}
	return h.next.UpdateInstance(ctx, cluster, instance)
}

// DeleteInstance deletes an instance, gated by RBAC.
func (h *rbacHandler) DeleteInstance(ctx context.Context, cluster, namespace, name string) error {
	object := rbac.ClusterNamespacedObjectName(cluster, namespace, name)
	if err := h.enforce(ctx, rbac.ResourceInstances, rbac.ActionDelete, object); err != nil {
		return err
	}
	return h.next.DeleteInstance(ctx, cluster, namespace, name)
}

// GetInstanceConnection returns connection details, gated by RBAC.
func (h *rbacHandler) GetInstanceConnection(ctx context.Context, cluster, namespace, name string) (*api.InstanceConnectionDetails, error) {
	object := rbac.ClusterNamespacedObjectName(cluster, namespace, name)
	if err := h.enforce(ctx, rbac.ResourceInstances, rbac.ActionRead, object); err != nil {
		return nil, err
	}
	return h.next.GetInstanceConnection(ctx, cluster, namespace, name)
}
