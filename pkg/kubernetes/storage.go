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
	err := listPaginated(ctx, k.k8sClient, result,
		func() *corev1.PersistentVolumeList { return &corev1.PersistentVolumeList{} },
		func(res, page *corev1.PersistentVolumeList) { res.Items = append(res.Items, page.Items...) },
		opts...,
	)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// ListStorageClasses returns list of storage classes that match the criteria.
// This method returns a list of full objects (meta and spec).
func (k *Kubernetes) ListStorageClasses(ctx context.Context, opts ...ctrlclient.ListOption) (*storagev1.StorageClassList, error) {
	result := &storagev1.StorageClassList{}
	err := listPaginated(ctx, k.k8sClient, result,
		func() *storagev1.StorageClassList { return &storagev1.StorageClassList{} },
		func(res, page *storagev1.StorageClassList) { res.Items = append(res.Items, page.Items...) },
		opts...,
	)
	if err != nil {
		return nil, err
	}
	return result, nil
}
