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

package backup

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

type testBackup struct {
	Metadata map[string]any `json:"metadata"`
	Spec     struct {
		InstanceRef struct {
			Name string `json:"name"`
		} `json:"instanceRef"`
		StorageRef struct {
			Name string `json:"name"`
		} `json:"storageRef"`
		ScheduleName string `json:"scheduleName,omitempty"`
	} `json:"spec"`
	Status *struct {
		State string `json:"state,omitempty"`
		Size  string `json:"size,omitempty"`
	} `json:"status,omitempty"`
}

func newTestBackup(name, namespace, instance, storage string, age time.Duration) testBackup {
	tb := testBackup{
		Metadata: map[string]any{
			"name":              name,
			"namespace":         namespace,
			"creationTimestamp": time.Now().Add(-age).UTC().Format(time.RFC3339),
		},
	}
	tb.Spec.InstanceRef.Name = instance
	tb.Spec.StorageRef.Name = storage
	tb.Status = &struct {
		State string `json:"state,omitempty"`
		Size  string `json:"size,omitempty"`
	}{State: "Succeeded", Size: "128Mi"}
	return tb
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

func TestBackupList_HappyPath(t *testing.T) {
	t.Parallel()

	items := []testBackup{
		newTestBackup("my-mongo-20260721", "everest", "my-mongo", "s3-primary", 25*time.Hour),
		newTestBackup("my-mongo-20260722", "everest", "my-mongo", "s3-primary", time.Hour),
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
	assert.Contains(t, requestPath, "/namespaces/everest/instances/my-mongo/backups")
}

func TestBackupList_EmptyResult(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"items": []testBackup{}})
	}))
	defer srv.Close()

	cfg := newTestConfig(srv.URL)
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, cfg.Save(cfgPath))

	runner := NewListRunner(Config{Pretty: true}, zap.NewNop().Sugar())
	err := runner.Run(t.Context(), ListOptions{Cluster: "main", Namespace: "everest", Instance: "my-mongo"}, cfgPath)
	require.NoError(t, err)
}

func TestBackupList_ServerError(t *testing.T) {
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

func TestBackupList_NoActiveContext(t *testing.T) {
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

func TestBackupList_JSONOutput(t *testing.T) {
	t.Parallel()

	items := []testBackup{newTestBackup("my-mongo-20260722", "everest", "my-mongo", "s3-primary", time.Hour)}

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

func backupWithAge(name string, age time.Duration) client.Backup {
	var b client.Backup
	b.Metadata = &map[string]any{
		"name":              name,
		"creationTimestamp": time.Now().Add(-age).UTC().Format(time.RFC3339),
	}
	return b
}

func backupWithoutTimestamp(name string) client.Backup {
	var b client.Backup
	b.Metadata = &map[string]any{"name": name}
	return b
}

func TestSortBackupsByRecency(t *testing.T) {
	t.Parallel()

	backups := []client.Backup{
		backupWithAge("oldest", 48*time.Hour),
		backupWithAge("newest", time.Hour),
		backupWithoutTimestamp("no-timestamp"),
		backupWithAge("middle", 24*time.Hour),
	}

	sortBackupsByRecency(backups)

	names := make([]string, len(backups))
	for i, b := range backups {
		names[i] = backupName(&b)
	}
	assert.Equal(t, []string{"newest", "middle", "oldest", "no-timestamp"}, names)
}
