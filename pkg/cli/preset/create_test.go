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

func TestPresetCreate_HappyPath(t *testing.T) {
	var gotPath string
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		require.Equal(t, http.MethodPost, r.Method)
		raw, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(raw, &body))
		writeJSON(w, http.StatusCreated, body)
	}))
	defer srv.Close()

	file := writeTempFile(t, "preset.yaml", minimalPresetYAML("mongodb-production", "everest"))
	cfgPath := newConfigPath(t, srv.URL)

	out := captureStdout(t, func() {
		require.NoError(t, NewCreateRunner(Config{Pretty: true}, zap.NewNop().Sugar()).Run(t.Context(), CreateOptions{
			File:    file,
			Cluster: "main",
		}, cfgPath))
	})

	assert.Contains(t, gotPath, "/namespaces/everest/instance-presets")
	assert.Equal(t, "mongodb-production", body["metadata"].(map[string]any)["name"])
	assert.Contains(t, out, "created")
}

func TestPresetCreate_FlagOverrides(t *testing.T) {
	t.Parallel()

	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(raw, &body))
		assert.Contains(t, r.URL.Path, "/namespaces/staging/instance-presets")
		writeJSON(w, http.StatusCreated, body)
	}))
	defer srv.Close()

	file := writeTempFile(t, "preset.yaml", minimalPresetYAML("mongodb-production", "everest"))
	cfgPath := newConfigPath(t, srv.URL)

	require.NoError(t, NewCreateRunner(Config{Pretty: true}, zap.NewNop().Sugar()).Run(t.Context(), CreateOptions{
		File:      file,
		Name:      "mongodb-staging",
		Namespace: "staging",
		Cluster:   "main",
	}, cfgPath))

	md := body["metadata"].(map[string]any)
	assert.Equal(t, "mongodb-staging", md["name"])
	assert.Equal(t, "staging", md["namespace"])
}

func TestPresetCreate_StripsServerFields(t *testing.T) {
	t.Parallel()

	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(raw, &body))
		writeJSON(w, http.StatusCreated, body)
	}))
	defer srv.Close()

	exported := `apiVersion: core.openeverest.io/v1alpha1
kind: InstancePreset
metadata:
  name: mongodb-production
  namespace: everest
  uid: abc-123
  resourceVersion: "42"
  generation: 3
spec:
  providerRef:
    name: percona-server-mongodb
status:
  conditions: []
`
	file := writeTempFile(t, "exported.yaml", exported)
	cfgPath := newConfigPath(t, srv.URL)

	require.NoError(t, NewCreateRunner(Config{Pretty: true}, zap.NewNop().Sugar()).Run(t.Context(), CreateOptions{
		File:    file,
		Cluster: "main",
	}, cfgPath))

	md := body["metadata"].(map[string]any)
	_, hasUID := md["uid"]
	_, hasRV := md["resourceVersion"]
	_, hasStatus := body["status"]
	assert.False(t, hasUID)
	assert.False(t, hasRV)
	assert.False(t, hasStatus)
}

func TestPresetCreate_Conflict(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
	}))
	defer srv.Close()

	file := writeTempFile(t, "preset.yaml", minimalPresetYAML("mongodb-production", "everest"))
	cfgPath := newConfigPath(t, srv.URL)

	err := NewCreateRunner(Config{Pretty: true}, zap.NewNop().Sugar()).Run(t.Context(), CreateOptions{
		File:    file,
		Cluster: "main",
	}, cfgPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestPresetCreate_MissingFile(t *testing.T) {
	t.Parallel()

	err := NewCreateRunner(Config{Pretty: true}, zap.NewNop().Sugar()).Run(t.Context(), CreateOptions{
		Cluster: "main",
	}, "/dev/null")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "file")
}

//nolint:paralleltest // captureStdout mutates global os.Stdout; must run serially
func TestPresetCreate_YAMLOutput(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var body map[string]any
		require.NoError(t, json.Unmarshal(raw, &body))
		writeJSON(w, http.StatusCreated, body)
	}))
	defer srv.Close()

	file := writeTempFile(t, "preset.yaml", minimalPresetYAML("mongodb-production", "everest"))
	cfgPath := newConfigPath(t, srv.URL)

	out := captureStdout(t, func() {
		require.NoError(t, NewCreateRunner(Config{YAML: true}, zap.NewNop().Sugar()).Run(t.Context(), CreateOptions{
			File:    file,
			Cluster: "main",
		}, cfgPath))
	})
	assert.Contains(t, out, "metadata:\n  name: mongodb-production\n")
	assert.NotContains(t, out, `"name":"mongodb-production"`)
}
