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

package agent

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/client-go/rest"
)

func TestProxy(t *testing.T) {
	t.Parallel()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/apis/core.openeverest.io/v1alpha1/instances", r.URL.Path)
		require.Equal(t, "Bearer spoke-service-account", r.Header.Get("Authorization"))
		require.Empty(t, r.Header.Get("Impersonate-User"))
		require.Empty(t, r.Header.Get("X-Forwarded-Authorization"))
		w.WriteHeader(http.StatusNoContent)
	}))
	defer upstream.Close()

	proxy, err := NewProxy(
		&rest.Config{Host: upstream.URL, BearerToken: "spoke-service-account"},
		"hub-agent-token",
	)
	require.NoError(t, err)

	t.Run("rejects unauthenticated requests", func(t *testing.T) {
		rec := httptest.NewRecorder()
		proxy.ServeHTTP(rec, httptest.NewRequest(
			http.MethodGet,
			"/apis/core.openeverest.io/v1alpha1/instances",
			nil,
		))
		require.Equal(t, http.StatusUnauthorized, rec.Code)
	})

	t.Run("rejects non Kubernetes paths", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/execute", nil)
		req.Header.Set("Authorization", "Bearer hub-agent-token")
		rec := httptest.NewRecorder()
		proxy.ServeHTTP(rec, req)
		require.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("rejects interactive Kubernetes subresources", func(t *testing.T) {
		req := httptest.NewRequest(
			http.MethodPost,
			"/api/v1/namespaces/default/pods/example/exec",
			nil,
		)
		req.Header.Set("Authorization", "Bearer hub-agent-token")
		rec := httptest.NewRecorder()
		proxy.ServeHTTP(rec, req)
		require.Equal(t, http.StatusNotFound, rec.Code)
	})

	t.Run("replaces hub identity with spoke credentials", func(t *testing.T) {
		req := httptest.NewRequest(
			http.MethodGet,
			"/apis/core.openeverest.io/v1alpha1/instances",
			nil,
		)
		req.Header.Set("Authorization", "Bearer hub-agent-token")
		req.Header.Set("Impersonate-User", "attacker")
		rec := httptest.NewRecorder()
		proxy.ServeHTTP(rec, req)
		require.Equal(t, http.StatusNoContent, rec.Code)
	})
}

func TestProxyRequiresBearerToken(t *testing.T) {
	t.Parallel()

	_, err := NewProxy(&rest.Config{Host: "https://example.test"}, "")
	require.ErrorIs(t, err, errMissingBearerToken)
}
