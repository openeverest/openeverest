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
	"io"
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

func TestBuildSpecPatch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		opts      UpdateOptions
		wantPatch string
		wantErr   string
	}{
		{
			name:      "nothing set",
			wantPatch: `{}`,
		},
		{
			name: "parameters and shorthands",
			opts: UpdateOptions{Parameters: ParameterOptions{
				Parameters: []string{"implementation=minio"}, Region: new("eu-west-1"), ForcePathStyle: new(true),
			}},
			wantPatch: `{"parameters":{"implementation":"minio","region":"eu-west-1","forcePathStyle":true}}`,
		},
		{
			name:      "credentials secret",
			opts:      UpdateOptions{CredentialsSecret: "my-s3-credentials-2"}, //nolint:gosec // a Secret name, not a credential
			wantPatch: `{"authentication":{"credentialsSecretRef":{"name":"my-s3-credentials-2"}}}`,
		},
		{
			name:      "auth mode and credentials secret",
			opts:      UpdateOptions{AuthMode: "AccessKey", CredentialsSecret: "my-s3-credentials-2"}, //nolint:gosec // a Secret name, not a credential
			wantPatch: `{"authentication":{"mode":"AccessKey","credentialsSecretRef":{"name":"my-s3-credentials-2"}}}`,
		},
		{
			name:      "CA secret clears insecure skip verify",
			opts:      UpdateOptions{CASecret: "my-ca"},
			wantPatch: `{"tls":{"ca":{"secretKeyRef":{"name":"my-ca","key":"ca.crt"}},"insecureSkipVerify":null}}`,
		},
		{
			name:      "CA secret with a key and insecure explicitly off",
			opts:      UpdateOptions{CASecret: "my-ca:bundle.pem", InsecureSkipTLSVerify: new(false)}, //nolint:gosec // a Secret name, not a credential
			wantPatch: `{"tls":{"ca":{"secretKeyRef":{"name":"my-ca","key":"bundle.pem"}},"insecureSkipVerify":false}}`,
		},
		{
			name:      "insecure on clears the CA",
			opts:      UpdateOptions{InsecureSkipTLSVerify: new(true)},
			wantPatch: `{"tls":{"insecureSkipVerify":true,"ca":null}}`,
		},
		{
			name:      "insecure off",
			opts:      UpdateOptions{InsecureSkipTLSVerify: new(false)},
			wantPatch: `{"tls":{"insecureSkipVerify":false}}`,
		},
		{
			name:    "CA secret and insecure on",
			opts:    UpdateOptions{CASecret: "my-ca", InsecureSkipTLSVerify: new(true)},
			wantErr: "--insecure-skip-tls-verify cannot be combined with --ca-secret",
		},
		{
			name:    "invalid CA secret reference",
			opts:    UpdateOptions{CASecret: ":ca.crt"},
			wantErr: `invalid --ca-secret ":ca.crt"`,
		},
		{
			name:    "shorthand conflict",
			opts:    UpdateOptions{Parameters: ParameterOptions{Parameters: []string{"region=a"}, Region: new("b")}},
			wantErr: "--region conflicts with --parameter region",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			spec, err := buildSpecPatch(tt.opts)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			got, err := json.Marshal(spec)
			require.NoError(t, err)
			assert.JSONEq(t, tt.wantPatch, string(got))
		})
	}
}

func TestUpdateRun_NothingToUpdate_ReturnsErrorWithoutCallingServer(t *testing.T) {
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
	assert.Contains(t, err.Error(), "nothing to update: pass at least one of --parameter")
}

func TestUpdateRun_SendsSpecMergePatch(t *testing.T) {
	t.Parallel()

	var gotPatch string
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/clusters/main/namespaces/everest/backup-storages/my-s3", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "application/merge-patch+json", r.Header.Get("Content-Type"))
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		gotPatch = string(body)
		writeJSON(w, http.StatusOK, backupStorageWithName(t, "my-s3", "everest"))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	opts := baseUpdateOptions()
	opts.CredentialsSecret = "my-s3-credentials-2"
	opts.Parameters.EndpointURL = new("https://s3.eu-west-1.amazonaws.com")

	u := NewUpdater(Config{Pretty: false}, zap.NewNop().Sugar())
	err := u.Run(context.Background(), opts, newConfigPath(t, srv.URL))

	require.NoError(t, err)
	want := `{"spec":{
		"parameters":{"endpointURL":"https://s3.eu-west-1.amazonaws.com"},
		"authentication":{"credentialsSecretRef":{"name":"my-s3-credentials-2"}}
	}}`
	assert.JSONEq(t, want, gotPatch)
}

func TestUpdateRun_ErrorResponses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		respond func(w http.ResponseWriter)
		wantErr string
	}{
		{
			name:    "not found",
			respond: func(w http.ResponseWriter) { w.WriteHeader(http.StatusNotFound) },
			wantErr: `backup storage "my-s3" not found in namespace "everest"`,
		},
		{
			name: "rejected patch",
			respond: func(w http.ResponseWriter) {
				writeJSON(w, http.StatusBadRequest, map[string]string{"message": "location is immutable"})
			},
			wantErr: "server error: location is immutable",
		},
		{
			name:    "undeclared status",
			respond: func(w http.ResponseWriter) { w.WriteHeader(http.StatusBadGateway) },
			wantErr: `unexpected response updating backup storage "my-s3"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mux := http.NewServeMux()
			mux.HandleFunc("/v1/clusters/main/namespaces/everest/backup-storages/my-s3", func(w http.ResponseWriter, _ *http.Request) {
				tt.respond(w)
			})
			srv := httptest.NewServer(mux)
			defer srv.Close()

			opts := baseUpdateOptions()
			opts.CredentialsSecret = "other-secret"

			u := NewUpdater(Config{Pretty: true}, zap.NewNop().Sugar())
			err := u.Run(context.Background(), opts, newConfigPath(t, srv.URL))

			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}
