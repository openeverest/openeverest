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

// CreateSecret validates the secret request and proxies it to the next handler.
func (h *validateHandler) CreateSecret(ctx context.Context, cluster, namespace string, secret *corev1.Secret) (*corev1.Secret, error) {
	if err := utils.ValidateEverestResourceName(secret.GetName(), "name"); err != nil {
		return nil, errors.Join(ErrInvalidRequest, err)
	}

	definition := secret.GetLabels()[common.OpenEverestDefinitionLabel]
	if definition == "" {
		return nil, errors.Join(ErrInvalidRequest, fmt.Errorf("missing %s label", common.OpenEverestDefinitionLabel))
	}

	providerName := secret.GetLabels()[common.OpenEverestProviderLabel]
	if providerName == "" {
		return nil, errors.Join(ErrInvalidRequest, fmt.Errorf("missing %s label", common.OpenEverestProviderLabel))
	}

	// Fetch the provider for validation.
	provider, err := h.kubeConnector.GetProvider(ctx, ctrlclient.ObjectKey{Name: providerName})
	if err != nil {
		return nil, errors.Join(ErrInvalidRequest, fmt.Errorf("failed to get provider %q: %w", providerName, err))
	}

	// Validate secret data against the schema.
	if err := controller.ValidateSecretSchema(secret, &provider.Spec); err != nil {
		return nil, errors.Join(ErrInvalidRequest, err)
	}

	return h.next.CreateSecret(ctx, cluster, namespace, secret)
}

// ListSecrets proxies the request to the next handler.
func (h *validateHandler) ListSecrets(ctx context.Context, cluster, namespace, provider, definition string) (*corev1.SecretList, error) {
	return h.next.ListSecrets(ctx, cluster, namespace, provider, definition)
}

// GetSecret proxies the request to the next handler.
func (h *validateHandler) GetSecret(ctx context.Context, cluster, namespace, name string) (*corev1.Secret, error) {
	return h.next.GetSecret(ctx, cluster, namespace, name)
}

// DeleteSecret proxies the request to the next handler.
func (h *validateHandler) DeleteSecret(ctx context.Context, cluster, namespace, name string) error {
	// TODO: prevent deletion of secrets that are still in use by an Instance (spec 011 §4.5).
	return h.next.DeleteSecret(ctx, cluster, namespace, name)
}
