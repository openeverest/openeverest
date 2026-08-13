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
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

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

// backupStorageFromJSON builds a BackupStorage fixture via JSON, so tests stay valid
// across schema changes instead of hand-spelling the generated anonymous Spec.S3 struct.
func backupStorageFromJSON(t *testing.T, body string) *client.BackupStorage {
	t.Helper()
	var bs client.BackupStorage
	require.NoError(t, json.Unmarshal([]byte(body), &bs))
	return &bs
}

func backupStorageWithName(t *testing.T, name, namespace string) *client.BackupStorage {
	t.Helper()
	body := fmt.Sprintf(
		`{"metadata":{"name":%q,"namespace":%q},"spec":{"type":"s3","s3":{"bucket":"my-bucket","region":"us-east-1","endpointURL":"https://s3.amazonaws.com","credentialsSecretRef":{"name":%q}}}}`,
		name, namespace, name+"-credentials",
	)
	return backupStorageFromJSON(t, body)
}

func baseCreateOptions() CreateOptions {
	return CreateOptions{
		Name:        "my-s3",
		Namespace:   "everest",
		Type:        "s3",
		Bucket:      "my-bucket",
		Region:      "us-east-1",
		EndpointURL: "https://s3.amazonaws.com",
		Cluster:     "main",
	}
}

func TestRun_HappyPath_AccessKeyID(t *testing.T) {
	t.Parallel()

	var gotBody client.BackupStorage
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/clusters/main/namespaces/everest/backup-storages", func(w http.ResponseWriter, r *http.Request) {
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(backupStorageWithName(t, "my-s3", "everest"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	opts := baseCreateOptions()
	opts.AccessKeyID = "AKIA123"
	opts.SecretAccessKey = "shh"

	cr := NewCreateRunner(Config{Pretty: true}, zap.NewNop().Sugar())
	err := cr.Run(context.Background(), opts, newConfigPath(t, srv.URL))

	require.NoError(t, err)
	require.NotNil(t, gotBody.Spec.S3)
	require.NotNil(t, gotBody.Spec.S3.AccessKeyId)
	assert.Equal(t, "AKIA123", *gotBody.Spec.S3.AccessKeyId)
	require.NotNil(t, gotBody.Spec.S3.SecretAccessKey)
	assert.Equal(t, "shh", *gotBody.Spec.S3.SecretAccessKey)
	assert.Equal(t, "backup-storage-my-s3-credentials", gotBody.Spec.S3.CredentialsSecretRef.Name)
}

func TestRun_HappyPath_CredentialsSecret(t *testing.T) {
	t.Parallel()

	var gotBody client.BackupStorage
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/clusters/main/namespaces/everest/backup-storages", func(w http.ResponseWriter, r *http.Request) {
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&gotBody))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(backupStorageWithName(t, "my-s3", "everest"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	opts := baseCreateOptions()
	opts.CredentialsSecret = "my-s3-creds"

	cr := NewCreateRunner(Config{Pretty: true}, zap.NewNop().Sugar())
	err := cr.Run(context.Background(), opts, newConfigPath(t, srv.URL))

	require.NoError(t, err)
	require.NotNil(t, gotBody.Spec.S3)
	assert.Equal(t, "my-s3-creds", gotBody.Spec.S3.CredentialsSecretRef.Name)
	assert.Nil(t, gotBody.Spec.S3.AccessKeyId)
	assert.Nil(t, gotBody.Spec.S3.SecretAccessKey)
}

func TestRun_Conflict_ReturnsFriendlyError(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/clusters/main/namespaces/everest/backup-storages", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	opts := baseCreateOptions()
	opts.CredentialsSecret = "my-s3-creds"

	cr := NewCreateRunner(Config{Pretty: true}, zap.NewNop().Sugar())
	err := cr.Run(context.Background(), opts, newConfigPath(t, srv.URL))

	require.Error(t, err)
	assert.Contains(t, err.Error(), `backup storage "my-s3" already exists in namespace "everest"`)
}

func TestRun_ValidationError(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/clusters/main/namespaces/everest/backup-storages", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "endpointURL must be a valid URL"})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	opts := baseCreateOptions()
	opts.CredentialsSecret = "my-s3-creds"

	cr := NewCreateRunner(Config{Pretty: true}, zap.NewNop().Sugar())
	err := cr.Run(context.Background(), opts, newConfigPath(t, srv.URL))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "endpointURL must be a valid URL")
}

func TestRun_ServerError(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/clusters/main/namespaces/everest/backup-storages", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "etcd unavailable"})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	opts := baseCreateOptions()
	opts.CredentialsSecret = "my-s3-creds"

	cr := NewCreateRunner(Config{Pretty: true}, zap.NewNop().Sugar())
	err := cr.Run(context.Background(), opts, newConfigPath(t, srv.URL))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "etcd unavailable")
}

func TestRun_JSONErrorsOnEmptyCreateBody(t *testing.T) {
	t.Parallel()

	// 201 with no parseable body must error rather than emit empty stdout.
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/clusters/main/namespaces/everest/backup-storages", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	opts := baseCreateOptions()
	opts.CredentialsSecret = "my-s3-creds"

	cr := NewCreateRunner(Config{Pretty: false}, zap.NewNop().Sugar())
	err := cr.Run(context.Background(), opts, newConfigPath(t, srv.URL))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected response creating backup storage")
}

func TestRun_UndeclaredStatus_SurfacesDefaultMessage(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/clusters/main/namespaces/everest/backup-storages", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "namespace not found"})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	opts := baseCreateOptions()
	opts.CredentialsSecret = "my-s3-creds"

	cr := NewCreateRunner(Config{Pretty: true}, zap.NewNop().Sugar())
	err := cr.Run(context.Background(), opts, newConfigPath(t, srv.URL))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "namespace not found")
}

func TestRun_UndeclaredStatus_NonJSONBody_FallsBackToStatusLine(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/clusters/main/namespaces/everest/backup-storages", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	opts := baseCreateOptions()
	opts.CredentialsSecret = "my-s3-creds"

	cr := NewCreateRunner(Config{Pretty: true}, zap.NewNop().Sugar())
	err := cr.Run(context.Background(), opts, newConfigPath(t, srv.URL))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected response creating backup storage")
}

func TestRun_UnsupportedType(t *testing.T) {
	t.Parallel()

	opts := baseCreateOptions()
	opts.Type = "gcs"
	opts.CredentialsSecret = "my-s3-creds"

	cr := NewCreateRunner(Config{Pretty: true}, zap.NewNop().Sugar())
	err := cr.Run(context.Background(), opts, newConfigPath(t, "http://unused.invalid"))

	require.Error(t, err)
	assert.Contains(t, err.Error(), `unsupported storage type "gcs"`)
}
