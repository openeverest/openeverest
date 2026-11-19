package kubernetes

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ListPersistentVolumes returns list of persistent volumes that match the criteria.
// This method returns a list of full objects (meta and spec).
func (k *Kubernetes) ListPersistentVolumes(ctx context.Context, opts ...ctrlclient.ListOption) (*corev1.PersistentVolumeList, error) {
	result := &corev1.PersistentVolumeList{}
	if err := k.k8sClient.List(ctx, result, opts...); err != nil {
		return nil, err
	}
	return result, nil
}

// ListStorageClasses returns list of storage classes that match the criteria.
// This method returns a list of full objects (meta and spec).
func (k *Kubernetes) ListStorageClasses(ctx context.Context, opts ...ctrlclient.ListOption) (*storagev1.StorageClassList, error) {
	result := &storagev1.StorageClassList{}
	if err := k.k8sClient.List(ctx, result, opts...); err != nil {
		return nil, err
	}
	return result, nil
}

// GetStorageClass returns the storage class with the given key.
func (k *Kubernetes) GetStorageClass(ctx context.Context, key ctrlclient.ObjectKey) (*storagev1.StorageClass, error) {
	result := &storagev1.StorageClass{}
	if err := k.k8sClient.Get(ctx, key, result); err != nil {
		return nil, err
	}
	return result, nil
}
