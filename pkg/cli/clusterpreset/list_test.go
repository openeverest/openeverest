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

package clusterpreset

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/openeverest/openeverest/v2/client"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	fn()
	require.NoError(t, w.Close())
	out, err := io.ReadAll(r)
	require.NoError(t, err)
	return string(out)
}

func TestClusterPresetList_HappyPath(t *testing.T) {
	items := []map[string]any{
		newTestPreset("mongodb-default", "percona-server-mongodb", "8.0.12"),
		newTestPreset("pg-default", "percona-postgresql", "16.0"),
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
		require.NoError(t, NewListRunner(Config{Pretty: true}, zap.NewNop().Sugar()).Run(t.Context(), ListOptions{
			Cluster:  "main",
			Provider: "percona-server-mongodb",
		}, cfgPath))
	})

	assert.Contains(t, requestPath, "/cluster-instance-presets")
	assert.Equal(t, "percona-server-mongodb", query.Get("provider"))
	assert.Contains(t, out, "mongodb-default")
	assert.Contains(t, out, "NAME")
}

func TestClusterPresetList_YAMLOutput(t *testing.T) {
	t.Parallel()

	presets := []client.ClusterInstancePreset{*presetFromJSON(t, `{"metadata":{"name":"mongodb-default"}}`)}
	var buf bytes.Buffer
	NewListRunner(Config{YAML: true}, zap.NewNop().Sugar()).render(&buf, presets)
	assert.Contains(t, buf.String(), "name: mongodb-default\n")
}
