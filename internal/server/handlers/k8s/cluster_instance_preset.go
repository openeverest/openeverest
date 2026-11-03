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

	"k8s.io/apimachinery/pkg/types"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

// ListClusterInstancePresets returns list of cluster instance presets, optionally filtered by provider.
func (h *k8sHandler) ListClusterInstancePresets(ctx context.Context, _, provider string) (*corev1alpha1.ClusterInstancePresetList, error) {
	list, err := h.kubeConnector.ListClusterInstancePresets(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list cluster instance presets: %w", err)
	}

	if provider != "" {
		filtered := []corev1alpha1.ClusterInstancePreset{}
		for _, preset := range list.Items {
			if preset.Spec.ProviderRef.Name == provider {
				filtered = append(filtered, preset)
			}
		}
		list.Items = filtered
	}

	return list, nil
}

// GetClusterInstancePreset returns a cluster instance preset that matches the criteria.
func (h *k8sHandler) GetClusterInstancePreset(ctx context.Context, _, name string) (*corev1alpha1.ClusterInstancePreset, error) {
	return h.kubeConnector.GetClusterInstancePreset(ctx, types.NamespacedName{Name: name})
}
