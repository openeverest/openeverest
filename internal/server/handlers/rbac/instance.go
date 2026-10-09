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
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	jsonpatch "github.com/evanphx/json-patch/v5"
	"github.com/google/go-cmp/cmp"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	api "github.com/openeverest/openeverest/v2/internal/server/api"
	valhandler "github.com/openeverest/openeverest/v2/internal/server/handlers/validation"
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

	var presetName string
	if instance.GetAnnotations() != nil {
		presetName = instance.GetAnnotations()["openeverest.io/instance-preset"]
	}

	// If no preset is specified, require standard create permission
	if presetName == "" {
		if err := h.enforce(ctx, rbac.ResourceInstances, rbac.ActionCreate, object); err != nil {
			return nil, err
		}
		if err := h.ensureInstanceReferencesAuthorized(ctx, cluster, nil, instance); err != nil {
			return nil, err
		}
		return h.next.CreateInstance(ctx, cluster, instance)
	}

	// User must have read permission on the InstancePreset
	presetObject := rbac.ClusterObjectName(cluster, presetName)
	if err := h.enforce(ctx, rbac.ResourceInstancePresets, rbac.ActionRead, presetObject); err != nil {
		return nil, err
	}

	var err error

	// If user has create permission on Instance, allow any customization
	if err = h.enforce(ctx, rbac.ResourceInstances, rbac.ActionCreate, object); err == nil {
		if err := h.ensureInstanceReferencesAuthorized(ctx, cluster, nil, instance); err != nil {
			return nil, err
		}
		return h.next.CreateInstance(ctx, cluster, instance)
	}

	if !errors.Is(err, ErrInsufficientPermissions) {
		return nil, fmt.Errorf("enforce create permission failed: %w", err)
	}

	// If user does not have deploy permission, reject
	if err = h.enforce(ctx, rbac.ResourceInstances, rbac.ActionDeploy, object); err != nil {
		return nil, err
	}

	// The preset vouches for the references it pins, but not for the user
	// secret: whose credentials to seed is the requester's own choice.
	if err := h.ensureUserSecretReadable(ctx, cluster, nil, instance); err != nil {
		return nil, err
	}

	// User has deploy but not create, check instance matches preset exactly
	if err := h.ensureInstanceMatchesPreset(ctx, cluster, instance, presetName); err != nil {
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
	current, err := h.next.GetInstance(ctx, cluster, instance.GetNamespace(), instance.GetName())
	if err != nil {
		return nil, err
	}
	if err := h.ensureInstanceReferencesAuthorized(ctx, cluster, current, instance); err != nil {
		return nil, err
	}
	// Pin the write to the version the checks ran against, so a reference
	// added concurrently can't ride in unchecked. A caller-supplied version is
	// left alone: the API server rejects it if it is not the one checked here.
	if instance.GetResourceVersion() == "" {
		instance.SetResourceVersion(current.GetResourceVersion())
	}
	return h.next.UpdateInstance(ctx, cluster, instance)
}

// PatchInstance patches an instance, gated by RBAC. A patch is authorised as an
// update on the instance rather than as a permission of its own. The patch is
// merged onto the stored instance first, so references it introduces get the
// same checks as an update.
func (h *rbacHandler) PatchInstance(ctx context.Context, cluster, namespace, name string, patch []byte) (*corev1alpha1.Instance, error) {
	object := rbac.ClusterNamespacedObjectName(cluster, namespace, name)
	if err := h.enforce(ctx, rbac.ResourceInstances, rbac.ActionUpdate, object); err != nil {
		return nil, err
	}
	current, err := h.next.GetInstance(ctx, cluster, namespace, name)
	if err != nil {
		return nil, err
	}
	patched, err := mergePatchInstance(current, patch)
	if err != nil {
		return nil, err
	}
	if err := h.ensureInstanceReferencesAuthorized(ctx, cluster, current, patched); err != nil {
		return nil, err
	}
	// Same pin as UpdateInstance. patched carries the caller's version if the
	// patch named one, and the stored one otherwise.
	pin, err := json.Marshal(map[string]any{"metadata": map[string]any{"resourceVersion": patched.GetResourceVersion()}})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal resource version: %w", err)
	}
	if patch, err = jsonpatch.MergePatch(patch, pin); err != nil {
		return nil, fmt.Errorf("failed to pin resource version: %w", err)
	}
	return h.next.PatchInstance(ctx, cluster, namespace, name, patch)
}

// mergePatchInstance returns the instance a JSON merge patch would produce.
func mergePatchInstance(current *corev1alpha1.Instance, patch []byte) (*corev1alpha1.Instance, error) {
	original, err := json.Marshal(current)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal instance: %w", err)
	}
	merged, err := jsonpatch.MergePatch(original, patch)
	if err != nil {
		return nil, errors.Join(valhandler.ErrInvalidRequest, err)
	}
	patched := &corev1alpha1.Instance{}
	if err := json.Unmarshal(merged, patched); err != nil {
		return nil, errors.Join(valhandler.ErrInvalidRequest, err)
	}
	return patched, nil
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

// ensureInstanceMatchesPreset checks if an instance spec matches its referenced preset.
// Returns an error if the instance has been customized beyond the preset values.
func (h *rbacHandler) ensureInstanceMatchesPreset(ctx context.Context, cluster string, instance *corev1alpha1.Instance, presetName string) error {
	// Fetch the preset with namespace defaults resolved
	preset, err := h.next.ResolveInstancePreset(ctx, cluster, presetName, instance.GetNamespace())
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

// ensureInstanceReferencesAuthorized checks the caller may use every resource the instance points at.
// On update, current is the stored instance and only references the request adds or changes are checked,
// so losing a grant on an untouched reference doesn't lock the caller out of editing unrelated fields.
func (h *rbacHandler) ensureInstanceReferencesAuthorized(ctx context.Context, cluster string, current, instance *corev1alpha1.Instance) error {
	if current == nil {
		current = &corev1alpha1.Instance{}
	}
	if name := instance.Spec.ProviderRef.Name; name != current.Spec.ProviderRef.Name {
		if err := h.enforce(ctx, rbac.ResourceProviders, rbac.ActionRead, rbac.ClusterObjectName(cluster, name)); err != nil {
			return err
		}
	}
	if err := h.ensureUserSecretReadable(ctx, cluster, current, instance); err != nil {
		return err
	}
	if err := h.ensureInstanceBackupAuthorized(ctx, cluster, current.Spec.Backup, instance); err != nil {
		return err
	}
	if dataSource := instance.Spec.DataSource; dataSource != nil && !reflect.DeepEqual(dataSource, current.Spec.DataSource) {
		object := rbac.ClusterNamespacedObjectName(cluster, instance.GetNamespace(), instance.GetName())
		if err := h.enforce(ctx, rbac.ResourceRestores, rbac.ActionCreate, object); err != nil {
			return err
		}
		if err := h.ensureDataSourceReadable(ctx, cluster, instance.GetNamespace(), *dataSource); err != nil {
			return err
		}
	}
	return nil
}

// ensureUserSecretReadable checks the caller may read a user secret the
// instance newly references, so they cannot seed it with credentials they
// could not otherwise see.
func (h *rbacHandler) ensureUserSecretReadable(ctx context.Context, cluster string, current, instance *corev1alpha1.Instance) error {
	secret := instance.Spec.UserSecretRef
	if secret == nil {
		return nil
	}
	if current != nil && current.Spec.UserSecretRef != nil && current.Spec.UserSecretRef.Name == secret.Name {
		return nil
	}
	object := rbac.ClusterNamespacedObjectName(cluster, instance.GetNamespace(), secret.Name)
	return h.enforce(ctx, rbac.ResourceSecrets, rbac.ActionRead, object)
}

// ensureInstanceBackupAuthorized checks the backup class and storages the
// instance newly references, and that the caller may create backups of it
// when it adds or changes a schedule or PITR.
func (h *rbacHandler) ensureInstanceBackupAuthorized(
	ctx context.Context,
	cluster string,
	current *corev1alpha1.InstanceBackupSpec,
	instance *corev1alpha1.Instance,
) error {
	backup := instance.Spec.Backup
	if backup == nil {
		return nil
	}
	if current == nil {
		current = &corev1alpha1.InstanceBackupSpec{}
	}
	if name := backup.ClassRef.Name; name != current.ClassRef.Name {
		if err := h.enforce(ctx, rbac.ResourceBackupClasses, rbac.ActionRead, rbac.ClusterObjectName(cluster, name)); err != nil {
			return err
		}
	}

	currentStorages := make(map[string]corev1alpha1.InstanceBackupStorage, len(current.Storages))
	for _, storage := range current.Storages {
		currentStorages[storage.StorageRef.Name] = storage
	}
	schedulesChanged := false
	for _, storage := range backup.Storages {
		currentStorage, referenced := currentStorages[storage.StorageRef.Name]
		if !referenced {
			object := rbac.ClusterNamespacedObjectName(cluster, instance.GetNamespace(), storage.StorageRef.Name)
			if err := h.enforce(ctx, rbac.ResourceBackupStorages, rbac.ActionRead, object); err != nil {
				return err
			}
		}
		schedulesSet := len(storage.Schedules) > 0 || storage.PITR != nil
		schedulesUnchanged := reflect.DeepEqual(storage.Schedules, currentStorage.Schedules) &&
			reflect.DeepEqual(storage.PITR, currentStorage.PITR)
		if schedulesSet && !schedulesUnchanged {
			schedulesChanged = true
		}
	}
	if !schedulesChanged {
		return nil
	}
	object := rbac.ClusterNamespacedObjectName(cluster, instance.GetNamespace(), instance.GetName())
	return h.enforce(ctx, rbac.ResourceBackups, rbac.ActionCreate, object)
}
