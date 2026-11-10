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

	"github.com/openeverest/openeverest/v2/internal/server/handlers"
	"github.com/openeverest/openeverest/v2/pkg/common"
)

// CreateSecret creates a Secret and labels it as managed by OpenEverest.
// Definition labels supplied by the caller are preserved; owner references
// and finalizers are not, since OpenEverest manages the Secret's lifecycle.
// Secret contents are stripped so only metadata is returned.
func (h *k8sHandler) CreateSecret(ctx context.Context, _ string, namespace string, secret *corev1.Secret) (*corev1.Secret, error) {
	if secret.Labels == nil {
		secret.Labels = map[string]string{}
	}
	secret.Labels[common.OpenEverestManagedLabel] = "true" //nolint:goconst
	secret.Namespace = namespace
	secret.OwnerReferences = nil
	secret.Finalizers = nil

	res, err := h.kubeConnector.CreateSecret(ctx, secret)
	if err != nil {
		return nil, err
	}

	return stripSecretData(res), nil
}

// ListSecrets lists OpenEverest-managed secrets matching the filter.
// Secret contents are stripped so only metadata is returned.
func (h *k8sHandler) ListSecrets(ctx context.Context, _ string, namespace string, filter handlers.SecretFilter) (*corev1.SecretList, error) {
	selector := ctrlclient.MatchingLabels{common.OpenEverestManagedLabel: "true"}
	if filter.Provider != "" {
		selector[common.OpenEverestProviderLabel] = filter.Provider
	}
	if filter.MonitoringClass != "" {
		selector[common.OpenEverestMonitoringClassLabel] = filter.MonitoringClass
	}
	if filter.Definition != "" {
		selector[common.OpenEverestDefinitionLabel] = filter.Definition
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

	if !isManagedSecret(secret) {
		return nil, ErrNotFound
	}

	return stripSecretData(secret), nil
}

// UpdateSecret replaces the data of the managed Secret named by the path and
// keeps everything else from the stored Secret. Secret contents are stripped
// so only metadata is returned.
func (h *k8sHandler) UpdateSecret(ctx context.Context, _ string, namespace, name string, secret *corev1.Secret) (*corev1.Secret, error) {
	stored, err := h.kubeConnector.GetSecret(ctx, ctrlclient.ObjectKey{
		Name:      name,
		Namespace: namespace,
	})
	if err != nil {
		return nil, err
	}

	if !isManagedSecret(stored) {
		return nil, ErrNotFound
	}

	// The client's version, when it sends one, so a stale document is refused.
	if rv := secret.GetResourceVersion(); rv != "" {
		stored.ResourceVersion = rv
	}
	stored.Data = secret.Data
	stored.StringData = secret.StringData

	res, err := h.kubeConnector.UpdateSecret(ctx, stored)
	if err != nil {
		return nil, err
	}

	return stripSecretData(res), nil
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

	if !isManagedSecret(secret) {
		return ErrNotFound
	}

	return h.kubeConnector.DeleteSecret(ctx, secret)
}

// isManagedSecret reports whether the HTTP API manages secret, i.e. it carries
// the label CreateSecret sets.
func isManagedSecret(secret *corev1.Secret) bool {
	return secret.Labels[common.OpenEverestManagedLabel] == "true"
}

// stripSecretData removes the contents of a secret, leaving only metadata.
func stripSecretData(secret *corev1.Secret) *corev1.Secret {
	secret.Data = nil
	secret.StringData = nil

	return secret
}
