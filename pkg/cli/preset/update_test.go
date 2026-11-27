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

package preset

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

//nolint:paralleltest // captureStdout mutates global os.Stdout; must run serially
func TestPresetUpdate_StampsResourceVersion(t *testing.T) {
	live := map[string]any{
		"metadata": map[string]any{
			"name":            "mongodb-production",
			"namespace":       "everest",
			"uid":             "live-uid",
			"resourceVersion": "99",
		},
		"spec": map[string]any{
			"providerRef": map[string]any{"name": "percona-server-mongodb"},
			"version":     "8.0.4",
		},
	}

	var putBody map[string]any
	mux := http.NewServeMux()
	path := "/v1/clusters/main/namespaces/everest/instance-presets/mongodb-production"
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, live)
		case http.MethodPut:
			raw, err := io.ReadAll(r.Body)
			if !assert.NoError(t, err) {
				return
			}
			if !assert.NoError(t, json.Unmarshal(raw, &putBody)) {
				return
			}
			writeJSON(w, http.StatusOK, putBody)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	file := writeTempFile(t, "preset.yaml", minimalPresetYAML("mongodb-production", "everest"))
	cfgPath := newConfigPath(t, srv.URL)

	out := captureStdout(t, func() {
		require.NoError(t, NewUpdater(Config{Pretty: true}, zap.NewNop().Sugar()).Run(t.Context(), UpdateOptions{
			File:    file,
			Cluster: "main",
		}, cfgPath,
		))
	})

	md, ok := putBody["metadata"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "99", md["resourceVersion"])
	assert.Equal(t, "live-uid", md["uid"])
	spec, ok := putBody["spec"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "8.0.12", spec["version"])
	assert.Contains(t, out, "updated")
}

func TestPresetUpdate_NotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	file := writeTempFile(t, "preset.yaml", minimalPresetYAML("missing", "everest"))
	cfgPath := newConfigPath(t, srv.URL)

	err := NewUpdater(Config{Pretty: true}, zap.NewNop().Sugar()).Run(t.Context(), UpdateOptions{
		File:    file,
		Cluster: "main",
	}, cfgPath,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

//nolint:paralleltest // captureStdout mutates global os.Stdout; must run serially
func TestPresetUpdate_YAMLOutput(t *testing.T) {
	live := map[string]any{
		"metadata": map[string]any{
			"name":            "mongodb-production",
			"namespace":       "everest",
			"uid":             "live-uid",
			"resourceVersion": "99",
		},
		"spec": map[string]any{
			"providerRef": map[string]any{"name": "percona-server-mongodb"},
			"version":     "8.0.4",
		},
	}

	mux := http.NewServeMux()
	path := "/v1/clusters/main/namespaces/everest/instance-presets/mongodb-production"
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, live)
		case http.MethodPut:
			raw, err := io.ReadAll(r.Body)
			if !assert.NoError(t, err) {
				return
			}
			var body map[string]any
			if !assert.NoError(t, json.Unmarshal(raw, &body)) {
				return
			}
			writeJSON(w, http.StatusOK, body)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	file := writeTempFile(t, "preset.yaml", minimalPresetYAML("mongodb-production", "everest"))
	cfgPath := newConfigPath(t, srv.URL)

	out := captureStdout(t, func() {
		require.NoError(t, NewUpdater(Config{YAML: true}, zap.NewNop().Sugar()).Run(t.Context(), UpdateOptions{
			File:    file,
			Cluster: "main",
		}, cfgPath,
		))
	})
	assert.Contains(t, out, "metadata:\n  name: mongodb-production\n")
	assert.NotContains(t, out, `"name":"mongodb-production"`)
}
