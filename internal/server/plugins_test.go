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
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/casbin/casbin/v2"
	casbinmodel "github.com/casbin/casbin/v2/model"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	pluginv1alpha1 "github.com/openeverest/openeverest/v2/api/extensions/v1alpha1"
	"github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
)

// rbacModel mirrors data/rbac/model.conf. The plugin proxy's RBAC checks rely
// solely on this matcher's glob semantics, so building an enforcer directly
// from the model lets us grant plugin-scoped resources ("plugin/<name>",
// "plugins") without going through the ConfigMap policy validator, which only
// knows the generated API resource names.
const rbacModel = `[request_definition]
r = sub, res, act, obj

[policy_definition]
p = sub, res, act, obj

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow)) && !some(where (p.eft == deny))

[matchers]
m = g(r.sub, p.sub) && globMatch(r.res, p.res) && globMatch(r.act, p.act) && globMatch(r.obj, p.obj)
`

// fakePluginConnector implements only the plugin-related methods of
// kubernetes.KubernetesConnector. The embedded nil interface satisfies the
// full contract; any other method call panics, which is exactly what we want
// for a focused unit test.
type fakePluginConnector struct {
	kubernetes.KubernetesConnector

	plugins []pluginv1alpha1.Plugin
	listErr error
}

func (f *fakePluginConnector) ListPlugins(_ context.Context, _ ...ctrlclient.ListOption) (*pluginv1alpha1.PluginList, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return &pluginv1alpha1.PluginList{Items: f.plugins}, nil
}

func (f *fakePluginConnector) GetPlugin(_ context.Context, key ctrlclient.ObjectKey) (*pluginv1alpha1.Plugin, error) {
	for i := range f.plugins {
		if f.plugins[i].Name == key.Name {
			return &f.plugins[i], nil
		}
	}
	return nil, fmt.Errorf("plugin %q not found", key.Name)
}

// newTestEnforcer builds a Casbin enforcer from the RBAC model and the given
// policy rules. Each rule is a {subject, resource, action, object} tuple.
func newTestEnforcer(t *testing.T, rules ...[]string) *casbin.Enforcer {
	t.Helper()
	m, err := casbinmodel.NewModelFromString(rbacModel)
	require.NoError(t, err)
	enf, err := casbin.NewEnforcer(m)
	require.NoError(t, err)
	for _, r := range rules {
		ok, err := enf.AddPolicy(r[0], r[1], r[2], r[3])
		require.NoError(t, err)
		require.True(t, ok)
	}
	return enf
}

// ctxWithUser returns a context carrying a JWT for the given subject/groups,
// shaped the way rbac.GetUser expects it.
func ctxWithUser(sub string, groups ...string) context.Context {
	claims := jwt.MapClaims{"sub": sub, "iss": "test-issuer"}
	if len(groups) > 0 {
		claims["groups"] = groups
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return context.WithValue(context.Background(), common.UserCtxKey, token) //nolint:staticcheck
}

// newEchoContext builds an echo.Context whose request carries reqCtx.
func newEchoContext(reqCtx context.Context, target string) (echo.Context, *httptest.ResponseRecorder) { //nolint:ireturn // echo.NewContext only returns the interface
	req := httptest.NewRequestWithContext(reqCtx, http.MethodGet, target, nil)
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(req, rec)
	return c, rec
}

func TestResolvePluginAssetPath(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name       string
		pluginName string
		assetPath  string
		want       string
	}{
		{name: "empty", pluginName: "my-plugin", assetPath: "", want: ""},
		{name: "http absolute", pluginName: "my-plugin", assetPath: "http://x/icon.png", want: "http://x/icon.png"},
		{name: "https absolute", pluginName: "my-plugin", assetPath: "https://x/icon.png", want: "https://x/icon.png"},
		{name: "data uri", pluginName: "my-plugin", assetPath: "data:image/png;base64,AAAA", want: "data:image/png;base64,AAAA"},
		{name: "already proxied", pluginName: "my-plugin", assetPath: "/v1/plugins/other/icon.png", want: "/v1/plugins/other/icon.png"},
		{name: "leading slash relative", pluginName: "my-plugin", assetPath: "/icon.png", want: "/v1/plugins/my-plugin/icon.png"},
		{name: "bare relative", pluginName: "my-plugin", assetPath: "icon.png", want: "/v1/plugins/my-plugin/icon.png"},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, resolvePluginAssetPath(tc.pluginName, tc.assetPath))
		})
	}
}

func TestResolveBackendURL(t *testing.T) {
	t.Parallel()
	pp := &pluginProxy{}

	t.Run("serviceRef resolves to in-cluster DNS", func(t *testing.T) {
		t.Parallel()
		u, tok, err := pp.resolveBackendURL(&pluginv1alpha1.PluginBackend{
			ServiceRef: &pluginv1alpha1.PluginBackendServiceRef{
				Namespace: "everest-system", Name: "plugin-hub", Port: 8080,
			},
		})
		require.NoError(t, err)
		assert.Equal(t, "http://plugin-hub.everest-system.svc.cluster.local:8080", u)
		assert.Empty(t, tok)
	})

	t.Run("serviceRef takes priority over externalUrl", func(t *testing.T) {
		t.Parallel()
		u, _, err := pp.resolveBackendURL(&pluginv1alpha1.PluginBackend{
			ServiceRef: &pluginv1alpha1.PluginBackendServiceRef{
				Namespace: "ns", Name: "svc", Port: 80,
			},
			ExternalURL: "https://external.example.com",
		})
		require.NoError(t, err)
		assert.Equal(t, "http://svc.ns.svc.cluster.local:80", u)
	})

	t.Run("serviceRef missing port errors", func(t *testing.T) {
		t.Parallel()
		_, _, err := pp.resolveBackendURL(&pluginv1alpha1.PluginBackend{
			ServiceRef: &pluginv1alpha1.PluginBackendServiceRef{Namespace: "ns", Name: "svc"},
		})
		require.Error(t, err)
	})

	t.Run("externalUrl used when no serviceRef", func(t *testing.T) {
		t.Parallel()
		u, _, err := pp.resolveBackendURL(&pluginv1alpha1.PluginBackend{
			ExternalURL: "https://external.example.com",
		})
		require.NoError(t, err)
		assert.Equal(t, "https://external.example.com", u)
	})

	t.Run("neither serviceRef nor externalUrl errors", func(t *testing.T) {
		t.Parallel()
		_, _, err := pp.resolveBackendURL(&pluginv1alpha1.PluginBackend{})
		require.Error(t, err)
	})
}

func TestCanUsePlugin(t *testing.T) {
	t.Parallel()
	pp := &pluginProxy{enforcer: newTestEnforcer(t,
		[]string{"bob", "plugin/plugin-hub", "use", "*"},
		[]string{"carol", "plugin/*", "use", "*"},
		[]string{"dave", "plugins", "*", "*"},
		[]string{"team-a", "plugin/plugin-hub", "use", "*"},
	)}

	testCases := []struct {
		name    string
		subject string
		groups  []string
		plugin  string
		want    bool
	}{
		{
			name:    "direct use grant",
			subject: "bob",
			plugin:  "plugin-hub",
			want:    true,
		},
		{
			name:    "direct grant does not leak to other plugin",
			subject: "bob",
			plugin:  "other",
			want:    false,
		},
		{
			name:    "wildcard plugin grant",
			subject: "carol",
			plugin:  "anything",
			want:    true,
		},
		{
			name:    "admin star on plugins",
			subject: "dave",
			plugin:  "plugin-hub",
			want:    true,
		},
		{
			name:    "group-based grant",
			subject: "eve",
			groups:  []string{"team-a"},
			plugin:  "plugin-hub",
			want:    true,
		},
		{
			name:    "no grant denied",
			subject: "nobody",
			plugin:  "plugin-hub",
			want:    false,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c, _ := newEchoContext(ctxWithUser(tc.subject, tc.groups...), "/v1/plugins")
			allowed, err := pp.canUsePlugin(c, tc.plugin)
			require.NoError(t, err)
			assert.Equal(t, tc.want, allowed)
		})
	}

	t.Run("missing user errors", func(t *testing.T) {
		t.Parallel()
		c, _ := newEchoContext(context.Background(), "/v1/plugins")
		_, err := pp.canUsePlugin(c, "plugin-hub")
		require.Error(t, err)
	})
}

func TestCheckPluginsReadAccess(t *testing.T) {
	t.Parallel()
	pp := &pluginProxy{enforcer: newTestEnforcer(t, []string{"reader", "plugins", "read", "*"})}

	t.Run("allowed with read grant", func(t *testing.T) {
		t.Parallel()
		c, _ := newEchoContext(ctxWithUser("reader"), "/v1/plugins")
		require.NoError(t, pp.checkPluginsReadAccess(c))
	})

	t.Run("forbidden without grant", func(t *testing.T) {
		t.Parallel()
		c, _ := newEchoContext(ctxWithUser("stranger"), "/v1/plugins")
		err := pp.checkPluginsReadAccess(c)
		var httpErr *echo.HTTPError
		require.ErrorAs(t, err, &httpErr)
		assert.Equal(t, http.StatusForbidden, httpErr.Code)
	})

	t.Run("unauthorized without user", func(t *testing.T) {
		t.Parallel()
		c, _ := newEchoContext(context.Background(), "/v1/plugins")
		err := pp.checkPluginsReadAccess(c)
		var httpErr *echo.HTTPError
		require.ErrorAs(t, err, &httpErr)
		assert.Equal(t, http.StatusUnauthorized, httpErr.Code)
	})
}

func TestListPluginsHandler(t *testing.T) {
	t.Parallel()

	plugins := []pluginv1alpha1.Plugin{
		{
			ObjectMeta: metav1.ObjectMeta{Name: "plugin-hub"},
			Spec: pluginv1alpha1.PluginSpec{
				DisplayName: "Plugin Hub",
				Enabled:     true,
				Backend:     &pluginv1alpha1.PluginBackend{ExternalURL: "http://example"},
				Frontend:    &pluginv1alpha1.PluginFrontend{BundlePath: "/hub.js"},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "disabled-plugin"},
			Spec: pluginv1alpha1.PluginSpec{
				DisplayName: "Disabled",
				Enabled:     false,
				Backend:     &pluginv1alpha1.PluginBackend{ExternalURL: "http://example"},
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "forbidden-plugin"},
			Spec: pluginv1alpha1.PluginSpec{
				DisplayName: "Forbidden",
				Enabled:     true,
				Backend:     &pluginv1alpha1.PluginBackend{ExternalURL: "http://example"},
			},
		},
	}

	pp := &pluginProxy{
		kubeConnector: &fakePluginConnector{plugins: plugins},
		enforcer: newTestEnforcer(t,
			[]string{"bob", "plugins", "read", "*"},
			[]string{"bob", "plugin/plugin-hub", "use", "*"},
		),
	}

	t.Run("returns only enabled and usable plugins", func(t *testing.T) {
		t.Parallel()
		c, rec := newEchoContext(ctxWithUser("bob"), "/v1/plugins")
		require.NoError(t, pp.listPluginsHandler(c))
		assert.Equal(t, http.StatusOK, rec.Code)

		var got []struct {
			Name      string `json:"name"`
			BundleURL string `json:"bundleUrl"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
		require.Len(t, got, 1)
		assert.Equal(t, "plugin-hub", got[0].Name)
		assert.Equal(t, "/v1/plugins/plugin-hub/hub.js", got[0].BundleURL)
	})

	t.Run("forbidden without read access", func(t *testing.T) {
		t.Parallel()
		c, _ := newEchoContext(ctxWithUser("stranger"), "/v1/plugins")
		err := pp.listPluginsHandler(c)
		var httpErr *echo.HTTPError
		require.ErrorAs(t, err, &httpErr)
		assert.Equal(t, http.StatusForbidden, httpErr.Code)
	})
}

func TestDoProxy(t *testing.T) {
	t.Parallel()

	t.Run("strips prefix and forwards to backend", func(t *testing.T) {
		t.Parallel()
		var gotPath string
		backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotPath = r.URL.Path
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		}))
		defer backend.Close()

		pp := &pluginProxy{kubeConnector: &fakePluginConnector{plugins: []pluginv1alpha1.Plugin{{
			ObjectMeta: metav1.ObjectMeta{Name: "myplugin"},
			Spec: pluginv1alpha1.PluginSpec{
				Enabled: true,
				Backend: &pluginv1alpha1.PluginBackend{ExternalURL: backend.URL},
			},
		}}}}

		c, rec := newEchoContext(context.Background(), "/v1/plugins/myplugin/api/summary")
		c.SetParamNames("name")
		c.SetParamValues("myplugin")

		require.NoError(t, pp.doProxy(c))
		assert.Equal(t, "/api/summary", gotPath)
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "ok", rec.Body.String())
	})

	t.Run("unknown plugin returns 404", func(t *testing.T) {
		t.Parallel()
		pp := &pluginProxy{kubeConnector: &fakePluginConnector{}}
		c, rec := newEchoContext(context.Background(), "/v1/plugins/missing/")
		c.SetParamNames("name")
		c.SetParamValues("missing")
		require.NoError(t, pp.doProxy(c))
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("disabled plugin returns 404", func(t *testing.T) {
		t.Parallel()
		pp := &pluginProxy{kubeConnector: &fakePluginConnector{plugins: []pluginv1alpha1.Plugin{{
			ObjectMeta: metav1.ObjectMeta{Name: "myplugin"},
			Spec: pluginv1alpha1.PluginSpec{
				Enabled: false,
				Backend: &pluginv1alpha1.PluginBackend{ExternalURL: "http://example"},
			},
		}}}}
		c, rec := newEchoContext(context.Background(), "/v1/plugins/myplugin/")
		c.SetParamNames("name")
		c.SetParamValues("myplugin")
		require.NoError(t, pp.doProxy(c))
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("plugin without backend returns 404", func(t *testing.T) {
		t.Parallel()
		pp := &pluginProxy{kubeConnector: &fakePluginConnector{plugins: []pluginv1alpha1.Plugin{{
			ObjectMeta: metav1.ObjectMeta{Name: "myplugin"},
			Spec:       pluginv1alpha1.PluginSpec{Enabled: true},
		}}}}
		c, rec := newEchoContext(context.Background(), "/v1/plugins/myplugin/")
		c.SetParamNames("name")
		c.SetParamValues("myplugin")
		require.NoError(t, pp.doProxy(c))
		assert.Equal(t, http.StatusNotFound, rec.Code)
	})
}

func TestAuthedProxyHandler(t *testing.T) {
	t.Parallel()

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(backend.Close)

	newProxy := func(t *testing.T, rules ...[]string) *pluginProxy {
		t.Helper()
		return &pluginProxy{
			kubeConnector: &fakePluginConnector{plugins: []pluginv1alpha1.Plugin{{
				ObjectMeta: metav1.ObjectMeta{Name: "plugin-hub"},
				Spec: pluginv1alpha1.PluginSpec{
					Enabled: true,
					Backend: &pluginv1alpha1.PluginBackend{ExternalURL: backend.URL},
				},
			}}},
			enforcer: newTestEnforcer(t, rules...),
		}
	}

	t.Run("forbidden without use grant", func(t *testing.T) {
		t.Parallel()
		pp := newProxy(t, []string{"bob", "plugins", "read", "*"})
		c, _ := newEchoContext(ctxWithUser("bob"), "/v1/plugins/plugin-hub/api/summary")
		c.SetParamNames("name")
		c.SetParamValues("plugin-hub")
		err := pp.authedProxyHandler(c)
		var httpErr *echo.HTTPError
		require.ErrorAs(t, err, &httpErr)
		assert.Equal(t, http.StatusForbidden, httpErr.Code)
	})

	t.Run("proxied with use grant", func(t *testing.T) {
		t.Parallel()
		pp := newProxy(t, []string{"bob", "plugin/plugin-hub", "use", "*"})
		c, rec := newEchoContext(ctxWithUser("bob"), "/v1/plugins/plugin-hub/api/summary")
		c.SetParamNames("name")
		c.SetParamValues("plugin-hub")
		require.NoError(t, pp.authedProxyHandler(c))
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "ok", rec.Body.String())
	})
}
