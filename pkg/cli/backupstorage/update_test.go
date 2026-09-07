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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func baseUpdateOptions() UpdateOptions {
	return UpdateOptions{Name: "my-s3", Namespace: "everest", Cluster: "main"}
}

func TestUpdateRun_NoFieldFlags_ReturnsErrorWithoutCallingServer(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/clusters/main/namespaces/everest/backup-storages/my-s3", func(w http.ResponseWriter, _ *http.Request) {
		assert.Fail(t, "PATCH should not have been called")
		w.WriteHeader(http.StatusInternalServerError)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	u := NewUpdater(Config{Pretty: true}, zap.NewNop().Sugar())
	err := u.Run(context.Background(), baseUpdateOptions(), newConfigPath(t, srv.URL))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one of --access-key-id, --credentials-secret, --verify-tls, or --force-path-style is required")
}

func TestUpdateRun_VerifyTLSOnly_PatchBodyNamesOnlyThatField(t *testing.T) {
	t.Parallel()

	var gotPatch map[string]any
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/clusters/main/namespaces/everest/backup-storages/my-s3", func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "application/merge-patch+json", r.Header.Get("Content-Type"))
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotPatch))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(backupStorageWithName(t, "my-s3", "everest"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	opts := baseUpdateOptions()
	verifyTLS := false
	opts.VerifyTLS = &verifyTLS

	u := NewUpdater(Config{Pretty: true}, zap.NewNop().Sugar())
	err := u.Run(context.Background(), opts, newConfigPath(t, srv.URL))

	require.NoError(t, err)
	spec, _ := gotPatch["spec"].(map[string]any)
	s3, _ := spec["s3"].(map[string]any)
	require.NotNil(t, s3)
	assert.Equal(t, map[string]any{"verifyTLS": false}, s3)
}

func TestUpdateRun_AccessKeyID_SendsCredentialPair(t *testing.T) {
	t.Parallel()

	var gotPatch map[string]any
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/clusters/main/namespaces/everest/backup-storages/my-s3", func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotPatch))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(backupStorageWithName(t, "my-s3", "everest"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	opts := baseUpdateOptions()
	opts.AccessKeyID = "AKIA123"
	opts.SecretAccessKey = "shh"

	u := NewUpdater(Config{Pretty: true}, zap.NewNop().Sugar())
	err := u.Run(context.Background(), opts, newConfigPath(t, srv.URL))

	require.NoError(t, err)
	spec, _ := gotPatch["spec"].(map[string]any)
	s3, _ := spec["s3"].(map[string]any)
	assert.Equal(t, "AKIA123", s3["accessKeyId"])
	assert.Equal(t, "shh", s3["secretAccessKey"])
}

func TestUpdateRun_CredentialsSecret_SendsSecretRefName(t *testing.T) {
	t.Parallel()

	var gotPatch map[string]any
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/clusters/main/namespaces/everest/backup-storages/my-s3", func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&gotPatch))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(backupStorageWithName(t, "my-s3", "everest"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	opts := baseUpdateOptions()
	opts.CredentialsSecret = "other-secret"

	u := NewUpdater(Config{Pretty: true}, zap.NewNop().Sugar())
	err := u.Run(context.Background(), opts, newConfigPath(t, srv.URL))

	require.NoError(t, err)
	spec, _ := gotPatch["spec"].(map[string]any)
	s3, _ := spec["s3"].(map[string]any)
	assert.Equal(t, map[string]any{"name": "other-secret"}, s3["credentialsSecretRef"])
	assert.NotContains(t, s3, "accessKeyId")
	assert.NotContains(t, s3, "secretAccessKey")
}

func TestUpdateRun_NotFound_ReturnsFriendlyError(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/clusters/main/namespaces/everest/backup-storages/my-s3", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	opts := baseUpdateOptions()
	opts.CredentialsSecret = "other-secret"

	u := NewUpdater(Config{Pretty: true}, zap.NewNop().Sugar())
	err := u.Run(context.Background(), opts, newConfigPath(t, srv.URL))

	require.Error(t, err)
	assert.Contains(t, err.Error(), `backup storage "my-s3" not found in namespace "everest"`)
}

func TestUpdateRun_ServerError_SurfacesMessage(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/clusters/main/namespaces/everest/backup-storages/my-s3", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "half a credential pair"})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	opts := baseUpdateOptions()
	opts.CredentialsSecret = "other-secret"

	u := NewUpdater(Config{Pretty: true}, zap.NewNop().Sugar())
	err := u.Run(context.Background(), opts, newConfigPath(t, srv.URL))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "half a credential pair")
}
