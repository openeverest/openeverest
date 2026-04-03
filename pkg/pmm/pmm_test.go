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

package pmm

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetPMMServerVersion tests GetPMMServerVersion against a mock PMM server.
func TestGetPMMServerVersion(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	tests := map[string]struct {
		serverResp any
		statusCode int
		expected   PMMServerVersion
	}{
		"PMM3 version": {
			serverResp: versionResponse{Version: "3.1.0"},
			statusCode: http.StatusOK,
			expected:   "3.1.0",
		},
		"PMM2 version": {
			serverResp: versionResponse{Version: "2.42.0"},
			statusCode: http.StatusOK,
			expected:   "2.42.0",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, r.URL.Path, "/v1/version")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)

				err := json.NewEncoder(w).Encode(tc.serverResp)
				require.NoError(t, err)
			}))

			defer srv.Close()

			version, err := GetPMMServerVersion(ctx, srv.URL, "test-token", false)
			require.NoError(t, err)
			require.Equal(t, tc.expected, version)
		})
	}
}
