// everest
// Copyright (C) 2023 Percona LLC
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

// Package kubernetes ...
package kubernetes

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
)

// ListBackupStorages returns list of managed backup storages in a given namespace.
// This method returns a list of full objects (meta and spec).
func (k *Kubernetes) ListBackupStorages(ctx context.Context, opts ...ctrlclient.ListOption) (*everestv1alpha1.BackupStorageList, error) {
	result := &everestv1alpha1.BackupStorageList{}
	if err := k.k8sClient.List(ctx, result, opts...); err != nil {
		return nil, err
	}
	return result, nil
}

// listBackupStoragesMeta returns list of managed backup storages in a given namespace.
// This method returns a list of simplified objects (meta only).
func (k *Kubernetes) listBackupStoragesMeta(ctx context.Context, opts ...ctrlclient.ListOption) (*metav1.PartialObjectMetadataList, error) {
	bsListMeta := &metav1.PartialObjectMetadataList{}
	bsListMeta.SetGroupVersionKind(everestv1alpha1.GroupVersion.WithKind("BackupStorageList"))
	if err := k.k8sClient.List(ctx, bsListMeta, opts...); err != nil {
		return nil, err
	}
	return bsListMeta, nil
}

// GetBackupStorage returns backup storages(full object) by provided name and namespace.
func (k *Kubernetes) GetBackupStorage(ctx context.Context, key ctrlclient.ObjectKey) (*everestv1alpha1.BackupStorage, error) {
	result := &everestv1alpha1.BackupStorage{}
	if err := k.k8sClient.Get(ctx, key, result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetBackupStorageMeta returns backup storages(metadata only) by provided name and namespace.
func (k *Kubernetes) GetBackupStorageMeta(ctx context.Context, key ctrlclient.ObjectKey) (*metav1.PartialObjectMetadata, error) {
	objMeta := &metav1.PartialObjectMetadata{}
	objMeta.SetGroupVersionKind(everestv1alpha1.GroupVersion.WithKind("BackupStorage"))
	if err := k.k8sClient.Get(ctx, key, objMeta); err != nil {
		return nil, err
	}
	return objMeta, nil
}

// CreateBackupStorage creates backup storages by provided object.
func (k *Kubernetes) CreateBackupStorage(ctx context.Context, storage *everestv1alpha1.BackupStorage) (*everestv1alpha1.BackupStorage, error) {
	if err := k.k8sClient.Create(ctx, storage); err != nil {
		return nil, err
	}
	return storage, nil
}

// UpdateBackupStorage updates backup storages by provided new object.
func (k *Kubernetes) UpdateBackupStorage(ctx context.Context, storage *everestv1alpha1.BackupStorage) (*everestv1alpha1.BackupStorage, error) {
	if err := k.k8sClient.Update(ctx, storage); err != nil {
		return nil, err
	}
	return storage, nil
}

// DeleteBackupStorage deletes backup storage by provided name and namespace.
func (k *Kubernetes) DeleteBackupStorage(ctx context.Context, obj *everestv1alpha1.BackupStorage) error {
	return k.k8sClient.Delete(ctx, obj)
}

// DeleteBackupStorages deletes all backup storages in provided namespace.
// This function will wait until all storages are deleted.
func (k *Kubernetes) DeleteBackupStorages(ctx context.Context, opts ...ctrlclient.ListOption) error {
	// No need to fetch full objects, we only need the fact there are objects that match the criteria(opts).
	delList, err := k.listBackupStoragesMeta(ctx, opts...)
	if err != nil {
		k.l.Errorf("Could not list backup storages: %s", err)
		return err
	}

	if delList == nil || len(delList.Items) == 0 {
		// Nothing to delete.
		return nil
	}

	// need to convert ListOptions to DeleteAllOfOptions
	delOpts := &ctrlclient.DeleteAllOfOptions{}
	for _, opt := range opts {
		opt.ApplyToList(&delOpts.ListOptions)
	}

	k.l.Debugf("Setting backup storages removal timeout to %s", pollTimeout)
	return wait.PollUntilContextTimeout(ctx, pollInterval, pollTimeout, true, func(ctx context.Context) (bool, error) {
		// Skip fetching the list of objects to delete again, we already have it (see code above).
		if delList == nil {
			var err error
			if delList, err = k.listBackupStoragesMeta(ctx, opts...); err != nil {
				k.l.Errorf("Could not list backup storages in polling: %s", err)
				return false, err
			}

			if delList == nil || len(delList.Items) == 0 {
				// Nothing to delete.
				return true, nil
			}
		}

		// Reset the list to nil to fetch it again on the next iteration.
		delList = nil

		if err := k.k8sClient.DeleteAllOf(ctx, &everestv1alpha1.BackupStorage{}, delOpts); err != nil {
			return false, err
		}
		return false, nil
	})
}
