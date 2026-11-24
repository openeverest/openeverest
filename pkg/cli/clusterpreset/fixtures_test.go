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
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openeverest/openeverest/v2/client"
	"github.com/openeverest/openeverest/v2/pkg/cli/config"
)

func newTestConfig(serverURL string) *config.Config {
	host := serverURL[len("http://"):]
	srvName := host
	userName := "admin@" + srvName
	return &config.Config{
		APIVersion:     "config.openeverest.io/v1alpha1",
		Kind:           "ClientConfig",
		CurrentContext: userName,
		Contexts: []config.NamedContext{
			{Name: userName, Context: config.Context{Server: srvName, User: userName}},
		},
		Servers: []config.NamedServer{
			{Name: srvName, Server: config.Server{URL: serverURL}},
		},
		Users: []config.NamedUser{
			{Name: userName, User: config.User{
				AccessToken:  "test-token",
				RefreshToken: "rt-test",
				ExpiresAt:    time.Now().Add(time.Hour),
			}},
		},
	}
}

func newConfigPath(t *testing.T, serverURL string) string {
	t.Helper()
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, newTestConfig(serverURL).Save(cfgPath))
	return cfgPath
}

func newTestPreset(name, provider, version string) map[string]any {
	return map[string]any{
		"metadata": map[string]any{
			"name":              name,
			"creationTimestamp": time.Now().Add(-time.Hour).UTC().Format(time.RFC3339),
		},
		"spec": map[string]any{
			"providerRef": map[string]any{"name": provider},
			"version":     version,
		},
	}
}

func presetFromJSON(t *testing.T, body string) *client.ClusterInstancePreset {
	t.Helper()
	var p client.ClusterInstancePreset
	require.NoError(t, json.Unmarshal([]byte(body), &p))
	return &p
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
