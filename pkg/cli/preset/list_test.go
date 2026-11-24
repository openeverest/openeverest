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
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/openeverest/openeverest/v2/client"
)

//nolint:paralleltest // captureStdout mutates global os.Stdout; must run serially
func TestPresetList_HappyPath(t *testing.T) {
	items := []map[string]any{
		newTestPreset("mongodb-production", "everest", "percona-server-mongodb", "8.0.12"),
		newTestPreset("mongodb-dev", "everest", "percona-server-mongodb", "8.0.4"),
	}

	var requestPath string
	var query url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.Path
		query = r.URL.Query()
		writeJSON(w, http.StatusOK, map[string]any{"items": items})
	}))
	defer srv.Close()

	cfgPath := newConfigPath(t, srv.URL)
	out := captureStdout(t, func() {
		runner := NewListRunner(Config{Pretty: true}, zap.NewNop().Sugar())
		require.NoError(t, runner.Run(t.Context(), ListOptions{
			Namespace: "everest",
			Cluster:   "main",
			Provider:  "percona-server-mongodb",
		}, cfgPath,
		))
	})

	assert.Contains(t, requestPath, "/namespaces/everest/instance-presets")
	assert.Equal(t, "percona-server-mongodb", query.Get("provider"))
	assert.Contains(t, out, "mongodb-production")
	assert.Contains(t, out, "mongodb-dev")
	assert.Contains(t, out, "NAME")
}

func TestPresetList_YAMLOutput(t *testing.T) {
	t.Parallel()

	presets := []client.InstancePreset{*presetFromJSON(t, `{"metadata":{"name":"mongodb-production","namespace":"everest"}}`)}
	var buf bytes.Buffer
	NewListRunner(Config{YAML: true}, zap.NewNop().Sugar()).render(&buf, presets)
	assert.Contains(t, buf.String(), "- metadata:\n    name: mongodb-production\n")
}

func TestPresetList_ServerError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	cfgPath := newConfigPath(t, srv.URL)
	err := NewListRunner(Config{Pretty: true}, zap.NewNop().Sugar()).Run(t.Context(), ListOptions{
		Namespace: "everest",
		Cluster:   "main",
	}, cfgPath,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected response")
}

func TestPresetList_AllNamespaces(t *testing.T) {
	t.Parallel()

	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		switch r.URL.Path {
		case "/v1/clusters/main/namespaces":
			writeJSON(w, http.StatusOK, []string{"ns-a", "ns-b"})
		default:
			writeJSON(w, http.StatusOK, map[string]any{"items": []any{}})
		}
	}))
	defer srv.Close()

	cfgPath := newConfigPath(t, srv.URL)
	require.NoError(t, NewListRunner(Config{Pretty: true}, zap.NewNop().Sugar()).Run(t.Context(), ListOptions{
		AllNamespaces: true,
		Cluster:       "main",
	}, cfgPath,
	))

	joined, err := json.Marshal(paths)
	require.NoError(t, err)
	assert.Contains(t, string(joined), `/namespaces/ns-a/instance-presets`)
	assert.Contains(t, string(joined), `/namespaces/ns-b/instance-presets`)
}
