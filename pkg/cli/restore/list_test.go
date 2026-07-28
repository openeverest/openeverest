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

package restore

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/openeverest/openeverest/v2/client"
	"github.com/openeverest/openeverest/v2/pkg/cli/config"
)

type testRestore struct {
	Metadata map[string]any `json:"metadata"`
	Spec     struct {
		InstanceRef struct {
			Name string `json:"name"`
		} `json:"instanceRef"`
		DataSource struct {
			Backup *struct {
				BackupRef struct {
					Name string `json:"name"`
				} `json:"backupRef"`
			} `json:"backup,omitempty"`
		} `json:"dataSource"`
	} `json:"spec"`
	Status *struct {
		State string `json:"state,omitempty"`
	} `json:"status,omitempty"`
}

func newTestRestore(name, namespace, instance, backup string, age time.Duration) testRestore {
	tr := testRestore{
		Metadata: map[string]any{
			"name":              name,
			"namespace":         namespace,
			"creationTimestamp": time.Now().Add(-age).UTC().Format(time.RFC3339),
		},
	}
	tr.Spec.InstanceRef.Name = instance
	tr.Spec.DataSource.Backup = &struct {
		BackupRef struct {
			Name string `json:"name"`
		} `json:"backupRef"`
	}{}
	tr.Spec.DataSource.Backup.BackupRef.Name = backup
	tr.Status = &struct {
		State string `json:"state,omitempty"`
	}{State: "Succeeded"}
	return tr
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

func TestRestoreList_HappyPath(t *testing.T) {
	t.Parallel()

	items := []testRestore{
		newTestRestore("my-mongo-restore-1", "everest", "my-mongo", "my-mongo-20260721", 25*time.Hour),
		newTestRestore("my-mongo-restore-2", "everest", "my-mongo", "my-mongo-20260722", time.Hour),
	}

	var requestPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"items": items})
	}))
	defer srv.Close()

	cfg := newTestConfig(srv.URL)
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, cfg.Save(cfgPath))

	runner := NewListRunner(Config{Pretty: true}, zap.NewNop().Sugar())
	err := runner.Run(t.Context(), ListOptions{Cluster: "main", Namespace: "everest", Instance: "my-mongo"}, cfgPath)
	require.NoError(t, err)
	assert.Contains(t, requestPath, "/namespaces/everest/instances/my-mongo/restores")
}

func TestRestoreList_EmptyResult(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"items": []testRestore{}})
	}))
	defer srv.Close()

	cfg := newTestConfig(srv.URL)
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, cfg.Save(cfgPath))

	runner := NewListRunner(Config{Pretty: true}, zap.NewNop().Sugar())
	err := runner.Run(t.Context(), ListOptions{Cluster: "main", Namespace: "everest", Instance: "my-mongo"}, cfgPath)
	require.NoError(t, err)
}

func TestRestoreList_ServerError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	cfg := newTestConfig(srv.URL)
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, cfg.Save(cfgPath))

	runner := NewListRunner(Config{Pretty: true}, zap.NewNop().Sugar())
	err := runner.Run(t.Context(), ListOptions{Cluster: "main", Namespace: "everest", Instance: "my-mongo"}, cfgPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected response")
}

func TestRestoreList_NoActiveContext(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		APIVersion: "config.openeverest.io/v1alpha1",
		Kind:       "ClientConfig",
	}
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, cfg.Save(cfgPath))

	runner := NewListRunner(Config{Pretty: true}, zap.NewNop().Sugar())
	err := runner.Run(t.Context(), ListOptions{Cluster: "main", Namespace: "everest", Instance: "my-mongo"}, cfgPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no active context")
}

func TestRestoreList_JSONOutput(t *testing.T) {
	t.Parallel()

	items := []testRestore{newTestRestore("my-mongo-restore-1", "everest", "my-mongo", "my-mongo-20260722", time.Hour)}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"items": items})
	}))
	defer srv.Close()

	cfg := newTestConfig(srv.URL)
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, cfg.Save(cfgPath))

	// Pretty=false → JSON path
	runner := NewListRunner(Config{Pretty: false}, zap.NewNop().Sugar())
	err := runner.Run(t.Context(), ListOptions{Cluster: "main", Namespace: "everest", Instance: "my-mongo"}, cfgPath)
	require.NoError(t, err)
}

func restoreWithAge(name string, age time.Duration) client.Restore {
	var r client.Restore
	r.Metadata = &map[string]any{
		"name":              name,
		"creationTimestamp": time.Now().Add(-age).UTC().Format(time.RFC3339),
	}
	return r
}

func restoreWithoutTimestamp(name string) client.Restore {
	var r client.Restore
	r.Metadata = &map[string]any{"name": name}
	return r
}

func TestSortRestoresByRecency(t *testing.T) {
	t.Parallel()

	restores := []client.Restore{
		restoreWithAge("oldest", 48*time.Hour),
		restoreWithAge("newest", time.Hour),
		restoreWithoutTimestamp("no-timestamp"),
		restoreWithAge("middle", 24*time.Hour),
	}

	sortRestoresByRecency(restores)

	names := make([]string, len(restores))
	for i, r := range restores {
		names[i] = restoreName(&r)
	}
	assert.Equal(t, []string{"newest", "middle", "oldest", "no-timestamp"}, names)
}

func TestRestoreBackup_NilDataSource(t *testing.T) {
	t.Parallel()

	var r client.Restore
	assert.Equal(t, "-", restoreBackup(&r))
}
