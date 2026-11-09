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

package backup_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	"github.com/openeverest/openeverest/v2/backupstorageclasses/s3"
	"github.com/openeverest/openeverest/v2/internal/controller/backup"
)

// TestS3Validations judges BackupStorages with the generated s3 class, whose
// validations Accept compiles against the embedded BackupStorage CRD, so a
// storage view that drifts from BackupStorageSpec fails here. Without tls,
// Accept and the s3 decoder executors use must agree.
func TestS3Validations(t *testing.T) {
	t.Parallel()

	class := builtinClass(t, s3.ClassName)
	assert.Len(t, class.Spec.Validations, 3)

	pem := selfSignedPEM(t)
	ca := &common.TLSConfig{CA: &common.CABundleSource{SecretKeyRef: &common.SecretKeyRef{Name: "ca", Key: "ca.crt"}}}
	tests := []struct {
		name       string
		bucket     string
		endpoint   string
		tls        *common.TLSConfig
		wantReason string
	}{
		{name: "valid", bucket: "team-a-backups", endpoint: "https://s3.example.com"},
		{name: "underscore and dots", bucket: "team_a.backups", endpoint: "https://s3.example.com"},
		{name: "63 characters", bucket: strings.Repeat("a", 63), endpoint: "https://s3.example.com"},
		{name: "64 characters", bucket: strings.Repeat("a", 64), endpoint: "https://s3.example.com", wantReason: backupv1alpha1.ReasonInvalidLocation},
		{name: "too short", bucket: "ab", endpoint: "https://s3.example.com", wantReason: backupv1alpha1.ReasonInvalidLocation},
		{name: "uppercase", bucket: "Backups", endpoint: "https://s3.example.com", wantReason: backupv1alpha1.ReasonInvalidLocation},
		{name: "double dot", bucket: "team..a", endpoint: "https://s3.example.com", wantReason: backupv1alpha1.ReasonInvalidLocation},
		{name: "starts with a dash", bucket: "-backups", endpoint: "https://s3.example.com", wantReason: backupv1alpha1.ReasonInvalidLocation},
		{name: "CA on https", bucket: "backups", endpoint: "https://s3.example.com", tls: ca},
		{name: "insecure on https", bucket: "backups", endpoint: "https://s3.example.com", tls: &common.TLSConfig{InsecureSkipVerify: true}},
		{name: "plain http", bucket: "backups", endpoint: "http://minio.minio.svc:9000"},
		{name: "empty tls on http", bucket: "backups", endpoint: "http://minio.minio.svc:9000", tls: &common.TLSConfig{}},
		{name: "CA on http", bucket: "backups", endpoint: "http://minio.minio.svc:9000", tls: ca, wantReason: backupv1alpha1.ReasonInvalidParameters},
		{
			name: "insecure on http", bucket: "backups", endpoint: "http://minio.minio.svc:9000",
			tls: &common.TLSConfig{InsecureSkipVerify: true}, wantReason: backupv1alpha1.ReasonInvalidParameters,
		},
		{name: "highest port", bucket: "backups", endpoint: "https://minio.example.com:65535"},
		{name: "IPv6 with port", bucket: "backups", endpoint: "http://[fd00::1]:9000"},
		{name: "port 0", bucket: "backups", endpoint: "https://minio.example.com:0", wantReason: backupv1alpha1.ReasonInvalidParameters},
		{name: "port out of range", bucket: "backups", endpoint: "https://minio.example.com:70000", wantReason: backupv1alpha1.ReasonInvalidParameters},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			params := []byte(`{"region":"us-east-1","endpointURL":"` + tt.endpoint + `"}`)
			verdict := backup.Accept(class, backup.AcceptInput{
				Location:       backupv1alpha1.BackupStorageLocation{ObjectStore: &backupv1alpha1.ObjectStoreLocation{Bucket: tt.bucket}},
				Parameters:     raw(string(params)),
				Authentication: common.Authentication{Mode: s3.AuthModeAccessKey, CredentialsSecretRef: &common.SecretRef{Name: "creds"}},
				TLS:            tt.tls,
				Credentials: backup.SecretContent{Found: true, Data: map[string][]byte{
					s3.KeyAWSAccessKeyID: []byte("AKIA"), s3.KeyAWSSecretAccessKey: []byte("secret"),
				}},
				CA: backup.SecretContent{Found: true, Data: map[string][]byte{"ca.crt": pem}},
			})
			want := tt.wantReason
			if want == "" {
				want = backupv1alpha1.ReasonAccepted
			}
			assert.Equal(t, want, verdict.Reason, "Accept: %s", verdict.Message)

			if tt.tls == nil {
				_, err := s3.Decode(s3.Location{Bucket: tt.bucket}, params)
				assert.Equal(t, tt.wantReason == "", err == nil, "the s3 decoder agrees: %v", err)
			}
		})
	}
}
