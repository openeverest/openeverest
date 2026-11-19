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

// Package k8s contains the Kubernetes handler.
package k8s

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"

	storagev1 "k8s.io/api/storage/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	api "github.com/openeverest/openeverest/v2/internal/server/api"
	"github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

const (
	annotationStorageClassDefault      = "storageclass.kubernetes.io/is-default-class"
	annotationStorageClassDefaultValue = "true"
)

func (h *k8sHandler) GetKubernetesClusterInfo(ctx context.Context, _ string) (*api.KubernetesClusterInfo, error) {
	clusterType, err := h.kubeConnector.GetClusterType(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to GetClusterType: %w", err)
	}
	storagesList, err := h.kubeConnector.ListStorageClasses(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to ListStorageClasses: %w", err)
	}

	// convert k8s storage class to api storage class
	storagesListAPI := []api.StorageClass{}
	for _, storageClass := range storagesList.Items {
		apiObj := api.StorageClass{}
		apiObj.FromCR(&storageClass)
		storagesListAPI = append(storagesListAPI, apiObj)
	}

	classNames := storageClasses(storagesList)
	return &api.KubernetesClusterInfo{
		ClusterType:       string(clusterType),
		StorageClassNames: classNames,
		StorageClasses:    &storagesListAPI,
	}, nil
}

func (h *k8sHandler) GetUserPermissions(ctx context.Context) (*api.UserPermissions, error) {
	cm, err := h.kubeConnector.GetConfigMap(ctx, types.NamespacedName{Namespace: h.kubeConnector.Namespace(), Name: common.EverestRBACConfigMapName})
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

// storageClasses extracts the names of StorageClasses, placing default StorageClasses
// at the beginning of the list (ordered by CreationTimestamp descending, with the most
// recently created default at index 0 to match Kubernetes PVC default resolution behavior),
// followed by non-default StorageClasses preserving their original order.
func storageClasses(storagesList *storagev1.StorageClassList) []string {
	if storagesList == nil || len(storagesList.Items) == 0 {
		return []string{}
	}

	items := slices.Clone(storagesList.Items)
	sort.SliceStable(items, func(i, j int) bool {
		iDefault := items[i].Annotations[annotationStorageClassDefault] == annotationStorageClassDefaultValue
		jDefault := items[j].Annotations[annotationStorageClassDefault] == annotationStorageClassDefaultValue

		if iDefault != jDefault {
			return iDefault
		}
		if iDefault {
			return items[i].CreationTimestamp.After(items[j].CreationTimestamp.Time)
		}
		return false
	})

	classNames := make([]string, len(items))
	for i, sc := range items {
		classNames[i] = sc.Name
	}

	return classNames
}
