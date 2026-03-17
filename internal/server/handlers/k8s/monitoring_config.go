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
	"fmt"

	"github.com/AlekSi/pointer"
	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
	api "github.com/openeverest/openeverest/v2/internal/server/api"
	"github.com/openeverest/openeverest/v2/pkg/pmm"
)

// ListMonitoringConfigs returns list of monitoring configs in a namespace.
func (h *k8sHandler) ListMonitoringConfigs(ctx context.Context, namespace string) (*monitoringv1alpha1.MonitoringConfigList, error) {
	return h.kubeConnector.ListMonitoringConfigsV2(ctx, ctrlclient.InNamespace(namespace))
}

// CreateMonitoringConfig creates a monitoring config.
func (h *k8sHandler) CreateMonitoringConfig(ctx context.Context, req *api.MonitoringConfigCreateParams) (*monitoringv1alpha1.MonitoringConfig, error) {
	m, err := h.kubeConnector.GetMonitoringConfigV2(ctx, types.NamespacedName{Namespace: req.Namespace, Name: req.Name})
	if err != nil && !k8serrors.IsNotFound(err) {
		return nil, err
	}

	if m != nil && m.GetName() != "" {
		return nil, k8serrors.NewAlreadyExists(schema.GroupResource{
			Group:    monitoringv1alpha1.GroupVersion.Group,
			Resource: "monitoringconfigs",
		}, req.Name,
		)
	}

	apiKey := req.Pmm.ApiKey
	if req.Pmm != nil && apiKey == "" {
		apiKeyName := fmt.Sprintf("everest-%s-%s", req.Name, uuid.NewString())
		skipVerifyTLS := !pointer.Get(req.VerifyTLS)

		if apiKey, err = pmm.CreateAPIKey(ctx, req.Url, apiKeyName, req.Pmm.User, req.Pmm.Password, skipVerifyTLS); err != nil {
			return nil, fmt.Errorf("failed to create PMM API key: %w", err)
		}
	}

	return h.createMonitoringConfig(ctx, req.Namespace, req, apiKey)
}

// DeleteMonitoringConfig deletes a monitoring config.
func (h *k8sHandler) DeleteMonitoringConfig(ctx context.Context, namespace, name string) error {
	delMCObj := &monitoringv1alpha1.MonitoringConfig{
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
	return h.kubeConnector.DeleteSecret(ctx, delSecObj)
}

// GetMonitoringConfig returns monitoring config that matches the criteria.
func (h *k8sHandler) GetMonitoringConfig(ctx context.Context, namespace, name string) (*monitoringv1alpha1.MonitoringConfig, error) {
	return h.kubeConnector.GetMonitoringConfigV2(ctx, types.NamespacedName{Namespace: namespace, Name: name})
}

// UpdateMonitoringConfig updates a monitoring config.
func (h *k8sHandler) UpdateMonitoringConfig(ctx context.Context, namespace, name string, req *api.MonitoringConfigUpdateParams) (*monitoringv1alpha1.MonitoringConfig, error) {
	m, err := h.kubeConnector.GetMonitoringConfigV2(ctx, types.NamespacedName{Namespace: namespace, Name: name})
	if err != nil {
		return nil, err
	}

	var apiKey string
	if req.Pmm != nil && req.Pmm.ApiKey != "" {
		apiKey = req.Pmm.ApiKey
	}

	if req.Pmm != nil && req.Pmm.User != "" && req.Pmm.Password != "" {
		apiKeyName := fmt.Sprintf("everest-%s-%s", name, uuid.NewString())
		skipVerifyTLS := !pointer.Get(req.VerifyTLS)

		if apiKey, err = pmm.CreateAPIKey(ctx, req.Url, apiKeyName, req.Pmm.User, req.Pmm.Password, skipVerifyTLS); err != nil {
			return nil, err
		}
	}

	if apiKey != "" {
		_, err := h.kubeConnector.UpdateSecret(ctx, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Type: corev1.SecretTypeOpaque,
			StringData: map[string]string{
				"apiKey":   apiKey,
				"username": "api_key",
			},
		})
		if err != nil {
			return nil, fmt.Errorf("could not update k8s secret %s", name)
		}
	}

	if req.Url != "" {
		m.Spec.PMM.URL = req.Url
	}

	if req.AllowedNamespaces != nil {
		m.Spec.AllowedNamespaces = *req.AllowedNamespaces
	}

	if req.VerifyTLS != nil {
		m.Spec.VerifyTLS = req.VerifyTLS
	}

	return h.kubeConnector.UpdateMonitoringConfigV2(ctx, m)
}

// createMonitoringConfig creates secret and monitoring config resources.
func (h *k8sHandler) createMonitoringConfig(
	c context.Context, namespace string, req *api.MonitoringConfigCreateParams, apiKey string,
) (*monitoringv1alpha1.MonitoringConfig, error) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: namespace,
		},
		Type:       corev1.SecretTypeOpaque,
		StringData: h.monitoringConfigSecretData(apiKey),
	}

	if _, err := h.kubeConnector.CreateSecret(c, secret); err != nil {
		if k8serrors.IsAlreadyExists(err) {
			_, err = h.kubeConnector.UpdateSecret(c, secret)
			if err != nil {
				return nil, fmt.Errorf("could not update k8s secret %s", req.Name)
			}
		} else {
			return nil, fmt.Errorf("failed creating secret in the Kubernetes cluster")
		}
	}

	created, err := h.kubeConnector.CreateMonitoringConfigV2(c, &monitoringv1alpha1.MonitoringConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: namespace,
		},
		Spec: monitoringv1alpha1.MonitoringConfigSpec{
			Type: monitoringv1alpha1.MonitoringType(req.Type),
			PMM: monitoringv1alpha1.PMMConfig{
				URL: req.Url,
			},
			CredentialsSecretName: req.Name,
			VerifyTLS:             req.VerifyTLS,
		},
	})
	if err != nil {
		delObj := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      req.Name,
				Namespace: namespace,
			},
		}

		if dErr := h.kubeConnector.DeleteSecret(c, delObj); dErr != nil {
			return nil, fmt.Errorf("failed cleaning up the secret because failed creating monitoring instance")
		}

		return nil, err
	}

	return created, nil
}
