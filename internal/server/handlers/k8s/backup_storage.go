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

package k8s

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
)

// ListBackupStorages returns the backup storages in a namespace: only those
// backupClass can write to, as published in status.backupClasses, and only
// those of backupStorageClass, when set.
func (h *k8sHandler) ListBackupStorages(ctx context.Context, _, namespace, backupClass, backupStorageClass string) (*backupv1alpha1.BackupStorageList, error) {
	list, err := h.kubeConnector.ListBackupStorages(ctx, ctrlclient.InNamespace(namespace))
	if err != nil || (backupClass == "" && backupStorageClass == "") {
		return list, err
	}
	filtered := make([]backupv1alpha1.BackupStorage, 0, len(list.Items))
	for _, bs := range list.Items {
		if backupStorageClass != "" && bs.Spec.BackupStorageClassRef.Name != backupStorageClass {
			continue
		}
		if backupClass != "" && !writableBy(&bs, backupClass) {
			continue
		}
		filtered = append(filtered, bs)
	}
	list.Items = filtered
	return list, nil
}

func writableBy(bs *backupv1alpha1.BackupStorage, backupClass string) bool {
	for _, c := range bs.Status.BackupClasses {
		if c.Name == backupClass {
			return c.Compatible
		}
	}
	return false
}

// GetBackupStorage returns a backup storage by name and namespace.
func (h *k8sHandler) GetBackupStorage(ctx context.Context, _, namespace, name string) (*backupv1alpha1.BackupStorage, error) {
	return h.kubeConnector.GetBackupStorage(ctx, types.NamespacedName{Namespace: namespace, Name: name})
}

// CreateBackupStorage creates a backup storage. Its credentials Secret is
// created beforehand, through the Secrets API.
func (h *k8sHandler) CreateBackupStorage(ctx context.Context, _ string, bs *backupv1alpha1.BackupStorage) (*backupv1alpha1.BackupStorage, error) {
	return h.kubeConnector.CreateBackupStorage(ctx, bs)
}

// UpdateBackupStorage updates a backup storage.
func (h *k8sHandler) UpdateBackupStorage(ctx context.Context, _ string, bs *backupv1alpha1.BackupStorage) (*backupv1alpha1.BackupStorage, error) {
	return h.kubeConnector.UpdateBackupStorage(ctx, bs)
}

// PatchBackupStorage applies a merge patch to a backup storage, leaving the read-modify-write to the API server.
// Strict validation is not optional: without it a misspelt path is pruned and returns 200 having changed nothing.
func (h *k8sHandler) PatchBackupStorage(ctx context.Context, _, namespace, name string, patch []byte) (*backupv1alpha1.BackupStorage, error) {
	bs := &backupv1alpha1.BackupStorage{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name},
	}
	return h.kubeConnector.PatchBackupStorage(ctx, bs,
		ctrlclient.RawPatch(types.MergePatchType, patch),
		ctrlclient.FieldValidation(metav1.FieldValidationStrict),
	)
}

// DeleteBackupStorage deletes a backup storage.
func (h *k8sHandler) DeleteBackupStorage(ctx context.Context, _, namespace, name string) error {
	bs, err := h.kubeConnector.GetBackupStorage(ctx, types.NamespacedName{Namespace: namespace, Name: name})
	if ctrlclient.IgnoreNotFound(err) != nil {
		return fmt.Errorf("failed to get backup storage: %w", err)
	}

	if bs == nil {
		// nothing to delete
		return nil
	}

	if err := h.kubeConnector.DeleteBackupStorage(ctx, bs); ctrlclient.IgnoreNotFound(err) != nil {
		return fmt.Errorf("failed to delete backup storage: %w", err)
	}

	return nil
}

// ListBackupStorageClasses returns the backup storage classes.
func (h *k8sHandler) ListBackupStorageClasses(ctx context.Context, _ string) (*backupv1alpha1.BackupStorageClassList, error) {
	return h.kubeConnector.ListBackupStorageClasses(ctx)
}

// GetBackupStorageClass returns a backup storage class by name.
func (h *k8sHandler) GetBackupStorageClass(ctx context.Context, _, name string) (*backupv1alpha1.BackupStorageClass, error) {
	return h.kubeConnector.GetBackupStorageClass(ctx, types.NamespacedName{Name: name})
}
