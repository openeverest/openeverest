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

package kubernetes

import (
	"context"

	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
)

// ListBackupStorages returns list of backup storages in a given namespace.
func (k *Kubernetes) ListBackupStorages(ctx context.Context, opts ...ctrlclient.ListOption) (*backupv1alpha1.BackupStorageList, error) {
	result := &backupv1alpha1.BackupStorageList{}
	if err := k.k8sClient.List(ctx, result, opts...); err != nil {
		return nil, err
	}
	return result, nil
}

// GetBackupStorage returns a backup storage by name and namespace.
func (k *Kubernetes) GetBackupStorage(ctx context.Context, key ctrlclient.ObjectKey) (*backupv1alpha1.BackupStorage, error) {
	result := &backupv1alpha1.BackupStorage{}
	if err := k.k8sClient.Get(ctx, key, result); err != nil {
		return nil, err
	}
	return result, nil
}

// CreateBackupStorage creates a backup storage.
func (k *Kubernetes) CreateBackupStorage(ctx context.Context, storage *backupv1alpha1.BackupStorage) (*backupv1alpha1.BackupStorage, error) {
	if err := k.k8sClient.Create(ctx, storage); err != nil {
		return nil, err
	}
	return storage, nil
}

// UpdateBackupStorage updates a backup storage.
func (k *Kubernetes) UpdateBackupStorage(ctx context.Context, storage *backupv1alpha1.BackupStorage) (*backupv1alpha1.BackupStorage, error) {
	if err := k.k8sClient.Update(ctx, storage); err != nil {
		return nil, err
	}
	return storage, nil
}

// DeleteBackupStorage deletes a backup storage.
func (k *Kubernetes) DeleteBackupStorage(ctx context.Context, obj *backupv1alpha1.BackupStorage) error {
	return k.k8sClient.Delete(ctx, obj)
}
