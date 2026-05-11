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

// Package validation provides the validation handler.
package validation

import (
	"context"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	api "github.com/openeverest/openeverest/v2/internal/server/api"
)

// ListInstances proxies the request to the next handler.
func (h *validateHandler) ListInstances(ctx context.Context, cluster, namespace string) (*corev1alpha1.InstanceList, error) {
	return h.next.ListInstances(ctx, cluster, namespace)
}

// GetInstance proxies the request to the next handler.
func (h *validateHandler) GetInstance(ctx context.Context, cluster, namespace, name string) (*corev1alpha1.Instance, error) {
	return h.next.GetInstance(ctx, cluster, namespace, name)
}

// CreateInstance proxies the request to the next handler.
func (h *validateHandler) CreateInstance(ctx context.Context, cluster string, instance *corev1alpha1.Instance) (*corev1alpha1.Instance, error) {
	// Add validation here if needed in the future
	return h.next.CreateInstance(ctx, cluster, instance)
}

// UpdateInstance proxies the request to the next handler.
func (h *validateHandler) UpdateInstance(ctx context.Context, cluster string, instance *corev1alpha1.Instance) (*corev1alpha1.Instance, error) {
	// Add validation here if needed in the future
	return h.next.UpdateInstance(ctx, cluster, instance)
}

// DeleteInstance proxies the request to the next handler.
func (h *validateHandler) DeleteInstance(ctx context.Context, cluster, namespace, name string) error {
	return h.next.DeleteInstance(ctx, cluster, namespace, name)
}

// GetInstanceConnection proxies the request to the next handler.
func (h *validateHandler) GetInstanceConnection(ctx context.Context, cluster, namespace, name string) (*api.InstanceConnectionDetails, error) {
	return h.next.GetInstanceConnection(ctx, cluster, namespace, name)
}
