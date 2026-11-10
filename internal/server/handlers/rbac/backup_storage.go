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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	jsonpatch "github.com/evanphx/json-patch/v5"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	valhandler "github.com/openeverest/openeverest/v2/internal/server/handlers/validation"
	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

// ListBackupStorages returns backup storages filtered by RBAC permissions.
func (h *rbacHandler) ListBackupStorages(ctx context.Context, cluster, namespace, backupClass, backupStorageClass string) (*backupv1alpha1.BackupStorageList, error) {
	list, err := h.next.ListBackupStorages(ctx, cluster, namespace, backupClass, backupStorageClass)
	if err != nil {
		return nil, fmt.Errorf("ListBackupStorages failed: %w", err)
	}
	filtered := make([]backupv1alpha1.BackupStorage, 0, len(list.Items))
	for _, bs := range list.Items {
		object := rbac.ClusterNamespacedObjectName(cluster, bs.GetNamespace(), bs.GetName())
		if err := h.enforce(ctx, rbac.ResourceBackupStorages, rbac.ActionRead, object); errors.Is(err, ErrInsufficientPermissions) {
			continue
		} else if err != nil {
			return nil, fmt.Errorf("enforce failed: %w", err)
		}
		filtered = append(filtered, bs)
	}
	list.Items = filtered
	return list, nil
}

// GetBackupStorage returns a backup storage, gated by RBAC.
func (h *rbacHandler) GetBackupStorage(ctx context.Context, cluster, namespace, name string) (*backupv1alpha1.BackupStorage, error) {
	object := rbac.ClusterNamespacedObjectName(cluster, namespace, name)
	if err := h.enforce(ctx, rbac.ResourceBackupStorages, rbac.ActionRead, object); err != nil {
		return nil, err
	}
	return h.next.GetBackupStorage(ctx, cluster, namespace, name)
}

// CreateBackupStorage creates a backup storage, gated by RBAC. It also
// requires read on the class it references and use on every Secret it names:
// the storage hands the Secrets' contents to whatever it is attached to.
func (h *rbacHandler) CreateBackupStorage(ctx context.Context, cluster string, bs *backupv1alpha1.BackupStorage) (*backupv1alpha1.BackupStorage, error) {
	object := rbac.ClusterNamespacedObjectName(cluster, bs.GetNamespace(), bs.GetName())
	if err := h.enforce(ctx, rbac.ResourceBackupStorages, rbac.ActionCreate, object); err != nil {
		return nil, err
	}
	if err := h.enforce(ctx, rbac.ResourceBackupStorageClasses, rbac.ActionRead, rbac.ClusterObjectName(cluster, bs.Spec.BackupStorageClassRef.Name)); err != nil {
		return nil, err
	}
	if err := h.enforceUseStorageSecrets(ctx, cluster, bs.GetNamespace(), bs.Spec.SecretNames()); err != nil {
		return nil, err
	}
	return h.next.CreateBackupStorage(ctx, cluster, bs)
}

// UpdateBackupStorage updates a backup storage, gated by RBAC. It also
// requires read on the class it references and, when the update changes
// spec, use on every Secret the updated storage names, including those it
// already named: pointing a storage at another endpoint while keeping its
// Secret sends the Secret's contents there. An update of labels or
// annotations only needs no use.
func (h *rbacHandler) UpdateBackupStorage(ctx context.Context, cluster string, bs *backupv1alpha1.BackupStorage) (*backupv1alpha1.BackupStorage, error) {
	object := rbac.ClusterNamespacedObjectName(cluster, bs.GetNamespace(), bs.GetName())
	if err := h.enforce(ctx, rbac.ResourceBackupStorages, rbac.ActionUpdate, object); err != nil {
		return nil, err
	}
	if err := h.enforce(ctx, rbac.ResourceBackupStorageClasses, rbac.ActionRead, rbac.ClusterObjectName(cluster, bs.Spec.BackupStorageClassRef.Name)); err != nil {
		return nil, err
	}
	current, err := h.next.GetBackupStorage(ctx, cluster, bs.GetNamespace(), bs.GetName())
	if err != nil {
		return nil, err
	}
	if err := h.enforceUseOnStorageSpecChange(ctx, cluster, current, bs); err != nil {
		return nil, err
	}
	// Pin the write to the version the check ran against, so a spec change
	// landing in between can't be overwritten unchecked. A caller-supplied
	// version is left alone: the API server rejects it if it is not current.
	if bs.GetResourceVersion() == "" {
		bs.SetResourceVersion(current.GetResourceVersion())
	}
	return h.next.UpdateBackupStorage(ctx, cluster, bs)
}

// PatchBackupStorage patches a backup storage, gated by RBAC. A patch is
// authorised as an update, and the use check of UpdateBackupStorage runs on
// the storage the patch produces, not on the patch: a patch that changes only
// parameters.endpointURL sends the stored Secrets elsewhere.
func (h *rbacHandler) PatchBackupStorage(ctx context.Context, cluster, namespace, name string, patch []byte) (*backupv1alpha1.BackupStorage, error) {
	object := rbac.ClusterNamespacedObjectName(cluster, namespace, name)
	if err := h.enforce(ctx, rbac.ResourceBackupStorages, rbac.ActionUpdate, object); err != nil {
		return nil, err
	}
	// Refuse a malformed patch before reading anything.
	var doc map[string]any
	if err := json.Unmarshal(patch, &doc); err != nil || doc == nil {
		return nil, errors.Join(valhandler.ErrInvalidRequest, errors.New("invalid patch: must be a JSON object"))
	}
	current, err := h.next.GetBackupStorage(ctx, cluster, namespace, name)
	if err != nil {
		return nil, err
	}
	patched, err := mergePatchBackupStorage(current, patch)
	if err != nil {
		return nil, err
	}
	if err := h.enforceUseOnStorageSpecChange(ctx, cluster, current, patched); err != nil {
		return nil, err
	}
	// Same pin as UpdateBackupStorage. patched carries the caller's version
	// if the patch named one, and the stored one otherwise.
	pin, err := json.Marshal(map[string]any{"metadata": map[string]any{"resourceVersion": patched.GetResourceVersion()}})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal resource version: %w", err)
	}
	if patch, err = jsonpatch.MergePatch(patch, pin); err != nil {
		return nil, fmt.Errorf("failed to pin resource version: %w", err)
	}
	return h.next.PatchBackupStorage(ctx, cluster, namespace, name, patch)
}

// mergePatchBackupStorage returns the storage a JSON merge patch would
// produce.
func mergePatchBackupStorage(current *backupv1alpha1.BackupStorage, patch []byte) (*backupv1alpha1.BackupStorage, error) {
	original, err := json.Marshal(current)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal backup storage: %w", err)
	}
	merged, err := jsonpatch.MergePatch(original, patch)
	if err != nil {
		return nil, errors.Join(valhandler.ErrInvalidRequest, fmt.Errorf("invalid patch: %w", err))
	}
	patched := &backupv1alpha1.BackupStorage{}
	if err := json.Unmarshal(merged, patched); err != nil {
		return nil, errors.Join(valhandler.ErrInvalidRequest, fmt.Errorf("invalid patch: %w", err))
	}
	return patched, nil
}

// enforceUseOnStorageSpecChange requires use on every Secret updated names
// when its spec differs from current's.
func (h *rbacHandler) enforceUseOnStorageSpecChange(ctx context.Context, cluster string, current, updated *backupv1alpha1.BackupStorage) error {
	if sameBackupStorageSpec(current.Spec, updated.Spec) {
		return nil
	}
	return h.enforceUseStorageSecrets(ctx, cluster, updated.GetNamespace(), updated.Spec.SecretNames())
}

// enforceUseStorageSecrets requires use on each of the named Secrets.
func (h *rbacHandler) enforceUseStorageSecrets(ctx context.Context, cluster, namespace string, secrets []string) error {
	for _, name := range secrets {
		if err := h.enforce(ctx, rbac.ResourceSecrets, rbac.ActionUse, rbac.ClusterNamespacedObjectName(cluster, namespace, name)); err != nil {
			return err
		}
	}
	return nil
}

// sameBackupStorageSpec reports whether two specs encode to the same JSON
// document. Comparing documents rather than Go values ignores how the request
// and the stored object happen to encode parameters (key order, whitespace).
// Numbers are compared as written, and a spec that fails to encode counts as
// changed, so any doubt requires the check.
func sameBackupStorageSpec(current, updated backupv1alpha1.BackupStorageSpec) bool {
	a, err := backupStorageSpecDocument(current)
	if err != nil {
		return false
	}
	b, err := backupStorageSpecDocument(updated)
	if err != nil {
		return false
	}
	return reflect.DeepEqual(a, b)
}

func backupStorageSpecDocument(spec backupv1alpha1.BackupStorageSpec) (any, error) {
	raw, err := json.Marshal(spec)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var document any
	if err := decoder.Decode(&document); err != nil {
		return nil, err
	}
	return document, nil
}

// ListBackupStorageClasses returns backup storage classes filtered by RBAC
// permissions.
func (h *rbacHandler) ListBackupStorageClasses(ctx context.Context, cluster string) (*backupv1alpha1.BackupStorageClassList, error) {
	list, err := h.next.ListBackupStorageClasses(ctx, cluster)
	if err != nil {
		return nil, fmt.Errorf("ListBackupStorageClasses failed: %w", err)
	}
	filtered := make([]backupv1alpha1.BackupStorageClass, 0, len(list.Items))
	for _, class := range list.Items {
		object := rbac.ClusterObjectName(cluster, class.GetName())
		if err := h.enforce(ctx, rbac.ResourceBackupStorageClasses, rbac.ActionRead, object); errors.Is(err, ErrInsufficientPermissions) {
			continue
		} else if err != nil {
			return nil, fmt.Errorf("enforce failed: %w", err)
		}
		filtered = append(filtered, class)
	}
	list.Items = filtered
	return list, nil
}

// GetBackupStorageClass returns a backup storage class, gated by RBAC.
func (h *rbacHandler) GetBackupStorageClass(ctx context.Context, cluster, name string) (*backupv1alpha1.BackupStorageClass, error) {
	if err := h.enforce(ctx, rbac.ResourceBackupStorageClasses, rbac.ActionRead, rbac.ClusterObjectName(cluster, name)); err != nil {
		return nil, err
	}
	return h.next.GetBackupStorageClass(ctx, cluster, name)
}

// DeleteBackupStorage deletes a backup storage, gated by RBAC.
func (h *rbacHandler) DeleteBackupStorage(ctx context.Context, cluster, namespace, name string) error {
	object := rbac.ClusterNamespacedObjectName(cluster, namespace, name)
	if err := h.enforce(ctx, rbac.ResourceBackupStorages, rbac.ActionDelete, object); err != nil {
		return err
	}
	return h.next.DeleteBackupStorage(ctx, cluster, namespace, name)
}
