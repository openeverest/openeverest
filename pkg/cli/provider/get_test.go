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

package provider

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/openeverest/openeverest/v2/client"
	"github.com/openeverest/openeverest/v2/pkg/cli/config"
)

func TestProviderGet_HappyPath(t *testing.T) {
	t.Parallel()

	item := newTestProvider("percona-server-mongodb")

	var requestPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(item)
	}))
	defer srv.Close()

	cfg := newTestConfig(srv.URL)
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, cfg.Save(cfgPath))

	runner := NewGetRunner(Config{Pretty: true}, zap.NewNop().Sugar())
	err := runner.Run(t.Context(), GetOptions{
		Name:    "percona-server-mongodb",
		Cluster: "main",
	}, cfgPath,
	)
	require.NoError(t, err)
	assert.Contains(t, requestPath, "/providers/percona-server-mongodb")
}

func TestProviderGet_NotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	cfg := newTestConfig(srv.URL)
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, cfg.Save(cfgPath))

	runner := NewGetRunner(Config{Pretty: true}, zap.NewNop().Sugar())
	err := runner.Run(t.Context(), GetOptions{
		Name:    "missing",
		Cluster: "main",
	}, cfgPath,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestProviderGet_ServerError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	cfg := newTestConfig(srv.URL)
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, cfg.Save(cfgPath))

	runner := NewGetRunner(Config{Pretty: true}, zap.NewNop().Sugar())
	err := runner.Run(t.Context(), GetOptions{
		Name:    "percona-server-mongodb",
		Cluster: "main",
	}, cfgPath,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected response")
}

func TestProviderGet_NoActiveContext(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		APIVersion: "config.openeverest.io/v1alpha1",
		Kind:       "ClientConfig",
	}
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, cfg.Save(cfgPath))

	runner := NewGetRunner(Config{Pretty: true}, zap.NewNop().Sugar())
	err := runner.Run(t.Context(), GetOptions{
		Name:    "percona-server-mongodb",
		Cluster: "main",
	}, cfgPath,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no active context")
}

func TestProviderGet_YAMLOutput(t *testing.T) {
	t.Parallel()

	var p client.Provider
	require.NoError(t, json.Unmarshal([]byte(`{"metadata":{"name":"percona-server-mongodb"}}`), &p))
	var buf bytes.Buffer
	require.NoError(t, NewGetRunner(Config{YAML: true}, zap.NewNop().Sugar()).render(&buf, &p))
	assert.Contains(t, buf.String(), "metadata:\n  name: percona-server-mongodb\n")
}
