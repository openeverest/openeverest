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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openeverest/openeverest/v2/pkg/common"
)

// CreateConfigMap creates a ConfigMap and labels it as managed by OpenEverest.
// Provider and definition labels supplied by the caller are preserved.
func (h *k8sHandler) CreateConfigMap(ctx context.Context, _ string, namespace string, configMap *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	if configMap.Labels == nil {
		configMap.Labels = map[string]string{}
	}

	configMap.Labels[common.OpenEverestManagedLabel] = "true" //nolint:goconst
	configMap.Namespace = namespace

	return h.kubeConnector.CreateConfigMap(ctx, configMap)
}

// ListConfigMaps lists OpenEverest-managed ConfigMaps matching the optional provider and
// definition filters.
func (h *k8sHandler) ListConfigMaps(ctx context.Context, _ string, namespace, provider, definition string) (*corev1.ConfigMapList, error) {
	selector := ctrlclient.MatchingLabels{common.OpenEverestManagedLabel: "true"}
	if provider != "" {
		selector[common.OpenEverestProviderLabel] = provider
	}

	if definition != "" {
		selector[common.OpenEverestDefinitionLabel] = definition
	}

	return h.kubeConnector.ListConfigMaps(ctx, ctrlclient.InNamespace(namespace), selector)
}

// GetConfigMap returns the ConfigMap specified by name.
// If the ConfigMap is not managed by OpenEverest, it returns ErrNotFound.
func (h *k8sHandler) GetConfigMap(ctx context.Context, _ string, namespace, name string) (*corev1.ConfigMap, error) {
	configMap, err := h.kubeConnector.GetConfigMap(ctx, ctrlclient.ObjectKey{
		Name:      name,
		Namespace: namespace,
	})
	if err != nil {
		return nil, err
	}

	if v, ok := configMap.Labels[common.OpenEverestManagedLabel]; !ok || v != "true" {
		return nil, ErrNotFound
	}

	return configMap, nil
}

// DeleteConfigMap deletes the ConfigMap specified by name in the namespace.
// It only allows deletion of OpenEverest-managed ConfigMaps.
// If the ConfigMap is not managed by OpenEverest, it returns ErrNotFound.
func (h *k8sHandler) DeleteConfigMap(ctx context.Context, _ string, namespace, name string) error {
	configMap, err := h.kubeConnector.GetConfigMap(ctx, ctrlclient.ObjectKey{
		Name:      name,
		Namespace: namespace,
	})
	if err != nil {
		return err
	}

	if v, ok := configMap.Labels[common.OpenEverestManagedLabel]; !ok || v != "true" {
		return ErrNotFound
	}

	return h.kubeConnector.DeleteConfigMap(ctx, &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	})
}
