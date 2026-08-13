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
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/openeverest/openeverest/v2/client"
)

func newConfigPath(t *testing.T, serverURL string) string {
	t.Helper()
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, newTestConfig(serverURL).Save(cfgPath))
	return cfgPath
}

// backupWithState builds a Backup fixture via JSON (see backupFromJSON in
// wait_test.go), so it stays valid across Backup schema changes instead of
// hand-spelling the generated anonymous Status struct.
func backupWithState(t *testing.T, name, state string) *client.Backup {
	t.Helper()
	statusField := ""
	if state != "" {
		statusField = fmt.Sprintf(`,"status":{"state":%q}`, state)
	}
	body := fmt.Sprintf(
		`{"metadata":{"name":%q,"namespace":"everest"},"spec":{"instanceRef":{"name":"my-mongo"},"classRef":{"name":"psmdb-backup"},"storageRef":{"name":"my-s3"}}%s}`,
		name, statusField,
	)
	return backupFromJSON(t, body)
}

func TestRun_HappyPath_GeneratesName(t *testing.T) {
	t.Parallel()

	var gotBody client.Backup
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/clusters/main/namespaces/everest/backups", func(w http.ResponseWriter, r *http.Request) {
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(backupWithState(t, "my-mongo-abcde", ""))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	cr := NewCreateRunner(Config{Pretty: true}, zap.NewNop().Sugar())
	err := cr.Run(context.Background(), CreateOptions{
		Instance:  "my-mongo",
		Namespace: "everest",
		Class:     "psmdb-backup",
		Storage:   "my-s3",
		Cluster:   "main",
	}, newConfigPath(t, srv.URL))

	require.NoError(t, err)
	require.NotNil(t, gotBody.Metadata)
	assert.Equal(t, "my-mongo-", gotBody.Metadata.GenerateName)
	assert.Empty(t, gotBody.Metadata.Name)
	assert.Equal(t, "psmdb-backup", gotBody.Spec.ClassRef.Name)
	assert.Equal(t, "my-s3", gotBody.Spec.StorageRef.Name)
}

func TestRun_ExplicitName(t *testing.T) {
	t.Parallel()

	var gotBody client.Backup
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/clusters/main/namespaces/everest/backups", func(w http.ResponseWriter, r *http.Request) {
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(backupWithState(t, "pre-upgrade", ""))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	cr := NewCreateRunner(Config{Pretty: true}, zap.NewNop().Sugar())
	err := cr.Run(context.Background(), CreateOptions{
		Instance:       "my-mongo",
		Namespace:      "everest",
		Class:          "psmdb-backup",
		Storage:        "my-s3",
		Name:           "pre-upgrade",
		DeletionPolicy: "Retain",
		Cluster:        "main",
	}, newConfigPath(t, srv.URL))

	require.NoError(t, err)
	require.NotNil(t, gotBody.Metadata)
	assert.Equal(t, "pre-upgrade", gotBody.Metadata.Name)
	assert.Equal(t, "Retain", gotBody.Spec.DeletionPolicy)
}

func TestRun_Conflict_ReturnsFriendlyError(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/clusters/main/namespaces/everest/backups", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	cr := NewCreateRunner(Config{Pretty: true}, zap.NewNop().Sugar())
	err := cr.Run(context.Background(), CreateOptions{
		Instance:  "my-mongo",
		Namespace: "everest",
		Class:     "psmdb-backup",
		Storage:   "my-s3",
		Name:      "pre-upgrade",
		Cluster:   "main",
	}, newConfigPath(t, srv.URL))

	require.Error(t, err)
	assert.Contains(t, err.Error(), `backup "pre-upgrade" already exists in namespace "everest"`)
}

func TestRun_Wait_SucceedsOnTerminalState(t *testing.T) {
	t.Parallel()

	var getCalls int
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/clusters/main/namespaces/everest/backups", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(backupWithState(t, "my-mongo-abcde", ""))
	})
	mux.HandleFunc("/v1/clusters/main/namespaces/everest/backups/my-mongo-abcde", func(w http.ResponseWriter, _ *http.Request) {
		getCalls++
		state := "Running"
		if getCalls >= 2 {
			state = "Succeeded"
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(backupWithState(t, "my-mongo-abcde", state))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	cr := NewCreateRunner(Config{Pretty: true}, zap.NewNop().Sugar())
	err := cr.Run(context.Background(), CreateOptions{
		Instance:  "my-mongo",
		Namespace: "everest",
		Class:     "psmdb-backup",
		Storage:   "my-s3",
		Cluster:   "main",
		Wait:      true,
		Timeout:   10 * time.Second,
	}, newConfigPath(t, srv.URL))

	require.NoError(t, err)
	assert.GreaterOrEqual(t, getCalls, 2)
}

func TestRun_Wait_FailsOnFailedState(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/clusters/main/namespaces/everest/backups", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(backupWithState(t, "my-mongo-abcde", ""))
	})
	mux.HandleFunc("/v1/clusters/main/namespaces/everest/backups/my-mongo-abcde", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		b := backupWithState(t, "my-mongo-abcde", "Failed")
		b.Status.Message = new("engine returned a non-zero exit code")
		_ = json.NewEncoder(w).Encode(b)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	cr := NewCreateRunner(Config{Pretty: true}, zap.NewNop().Sugar())
	err := cr.Run(context.Background(), CreateOptions{
		Instance:  "my-mongo",
		Namespace: "everest",
		Class:     "psmdb-backup",
		Storage:   "my-s3",
		Cluster:   "main",
		Wait:      true,
		Timeout:   10 * time.Second,
	}, newConfigPath(t, srv.URL))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "engine returned a non-zero exit code")
}

func TestRun_Wait_RetriesTransientFetchError(t *testing.T) {
	t.Parallel()

	// First poll returns 500 (transient), second returns Succeeded — the wait
	// must retry rather than fail on the transient status.
	var getCalls int
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/clusters/main/namespaces/everest/backups", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(backupWithState(t, "my-mongo-abcde", ""))
	})
	mux.HandleFunc("/v1/clusters/main/namespaces/everest/backups/my-mongo-abcde", func(w http.ResponseWriter, _ *http.Request) {
		getCalls++
		if getCalls == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(backupWithState(t, "my-mongo-abcde", "Succeeded"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	cr := NewCreateRunner(Config{Pretty: true}, zap.NewNop().Sugar())
	err := cr.Run(context.Background(), CreateOptions{
		Instance:  "my-mongo",
		Namespace: "everest",
		Class:     "psmdb-backup",
		Storage:   "my-s3",
		Cluster:   "main",
		Wait:      true,
		Timeout:   10 * time.Second,
	}, newConfigPath(t, srv.URL))

	require.NoError(t, err)
	assert.GreaterOrEqual(t, getCalls, 2)
}

func TestRun_JSONErrorsOnEmptyCreateBody(t *testing.T) {
	t.Parallel()

	// 201 with no parseable body must error rather than emit empty stdout.
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/clusters/main/namespaces/everest/backups", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	cr := NewCreateRunner(Config{Pretty: false}, zap.NewNop().Sugar())
	err := cr.Run(context.Background(), CreateOptions{
		Instance:  "my-mongo",
		Namespace: "everest",
		Class:     "psmdb-backup",
		Storage:   "my-s3",
		Cluster:   "main",
	}, newConfigPath(t, srv.URL))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected response creating backup")
}

func TestRun_Wait_TimesOut(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/clusters/main/namespaces/everest/backups", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(backupWithState(t, "my-mongo-abcde", ""))
	})
	mux.HandleFunc("/v1/clusters/main/namespaces/everest/backups/my-mongo-abcde", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(backupWithState(t, "my-mongo-abcde", "Running"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	cr := NewCreateRunner(Config{Pretty: true}, zap.NewNop().Sugar())
	err := cr.Run(context.Background(), CreateOptions{
		Instance:  "my-mongo",
		Namespace: "everest",
		Class:     "psmdb-backup",
		Storage:   "my-s3",
		Cluster:   "main",
		Wait:      true,
		Timeout:   3 * time.Second,
	}, newConfigPath(t, srv.URL))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "timed out")
}
