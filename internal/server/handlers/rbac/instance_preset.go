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

// ListInstancePresets returns namespace-scoped instance presets filtered by RBAC permissions.
func (h *rbacHandler) ListInstancePresets(ctx context.Context, cluster, namespace, provider string) (*corev1alpha1.InstancePresetList, error) {
	list, err := h.next.ListInstancePresets(ctx, cluster, namespace, provider)
	if err != nil {
		return nil, fmt.Errorf("ListInstancePresets failed: %w", err)
	}
	filtered := make([]corev1alpha1.InstancePreset, 0, len(list.Items))
	for _, preset := range list.Items {
		object := rbac.ClusterNamespacedObjectName(cluster, preset.GetNamespace(), preset.GetName())
		if err := h.enforce(ctx, rbac.ResourceInstancePresets, rbac.ActionRead, object); errors.Is(err, ErrInsufficientPermissions) {
			continue
		} else if err != nil {
			return nil, fmt.Errorf("enforce failed: %w", err)
		}
		filtered = append(filtered, preset)
	}
	list.Items = filtered
	return list, nil
}

// GetInstancePreset returns a namespace-scoped instance preset, gated by RBAC.
func (h *rbacHandler) GetInstancePreset(ctx context.Context, cluster, namespace, name string) (*corev1alpha1.InstancePreset, error) {
	object := rbac.ClusterNamespacedObjectName(cluster, namespace, name)
	if err := h.enforce(ctx, rbac.ResourceInstancePresets, rbac.ActionRead, object); err != nil {
		return nil, err
	}
	return h.next.GetInstancePreset(ctx, cluster, namespace, name)
}

// CreateInstancePreset creates a namespace-scoped instance preset, gated by RBAC.
func (h *rbacHandler) CreateInstancePreset(ctx context.Context, cluster string, preset *corev1alpha1.InstancePreset) (*corev1alpha1.InstancePreset, error) {
	object := rbac.ClusterNamespacedObjectName(cluster, preset.GetNamespace(), preset.GetName())
	if err := h.enforce(ctx, rbac.ResourceInstancePresets, rbac.ActionCreate, object); err != nil {
		return nil, err
	}
	if err := h.ensureInstancePresetReferencesAuthorized(ctx, cluster, nil, preset); err != nil {
		return nil, err
	}
	return h.next.CreateInstancePreset(ctx, cluster, preset)
}

// ensureInstancePresetReferencesAuthorized checks the caller may read resource
// the preset pins, since users deploying from the preset are not checked against them.
// On update, current is the stored preset and only added or changed references are checked.
//
// backups:create and restores:create are not checked: they are keyed on an
// instance that does not exist until someone deploys from the preset, and
// instances:deploy implies them.
func (h *rbacHandler) ensureInstancePresetReferencesAuthorized(ctx context.Context, cluster string, current, preset *corev1alpha1.InstancePreset) error {
	var currentSpec *corev1alpha1.InstanceSpec
	var currentMonitoring *corev1alpha1.InstanceMonitoringSpec
	if current != nil {
		currentSpec = &current.Spec.InstanceSpec
		currentMonitoring = current.Spec.Monitoring
	}
	if err := h.enforceReadMonitoringDestinations(ctx, cluster, preset.GetNamespace(), preset.Spec.Monitoring, currentMonitoring); err != nil {
		return err
	}
	return h.ensureInstanceReferencesAuthorized(ctx, cluster, preset.GetNamespace(), preset.GetName(), currentSpec, preset.Spec.InstanceSpec, false)
}

// UpdateInstancePreset updates a namespace-scoped instance preset, gated by RBAC.
func (h *rbacHandler) UpdateInstancePreset(ctx context.Context, cluster string, preset *corev1alpha1.InstancePreset) (*corev1alpha1.InstancePreset, error) {
	object := rbac.ClusterNamespacedObjectName(cluster, preset.GetNamespace(), preset.GetName())
	if err := h.enforce(ctx, rbac.ResourceInstancePresets, rbac.ActionUpdate, object); err != nil {
		return nil, err
	}
	current, err := h.next.GetInstancePreset(ctx, cluster, preset.GetNamespace(), preset.GetName())
	if err != nil {
		return nil, err
	}
	if err := h.ensureInstancePresetReferencesAuthorized(ctx, cluster, current, preset); err != nil {
		return nil, err
	}
	// Pin the write to the version the checks ran against.
	if preset.GetResourceVersion() == "" {
		preset.SetResourceVersion(current.GetResourceVersion())
	}
	return h.next.UpdateInstancePreset(ctx, cluster, preset)
}

// DeleteInstancePreset deletes a namespace-scoped instance preset, gated by RBAC.
func (h *rbacHandler) DeleteInstancePreset(ctx context.Context, cluster, namespace, name string) error {
	object := rbac.ClusterNamespacedObjectName(cluster, namespace, name)
	if err := h.enforce(ctx, rbac.ResourceInstancePresets, rbac.ActionDelete, object); err != nil {
		return err
	}
	return h.next.DeleteInstancePreset(ctx, cluster, namespace, name)
}
