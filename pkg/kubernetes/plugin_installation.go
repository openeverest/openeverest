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

package kubernetes

import (
	"context"

	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	pluginv1alpha1 "github.com/openeverest/openeverest/v2/api/plugin/v1alpha1"
)

// ListPluginInstallations returns plugin installations that match the criteria.
func (k *Kubernetes) ListPluginInstallations(ctx context.Context, opts ...ctrlclient.ListOption) (*pluginv1alpha1.PluginInstallationList, error) {
	result := &pluginv1alpha1.PluginInstallationList{}
	if err := k.k8sClient.List(ctx, result, opts...); err != nil {
		return nil, err
	}
	return result, nil
}

// GetPluginInstallation returns a plugin installation that matches the criteria.
func (k *Kubernetes) GetPluginInstallation(ctx context.Context, key ctrlclient.ObjectKey) (*pluginv1alpha1.PluginInstallation, error) {
	result := &pluginv1alpha1.PluginInstallation{}
	if err := k.k8sClient.Get(ctx, key, result); err != nil {
		return nil, err
	}
	return result, nil
}

// CreatePluginInstallation creates a new plugin installation.
func (k *Kubernetes) CreatePluginInstallation(ctx context.Context, pi *pluginv1alpha1.PluginInstallation) (*pluginv1alpha1.PluginInstallation, error) {
	if err := k.k8sClient.Create(ctx, pi); err != nil {
		return nil, err
	}
	return pi, nil
}

// DeletePluginInstallation deletes a plugin installation.
func (k *Kubernetes) DeletePluginInstallation(ctx context.Context, obj *pluginv1alpha1.PluginInstallation) error {
	return k.k8sClient.Delete(ctx, obj)
}
