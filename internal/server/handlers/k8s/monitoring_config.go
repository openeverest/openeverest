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
	"errors"
	"fmt"

	"github.com/AlekSi/pointer"
	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	monitoringv1alpha2 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha2"
	api "github.com/openeverest/openeverest/v2/internal/server/api"
	"github.com/openeverest/openeverest/v2/pkg/pmm"
)

// ListMonitoringConfigs returns list of monitoring configs in a namespace.
func (h *k8sHandler) ListMonitoringConfigs(ctx context.Context, cluster, namespace string) (*monitoringv1alpha2.MonitoringConfigList, error) {
	return h.kubeConnector.ListMonitoringConfigsV2(ctx, ctrlclient.InNamespace(namespace))
}

// CreateMonitoringConfig creates a monitoring config.
func (h *k8sHandler) CreateMonitoringConfig(ctx context.Context, cluster, namespace string, req *api.MonitoringConfigCreateParams) (*monitoringv1alpha2.MonitoringConfig, error) {
	_, err := h.kubeConnector.GetMonitoringConfigV2(ctx,
		types.NamespacedName{
			Namespace: namespace,
			Name:      req.Name,
		},
	)

	if err == nil {
		return nil, k8serrors.NewAlreadyExists(schema.GroupResource{
			Group:    monitoringv1alpha2.GroupVersion.Group,
			Resource: "monitoringconfigs",
		}, req.Name)
	}

	if !k8serrors.IsNotFound(err) {
		return nil, err
	}

	if req.Pmm != nil {
		apiKey := req.Pmm.ApiKey
		if apiKey == "" {
			apiKeyName := fmt.Sprintf("everest-%s-%s", req.Name, uuid.NewString())
			if apiKey, err = pmm.CreateAPIKey(ctx, req.Url, apiKeyName, req.Pmm.User, req.Pmm.Password, !pointer.Get(req.VerifyTLS)); err != nil {
				return nil, fmt.Errorf("failed to create PMM API key: %w", err)
			}
		}

		if err := h.createOrUpdateMonitoringConfigSecret(ctx, req.Name, namespace, apiKey); err != nil {
			return nil, err
		}
	}

	mc := &monitoringv1alpha2.MonitoringConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: namespace,
		},
		Spec: monitoringv1alpha2.MonitoringConfigSpec{
			Type:                  monitoringv1alpha2.MonitoringType(req.Type),
			URL:                   req.Url,
			CredentialsSecretName: req.Name,
			VerifyTLS:             req.VerifyTLS,
		},
	}

	result, err := h.kubeConnector.CreateMonitoringConfigV2(ctx, mc)
	if err != nil {
		if dErr := h.kubeConnector.DeleteSecret(ctx, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      req.Name,
				Namespace: namespace,
			},
		}); dErr != nil {
			return nil, fmt.Errorf("failed to clean up secret: %w", errors.Join(dErr, err))
		}

		return nil, err
	}

	return result, nil
}

// DeleteMonitoringConfig deletes a monitoring config.
func (h *k8sHandler) DeleteMonitoringConfig(ctx context.Context, cluster, namespace, name string) error {
	delMCObj := &monitoringv1alpha2.MonitoringConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	if err := h.kubeConnector.DeleteMonitoringConfigV2(ctx, delMCObj); err != nil {
		return err
	}

	delSecObj := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	err := h.kubeConnector.DeleteSecret(ctx, delSecObj)
	if k8serrors.IsNotFound(err) {
		return nil
	}

	return err
}

// GetMonitoringConfig returns monitoring config that matches the criteria.
func (h *k8sHandler) GetMonitoringConfig(ctx context.Context, cluster, namespace, name string) (*monitoringv1alpha2.MonitoringConfig, error) {
	return h.kubeConnector.GetMonitoringConfigV2(ctx,
		types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		},
	)
}

// UpdateMonitoringConfig updates a monitoring config.
func (h *k8sHandler) UpdateMonitoringConfig(ctx context.Context, cluster, namespace, name string, req *api.MonitoringConfigUpdateParams) (*monitoringv1alpha2.MonitoringConfig, error) {
	m, err := h.kubeConnector.GetMonitoringConfigV2(ctx,
		types.NamespacedName{
			Namespace: namespace,
			Name:      name,
		},
	)
	if err != nil {
		return nil, err
	}

	if req.Url != "" {
		m.Spec.URL = req.Url
	}

	if req.VerifyTLS != nil {
		m.Spec.VerifyTLS = req.VerifyTLS
	}

	if req.Pmm != nil {
		apiKey := req.Pmm.ApiKey

		if req.Pmm.User != "" && req.Pmm.Password != "" {
			verifyTLS := pointer.Get(m.Spec.VerifyTLS)
			apiKeyName := fmt.Sprintf("everest-%s-%s", name, uuid.NewString())

			if apiKey, err = pmm.CreateAPIKey(ctx, m.Spec.URL, apiKeyName, req.Pmm.User, req.Pmm.Password, !verifyTLS); err != nil {
				return nil, err
			}
		}

		if apiKey != "" {
			secret := newMonitoringConfigSecret(name, namespace, apiKey)

			if _, err = h.kubeConnector.UpdateSecret(ctx, secret); err != nil {
				return nil, fmt.Errorf("could not update k8s secret %s", name)
			}
		}
	}

	return h.kubeConnector.UpdateMonitoringConfigV2(ctx, m)
}

// createOrUpdateMonitoringConfigSecret creates a secret, or updates it if it already exists.
func (h *k8sHandler) createOrUpdateMonitoringConfigSecret(ctx context.Context, name, namespace, apiKey string) error {
	secret := newMonitoringConfigSecret(name, namespace, apiKey)
	_, err := h.kubeConnector.CreateSecret(ctx, secret)
	if err == nil {
		return nil
	}

	if !k8serrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed creating secret: %w", err)
	}

	if _, err = h.kubeConnector.UpdateSecret(ctx, secret); err != nil {
		return fmt.Errorf("failed to update secret: %w", err)
	}

	return nil
}

func newMonitoringConfigSecret(name, namespace string, apiKey string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Type: corev1.SecretTypeOpaque,
		StringData: map[string]string{
			"apiKey":   apiKey,
			"username": "api_key",
		},
	}
}
