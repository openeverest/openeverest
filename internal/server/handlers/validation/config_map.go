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
	"errors"
	"fmt"

	"github.com/percona/everest-operator/utils"
	corev1 "k8s.io/api/core/v1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

// CreateConfigMap validates and creates a ConfigMap.
func (h *validateHandler) CreateConfigMap(ctx context.Context, cluster, namespace string, configMap *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	if err := utils.ValidateEverestResourceName(configMap.Name, "name"); err != nil {
		return nil, errors.Join(ErrInvalidRequest, fmt.Errorf("invalid ConfigMap name: %w", err))
	}

	definition := configMap.GetLabels()[common.OpenEverestDefinitionLabel]
	if definition == "" {
		return nil, errors.Join(ErrInvalidRequest, fmt.Errorf("missing %s label", common.OpenEverestDefinitionLabel))
	}

	providerName := configMap.GetLabels()[common.OpenEverestProviderLabel]
	if providerName == "" {
		return nil, errors.Join(ErrInvalidRequest, fmt.Errorf("missing %s label", common.OpenEverestProviderLabel))
	}

	// Fetch the provider for validation.
	provider, err := h.kubeConnector.GetProvider(ctx, ctrlclient.ObjectKey{Name: providerName})
	if err != nil {
		return nil, errors.Join(ErrInvalidRequest, fmt.Errorf("failed to get provider %q: %w", providerName, err))
	}

	// Validate configmap data against the schema.
	if err := controller.ValidateConfigMapSchema(configMap, &provider.Spec); err != nil {
		return nil, errors.Join(ErrInvalidRequest, err)
	}

	return h.next.CreateConfigMap(ctx, cluster, namespace, configMap)
}

// ListConfigMaps proxies the request to the next handler.
func (h *validateHandler) ListConfigMaps(ctx context.Context, cluster, namespace, provider, definition string) (*corev1.ConfigMapList, error) {
	return h.next.ListConfigMaps(ctx, cluster, namespace, provider, definition)
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
