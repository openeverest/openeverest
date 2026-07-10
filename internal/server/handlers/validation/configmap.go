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

package validation

import (
	"context"
	"fmt"

	"github.com/percona/everest-operator/utils"
	corev1 "k8s.io/api/core/v1"

	"github.com/openeverest/openeverest/v2/pkg/common"
)

// CreateConfigMap validates and creates a ConfigMap.
func (h *validateHandler) CreateConfigMap(ctx context.Context, cluster, namespace string, configMap *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	if err := utils.ValidateEverestResourceName(configMap.Name, "name"); err != nil {
		return nil, fmt.Errorf("invalid ConfigMap name: %w", err)
	}

	// Validate required category label.
	if configMap.Labels == nil || configMap.Labels[common.OpenEverestCategoryLabel] == "" {
		return nil, fmt.Errorf("ConfigMap must have a '%s' label", common.OpenEverestCategoryLabel)
	}

	return h.next.CreateConfigMap(ctx, cluster, namespace, configMap)
}

// ListConfigMaps proxies the request to the next handler.
func (h *validateHandler) ListConfigMaps(ctx context.Context, cluster, namespace, provider, category string) (*corev1.ConfigMapList, error) {
	return h.next.ListConfigMaps(ctx, cluster, namespace, provider, category)
}

// GetConfigMap proxies the request to the next handler.
func (h *validateHandler) GetConfigMap(ctx context.Context, cluster, namespace, name string) (*corev1.ConfigMap, error) {
	return h.next.GetConfigMap(ctx, cluster, namespace, name)
}

// DeleteConfigMap proxies the request to the next handler.
func (h *validateHandler) DeleteConfigMap(ctx context.Context, cluster, namespace, name string) error {
	// TODO: Check if configmap is in use by any instances before allowing deletion.
	return h.next.DeleteConfigMap(ctx, cluster, namespace, name)
}
