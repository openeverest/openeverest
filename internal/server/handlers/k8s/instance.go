// everest
// Copyright (C) 2023 Percona LLC
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
	"net/http"

	echo "github.com/labstack/echo/v4"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	api "github.com/openeverest/openeverest/v2/internal/server/api"
)

// ListInstances returns list of instances in a namespace.
func (h *k8sHandler) ListInstances(ctx context.Context, cluster, namespace string) (*corev1alpha1.InstanceList, error) {
	return h.kubeConnector.ListInstances(ctx, ctrlclient.InNamespace(namespace))
}

// GetInstance returns instance that matches the criteria.
func (h *k8sHandler) GetInstance(ctx context.Context, cluster, namespace, name string) (*corev1alpha1.Instance, error) {
	return h.kubeConnector.GetInstance(ctx, types.NamespacedName{Namespace: namespace, Name: name})
}

// CreateInstance creates an instance.
func (h *k8sHandler) CreateInstance(ctx context.Context, cluster string, instance *corev1alpha1.Instance) (*corev1alpha1.Instance, error) {
	return h.kubeConnector.CreateInstance(ctx, instance)
}

// UpdateInstance updates an instance.
func (h *k8sHandler) UpdateInstance(ctx context.Context, cluster string, instance *corev1alpha1.Instance) (*corev1alpha1.Instance, error) {
	return h.kubeConnector.UpdateInstance(ctx, instance)
}

// DeleteInstance deletes an instance.
func (h *k8sHandler) DeleteInstance(ctx context.Context, cluster, namespace, name string) error {
	instance, err := h.kubeConnector.GetInstance(ctx, types.NamespacedName{Namespace: namespace, Name: name})
	if err != nil {
		return err
	}
	return h.kubeConnector.DeleteInstance(ctx, instance)
}

// GetInstanceConnection returns connection details for an instance by reading
// the connection Secret referenced in the Instance status.
func (h *k8sHandler) GetInstanceConnection(ctx context.Context, cluster, namespace, name string) (*api.InstanceConnectionDetails, error) {
	instance, err := h.kubeConnector.GetInstance(ctx, types.NamespacedName{Namespace: namespace, Name: name})
	if err != nil {
		return nil, err
	}

	secretName := instance.Status.ConnectionSecretRef.Name
	if secretName == "" {
		return nil, echo.NewHTTPError(
			http.StatusNotFound,
			fmt.Sprintf("connection details are not yet available for instance %q", name),
		)
	}

	secret, err := h.kubeConnector.GetSecret(ctx, types.NamespacedName{Namespace: namespace, Name: secretName})
	if err != nil {
		return nil, err
	}

	result := &api.InstanceConnectionDetails{}

	// Map well-known keys to struct fields
	if v, ok := secret.Data["type"]; ok {
		s := string(v)
		result.Type = &s
	}
	if v, ok := secret.Data["provider"]; ok {
		s := string(v)
		result.Provider = &s
	}
	if v, ok := secret.Data["host"]; ok {
		s := string(v)
		result.Host = &s
	}
	if v, ok := secret.Data["port"]; ok {
		s := string(v)
		result.Port = &s
	}
	if v, ok := secret.Data["username"]; ok {
		s := string(v)
		result.Username = &s
	}
	if v, ok := secret.Data["password"]; ok {
		s := string(v)
		result.Password = &s
	}
	if v, ok := secret.Data["uri"]; ok {
		s := string(v)
		result.Uri = &s
	}

	// Store any additional provider-specific keys
	wellKnownKeys := map[string]bool{
		"type": true, "provider": true, "host": true, "port": true,
		"username": true, "password": true, "uri": true,
	}
	for k, v := range secret.Data {
		if !wellKnownKeys[k] {
			result.Set(k, string(v))
		}
	}

	return result, nil
}
