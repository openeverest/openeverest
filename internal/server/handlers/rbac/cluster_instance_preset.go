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
	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

// ListClusterInstancePresets returns cluster instance presets filtered by RBAC permissions.
func (h *rbacHandler) ListClusterInstancePresets(ctx context.Context, cluster string, provider string) (*corev1alpha1.ClusterInstancePresetList, error) {
	list, err := h.next.ListClusterInstancePresets(ctx, cluster, provider)
	if err != nil {
		return nil, fmt.Errorf("ListClusterInstancePresets failed: %w", err)
	}
	filtered := make([]corev1alpha1.ClusterInstancePreset, 0, len(list.Items))
	for _, preset := range list.Items {
		object := rbac.ClusterObjectName(cluster, preset.GetName())
		if err := h.enforce(ctx, rbac.ResourceClusterInstancePresets, rbac.ActionRead, object); errors.Is(err, ErrInsufficientPermissions) {
			continue
		} else if err != nil {
			return nil, fmt.Errorf("enforce failed: %w", err)
		}
		filtered = append(filtered, preset)
	}
	list.Items = filtered
	return list, nil
}

// GetClusterInstancePreset returns a cluster instance preset, gated by RBAC.
func (h *rbacHandler) GetClusterInstancePreset(ctx context.Context, cluster, name string) (*corev1alpha1.ClusterInstancePreset, error) {
	object := rbac.ClusterObjectName(cluster, name)
	if err := h.enforce(ctx, rbac.ResourceClusterInstancePresets, rbac.ActionRead, object); err != nil {
		return nil, err
	}
	return h.next.GetClusterInstancePreset(ctx, cluster, name)
}

// ResolveClusterInstancePreset returns a cluster instance preset with namespace defaults, gated by RBAC.
func (h *rbacHandler) ResolveClusterInstancePreset(ctx context.Context, cluster, name, namespace string) (*corev1alpha1.ClusterInstancePreset, error) {
	object := rbac.ClusterObjectName(cluster, name)
	if err := h.enforce(ctx, rbac.ResourceClusterInstancePresets, rbac.ActionRead, object); err != nil {
		return nil, err
	}
	// Also check namespace access
	nsObject := rbac.ClusterObjectName(cluster, namespace)
	if err := h.enforce(ctx, rbac.ResourceNamespaces, rbac.ActionRead, nsObject); err != nil {
		return nil, err
	}
	return h.next.ResolveClusterInstancePreset(ctx, cluster, name, namespace)
}
