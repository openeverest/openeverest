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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

// ListInstancePresets returns namespace-scoped instance presets, optionally filtered by provider.
// The provider filter is evaluated server-side via the CRD's selectable fields.
func (h *k8sHandler) ListInstancePresets(ctx context.Context, _, namespace, provider string) (*corev1alpha1.InstancePresetList, error) {
	opts := []ctrlclient.ListOption{ctrlclient.InNamespace(namespace)}
	if provider != "" {
		opts = append(opts, ctrlclient.MatchingFields{"spec.providerRef.name": provider})
	}

	list, err := h.kubeConnector.ListInstancePresets(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to list instance presets: %w", err)
	}

	return list, nil
}

// GetInstancePreset returns a namespace-scoped instance preset by name and namespace.
func (h *k8sHandler) GetInstancePreset(ctx context.Context, _, namespace, name string) (*corev1alpha1.InstancePreset, error) {
	return h.kubeConnector.GetInstancePreset(ctx, types.NamespacedName{Namespace: namespace, Name: name})
}

// CreateInstancePreset creates a namespace-scoped instance preset.
func (h *k8sHandler) CreateInstancePreset(ctx context.Context, _ string, preset *corev1alpha1.InstancePreset) (*corev1alpha1.InstancePreset, error) {
	return h.kubeConnector.CreateInstancePreset(ctx, preset)
}

// UpdateInstancePreset updates a namespace-scoped instance preset.
func (h *k8sHandler) UpdateInstancePreset(ctx context.Context, _ string, preset *corev1alpha1.InstancePreset) (*corev1alpha1.InstancePreset, error) {
	return h.kubeConnector.UpdateInstancePreset(ctx, preset)
}

// DeleteInstancePreset deletes a namespace-scoped instance preset.
func (h *k8sHandler) DeleteInstancePreset(ctx context.Context, _, namespace, name string) error {
	preset := &corev1alpha1.InstancePreset{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name},
	}

	if err := h.kubeConnector.DeleteInstancePreset(ctx, preset); ctrlclient.IgnoreNotFound(err) != nil {
		return fmt.Errorf("failed to delete instance preset: %w", err)
	}

	return nil
}
