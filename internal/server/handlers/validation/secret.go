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
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/pkg/common"
	schemavalidation "github.com/openeverest/openeverest/v2/pkg/openapi"
)

// CreateSecret validates the secret request and proxies it to the next handler.
func (h *validateHandler) CreateSecret(ctx context.Context, cluster, namespace string, secret *corev1.Secret) (*corev1.Secret, error) {
	if err := utils.ValidateEverestResourceName(secret.GetName(), "name"); err != nil {
		return nil, errors.Join(ErrInvalidRequest, err)
	}

	labels := secret.GetLabels()

	if labels == nil || labels[common.OpenEverestDefinitionLabel] == "" {
		return nil, errors.Join(ErrInvalidRequest, errors.New("secret must have a definition label"))
	}

	definition := labels[common.OpenEverestDefinitionLabel]
	providerName := labels[common.OpenEverestProviderLabel]

	// Get the secret definition schema for validation.
	secretDef, err := h.getSecretDefinition(ctx, providerName, definition)
	if err != nil {
		return nil, err
	}

	// Validate secret data against the schema if one exists.
	if secretDef != nil && secretDef.OpenAPIV3Schema != nil {
		if err := h.validateSecretData(secret, secretDef.OpenAPIV3Schema); err != nil {
			return nil, errors.Join(ErrInvalidRequest, err)
		}
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

// getSecretDefinition retrieves the secret definition from provider(s).
// If providerName is set, it fetches that specific provider.
// If providerName is empty, it lists all providers and finds one with the definition marked as shared.
func (h *validateHandler) getSecretDefinition(ctx context.Context, providerName, definition string) (*corev1alpha1.SecretDefinition, error) {
	if providerName != "" {
		// Fetch the specific provider.
		provider, err := h.kubeConnector.GetProvider(ctx, ctrlclient.ObjectKey{Name: providerName})
		if err != nil {
			return nil, errors.Join(ErrInvalidRequest, fmt.Errorf("failed to get provider %q: %w", providerName, err))
		}

		secretDef, ok := provider.Spec.Secrets[definition]
		if !ok {
			return nil, errors.Join(ErrInvalidRequest, fmt.Errorf("secret definition %q not found in provider %q", definition, providerName))
		}

		return &secretDef, nil
	}

	// Provider not set - list all providers and find one with shared definition.
	providers, err := h.kubeConnector.ListProviders(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list providers: %w", err)
	}

	for _, provider := range providers.Items {
		if secretDef, ok := provider.Spec.Secrets[definition]; ok && secretDef.Shared {
			return &secretDef, nil
		}
	}

	// No secret definition found
	return nil, errors.Join(ErrInvalidRequest, fmt.Errorf("shared secret definition %q not found for any provider", definition))
}

// validateSecretData validates the secret's data or stringData against the OpenAPI v3 schema.
func (h *validateHandler) validateSecretData(secret *corev1.Secret, schema *apiextensionsv1.JSONSchemaProps) error {
	// Merge data and stringData into a single map for validation.
	// stringData takes precedence (same as Kubernetes behavior).
	data := make(map[string]any)

	// Add decoded data entries.
	for k, v := range secret.Data {
		data[k] = string(v)
	}

	// Add stringData entries (overrides data if same key).
	for k, v := range secret.StringData {
		data[k] = v
	}

	if len(data) == 0 {
		return errors.New("secret must have data or stringData")
	}

	return schemavalidation.Validate(schema, data)
}
