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

package cli

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/openeverest/openeverest/v2/client"
)

func TestResolveCluster(t *testing.T) {
	t.Parallel()

	t.Run("explicit cluster skips discovery", func(t *testing.T) {
		t.Parallel()

		requested := false
		srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			requested = true
		}))
		defer srv.Close()

		c, err := client.NewClientWithResponses(srv.URL)
		require.NoError(t, err)

		cluster, err := ResolveCluster(t.Context(), c, "prod")
		require.NoError(t, err)
		require.Equal(t, "prod", cluster)
		require.False(t, requested)
	})

	tests := []struct {
		name     string
		response string
		expected string
		err      string
	}{
		{
			name:     "discovers the only cluster",
			response: `{"items":[{"name":"prod-eu","server":"https://example.com"}]}`,
			expected: "prod-eu",
		},
		{
			name: "rejects ambiguous selection",
			response: `{"items":[` +
				`{"name":"prod-us","server":"https://us.example.com"},` +
				`{"name":"prod-eu","server":"https://eu.example.com"}]}`,
			err: "multiple clusters are available (prod-eu, prod-us); specify --cluster",
		},
		{
			name:     "rejects empty registry",
			response: `{"items":[]}`,
			err:      "no clusters are registered",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "/clusters", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				_, err := w.Write([]byte(tc.response))
				require.NoError(t, err)
			}))
			defer srv.Close()

			c, err := client.NewClientWithResponses(srv.URL)
			require.NoError(t, err)

			cluster, err := ResolveCluster(t.Context(), c, "")
			if tc.err != "" {
				require.EqualError(t, err, tc.err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.expected, cluster)
		})
	}
}
