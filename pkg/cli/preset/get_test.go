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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestPresetGet_HappyPath(t *testing.T) {
	t.Parallel()

	item := newTestPreset("mongodb-production", "everest", "percona-server-mongodb", "8.0.12")

	var requestPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.Path
		writeJSON(w, http.StatusOK, item)
	}))
	defer srv.Close()

	cfgPath := newConfigPath(t, srv.URL)
	require.NoError(t, NewGetRunner(Config{Pretty: true}, zap.NewNop().Sugar()).Run(t.Context(), GetOptions{
		Name:      "mongodb-production",
		Namespace: "everest",
		Cluster:   "main",
	}, cfgPath))
	assert.Contains(t, requestPath, "/namespaces/everest/instance-presets/mongodb-production")
}

func TestPresetGet_NotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	cfgPath := newConfigPath(t, srv.URL)
	err := NewGetRunner(Config{Pretty: true}, zap.NewNop().Sugar()).Run(t.Context(), GetOptions{
		Name:      "missing",
		Namespace: "everest",
		Cluster:   "main",
	}, cfgPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestPresetGet_YAMLOutput(t *testing.T) {
	t.Parallel()

	preset := presetFromJSON(t, `{"metadata":{"name":"mongodb-production"}}`)
	var buf bytes.Buffer
	require.NoError(t, NewGetRunner(Config{YAML: true}, zap.NewNop().Sugar()).render(&buf, preset))
	assert.Contains(t, buf.String(), "metadata:\n  name: mongodb-production\n")
}
