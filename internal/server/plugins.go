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
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"strings"

	"github.com/casbin/casbin/v2"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	pluginv1alpha1 "github.com/openeverest/openeverest/v2/api/extensions/v1alpha1"
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

// canUsePlugin returns true when the caller has the "use" verb on
// "plugin/<name>". Admins that have "*" on "plugins" are also permitted.
func (pp *pluginProxy) canUsePlugin(c echo.Context, name string) (bool, error) {
	user, err := rbac.GetUser(c.Request().Context())
	if err != nil {
		return false, err
	}
	// resource is "plugin/<name>" per the design (§9.2).
	resource := "plugin/" + name
	for _, sub := range append([]string{user.Subject}, user.Groups...) {
		// Direct "use" grant on the specific plugin.
		if ok, err := pp.enforcer.Enforce(sub, resource, rbac.ActionUse, "*"); err != nil {
			return false, err
		} else if ok {
			return true, nil
		}
		// Wildcard "use" grant (e.g. "plugin/*" → use).
		if ok, err := pp.enforcer.Enforce(sub, "plugin/*", rbac.ActionUse, "*"); err != nil {
			return false, err
		} else if ok {
			return true, nil
		}
		// Admin shortcut: "*" action on "plugins" resource also grants use.
		if ok, err := pp.enforcer.Enforce(sub, rbac.ResourcePlugins, rbac.ActionAll, "*"); err != nil {
			return false, err
		} else if ok {
			return true, nil
		}
	}
	return false, nil
}

// proxyHandler reverse-proxies requests to a plugin's backend (no RBAC).
// Used for unauthenticated bundle serving.
// Route: /v1/clusters/:cluster/plugins/:name/*
func (pp *pluginProxy) proxyHandler(c echo.Context) error {
	return pp.doProxy(c)
}

// authedProxyHandler reverse-proxies requests to a plugin's backend with RBAC.
// The caller must have the "use" verb on "plugin/<name>" (§9.2).
// Route: /v1/clusters/:cluster/plugins/:name (JWT-protected group)
func (pp *pluginProxy) authedProxyHandler(c echo.Context) error {
	name := c.Param("name")
	allowed, err := pp.canUsePlugin(c, name)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}
	if !allowed {
		return echo.NewHTTPError(http.StatusForbidden, "insufficient permissions")
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

	if plugin.Spec.Backend == nil {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "plugin has no backend configured: " + name,
		})
	}

	targetURL, credToken, err := pp.resolveBackendURL(plugin.Spec.Backend)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "cannot resolve plugin backend: " + err.Error(),
		})
	}

	target, err := url.Parse(targetURL)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "invalid plugin backend URL",
		})
	}

	// Strip the prefix /v1/clusters/:cluster/plugins/:name from the request
	// path before proxying.
	prefix := path.Join("/v1/clusters", c.Param("cluster"), "plugins", name)
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			req.URL.Path = strings.TrimPrefix(req.URL.Path, prefix)
			if req.URL.Path == "" {
				req.URL.Path = "/"
			}
			req.Host = target.Host
			// Forward external-backend credentials if present.
			if credToken != "" {
				req.Header.Set("Authorization", "Bearer "+credToken)
			}
		},
	}

	proxy.ServeHTTP(c.Response(), c.Request())
	return nil
}

// resolveBackendURL returns the base URL and optional bearer token for the plugin backend.
// Priority: ServiceRef > ExternalURL.
func (pp *pluginProxy) resolveBackendURL(backend *pluginv1alpha1.PluginBackend) (string, string, error) {
	if backend.ServiceRef != nil {
		ref := backend.ServiceRef
		if ref.Namespace == "" || ref.Name == "" || ref.Port == 0 {
			return "", "", fmt.Errorf("serviceRef is missing namespace, name, or port")
		}
		// Standard in-cluster DNS: <name>.<namespace>.svc.cluster.local
		u := fmt.Sprintf("http://%s.%s.svc.cluster.local:%d", ref.Name, ref.Namespace, ref.Port)
		return u, "", nil
	}
	if backend.ExternalURL != "" {
		return backend.ExternalURL, "", nil
	}
	return "", "", fmt.Errorf("backend has neither serviceRef nor externalUrl")
}

func pluginKey(name string) ctrlclient.ObjectKey {
	return ctrlclient.ObjectKey{Name: name}
}
