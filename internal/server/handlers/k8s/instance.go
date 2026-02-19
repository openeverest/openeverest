// everest
// Copyright (C) 2023 Percona LLC
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

	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

// ListInstances returns list of instances in a namespace.
func (h *k8sHandler) ListInstances(ctx context.Context, namespace string) (*corev1alpha1.InstanceList, error) {
	return h.kubeConnector.ListInstances(ctx, ctrlclient.InNamespace(namespace))
}

// GetInstance returns instance that matches the criteria.
func (h *k8sHandler) GetInstance(ctx context.Context, namespace, name string) (*corev1alpha1.Instance, error) {
	return h.kubeConnector.GetInstance(ctx, types.NamespacedName{Namespace: namespace, Name: name})
}

// CreateInstance creates an instance.
func (h *k8sHandler) CreateInstance(ctx context.Context, instance *corev1alpha1.Instance) (*corev1alpha1.Instance, error) {
	return h.kubeConnector.CreateInstance(ctx, instance)
}

// UpdateInstance updates an instance.
func (h *k8sHandler) UpdateInstance(ctx context.Context, instance *corev1alpha1.Instance) (*corev1alpha1.Instance, error) {
	return h.kubeConnector.UpdateInstance(ctx, instance)
}

// DeleteInstance deletes an instance.
func (h *k8sHandler) DeleteInstance(ctx context.Context, namespace, name string) error {
	instance, err := h.kubeConnector.GetInstance(ctx, types.NamespacedName{Namespace: namespace, Name: name})
	if err != nil {
		return err
	}
	return h.kubeConnector.DeleteInstance(ctx, instance)
}
