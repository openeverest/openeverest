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

func newConfigPath(t *testing.T, serverURL string) string {
	t.Helper()
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, newTestConfig(serverURL).Save(cfgPath))
	return cfgPath
}

func backupWithState(name, state string) *client.Backup {
	b := &client.Backup{
		Metadata: &map[string]interface{}{"name": name, "namespace": "everest"},
	}
	b.Spec.InstanceName = "my-mongo"
	b.Spec.BackupClassName = "psmdb-backup"
	b.Spec.StorageName = "my-s3"
	if state != "" {
		b.Status = &struct {
			CompletedAt *time.Time `json:"completedAt,omitempty"`
			Conditions  *[]struct {
				LastTransitionTime time.Time                           `json:"lastTransitionTime"`
				Message            string                              `json:"message"`
				ObservedGeneration *int64                              `json:"observedGeneration,omitempty"`
				Reason             string                              `json:"reason"`
				Status             client.BackupStatusConditionsStatus `json:"status"`
				Type               string                              `json:"type"`
			} `json:"conditions,omitempty"`
			ExecutionMode          *client.BackupStatusExecutionMode `json:"executionMode,omitempty"`
			JobName                *string                           `json:"jobName,omitempty"`
			LastObservedGeneration *int64                            `json:"lastObservedGeneration,omitempty"`
			Message                *string                           `json:"message,omitempty"`
			OperatorBackupRef      *struct {
				ApiGroup *string `json:"apiGroup,omitempty"`
				Kind     string  `json:"kind"`
				Name     string  `json:"name"`
			} `json:"operatorBackupRef,omitempty"`
			Size      *string    `json:"size,omitempty"`
			StartedAt *time.Time `json:"startedAt,omitempty"`
			State     *string    `json:"state,omitempty"`
		}{State: &state}
	}
	return b
}

func TestRun_HappyPath_GeneratesName(t *testing.T) {
	t.Parallel()

	var gotBody client.Backup
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/clusters/main/namespaces/everest/backups", func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(backupWithState("my-mongo-abcde", ""))
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
	assert.Equal(t, "my-mongo-", (*gotBody.Metadata)["generateName"])
	assert.Nil(t, (*gotBody.Metadata)["name"])
	assert.Equal(t, "psmdb-backup", gotBody.Spec.BackupClassName)
	assert.Equal(t, "my-s3", gotBody.Spec.StorageName)
}

func TestRun_ExplicitName(t *testing.T) {
	t.Parallel()

	var gotBody client.Backup
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/clusters/main/namespaces/everest/backups", func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(backupWithState("pre-upgrade", ""))
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
	assert.Equal(t, "pre-upgrade", (*gotBody.Metadata)["name"])
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
		_ = json.NewEncoder(w).Encode(backupWithState("my-mongo-abcde", ""))
	})
	mux.HandleFunc("/v1/clusters/main/namespaces/everest/backups/my-mongo-abcde", func(w http.ResponseWriter, _ *http.Request) {
		getCalls++
		state := "Running"
		if getCalls >= 2 {
			state = "Succeeded"
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(backupWithState("my-mongo-abcde", state))
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
		_ = json.NewEncoder(w).Encode(backupWithState("my-mongo-abcde", ""))
	})
	mux.HandleFunc("/v1/clusters/main/namespaces/everest/backups/my-mongo-abcde", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		b := backupWithState("my-mongo-abcde", "Failed")
		b.Status.Message = strPtr("engine returned a non-zero exit code")
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

func TestRun_Wait_TimesOut(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/clusters/main/namespaces/everest/backups", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(backupWithState("my-mongo-abcde", ""))
	})
	mux.HandleFunc("/v1/clusters/main/namespaces/everest/backups/my-mongo-abcde", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(backupWithState("my-mongo-abcde", "Running"))
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

func strPtr(s string) *string { return &s }
