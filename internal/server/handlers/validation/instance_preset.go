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
	"github.com/openeverest/openeverest/v2/internal/preset"
)

// ListInstancePresets proxies the request to the next handler.
func (h *validateHandler) ListInstancePresets(ctx context.Context, cluster string, provider string) (*corev1alpha1.InstancePresetList, error) {
	return h.next.ListInstancePresets(ctx, cluster, provider)
}

// GetInstancePreset proxies the request to the next handler.
func (h *validateHandler) GetInstancePreset(ctx context.Context, cluster, name string) (*corev1alpha1.InstancePreset, error) {
	return h.next.GetInstancePreset(ctx, cluster, name)
}

// ResolveInstancePreset proxies the request to the next handler.
func (h *validateHandler) ResolveInstancePreset(ctx context.Context, cluster, name, namespace string) (*corev1alpha1.InstancePreset, error) {
	return h.next.ResolveInstancePreset(ctx, cluster, name, namespace)
}

// CreateInstancePreset validates and creates an instance preset.
func (h *validateHandler) CreateInstancePreset(ctx context.Context, cluster string, instancePreset *corev1alpha1.InstancePreset) (*corev1alpha1.InstancePreset, error) {
	if err := preset.EnsureNamespaceRefsEmpty(&instancePreset.Spec.InstanceSpec); err != nil {
		return nil, errors.Join(ErrInvalidRequest, err)
	}

	return h.next.CreateInstancePreset(ctx, cluster, instancePreset)
}

// UpdateInstancePreset validates and updates an instance preset.
func (h *validateHandler) UpdateInstancePreset(ctx context.Context, cluster string, instancePreset *corev1alpha1.InstancePreset) (*corev1alpha1.InstancePreset, error) {
	if err := preset.EnsureNamespaceRefsEmpty(&instancePreset.Spec.InstanceSpec); err != nil {
		return nil, errors.Join(ErrInvalidRequest, err)
	}

	return h.next.UpdateInstancePreset(ctx, cluster, instancePreset)
}

// DeleteInstancePreset proxies the request to the next handler.
func (h *validateHandler) DeleteInstancePreset(ctx context.Context, cluster, name string) error {
	return h.next.DeleteInstancePreset(ctx, cluster, name)
}

// CreateInstancePresetFromInstance validates and creates an instance preset from an instance.
func (h *validateHandler) CreateInstancePresetFromInstance(ctx context.Context, cluster, namespace, instanceName, presetName string) (*corev1alpha1.InstancePreset, error) {
	if presetName == "" {
		return nil, errors.Join(ErrInvalidRequest, errors.New("presetName cannot be empty"))
	}

	return h.next.CreateInstancePresetFromInstance(ctx, cluster, namespace, instanceName, presetName)
}
