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

	corev1 "k8s.io/api/core/v1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ListConfigMaps returns list of configmaps that match the criteria.
func (k *Kubernetes) ListConfigMaps(ctx context.Context, opts ...ctrlclient.ListOption) (*corev1.ConfigMapList, error) {
	result := &corev1.ConfigMapList{}
	if err := k.k8sClient.List(ctx, result, opts...); err != nil {
		return nil, err
	}
	return result, nil
}

// GetConfigMap returns k8s configmap that matches the criteria.
func (k *Kubernetes) GetConfigMap(ctx context.Context, key ctrlclient.ObjectKey) (*corev1.ConfigMap, error) {
	result := &corev1.ConfigMap{}
	if err := k.k8sClient.Get(ctx, key, result); err != nil {
		return nil, err
	}
	return result, nil
}

// CreateConfigMap creates k8s configmap.
func (k *Kubernetes) CreateConfigMap(ctx context.Context, config *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	if err := k.k8sClient.Create(ctx, config); err != nil {
		return nil, err
	}
	return config, nil
}

// UpdateConfigMap updates k8s configmap.
func (k *Kubernetes) UpdateConfigMap(ctx context.Context, config *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	if err := k.k8sClient.Update(ctx, config); err != nil {
		return nil, err
	}
	return config, nil
}

// DeleteConfigMap deletes a configmap.
func (k *Kubernetes) DeleteConfigMap(ctx context.Context, obj *corev1.ConfigMap) error {
	return k.k8sClient.Delete(ctx, obj)
}
