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

// Package rbac provides the RBAC handler.
package rbac

import (
	"context"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	api "github.com/openeverest/openeverest/v2/internal/server/api"
)

// ListInstances proxies the request to the next handler.
func (h *rbacHandler) ListInstances(ctx context.Context, namespace string) (*corev1alpha1.InstanceList, error) {
	// Add RBAC checks here if needed in the future
	return h.next.ListInstances(ctx, namespace)
}

// GetInstance proxies the request to the next handler.
func (h *rbacHandler) GetInstance(ctx context.Context, namespace, name string) (*corev1alpha1.Instance, error) {
	// Add RBAC checks here if needed in the future
	return h.next.GetInstance(ctx, namespace, name)
}

// CreateInstance proxies the request to the next handler.
func (h *rbacHandler) CreateInstance(ctx context.Context, instance *corev1alpha1.Instance) (*corev1alpha1.Instance, error) {
	// Add RBAC checks here if needed in the future
	return h.next.CreateInstance(ctx, instance)
}

// UpdateInstance proxies the request to the next handler.
func (h *rbacHandler) UpdateInstance(ctx context.Context, instance *corev1alpha1.Instance) (*corev1alpha1.Instance, error) {
	// Add RBAC checks here if needed in the future
	return h.next.UpdateInstance(ctx, instance)
}

// DeleteInstance proxies the request to the next handler.
func (h *rbacHandler) DeleteInstance(ctx context.Context, namespace, name string) error {
	// Add RBAC checks here if needed in the future
	return h.next.DeleteInstance(ctx, namespace, name)
}

// GetInstanceConnection proxies the request to the next handler.
func (h *rbacHandler) GetInstanceConnection(ctx context.Context, namespace, name string) (*api.InstanceConnectionDetails, error) {
	// Add RBAC checks here if needed in the future
	return h.next.GetInstanceConnection(ctx, namespace, name)
}
