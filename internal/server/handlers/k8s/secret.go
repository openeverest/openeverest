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
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openeverest/openeverest/v2/pkg/common"
)

// CreateSecret creates a Secret and labels it as managed by OpenEverest.
// Provider and definition labels supplied by the caller are preserved.
// Secret contents are stripped so only metadata is returned.
func (h *k8sHandler) CreateSecret(ctx context.Context, _ string, namespace string, secret *corev1.Secret) (*corev1.Secret, error) {
	if secret.Labels == nil {
		secret.Labels = map[string]string{}
	}
	secret.Labels[common.OpenEverestManagedLabel] = "true" //nolint:goconst
	secret.Namespace = namespace

	res, err := h.kubeConnector.CreateSecret(ctx, secret)
	if err != nil {
		return nil, err
	}

	return stripSecretData(res), nil
}

// ListSecrets lists OpenEverest-managed secrets matching the optional provider and
// definition filters. Secret contents are stripped so only metadata is returned.
func (h *k8sHandler) ListSecrets(ctx context.Context, _ string, namespace, provider, definition string) (*corev1.SecretList, error) {
	selector := ctrlclient.MatchingLabels{common.OpenEverestManagedLabel: "true"}
	if provider != "" {
		selector[common.OpenEverestProviderLabel] = provider
	}
	if definition != "" {
		selector[common.OpenEverestDefinitionLabel] = definition
	}

	list, err := h.kubeConnector.ListSecrets(ctx, ctrlclient.InNamespace(namespace), selector)
	if err != nil {
		return nil, err
	}

	// Never expose secret contents in list responses.
	for i := range list.Items {
		list.Items[i] = *stripSecretData(&list.Items[i])
	}

	return list, nil
}

// GetSecret returns the secret specified by name. Secret contents are stripped
// so only metadata is returned.
// Secret contents are stripped so only metadata is returned.
func (h *k8sHandler) GetSecret(ctx context.Context, _ string, namespace, name string) (*corev1.Secret, error) {
	secret, err := h.kubeConnector.GetSecret(ctx, ctrlclient.ObjectKey{
		Name:      name,
		Namespace: namespace,
	})
	if err != nil {
		return nil, err
	}

	if v, ok := secret.Labels[common.OpenEverestManagedLabel]; !ok || v != "true" {
		return nil, ErrNotFound
	}

	return stripSecretData(secret), nil
}

// DeleteSecret deletes the secret specified by name in the namespace.
func (h *k8sHandler) DeleteSecret(ctx context.Context, _ string, namespace, name string) error {
	secret, err := h.kubeConnector.GetSecret(ctx, ctrlclient.ObjectKey{
		Name:      name,
		Namespace: namespace,
	})
	if err != nil {
		return err
	}

	if v, ok := secret.Labels[common.OpenEverestManagedLabel]; !ok || v != "true" {
		return ErrNotFound
	}

	return h.kubeConnector.DeleteSecret(ctx, secret)
}

// stripSecretData removes the contents of a secret, leaving only metadata.
func stripSecretData(secret *corev1.Secret) *corev1.Secret {
	secret.Data = nil
	secret.StringData = nil

	return secret
}
