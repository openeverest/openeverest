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

package backupstorage

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/openeverest/openeverest/v2/pkg/cli/config"
)

// testBackupStorage mirrors just the JSON shape the list runner reads off of
// client.BackupStorage, so tests can build fixtures without the generated
// type's large anonymous Spec struct.
type testBackupStorage struct {
	Metadata map[string]any `json:"metadata"`
	Spec     struct {
		Type string `json:"type"`
		S3   *struct {
			Bucket string `json:"bucket"`
			Region string `json:"region"`
		} `json:"s3,omitempty"`
	} `json:"spec"`
}

func newTestBackupStorage(name, namespace string) testBackupStorage {
	tbs := testBackupStorage{
		Metadata: map[string]any{
			"name":              name,
			"namespace":         namespace,
			"creationTimestamp": time.Now().Add(-time.Hour).UTC().Format(time.RFC3339),
		},
	}
	tbs.Spec.Type = "s3"
	tbs.Spec.S3 = &struct {
		Bucket string `json:"bucket"`
		Region string `json:"region"`
	}{Bucket: "my-bucket", Region: "us-east-1"}
	return tbs
}

// newTestConfig returns a config with a single context pointing at serverURL.
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

func TestBackupStorageList_HappyPath(t *testing.T) {
	t.Parallel()

	items := []testBackupStorage{
		newTestBackupStorage("s3-primary", "everest"),
		newTestBackupStorage("s3-secondary", "everest"),
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
	err := runner.Run(t.Context(), ListOptions{Cluster: "main", Namespace: "everest"}, cfgPath)
	require.NoError(t, err)
	assert.Contains(t, requestPath, "/namespaces/everest/backup-storages")
}

func TestBackupStorageList_AllNamespaces(t *testing.T) {
	t.Parallel()

	nsBackupStorages := map[string][]testBackupStorage{
		"everest":  {newTestBackupStorage("s3-primary", "everest")},
		"everest2": {newTestBackupStorage("s3-secondary", "everest2")},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if strings.HasSuffix(r.URL.Path, "/namespaces") {
			_ = json.NewEncoder(w).Encode([]string{"everest", "everest2"})
			return
		}

		for ns, items := range nsBackupStorages {
			if strings.Contains(r.URL.Path, "/namespaces/"+ns+"/backup-storages") {
				_ = json.NewEncoder(w).Encode(map[string]any{"items": items})
				return
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"items": []testBackupStorage{}})
	}))
	defer srv.Close()

	cfg := newTestConfig(srv.URL)
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, cfg.Save(cfgPath))

	runner := NewListRunner(Config{Pretty: true}, zap.NewNop().Sugar())
	err := runner.Run(t.Context(), ListOptions{
		AllNamespaces: true,
		Cluster:       "main",
	}, cfgPath)
	require.NoError(t, err)
}

func TestBackupStorageList_EmptyResult(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"items": []testBackupStorage{}})
	}))
	defer srv.Close()

	cfg := newTestConfig(srv.URL)
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, cfg.Save(cfgPath))

	runner := NewListRunner(Config{Pretty: true}, zap.NewNop().Sugar())
	err := runner.Run(t.Context(), ListOptions{Cluster: "main", Namespace: "everest"}, cfgPath)
	require.NoError(t, err)
}

func TestBackupStorageList_ServerError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	cfg := newTestConfig(srv.URL)
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, cfg.Save(cfgPath))

	runner := NewListRunner(Config{Pretty: true}, zap.NewNop().Sugar())
	err := runner.Run(t.Context(), ListOptions{Cluster: "main", Namespace: "everest"}, cfgPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected response")
}

func TestBackupStorageList_NoActiveContext(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		APIVersion: "config.openeverest.io/v1alpha1",
		Kind:       "ClientConfig",
	}
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, cfg.Save(cfgPath))

	runner := NewListRunner(Config{Pretty: true}, zap.NewNop().Sugar())
	err := runner.Run(t.Context(), ListOptions{Cluster: "main", Namespace: "everest"}, cfgPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no active context")
}

func TestBackupStorageList_JSONOutput(t *testing.T) {
	t.Parallel()

	items := []testBackupStorage{newTestBackupStorage("s3-primary", "everest")}

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
	err := runner.Run(t.Context(), ListOptions{Cluster: "main", Namespace: "everest"}, cfgPath)
	require.NoError(t, err)
}
