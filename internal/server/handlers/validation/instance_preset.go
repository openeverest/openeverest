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
)

// ListClusterInstancePresets proxies the request to the next handler.
func (h *validateHandler) ListClusterInstancePresets(ctx context.Context, cluster string, provider string) (*corev1alpha1.ClusterInstancePresetList, error) {
	return h.next.ListClusterInstancePresets(ctx, cluster, provider)
}

// GetClusterInstancePreset proxies the request to the next handler.
func (h *validateHandler) GetClusterInstancePreset(ctx context.Context, cluster, name string) (*corev1alpha1.ClusterInstancePreset, error) {
	return h.next.GetClusterInstancePreset(ctx, cluster, name)
}

// ResolveClusterInstancePreset proxies the request to the next handler.
func (h *validateHandler) ResolveClusterInstancePreset(ctx context.Context, cluster, name, namespace string) (*corev1alpha1.ClusterInstancePreset, error) {
	return h.next.ResolveClusterInstancePreset(ctx, cluster, name, namespace)
}
