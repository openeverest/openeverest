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
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/openeverest/openeverest/v2/client"
	"github.com/openeverest/openeverest/v2/pkg/cli/config"
)

// newTestBackupStorage builds the JSON of a BackupStorage, so tests need not
// spell the generated type's anonymous Spec structs.
func newTestBackupStorage(name, namespace string) map[string]any {
	return map[string]any{
		"metadata": map[string]any{
			"name":              name,
			"namespace":         namespace,
			"creationTimestamp": time.Now().Add(-time.Hour).UTC().Format(time.RFC3339),
		},
		"spec": map[string]any{
			"backupStorageClassRef": map[string]any{"name": "s3"},
			"location":              map[string]any{"bucket": "my-bucket"},
			"authentication": map[string]any{
				"mode":                 "AccessKey",
				"credentialsSecretRef": map[string]any{"name": name + "-credentials"},
			},
		},
	}
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

	items := []map[string]any{
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

	nsBackupStorages := map[string][]map[string]any{
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
		_ = json.NewEncoder(w).Encode(map[string]any{"items": []map[string]any{}})
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
		_ = json.NewEncoder(w).Encode(map[string]any{"items": []map[string]any{}})
	}))
	defer srv.Close()

	cfg := newTestConfig(srv.URL)
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, cfg.Save(cfgPath))

	runner := NewListRunner(Config{Pretty: true}, zap.NewNop().Sugar())
	err := runner.Run(t.Context(), ListOptions{Cluster: "main", Namespace: "everest"}, cfgPath)
	require.NoError(t, err)
}

func TestBackupStorageList_ErrorResponses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		status     int
		message    string
		withBody   bool
		wantSubstr string
	}{
		{
			name:       "400",
			status:     http.StatusBadRequest,
			message:    "invalid namespace",
			withBody:   true,
			wantSubstr: "server error: invalid namespace",
		},
		{
			name:       "500 with body",
			status:     http.StatusInternalServerError,
			message:    "database unavailable",
			withBody:   true,
			wantSubstr: "server error: database unavailable",
		},
		{
			name:       "403 via default",
			status:     http.StatusForbidden,
			message:    "access denied",
			withBody:   true,
			wantSubstr: "server error: access denied",
		},
		{
			name:       "500 without body",
			status:     http.StatusInternalServerError,
			withBody:   false,
			wantSubstr: "unexpected response listing backup storages",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if tt.withBody {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(tt.status)
					_ = json.NewEncoder(w).Encode(map[string]string{"message": tt.message})
					return
				}
				w.WriteHeader(tt.status)
			}))
			defer srv.Close()

			c, err := client.NewClientWithResponses(srv.URL)
			require.NoError(t, err)

			_, err = NewListRunner(Config{}, zap.NewNop().Sugar()).listBackupStorages(
				t.Context(), c, "main", "everest", nil,
			)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantSubstr)
		})
	}
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

	items := []map[string]any{newTestBackupStorage("s3-primary", "everest")}

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

func TestBackupStorageList_Filters(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		opts      ListOptions
		wantQuery url.Values
	}{
		{
			name:      "no filters",
			opts:      ListOptions{Cluster: "main", Namespace: "everest"},
			wantQuery: url.Values{},
		},
		{
			name: "backup class and backup storage class",
			opts: ListOptions{Cluster: "main", Namespace: "everest", BackupClass: "pbm", Class: "s3"},
			wantQuery: url.Values{
				"backupClass":        []string{"pbm"},
				"backupStorageClass": []string{"s3"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			queries := make(chan url.Values, 1)
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				queries <- r.URL.Query()
				writeJSON(w, http.StatusOK, map[string]any{"items": []map[string]any{}})
			}))
			defer srv.Close()

			runner := NewListRunner(Config{Pretty: true}, zap.NewNop().Sugar())
			require.NoError(t, runner.Run(t.Context(), tt.opts, newConfigPath(t, srv.URL)))
			assert.Equal(t, tt.wantQuery, <-queries)
		})
	}
}

func TestPrintBackupStorageTable(t *testing.T) {
	t.Parallel()

	created := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)
	storages := []client.BackupStorage{
		*backupStorageFromJSON(t, `{"metadata":{"name":"s3-prod","namespace":"team-a","creationTimestamp":"`+created+`"},`+
			`"spec":{"backupStorageClassRef":{"name":"s3"},"location":{"bucket":"my-backups","prefix":"team-a/prod"},`+
			`"authentication":{"mode":"AccessKey"}},`+
			`"status":{"conditions":[{"type":"Accepted","status":"True","reason":"Accepted","message":"","lastTransitionTime":"`+created+`"}]}}`),
		*backupStorageFromJSON(t, `{"metadata":{"name":"s3-bad","namespace":"team-a","creationTimestamp":"`+created+`"},`+
			`"spec":{"backupStorageClassRef":{"name":"s3"},"location":{"bucket":"other"},"authentication":{"mode":"AccessKey"}},`+
			`"status":{"conditions":[`+
			`{"type":"Deleting","status":"True","reason":"WaitingForReferrers","message":"","lastTransitionTime":"`+created+`"},`+
			`{"type":"Accepted","status":"False","reason":"CredentialsNotFound","message":"","lastTransitionTime":"`+created+`"}]}}`),
		*backupStorageFromJSON(t, `{"metadata":{"name":"new","namespace":"team-b"},`+
			`"spec":{"backupStorageClassRef":{"name":"s3"},"location":{"bucket":"fresh"},"authentication":{"mode":""}}}`),
	}

	var buf bytes.Buffer
	printBackupStorageTable(&buf, storages)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	require.Len(t, lines, 4)
	assert.Equal(t, []string{"NAME", "NAMESPACE", "CLASS", "BUCKET", "AUTH", "ACCEPTED", "AGE"}, strings.Fields(lines[0]))
	assert.Equal(t, []string{"s3-prod", "team-a", "s3", "my-backups/team-a/prod", "AccessKey", "True", "1h"}, strings.Fields(lines[1]))
	assert.Equal(t, []string{"s3-bad", "team-a", "s3", "other", "AccessKey", "False", "1h"}, strings.Fields(lines[2]))
	assert.Equal(t, []string{"new", "team-b", "s3", "fresh", "-", "Unknown", "-"}, strings.Fields(lines[3]))
}
