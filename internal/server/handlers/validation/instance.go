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
	"errors"

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

// UpdateInstance rejects a change to the bootstrap credentials reference, then proxies to the next handler.
func (h *validateHandler) UpdateInstance(ctx context.Context, cluster string, instance *corev1alpha1.Instance) (*corev1alpha1.Instance, error) {
	if instance.Spec.UserSecretRef != nil {
		current, err := h.next.GetInstance(ctx, cluster, instance.GetNamespace(), instance.GetName())
		if err != nil {
			return nil, err
		}

		// A update could set userSecretRef when it was previously unset, which
		// kubebuilder's immutable validation does not catch.
		if current.Spec.UserSecretRef == nil {
			return nil, errors.Join(
				ErrInvalidRequest,
				errors.New("spec.userSecretRef may not be modified"),
			)
		}
	}
	// Add validation here if needed in the future
	return h.next.UpdateInstance(ctx, cluster, instance)
}

// PatchInstance rejects a patch naming a member the caller may not write, then proxies to the next handler.
func (h *validateHandler) PatchInstance(ctx context.Context, cluster, namespace, name string, patch []byte) (*corev1alpha1.Instance, error) {
	doc, err := validateMergePatch(patch)
	if err != nil {
		return nil, err
	}
	if spec, isObject := doc["spec"].(map[string]any); isObject {
		// A patch could set userSecretRef when it was previously unset, which
		// kubebuilder's immutable validation does not catch.
		if _, ok := spec["userSecretRef"]; ok {
			return nil, errors.Join(
				ErrInvalidRequest,
				errors.New("spec.userSecretRef may not be patched"),
			)
		}
	}
	return h.next.PatchInstance(ctx, cluster, namespace, name, patch)
}

// DeleteInstance proxies the request to the next handler.
func (h *validateHandler) DeleteInstance(ctx context.Context, cluster, namespace, name string, params *api.DeleteInstanceParams) error {
	return h.next.DeleteInstance(ctx, cluster, namespace, name, params)
}

// GetInstanceConnection proxies the request to the next handler.
func (h *validateHandler) GetInstanceConnection(ctx context.Context, cluster, namespace, name string) (*api.InstanceConnectionDetails, error) {
	return h.next.GetInstanceConnection(ctx, cluster, namespace, name)
}
