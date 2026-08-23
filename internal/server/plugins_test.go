// everest
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
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/stretchr/testify/assert"
)

func TestResolvePluginAssetPath(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		cluster    string
		pluginName string
		assetPath  string
		expected   string
	}{
		{
			name:       "empty asset path",
			cluster:    "main",
			pluginName: "my-plugin",
			assetPath:  "",
			expected:   "",
		},
		{
			name:       "relative path with leading slash",
			cluster:    "main",
			pluginName: "my-plugin",
			assetPath:  "/icon.png",
			expected:   "/v1/clusters/main/plugins/my-plugin/icon.png",
		},
		{
			name:       "relative path without leading slash",
			cluster:    "remote-cluster",
			pluginName: "my-plugin",
			assetPath:  "assets/icon.svg",
			expected:   "/v1/clusters/remote-cluster/plugins/my-plugin/assets/icon.svg",
		},
		{
			name:       "absolute http URL",
			cluster:    "main",
			pluginName: "my-plugin",
			assetPath:  "http://example.com/icon.png",
			expected:   "http://example.com/icon.png",
		},
		{
			name:       "absolute https URL",
			cluster:    "main",
			pluginName: "my-plugin",
			assetPath:  "https://cdn.example.com/icon.png",
			expected:   "https://cdn.example.com/icon.png",
		},
		{
			name:       "data URI",
			cluster:    "main",
			pluginName: "my-plugin",
			assetPath:  "data:image/svg+xml;base64,PHN2Zz48L3N2Zz4=",
			expected:   "data:image/svg+xml;base64,PHN2Zz48L3N2Zz4=",
		},
		{
			name:       "already prefixed with cluster path",
			cluster:    "main",
			pluginName: "my-plugin",
			assetPath:  "/v1/clusters/main/plugins/my-plugin/icon.png",
			expected:   "/v1/clusters/main/plugins/my-plugin/icon.png",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			e := echo.New()
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			c.SetParamNames("cluster")
			c.SetParamValues(tc.cluster)

			result := resolvePluginAssetPath(c, tc.pluginName, tc.assetPath)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestLegacyPluginRouteRewrite(t *testing.T) {
	t.Parallel()

	newEchoWithRewrite := func() *echo.Echo {
		e := echo.New()
		e.Pre(echomiddleware.Rewrite(map[string]string{
			"/v1/plugins/*": "/v1/clusters/main/plugins/$1",
			"/v1/plugins":   "/v1/clusters/main/plugins",
		}))
		return e
	}

	t.Run("rewrites /v1/plugins to /v1/clusters/main/plugins", func(t *testing.T) {
		t.Parallel()
		e := newEchoWithRewrite()
		var capturedPath string
		e.GET("/v1/clusters/:cluster/plugins", func(c echo.Context) error {
			capturedPath = c.Request().URL.Path
			return c.String(http.StatusOK, "list-plugins")
		})

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v1/plugins", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "list-plugins", rec.Body.String())
		assert.Equal(t, "/v1/clusters/main/plugins", capturedPath)
	})

	t.Run("rewrites /v1/plugins/:name/* to /v1/clusters/main/plugins/:name/*", func(t *testing.T) {
		t.Parallel()
		e := newEchoWithRewrite()
		var capturedPath string
		e.GET("/v1/clusters/:cluster/plugins/:name/*", func(c echo.Context) error {
			capturedPath = c.Request().URL.Path
			return c.String(http.StatusOK, "plugin-asset")
		})

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v1/plugins/my-plugin/bundle.js", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "plugin-asset", rec.Body.String())
		assert.Equal(t, "/v1/clusters/main/plugins/my-plugin/bundle.js", capturedPath)
	})

	t.Run("rewrites /v1/plugins/context to /v1/clusters/main/plugins/context", func(t *testing.T) {
		t.Parallel()
		e := newEchoWithRewrite()
		var capturedPath string
		e.GET("/v1/clusters/:cluster/plugins/context", func(c echo.Context) error {
			capturedPath = c.Request().URL.Path
			return c.String(http.StatusOK, "plugin-context")
		})

		req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v1/plugins/context", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "plugin-context", rec.Body.String())
		assert.Equal(t, "/v1/clusters/main/plugins/context", capturedPath)
	})

	t.Run("rewrites /v1/plugins/:name to /v1/clusters/main/plugins/:name", func(t *testing.T) {
		t.Parallel()
		e := newEchoWithRewrite()
		var capturedPath string
		e.Any("/v1/clusters/:cluster/plugins/:name", func(c echo.Context) error {
			capturedPath = c.Request().URL.Path
			return c.String(http.StatusOK, "plugin-authed")
		})

		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/v1/plugins/my-plugin", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "plugin-authed", rec.Body.String())
		assert.Equal(t, "/v1/clusters/main/plugins/my-plugin", capturedPath)
	})
}

// TestRewriteAllLegacyURLs verifies every URL pattern the UI actually emits
// (plugins.context.tsx, useSubmitPluginInstanceConfig.ts, extension-catalog.ts)
// is correctly rewritten by the backward-compatibility Pre middleware.
// These cases were derived by grepping the UI source for /v1/plugins.
func TestRewriteAllLegacyURLs(t *testing.T) {
	t.Parallel()

	cases := []struct {
		method   string
		from     string
		wantPath string
	}{
		// plugins.context.tsx:104 — list all plugins
		{"GET", "/v1/plugins", "/v1/clusters/main/plugins"},
		// plugins.context.tsx:88 — generic proxy call to plugin backend
		{"GET", "/v1/plugins/my-plugin/api/data", "/v1/clusters/main/plugins/my-plugin/api/data"},
		// useSubmitPluginInstanceConfig.ts:36 — POST to plugin sub-path
		{"POST", "/v1/plugins/my-plugin/instance-config", "/v1/clusters/main/plugins/my-plugin/instance-config"},
		// bundle serving — default bundle path
		{"GET", "/v1/plugins/my-plugin/main.js", "/v1/clusters/main/plugins/my-plugin/main.js"},
		// extension-catalog.ts — plugin-hub api call
		{"GET", "/v1/plugins/plugin-hub/api/summary", "/v1/clusters/main/plugins/plugin-hub/api/summary"},
		// authedProxy — single-segment plugin name (no sub-path)
		{"GET", "/v1/plugins/my-plugin", "/v1/clusters/main/plugins/my-plugin"},
		// /context sub-route registered on pluginGroup
		{"GET", "/v1/plugins/context", "/v1/clusters/main/plugins/context"},
	}

	for _, tc := range cases {
		t.Run(tc.from, func(t *testing.T) {
			t.Parallel()

			e := echo.New()
			e.Pre(echomiddleware.Rewrite(map[string]string{
				"/v1/plugins/*": "/v1/clusters/main/plugins/$1",
				"/v1/plugins":   "/v1/clusters/main/plugins",
			}))

			var got string
			e.Any("/*", func(c echo.Context) error {
				got = c.Request().URL.Path
				return c.NoContent(http.StatusOK)
			})

			req := httptest.NewRequestWithContext(t.Context(), tc.method, tc.from, nil)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			assert.Equal(t, tc.wantPath, got, "URL not rewritten correctly: %s %s", tc.method, tc.from)
		})
	}
}
