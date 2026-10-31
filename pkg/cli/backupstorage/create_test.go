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
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/openeverest/openeverest/v2/client"
)

const (
	testCredentials = "AWS_ACCESS_KEY_ID=AKIA123\nAWS_SECRET_ACCESS_KEY=shh\n" //nolint:gosec // fake keys for the fake API
	testPEM         = "-----BEGIN CERTIFICATE-----\naGVsbG8=\n-----END CERTIFICATE-----\n"
	s3ClassJSON     = `{"metadata":{"name":"s3"},"spec":{"authModes":[` +
		`{"name":"AccessKey","credentials":{"required":["AWS_ACCESS_KEY_ID","AWS_SECRET_ACCESS_KEY"]}}]}}`
)

func newConfigPath(t *testing.T, serverURL string) string {
	t.Helper()
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, newTestConfig(serverURL).Save(cfgPath))
	return cfgPath
}

// backupStorageFromJSON builds a BackupStorage fixture via JSON, so tests stay valid
// across schema changes instead of hand-spelling the generated anonymous Spec structs.
func backupStorageFromJSON(t *testing.T, body string) *client.BackupStorage {
	t.Helper()
	var bs client.BackupStorage
	require.NoError(t, json.Unmarshal([]byte(body), &bs))
	return &bs
}

func backupStorageWithName(t *testing.T, name, namespace string) *client.BackupStorage {
	t.Helper()
	body := fmt.Sprintf(
		`{"metadata":{"name":%q,"namespace":%q},"spec":{"backupStorageClassRef":{"name":"s3"},"location":{"bucket":"my-bucket"},`+
			`"authentication":{"mode":"AccessKey","credentialsSecretRef":{"name":%q}}}}`,
		name, namespace, "backup-storage-"+name+"-credentials",
	)
	return backupStorageFromJSON(t, body)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v) //nolint:errchkjson // fixtures are plain maps and API types
}

func writeTempFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func baseCreateOptions() CreateOptions {
	return CreateOptions{
		Name:      "my-s3",
		Namespace: "everest",
		Cluster:   "main",
		Class:     "s3",
		Bucket:    "my-bucket",
		AuthMode:  "AccessKey",
		Parameters: ParameterOptions{
			Region:      new("us-east-1"),
			EndpointURL: new("https://s3.amazonaws.com"),
		},
	}
}

// fakeCreateAPI fakes the endpoints create calls, in the order it calls them.
type fakeCreateAPI struct {
	mu sync.Mutex
	// storageResponse answers the BackupStorage POST; nil echoes it with 201.
	storageResponse func(w http.ResponseWriter)
	// secretStatus overrides the 201 answer to a Secret POST, by Secret name.
	secretStatus map[string]int

	calls   []string
	secrets []client.Secret
	storage *client.BackupStorage
}

// newFakeCreateAPI starts the fake; configure runs before the server does.
func newFakeCreateAPI(t *testing.T, configure ...func(*fakeCreateAPI)) (*fakeCreateAPI, string) {
	t.Helper()
	f := &fakeCreateAPI{secretStatus: make(map[string]int)}
	for _, c := range configure {
		c(f)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/clusters/main/backup-storage-classes/{name}", func(w http.ResponseWriter, r *http.Request) {
		if r.PathValue("name") != "s3" {
			writeJSON(w, http.StatusNotFound, map[string]string{"message": "not found"})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(s3ClassJSON))
	})
	mux.HandleFunc("POST /v1/clusters/main/namespaces/everest/secrets", func(w http.ResponseWriter, r *http.Request) {
		var s client.Secret
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&s))
		f.mu.Lock()
		defer f.mu.Unlock()
		f.calls = append(f.calls, "create secret "+s.Metadata.Name)
		if status, ok := f.secretStatus[s.Metadata.Name]; ok {
			writeJSON(w, status, map[string]string{"message": "secret rejected"})
			return
		}
		f.secrets = append(f.secrets, s)
		writeJSON(w, http.StatusCreated, s)
	})
	mux.HandleFunc("DELETE /v1/clusters/main/namespaces/everest/secrets/{name}", func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.calls = append(f.calls, "delete secret "+r.PathValue("name"))
		w.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("POST /v1/clusters/main/namespaces/everest/backup-storages", func(w http.ResponseWriter, r *http.Request) {
		var bs client.BackupStorage
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&bs))
		f.mu.Lock()
		defer f.mu.Unlock()
		f.calls = append(f.calls, "create backup storage")
		f.storage = &bs
		if f.storageResponse != nil {
			f.storageResponse(w)
			return
		}
		writeJSON(w, http.StatusCreated, bs)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return f, newConfigPath(t, srv.URL)
}

func (f *fakeCreateAPI) recorded() ([]string, []client.Secret, *client.BackupStorage) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls, f.secrets, f.storage
}

func runCreate(t *testing.T, pretty bool, opts CreateOptions, cfgPath string) error {
	t.Helper()
	return NewCreateRunner(Config{Pretty: pretty}, zap.NewNop().Sugar()).Run(context.Background(), opts, cfgPath)
}

func TestCreateRun_CredentialsSecret_SendsOnlyTheStorage(t *testing.T) {
	t.Parallel()

	f, cfgPath := newFakeCreateAPI(t)
	opts := baseCreateOptions()
	opts.CredentialsSecret = "my-s3-creds"

	require.NoError(t, runCreate(t, true, opts, cfgPath))

	calls, _, storage := f.recorded()
	assert.Equal(t, []string{"create backup storage"}, calls)
	require.NotNil(t, storage)
	assert.Equal(t, "s3", storage.Spec.BackupStorageClassRef.Name)
	assert.Equal(t, "my-bucket", storage.Spec.Location.Bucket)
	assert.Nil(t, storage.Spec.Location.Prefix)
	require.NotNil(t, storage.Spec.Parameters)
	assert.Equal(t, map[string]any{"region": "us-east-1", "endpointURL": "https://s3.amazonaws.com"}, *storage.Spec.Parameters)
	assert.Equal(t, "AccessKey", storage.Spec.Authentication.Mode)
	require.NotNil(t, storage.Spec.Authentication.CredentialsSecretRef)
	assert.Equal(t, "my-s3-creds", storage.Spec.Authentication.CredentialsSecretRef.Name)
	assert.Nil(t, storage.Spec.Tls)
}

func TestCreateRun_CredentialsAndCAFiles_CreateSecretsBeforeStorage(t *testing.T) {
	t.Parallel()

	f, cfgPath := newFakeCreateAPI(t)
	opts := baseCreateOptions()
	opts.CredentialsFile = writeTempFile(t, "s3.env", "# prod keys\n"+testCredentials)
	opts.CAFile = writeTempFile(t, "ca.pem", testPEM)

	require.NoError(t, runCreate(t, false, opts, cfgPath))

	calls, secrets, storage := f.recorded()
	assert.Equal(t, []string{"create secret backup-storage-my-s3-credentials", "create secret backup-storage-my-s3-ca", "create backup storage"}, calls)
	require.Len(t, secrets, 2)

	creds := secrets[0]
	require.NotNil(t, creds.Metadata)
	assert.Equal(t, "backup-storage-my-s3-credentials", creds.Metadata.Name)
	assert.Equal(t, "everest", creds.Metadata.Namespace)
	require.NotNil(t, creds.Type)
	assert.Equal(t, "Opaque", *creds.Type)
	require.NotNil(t, creds.StringData)
	assert.Equal(t, map[string]string{"AWS_ACCESS_KEY_ID": "AKIA123", "AWS_SECRET_ACCESS_KEY": "shh"}, *creds.StringData)

	ca := secrets[1]
	assert.Equal(t, "backup-storage-my-s3-ca", ca.Metadata.Name)
	require.NotNil(t, ca.StringData)
	assert.Equal(t, map[string]string{"ca.crt": testPEM}, *ca.StringData)

	require.NotNil(t, storage)
	require.NotNil(t, storage.Spec.Authentication.CredentialsSecretRef)
	assert.Equal(t, "backup-storage-my-s3-credentials", storage.Spec.Authentication.CredentialsSecretRef.Name)
	require.NotNil(t, storage.Spec.Tls)
	require.NotNil(t, storage.Spec.Tls.Ca)
	require.NotNil(t, storage.Spec.Tls.Ca.SecretKeyRef)
	assert.Equal(t, "backup-storage-my-s3-ca", storage.Spec.Tls.Ca.SecretKeyRef.Name)
	assert.Equal(t, "ca.crt", storage.Spec.Tls.Ca.SecretKeyRef.Key)
}

func TestCreateRun_StorageNotCreated_DeletesCreatedSecrets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		respond  func(w http.ResponseWriter)
		wantErrs string
	}{
		{
			name: "validation error",
			respond: func(w http.ResponseWriter) {
				writeJSON(w, http.StatusBadRequest, map[string]string{"message": "bucket is invalid"})
			},
			wantErrs: "server error: bucket is invalid",
		},
		{
			name: "conflict",
			respond: func(w http.ResponseWriter) {
				writeJSON(w, http.StatusConflict, map[string]string{"reason": "AlreadyExists"})
			},
			wantErrs: `backup storage "my-s3" already exists in namespace "everest"`,
		},
		{
			name: "server error",
			respond: func(w http.ResponseWriter) {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"message": "etcd unavailable"})
			},
			wantErrs: "server error: etcd unavailable",
		},
		{
			name: "undeclared status, non-JSON body",
			respond: func(w http.ResponseWriter) {
				w.WriteHeader(http.StatusBadGateway)
				_, _ = w.Write([]byte("bad gateway"))
			},
			wantErrs: "unexpected response creating backup storage",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f, cfgPath := newFakeCreateAPI(t, func(f *fakeCreateAPI) { f.storageResponse = tt.respond })
			opts := baseCreateOptions()
			opts.CredentialsFile = writeTempFile(t, "s3.env", testCredentials)
			opts.CAFile = writeTempFile(t, "ca.pem", testPEM)

			err := runCreate(t, true, opts, cfgPath)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErrs)
			calls, _, _ := f.recorded()
			want := []string{
				"create secret backup-storage-my-s3-credentials", "create secret backup-storage-my-s3-ca", "create backup storage",
				"delete secret backup-storage-my-s3-credentials", "delete secret backup-storage-my-s3-ca",
			}
			assert.Equal(t, want, calls)
		})
	}
}

func TestCreateRun_CASecretRejected_DeletesCredentialsSecret(t *testing.T) {
	t.Parallel()

	f, cfgPath := newFakeCreateAPI(t, func(f *fakeCreateAPI) { f.secretStatus["backup-storage-my-s3-ca"] = http.StatusBadRequest })
	opts := baseCreateOptions()
	opts.CredentialsFile = writeTempFile(t, "s3.env", testCredentials)
	opts.CAFile = writeTempFile(t, "ca.pem", testPEM)

	err := runCreate(t, true, opts, cfgPath)

	require.Error(t, err)
	assert.Contains(t, err.Error(), `server error creating secret "backup-storage-my-s3-ca": secret rejected`)
	calls, _, _ := f.recorded()
	assert.Equal(t, []string{"create secret backup-storage-my-s3-credentials", "create secret backup-storage-my-s3-ca", "delete secret backup-storage-my-s3-credentials"}, calls)
}

func TestCreateRun_CredentialsSecretExists_PointsAtCredentialsSecretFlag(t *testing.T) {
	t.Parallel()

	f, cfgPath := newFakeCreateAPI(t, func(f *fakeCreateAPI) { f.secretStatus["backup-storage-my-s3-credentials"] = http.StatusConflict })
	opts := baseCreateOptions()
	opts.CredentialsFile = writeTempFile(t, "s3.env", testCredentials)

	err := runCreate(t, true, opts, cfgPath)

	require.Error(t, err)
	assert.Contains(t, err.Error(), `secret "backup-storage-my-s3-credentials" already exists in namespace "everest": reference it with --credentials-secret`)
	calls, _, _ := f.recorded()
	assert.Equal(t, []string{"create secret backup-storage-my-s3-credentials"}, calls, "a Secret this run did not create must not be deleted")
}

func TestCreateRun_CreatedWithUnreadableBody_KeepsSecrets(t *testing.T) {
	t.Parallel()

	// A 201 means the storage exists, so its Secrets must stay even though
	// the JSON output cannot be produced.
	f, cfgPath := newFakeCreateAPI(t, func(f *fakeCreateAPI) {
		f.storageResponse = func(w http.ResponseWriter) { w.WriteHeader(http.StatusCreated) }
	})
	opts := baseCreateOptions()
	opts.CredentialsFile = writeTempFile(t, "s3.env", testCredentials)

	err := runCreate(t, false, opts, cfgPath)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected response creating backup storage")
	calls, _, _ := f.recorded()
	assert.Equal(t, []string{"create secret backup-storage-my-s3-credentials", "create backup storage"}, calls)
}

func TestCreateRun_StorageClassChecks_FailBeforeAnySecret(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(*CreateOptions)
		envFile string
		wantErr string
	}{
		{
			name:    "unknown class",
			mutate:  func(o *CreateOptions) { o.Class = "gcs" },
			wantErr: `backup storage class "gcs" not found`,
		},
		{
			name:    "auth mode the class does not list",
			mutate:  func(o *CreateOptions) { o.AuthMode = "WorkloadIdentity" },
			wantErr: `backup storage class "s3" has no authentication mode "WorkloadIdentity" (supported: AccessKey)`,
		},
		{
			name:    "credentials file without a required key",
			envFile: "AWS_ACCESS_KEY_ID=AKIA123\n",
			wantErr: "credentials file is missing AWS_SECRET_ACCESS_KEY, required by authentication mode AccessKey",
		},
		{
			name:    "credentials file with an empty required key",
			envFile: "AWS_ACCESS_KEY_ID=AKIA123\nAWS_SECRET_ACCESS_KEY=\n",
			wantErr: "credentials file is missing AWS_SECRET_ACCESS_KEY",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f, cfgPath := newFakeCreateAPI(t)
			opts := baseCreateOptions()
			envFile := testCredentials
			if tt.envFile != "" {
				envFile = tt.envFile
			}
			opts.CredentialsFile = writeTempFile(t, "s3.env", envFile)
			if tt.mutate != nil {
				tt.mutate(&opts)
			}

			err := runCreate(t, true, opts, cfgPath)

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
			calls, _, _ := f.recorded()
			assert.Empty(t, calls)
		})
	}
}

func TestCreateRun_InvalidInput_FailsBeforeAnyRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(t *testing.T, o *CreateOptions)
		wantErr string
	}{
		{
			name:    "no credentials",
			mutate:  func(*testing.T, *CreateOptions) {},
			wantErr: "credentials are required: pass --credentials-secret NAME to reference an existing Secret, or --credentials-file PATH to create one",
		},
		{
			name: "both credentials sources",
			mutate: func(t *testing.T, o *CreateOptions) {
				t.Helper()
				o.CredentialsSecret = "my-s3-creds"
				o.CredentialsFile = writeTempFile(t, "s3.env", testCredentials)
			},
			wantErr: "--credentials-secret and --credentials-file are mutually exclusive",
		},
		{
			name: "both CA sources",
			mutate: func(_ *testing.T, o *CreateOptions) {
				o.CredentialsSecret = "my-s3-creds"
				o.CASecret = "my-ca"
				o.CAFile = "ca.pem"
			},
			wantErr: "--ca-secret and --ca-file are mutually exclusive",
		},
		{
			name: "CA and insecure",
			mutate: func(_ *testing.T, o *CreateOptions) {
				o.CredentialsSecret = "my-s3-creds"
				o.CASecret = "my-ca"
				o.InsecureSkipTLSVerify = true
			},
			wantErr: "--insecure-skip-tls-verify cannot be combined with --ca-secret or --ca-file",
		},
		{
			name: "invalid CA secret reference",
			mutate: func(_ *testing.T, o *CreateOptions) {
				o.CredentialsSecret = "my-s3-creds"
				o.CASecret = "my-ca:"
			},
			wantErr: `invalid --ca-secret "my-ca:": must be NAME or NAME:KEY`,
		},
		{
			name: "parameter conflicting with a shorthand",
			mutate: func(_ *testing.T, o *CreateOptions) {
				o.CredentialsSecret = "my-s3-creds"
				o.Parameters.Parameters = []string{"region=eu-west-1"}
			},
			wantErr: "--region conflicts with --parameter region",
		},
		{
			name: "missing credentials file",
			mutate: func(t *testing.T, o *CreateOptions) {
				t.Helper()
				o.CredentialsFile = filepath.Join(t.TempDir(), "missing.env")
			},
			wantErr: "failed to read credentials file",
		},
		{
			name: "malformed credentials file",
			mutate: func(t *testing.T, o *CreateOptions) {
				t.Helper()
				o.CredentialsFile = writeTempFile(t, "s3.env", "AWS_ACCESS_KEY_ID AKIA123\n")
			},
			wantErr: "line 1: expected KEY=VALUE",
		},
		{
			name: "CA file without PEM data",
			mutate: func(t *testing.T, o *CreateOptions) {
				t.Helper()
				o.CredentialsSecret = "my-s3-creds"
				o.CAFile = writeTempFile(t, "ca.pem", "not a certificate")
			},
			wantErr: "holds no PEM data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := baseCreateOptions()
			tt.mutate(t, &opts)

			err := runCreate(t, true, opts, newConfigPath(t, "http://unused.invalid"))

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestNewBackupStorage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		mutate   func(*CreateOptions)
		params   map[string]any
		wantSpec string
	}{
		{
			name:   "prefix and parameters",
			mutate: func(o *CreateOptions) { o.Prefix = "team-a/prod" },
			params: map[string]any{"region": "us-east-1", "forcePathStyle": true},
			wantSpec: `{"backupStorageClassRef":{"name":"s3"},"location":{"bucket":"my-bucket","prefix":"team-a/prod"},` +
				`"parameters":{"region":"us-east-1","forcePathStyle":true},` +
				`"authentication":{"mode":"AccessKey","credentialsSecretRef":{"name":"my-s3-creds"}}}`,
		},
		{
			name:   "credentials file names the Secret the CLI creates",
			mutate: func(o *CreateOptions) { o.CredentialsSecret, o.CredentialsFile = "", "s3.env" },
			wantSpec: `{"backupStorageClassRef":{"name":"s3"},"location":{"bucket":"my-bucket"},` +
				`"authentication":{"mode":"AccessKey","credentialsSecretRef":{"name":"backup-storage-my-s3-credentials"}}}`,
		},
		{
			name:   "CA secret with the default key",
			mutate: func(o *CreateOptions) { o.CASecret = "my-ca" },
			wantSpec: `{"backupStorageClassRef":{"name":"s3"},"location":{"bucket":"my-bucket"},` +
				`"authentication":{"mode":"AccessKey","credentialsSecretRef":{"name":"my-s3-creds"}},` +
				`"tls":{"ca":{"secretKeyRef":{"name":"my-ca","key":"ca.crt"}}}}`,
		},
		{
			name:   "CA secret with a key",
			mutate: func(o *CreateOptions) { o.CASecret = "my-ca:bundle.pem" },
			wantSpec: `{"backupStorageClassRef":{"name":"s3"},"location":{"bucket":"my-bucket"},` +
				`"authentication":{"mode":"AccessKey","credentialsSecretRef":{"name":"my-s3-creds"}},` +
				`"tls":{"ca":{"secretKeyRef":{"name":"my-ca","key":"bundle.pem"}}}}`,
		},
		{
			name:   "insecure skip verify",
			mutate: func(o *CreateOptions) { o.InsecureSkipTLSVerify = true },
			wantSpec: `{"backupStorageClassRef":{"name":"s3"},"location":{"bucket":"my-bucket"},` +
				`"authentication":{"mode":"AccessKey","credentialsSecretRef":{"name":"my-s3-creds"}},` +
				`"tls":{"insecureSkipVerify":true}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			opts := baseCreateOptions()
			opts.CredentialsSecret = "my-s3-creds"
			tt.mutate(&opts)

			bs, err := newBackupStorage(opts, tt.params)

			require.NoError(t, err)
			require.NotNil(t, bs.Metadata)
			assert.Equal(t, "my-s3", bs.Metadata.Name)
			assert.Equal(t, "everest", bs.Metadata.Namespace)
			spec, err := json.Marshal(bs.Spec)
			require.NoError(t, err)
			assert.JSONEq(t, tt.wantSpec, string(spec))
		})
	}
}
