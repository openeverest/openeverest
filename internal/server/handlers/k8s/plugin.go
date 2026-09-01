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
	"path"
	"strings"

	api "github.com/openeverest/openeverest/v2/internal/server/api"
	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

// ListPlugins returns the enabled plugins advertised to the frontend loader.
// The cluster param follows the multi-cluster routing convention; today it only
// shapes the asset URLs the descriptors point back to.
func (h *k8sHandler) ListPlugins(ctx context.Context, cluster string) (api.PluginDescriptorList, error) {
	plugins, err := h.kubeConnector.ListPlugins(ctx)
	if err != nil {
		return nil, err
	}

	descriptors := make(api.PluginDescriptorList, 0, len(plugins.Items))
	for _, p := range plugins.Items {
		if !p.Spec.Enabled {
			continue
		}
		bundlePath := "/main.js"
		var extPoints []api.PluginExtensionPoint
		if p.Spec.Frontend != nil {
			if p.Spec.Frontend.BundlePath != "" {
				bundlePath = p.Spec.Frontend.BundlePath
			}
			for _, ep := range p.Spec.Frontend.ExtensionPoints {
				extPoints = append(extPoints, api.PluginExtensionPoint{
					Type:      ep.Type,
					Label:     ep.Label,
					Path:      ep.Path,
					Icon:      resolvePluginAssetPath(cluster, p.Name, ep.Icon),
					Providers: ep.Providers,
				})
			}
		}
		descriptors = append(descriptors, api.PluginDescriptor{
			Name:            p.Name,
			DisplayName:     p.Spec.DisplayName,
			Description:     p.Spec.Description,
			Version:         p.Spec.Version,
			Vendor:          p.Spec.Vendor,
			Icon:            resolvePluginAssetPath(cluster, p.Name, p.Spec.Icon),
			BundleUrl:       path.Join(pluginBasePath(cluster, p.Name), bundlePath),
			ExtensionPoints: extPoints,
		})
	}
	return descriptors, nil
}

// GetPluginContext returns the calling user's identity and the namespaces they
// can access in the cluster. Plugins use it to scope their queries per tenant.
func (h *k8sHandler) GetPluginContext(ctx context.Context, _ string) (*api.PluginContext, error) {
	user, err := rbac.GetUser(ctx)
	if err != nil {
		return nil, err
	}

	nsList, err := h.kubeConnector.GetDBNamespaces(ctx)
	if err != nil {
		return nil, err
	}
	namespaces := make([]string, 0, len(nsList.Items))
	for _, ns := range nsList.Items {
		namespaces = append(namespaces, ns.Name)
	}

	return &api.PluginContext{
		User:       user.Subject,
		Groups:     user.Groups,
		Namespaces: namespaces,
	}, nil
}

// pluginBasePath returns the proxy base a plugin's assets are served under.
func pluginBasePath(cluster, name string) string {
	return path.Join("/v1/clusters", cluster, "plugins", name)
}

// resolvePluginAssetPath resolves a relative asset path (e.g. "icon.png") to the
// full plugin proxy URL. Absolute URLs, data URIs, and already-resolved /v1/
// paths are returned unchanged. Empty strings are returned as-is.
func resolvePluginAssetPath(cluster, pluginName, assetPath string) string {
	if assetPath == "" {
		return ""
	}
	if strings.HasPrefix(assetPath, "http://") ||
		strings.HasPrefix(assetPath, "https://") ||
		strings.HasPrefix(assetPath, "data:") ||
		strings.HasPrefix(assetPath, "/v1/") {
		return assetPath
	}
	return path.Join(pluginBasePath(cluster, pluginName), assetPath)
}
