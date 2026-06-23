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

	extensionsv1alpha1 "github.com/openeverest/openeverest/v2/api/extensions/v1alpha1"
)

// ListPlugins returns list of plugins that match the criteria.
func (k *Kubernetes) ListPlugins(ctx context.Context, opts ...ctrlclient.ListOption) (*extensionsv1alpha1.PluginList, error) {
	result := &extensionsv1alpha1.PluginList{}
	if err := k.k8sClient.List(ctx, result, opts...); err != nil {
		return nil, err
	}
	return result, nil
}

// GetPlugin returns plugin that matches the criteria.
func (k *Kubernetes) GetPlugin(ctx context.Context, key ctrlclient.ObjectKey) (*extensionsv1alpha1.Plugin, error) {
	result := &extensionsv1alpha1.Plugin{}
	if err := k.k8sClient.Get(ctx, key, result); err != nil {
		return nil, err
	}
	return result, nil
}

// CreatePlugin creates a new plugin.
func (k *Kubernetes) CreatePlugin(ctx context.Context, plugin *extensionsv1alpha1.Plugin) (*extensionsv1alpha1.Plugin, error) {
	if err := k.k8sClient.Create(ctx, plugin); err != nil {
		return nil, err
	}
	return plugin, nil
}

// DeletePlugin deletes a plugin.
func (k *Kubernetes) DeletePlugin(ctx context.Context, obj *extensionsv1alpha1.Plugin) error {
	return k.k8sClient.Delete(ctx, obj)
}
