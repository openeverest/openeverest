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
	"context"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"strings"

	"github.com/casbin/casbin/v2"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

// pluginProxy handles plugin discovery and reverse-proxying.
// It reads Plugin CRs from the Kubernetes API on every request
// so newly created/deleted CRs take effect immediately.
type pluginProxy struct {
	kubeConnector kubernetes.KubernetesConnector
	enforcer      casbin.IEnforcer
}

func newPluginProxy(ctx context.Context, log *zap.SugaredLogger, kc kubernetes.KubernetesConnector) (*pluginProxy, error) {
	enf, err := rbac.NewEnforcerWithRefresh(ctx, kc, log)
	if err != nil {
		return nil, err
	}
	return &pluginProxy{kubeConnector: kc, enforcer: enf}, nil
}

// checkPluginAccess verifies the caller has "read" permission on the "plugins" resource.
func (pp *pluginProxy) checkPluginAccess(c echo.Context) error {
	user, err := rbac.GetUser(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}
	for _, sub := range append([]string{user.Subject}, user.Groups...) {
		ok, err := pp.enforcer.Enforce(sub, rbac.ResourcePlugins, rbac.ActionRead, "*")
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "rbac error")
		}
		if ok {
			return nil
		}
	}
	return echo.NewHTTPError(http.StatusForbidden, "insufficient permissions")
}

// listPluginsHandler returns the list of enabled plugins as JSON.
// This is consumed by the frontend PluginProvider to discover available plugins.
// The bundleUrl is a relative path that goes through the server's reverse proxy,
// so the browser never needs direct access to the plugin backend.
func (pp *pluginProxy) listPluginsHandler(c echo.Context) error {
	if err := pp.checkPluginAccess(c); err != nil {
		return err
	}

	type extensionPointDescriptor struct {
		Type  string `json:"type"`
		Label string `json:"label,omitempty"`
		Path  string `json:"path,omitempty"`
		Icon  string `json:"icon,omitempty"`
	}

	type pluginDescriptor struct {
		Name            string                     `json:"name"`
		DisplayName     string                     `json:"displayName"`
		BundleURL       string                     `json:"bundleUrl"`
		ExtensionPoints []extensionPointDescriptor `json:"extensionPoints,omitempty"`
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
		bundlePath := "/main.js"
		var extPoints []extensionPointDescriptor
		if p.Spec.Frontend != nil {
			if p.Spec.Frontend.BundlePath != "" {
				bundlePath = p.Spec.Frontend.BundlePath
			}
			for _, ep := range p.Spec.Frontend.ExtensionPoints {
				extPoints = append(extPoints, extensionPointDescriptor{
					Type:  ep.Type,
					Label: ep.Label,
					Path:  ep.Path,
					Icon:  ep.Icon,
				})
			}
		}
		descriptors = append(descriptors, pluginDescriptor{
			Name:            p.Name,
			DisplayName:     p.Spec.DisplayName,
			BundleURL:       path.Join("/v1/plugins", p.Name, bundlePath),
			ExtensionPoints: extPoints,
		})
	}
	return c.JSON(http.StatusOK, descriptors)
}

// proxyHandler reverse-proxies requests to a plugin's backend (no RBAC).
// Used for unauthenticated bundle serving.
// Route: /v1/plugins/:name/*
func (pp *pluginProxy) proxyHandler(c echo.Context) error {
	return pp.doProxy(c)
}

// authedProxyHandler reverse-proxies requests to a plugin's backend with RBAC.
// Route: /v1/plugins/:name (JWT-protected group)
func (pp *pluginProxy) authedProxyHandler(c echo.Context) error {
	if err := pp.checkPluginAccess(c); err != nil {
		return err
	}
	return pp.doProxy(c)
}

// doProxy performs the actual reverse proxy to a plugin backend.
func (pp *pluginProxy) doProxy(c echo.Context) error {
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

	if plugin.Spec.Backend == nil || plugin.Spec.Backend.URL == "" {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "plugin has no backend configured: " + name,
		})
	}

	target, err := url.Parse(plugin.Spec.Backend.URL)
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
