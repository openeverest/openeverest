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
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/openeverest/openeverest/v2/client"
	"github.com/openeverest/openeverest/v2/pkg/cli/config"
)

// captureStdout returns what fn writes to os.Stdout. Not parallel-safe
// (os.Stdout is global), so callers must not use t.Parallel.
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

func newTestPreset(name, namespace, provider, version string) map[string]any {
	return map[string]any{
		"metadata": map[string]any{
			"name":              name,
			"namespace":         namespace,
			"creationTimestamp": time.Now().Add(-time.Hour).UTC().Format(time.RFC3339),
		},
		"spec": map[string]any{
			"providerRef": map[string]any{"name": provider},
			"version":     version,
		},
	}
}

func presetFromJSON(t *testing.T, body string) *client.InstancePreset {
	t.Helper()
	var p client.InstancePreset
	require.NoError(t, json.Unmarshal([]byte(body), &p))
	return &p
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeTempFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func minimalPresetYAML(name, namespace string) string {
	return fmt.Sprintf(`apiVersion: core.openeverest.io/v1alpha1
kind: InstancePreset
metadata:
  name: %s
  namespace: %s
spec:
  providerRef:
    name: percona-server-mongodb
  version: "8.0.12"
`, name, namespace)
}
