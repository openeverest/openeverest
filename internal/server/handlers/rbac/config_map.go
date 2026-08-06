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

	corev1 "k8s.io/api/core/v1"

	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

// CreateConfigMap creates a ConfigMap with RBAC enforcement.
func (h *rbacHandler) CreateConfigMap(ctx context.Context, cluster, namespace string, configMap *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	if err := h.enforce(ctx, rbac.ResourceConfigMaps, rbac.ActionCreate, rbac.ClusterNamespacedObjectName(cluster, namespace, configMap.Name)); err != nil {
		return nil, err
	}

	return h.next.CreateConfigMap(ctx, cluster, namespace, configMap)
}

// ListConfigMaps lists ConfigMaps with RBAC filtering.
func (h *rbacHandler) ListConfigMaps(ctx context.Context, cluster, namespace, provider, definition string) (*corev1.ConfigMapList, error) {
	list, err := h.next.ListConfigMaps(ctx, cluster, namespace, provider, definition)
	if err != nil {
		return nil, err
	}

	// Filter the list based on user's read permissions.
	filtered := make([]corev1.ConfigMap, 0, len(list.Items))
	for _, cm := range list.Items {
		if err := h.enforce(ctx, rbac.ResourceConfigMaps, rbac.ActionRead, rbac.ClusterNamespacedObjectName(cluster, namespace, cm.Name)); err != nil {
			if errors.Is(err, ErrInsufficientPermissions) {
				continue // Skip items user cannot access.
			}
			return nil, err
		}
		filtered = append(filtered, cm)
	}

	list.Items = filtered
	return list, nil
}

// GetConfigMap gets a ConfigMap with RBAC enforcement.
func (h *rbacHandler) GetConfigMap(ctx context.Context, cluster, namespace, name string) (*corev1.ConfigMap, error) {
	if err := h.enforce(ctx, rbac.ResourceConfigMaps, rbac.ActionRead, rbac.ClusterNamespacedObjectName(cluster, namespace, name)); err != nil {
		return nil, err
	}

	return h.next.GetConfigMap(ctx, cluster, namespace, name)
}

// DeleteConfigMap deletes a ConfigMap with RBAC enforcement.
func (h *rbacHandler) DeleteConfigMap(ctx context.Context, cluster, namespace, name string) error {
	if err := h.enforce(ctx, rbac.ResourceConfigMaps, rbac.ActionDelete, rbac.ClusterNamespacedObjectName(cluster, namespace, name)); err != nil {
		return err
	}

	return h.next.DeleteConfigMap(ctx, cluster, namespace, name)
}
