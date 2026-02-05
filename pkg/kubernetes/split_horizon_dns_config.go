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

package kubernetes

import (
	"context"

	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	enginefeaturesv1alpha1 "github.com/percona/everest-operator/api/enginefeatures.everest/v1alpha1"
)

// CreateSplitHorizonDNSConfig creates a SplitHorizonDNSConfig resource in Kubernetes.
func (k *Kubernetes) CreateSplitHorizonDNSConfig(ctx context.Context, shdc *enginefeaturesv1alpha1.SplitHorizonDNSConfig) (*enginefeaturesv1alpha1.SplitHorizonDNSConfig, error) { //nolint:lll
	if err := k.k8sClient.Create(ctx, shdc); err != nil {
		return nil, err
	}
	return shdc, nil
}

// UpdateSplitHorizonDNSConfig updates an existing SplitHorizonDNSConfig resource in Kubernetes.
func (k *Kubernetes) UpdateSplitHorizonDNSConfig(ctx context.Context, shdc *enginefeaturesv1alpha1.SplitHorizonDNSConfig) (*enginefeaturesv1alpha1.SplitHorizonDNSConfig, error) { //nolint:lll
	if err := k.k8sClient.Update(ctx, shdc); err != nil {
		return nil, err
	}
	return shdc, nil
}

// ListSplitHorizonDNSConfigs lists all SplitHorizonDNSConfig resources in Kubernetes that match the provided options.
func (k *Kubernetes) ListSplitHorizonDNSConfigs(ctx context.Context, opts ...ctrlclient.ListOption) (*enginefeaturesv1alpha1.SplitHorizonDNSConfigList, error) {
	result := &enginefeaturesv1alpha1.SplitHorizonDNSConfigList{}
	if err := k.k8sClient.List(ctx, result, opts...); err != nil {
		return nil, err
	}
	return result, nil
}

// DeleteSplitHorizonDNSConfig deletes a SplitHorizonDNSConfig resource in Kubernetes.
func (k *Kubernetes) DeleteSplitHorizonDNSConfig(ctx context.Context, shdc *enginefeaturesv1alpha1.SplitHorizonDNSConfig) error {
	return k.k8sClient.Delete(ctx, shdc)
}

// GetSplitHorizonDNSConfig retrieves a SplitHorizonDNSConfig resource from Kubernetes by its namespaced name.
func (k *Kubernetes) GetSplitHorizonDNSConfig(ctx context.Context, key ctrlclient.ObjectKey) (*enginefeaturesv1alpha1.SplitHorizonDNSConfig, error) {
	result := &enginefeaturesv1alpha1.SplitHorizonDNSConfig{}
	if err := k.k8sClient.Get(ctx, key, result); err != nil {
		return nil, err
	}
	return result, nil
}
