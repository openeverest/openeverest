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

	"github.com/google/go-cmp/cmp"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	api "github.com/openeverest/openeverest/v2/internal/server/api"
	"github.com/openeverest/openeverest/v2/pkg/common"
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
	// If the instance references a user secret, ensure the user has read
	// permission on that secret to prevent privilege escalation.
	if userSecret := instance.Spec.UserSecretRef; userSecret != nil {
		object := rbac.ClusterNamespacedObjectName(cluster, instance.GetNamespace(), userSecret.Name)
		if err := h.enforce(ctx, rbac.ResourceSecrets, rbac.ActionRead, object); err != nil {
			return nil, err
		}
	}

	object := rbac.ClusterNamespacedObjectName(cluster, instance.GetNamespace(), instance.GetName())

	// If user has create permission on Instance, allow any customization
	err := h.enforce(ctx, rbac.ResourceInstances, rbac.ActionCreate, object)
	if err == nil {
		return h.next.CreateInstance(ctx, cluster, instance)
	}

	if !errors.Is(err, ErrInsufficientPermissions) {
		return nil, fmt.Errorf("enforce create permission failed: %w", err)
	}

	// If user does not have deploy permission, reject
	if err = h.enforce(ctx, rbac.ResourceInstances, rbac.ActionDeploy, object); err != nil {
		return nil, err
	}

	var instancePreset, clusterInstancePreset string
	if annotations := instance.GetAnnotations(); annotations != nil {
		instancePreset = annotations[common.InstancePresetAnnotation]
		clusterInstancePreset = annotations[common.ClusterInstancePresetAnnotation]
	}

	switch {
	case instancePreset != "":
		// User must have read permission on the InstancePreset
		presetObj := rbac.ClusterNamespacedObjectName(cluster, instance.GetNamespace(), instancePreset)
		if err = h.enforce(ctx, rbac.ResourceInstancePresets, rbac.ActionRead, presetObj); err != nil {
			return nil, err
		}

		// User has deploy permission, check instance matches preset exactly
		if err = h.ensureMatchesInstancePreset(ctx, cluster, instance, instancePreset); err != nil {
			return nil, err
		}
	case clusterInstancePreset != "":
		// User must have read permission on the ClusterInstancePreset
		clusterPresetObj := rbac.ClusterObjectName(cluster, clusterInstancePreset)
		if err = h.enforce(ctx, rbac.ResourceClusterInstancePresets, rbac.ActionRead, clusterPresetObj); err != nil {
			return nil, err
		}

		// User has deploy permission, check instance matches preset exactly
		if err = h.ensureMatchesClusterInstancePreset(ctx, cluster, instance, clusterInstancePreset); err != nil {
			return nil, err
		}
	default:
		// no preset specified
		return nil, ErrInsufficientPermissions
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

// PatchInstance patches an instance, gated by RBAC. A patch is authorised as an
// update on the instance rather than as a permission of its own.
func (h *rbacHandler) PatchInstance(ctx context.Context, cluster, namespace, name string, patch []byte) (*corev1alpha1.Instance, error) {
	object := rbac.ClusterNamespacedObjectName(cluster, namespace, name)
	if err := h.enforce(ctx, rbac.ResourceInstances, rbac.ActionUpdate, object); err != nil {
		return nil, err
	}
	return h.next.PatchInstance(ctx, cluster, namespace, name, patch)
}

// DeleteInstance deletes an instance, gated by RBAC.
func (h *rbacHandler) DeleteInstance(ctx context.Context, cluster, namespace, name string, params *api.DeleteInstanceParams) error {
	object := rbac.ClusterNamespacedObjectName(cluster, namespace, name)
	if err := h.enforce(ctx, rbac.ResourceInstances, rbac.ActionDelete, object); err != nil {
		return err
	}
	return h.next.DeleteInstance(ctx, cluster, namespace, name, params)
}

// GetInstanceConnection returns connection details, gated by the read-connection action.
func (h *rbacHandler) GetInstanceConnection(ctx context.Context, cluster, namespace, name string) (*api.InstanceConnectionDetails, error) {
	object := rbac.ClusterNamespacedObjectName(cluster, namespace, name)
	if err := h.enforce(ctx, rbac.ResourceInstances, rbac.ActionReadConnection, object); err != nil {
		return nil, err
	}
	return h.next.GetInstanceConnection(ctx, cluster, namespace, name)
}

// ensureMatchesInstancePreset checks if an instance spec matches its referenced instance preset.
// Returns an error if the instance has been customized beyond the preset values.
func (h *rbacHandler) ensureMatchesInstancePreset(ctx context.Context, cluster string, instance *corev1alpha1.Instance, presetName string) error {
	preset, err := h.next.GetInstancePreset(ctx, cluster, instance.GetNamespace(), presetName)
	if err != nil {
		return fmt.Errorf("failed to get preset %s: %w", presetName, err)
	}

	if diff := cmp.Diff(preset.Spec.InstanceSpec, instance.Spec); diff != "" {
		h.log.Warnf("Instance spec customization detected for preset %s: %s", presetName, diff)
		return ErrInsufficientPermissions
	}

	return nil
}

// ensureMatchesClusterInstancePreset checks if an instance spec matches its referenced cluster instance preset.
// Returns an error if the instance has been customized beyond the preset values.
func (h *rbacHandler) ensureMatchesClusterInstancePreset(ctx context.Context, cluster string, instance *corev1alpha1.Instance, presetName string) error {
	// Fetch the preset with namespace defaults resolved
	preset, err := h.next.ResolveClusterInstancePreset(ctx, cluster, presetName, instance.GetNamespace())
	if err != nil {
		return fmt.Errorf("failed to resolve preset %s for namespace %s: %w", presetName, instance.GetNamespace(), err)
	}

	// Compare the instance spec with the preset's resolved instance spec
	if diff := cmp.Diff(preset.Spec.InstanceSpec, instance.Spec); diff != "" {
		h.log.Warnf("Instance spec customization detected for preset %s: %s", presetName, diff)
		return ErrInsufficientPermissions
	}

	return nil
}
