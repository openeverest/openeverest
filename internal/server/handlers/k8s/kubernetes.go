// Package k8s contains the Kubernetes handler.
package k8s

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/AlekSi/pointer"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"github.com/openeverest/openeverest/v2/api"
	"github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

const (
	annotationStorageClassDefault = "storageclass.kubernetes.io/is-default-class"
)

func (h *k8sHandler) GetKubernetesClusterResources(ctx context.Context) (*api.KubernetesClusterResources, error) {
	// Get cluster type
	clusterType, err := h.kubeConnector.GetClusterType(ctx)
	if err != nil {
		// Instead of failing we switch to a generic cluster type.
		clusterType = kubernetes.ClusterTypeGeneric
	}

	var volumes *corev1.PersistentVolumeList
	if clusterType == kubernetes.ClusterTypeEKS {
		volumes, err = h.kubeConnector.ListPersistentVolumes(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to ListPersistentVolumes: %w", err)
		}
	}

	res, err := h.calculateClusterResources(ctx, h.kubeConnector, clusterType, volumes)
	if err != nil {
		return nil, fmt.Errorf("failed to calculateClusterResources: %w", err)
	}
	return res, nil
}

func (h *k8sHandler) GetKubernetesClusterInfo(ctx context.Context) (*api.KubernetesClusterInfo, error) {
	clusterType, err := h.kubeConnector.GetClusterType(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to GetClusterType: %w", err)
	}
	storagesList, err := h.kubeConnector.ListStorageClasses(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to ListStorageClasses: %w", err)
	}

	// convert k8s storage class to api storage class
	storagesListApi := []api.StorageClass{}
	for _, storageClass := range storagesList.Items {
		apiObj := api.StorageClass{}
		apiObj.FromCR(&storageClass)
		storagesListApi = append(storagesListApi, apiObj)
	}

	classNames := storageClasses(storagesList)
	return &api.KubernetesClusterInfo{
		ClusterType:       string(clusterType),
		StorageClassNames: classNames,
		StorageClasses:    &storagesListApi,
	}, nil
}

func (h *k8sHandler) GetUserPermissions(ctx context.Context) (*api.UserPermissions, error) {
	cm, err := h.kubeConnector.GetConfigMap(ctx, types.NamespacedName{Namespace: common.SystemNamespace, Name: common.EverestRBACConfigMapName})
	if err != nil {
		return nil, errors.Join(err, errors.New("could not get Everest RBAC ConfigMap"))
	}
	enabled := rbac.IsEnabled(cm)
	return &api.UserPermissions{
		Enabled: enabled,
	}, nil
}

func (h *k8sHandler) GetSettings(ctx context.Context) (*api.Settings, error) {
	settings, err := h.kubeConnector.GetEverestSettings(ctx)
	if err != nil && !k8serrors.IsNotFound(err) {
		return nil, err
	}
	config, err := settings.OIDCConfig()
	if err != nil {
		return nil, errors.Join(err, errors.New("cannot parse OIDC raw config"))
	}
	return &api.Settings{
		OidcConfig: api.OIDCConfig{
			ClientId:  config.ClientID,
			IssuerURL: config.IssuerURL,
			Scopes:    config.Scopes,
		},
	}, nil
}

func storageClasses(storagesList *storagev1.StorageClassList) []string {
	classNames := make([]string, len(storagesList.Items))
	swap := reflect.Swapper(classNames)
	for i, storageClass := range storagesList.Items {
		classNames[i] = storageClass.Name
		if _, ok := storageClass.Annotations[annotationStorageClassDefault]; ok {
			if i != 0 {
				swap(i, 0)
			}
		}
	}
	return classNames
}

func (h *k8sHandler) calculateClusterResources(
	ctx context.Context, kubeClient kubernetes.KubernetesConnector, clusterType kubernetes.ClusterType,
	volumes *corev1.PersistentVolumeList,
) (*api.KubernetesClusterResources, error) {
	allCPUMillis, allMemoryBytes, allDiskBytes, err := kubeClient.GetAllClusterResources(
		ctx, clusterType, volumes,
	)
	if err != nil {
		return nil, errors.New("could not get cluster resources")
	}

	consumedCPUMillis, consumedMemoryBytes, err := kubeClient.GetConsumedCPUAndMemory(ctx, "")
	if err != nil {
		return nil, errors.New("could not get consumed cpu and memory")
	}

	consumedDiskBytes, err := kubeClient.GetConsumedDiskBytes(ctx, clusterType, volumes)
	if err != nil {
		return nil, errors.New("could not get consumed disk bytes")
	}

	availableCPUMillis := allCPUMillis - consumedCPUMillis
	// handle underflow
	if availableCPUMillis > allCPUMillis {
		availableCPUMillis = 0
	}
	availableMemoryBytes := allMemoryBytes - consumedMemoryBytes
	// handle underflow
	if availableMemoryBytes > allMemoryBytes {
		availableMemoryBytes = 0
	}
	availableDiskBytes := allDiskBytes - consumedDiskBytes
	// handle underflow
	if availableDiskBytes > allDiskBytes {
		availableDiskBytes = 0
	}

	res := &api.KubernetesClusterResources{
		Capacity: api.ResourcesCapacity{
			CpuMillis:   pointer.ToUint64OrNil(allCPUMillis),
			MemoryBytes: pointer.ToUint64OrNil(allMemoryBytes),
			DiskSize:    pointer.ToUint64OrNil(allDiskBytes),
		},
		Available: api.ResourcesAvailable{
			CpuMillis:   pointer.ToUint64OrNil(availableCPUMillis),
			MemoryBytes: pointer.ToUint64OrNil(availableMemoryBytes),
			DiskSize:    pointer.ToUint64OrNil(availableDiskBytes),
		},
	}

	return res, nil
}
