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

// Package kubernetes ...
package kubernetes

import (
	"context"

	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

// ListClusterInstancePresets returns list of instance presets that match the criteria.
func (k *Kubernetes) ListClusterInstancePresets(ctx context.Context, opts ...ctrlclient.ListOption) (*v1alpha1.ClusterInstancePresetList, error) {
	result := &v1alpha1.ClusterInstancePresetList{}
	if err := k.k8sClient.List(ctx, result, opts...); err != nil {
		return nil, err
	}
	return result, nil
}

// GetClusterInstancePreset returns instance preset that matches the criteria.
func (k *Kubernetes) GetClusterInstancePreset(ctx context.Context, key ctrlclient.ObjectKey) (*v1alpha1.ClusterInstancePreset, error) {
	result := &v1alpha1.ClusterInstancePreset{}
	if err := k.k8sClient.Get(ctx, key, result); err != nil {
		return nil, err
	}
	return result, nil
}

// ListInstancePresets returns list of namespace-scoped instance presets that match the criteria.
func (k *Kubernetes) ListInstancePresets(ctx context.Context, opts ...ctrlclient.ListOption) (*v1alpha1.InstancePresetList, error) {
	result := &v1alpha1.InstancePresetList{}
	if err := k.k8sClient.List(ctx, result, opts...); err != nil {
		return nil, err
	}
	return result, nil
}

// GetInstancePreset returns a namespace-scoped instance preset that matches the criteria.
func (k *Kubernetes) GetInstancePreset(ctx context.Context, key ctrlclient.ObjectKey) (*v1alpha1.InstancePreset, error) {
	result := &v1alpha1.InstancePreset{}
	if err := k.k8sClient.Get(ctx, key, result); err != nil {
		return nil, err
	}
	return result, nil
}

// CreateInstancePreset creates a namespace-scoped instance preset.
func (k *Kubernetes) CreateInstancePreset(ctx context.Context, preset *v1alpha1.InstancePreset) (*v1alpha1.InstancePreset, error) {
	if err := k.k8sClient.Create(ctx, preset); err != nil {
		return nil, err
	}
	return preset, nil
}

// UpdateInstancePreset updates a namespace-scoped instance preset.
func (k *Kubernetes) UpdateInstancePreset(ctx context.Context, preset *v1alpha1.InstancePreset) (*v1alpha1.InstancePreset, error) {
	if err := k.k8sClient.Update(ctx, preset); err != nil {
		return nil, err
	}
	return preset, nil
}

// DeleteInstancePreset deletes a namespace-scoped instance preset.
func (k *Kubernetes) DeleteInstancePreset(ctx context.Context, preset *v1alpha1.InstancePreset) error {
	return k.k8sClient.Delete(ctx, preset)
}
