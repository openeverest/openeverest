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

package server

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"strings"

	"github.com/labstack/echo/v4"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
)

// pluginProxy handles plugin discovery and reverse-proxying.
// It reads Plugin CRs from the Kubernetes API on every request
// so newly created/deleted CRs take effect immediately.
type pluginProxy struct {
	kubeConnector kubernetes.KubernetesConnector
}

func newPluginProxy(kc kubernetes.KubernetesConnector) *pluginProxy {
	return &pluginProxy{kubeConnector: kc}
}

// listPluginsHandler returns the list of enabled plugins as JSON.
// This is consumed by the frontend PluginProvider to discover available plugins.
// The bundleUrl is a relative path that goes through the server's reverse proxy,
// so the browser never needs direct access to the plugin backend.
func (pp *pluginProxy) listPluginsHandler(c echo.Context) error {
	type pluginDescriptor struct {
		Name        string `json:"name"`
		DisplayName string `json:"displayName"`
		BundleURL   string `json:"bundleUrl"`
	}

	plugins, err := pp.kubeConnector.ListPlugins(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "failed to list plugins: " + err.Error(),
		})
	}

	descriptors := make([]pluginDescriptor, 0, len(plugins.Items))
	for _, p := range plugins.Items {
		if !p.Spec.Enabled {
			continue
		}
		bundlePath := p.Spec.BundlePath
		if bundlePath == "" {
			bundlePath = "/main.js"
		}
		descriptors = append(descriptors, pluginDescriptor{
			Name:        p.Name,
			DisplayName: p.Spec.DisplayName,
			BundleURL:   path.Join("/v1/plugins", p.Name, bundlePath),
		})
	}
	return c.JSON(http.StatusOK, descriptors)
}

// proxyHandler reverse-proxies requests to a plugin's backend.
// It looks up the Plugin CR by name on each request.
// Route: /v1/plugins/:name/*
func (pp *pluginProxy) proxyHandler(c echo.Context) error {
	name := c.Param("name")

	plugin, err := pp.kubeConnector.GetPlugin(c.Request().Context(), pluginKey(name))
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "plugin not found: " + name,
		})
	}

	if !plugin.Spec.Enabled {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "plugin is disabled: " + name,
		})
	}

	target, err := url.Parse(plugin.Spec.BackendURL)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "invalid plugin backend URL",
		})
	}

	// Strip the prefix /v1/plugins/:name from the request path before proxying.
	prefix := "/v1/plugins/" + name
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			req.URL.Path = strings.TrimPrefix(req.URL.Path, prefix)
			if req.URL.Path == "" {
				req.URL.Path = "/"
			}
			req.Host = target.Host
		},
	}

	proxy.ServeHTTP(c.Response(), c.Request())
	return nil
}

func pluginKey(name string) ctrlclient.ObjectKey {
	return ctrlclient.ObjectKey{Name: name}
}
