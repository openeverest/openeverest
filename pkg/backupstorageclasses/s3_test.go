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

package backupstorageclasses_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	"github.com/openeverest/openeverest/v2/pkg/backupstorageclasses"
)

func raw(s string) *runtime.RawExtension {
	return &runtime.RawExtension{Raw: []byte(s)}
}

func TestS3From(t *testing.T) {
	t.Parallel()

	location := backupv1alpha1.BackupStorageLocation{ObjectStore: &backupv1alpha1.ObjectStoreLocation{Bucket: "team-a-backups", Prefix: "everest"}}
	keys := map[string][]byte{
		backupstorageclasses.KeyAWSAccessKeyID:     []byte("AKIA"),
		backupstorageclasses.KeyAWSSecretAccessKey: []byte("secret"),
		"unrelated": []byte("ignored"),
	}

	t.Run("decodes parameters and credentials", func(t *testing.T) {
		t.Parallel()

		params := raw(`{"region":"us-east-1","endpointURL":"https://minio.example.com:9000","forcePathStyle":true,"implementation":"minio"}`)
		s3, creds, err := backupstorageclasses.S3From(location, params, backupstorageclasses.AuthModeAccessKey, keys)
		require.NoError(t, err)
		want := backupstorageclasses.S3{
			Region:         "us-east-1",
			EndpointURL:    "https://minio.example.com:9000",
			ForcePathStyle: true,
			Implementation: backupstorageclasses.S3ImplementationMinIO,
		}
		assert.Equal(t, want, s3)
		assert.Equal(t, backupstorageclasses.S3Credentials{AccessKeyID: "AKIA", SecretAccessKey: "secret"}, creds)
		ep, err := s3.Endpoint()
		require.NoError(t, err)
		assert.Equal(t, "minio.example.com:9000", ep.HostPort())
	})

	tests := []struct {
		name       string
		location   *backupv1alpha1.BackupStorageLocation
		parameters string
		mode       string
		data       map[string][]byte
		wantErr    error
	}{
		{name: "unknown parameter", parameters: `{"region":"r","endpointURL":"https://s3","checksumAlgorithm":"crc32"}`, wantErr: backupstorageclasses.ErrUnsupportedParameter},
		{name: "wrong type", parameters: `{"region":"r","endpointURL":"https://s3","forcePathStyle":"yes"}`, wantErr: backupstorageclasses.ErrInvalidParameters},
		{name: "missing region", parameters: `{"endpointURL":"https://s3"}`, wantErr: backupstorageclasses.ErrInvalidParameters},
		{name: "missing endpoint", parameters: `{"region":"r"}`, wantErr: backupstorageclasses.ErrInvalidParameters},
		{name: "endpoint not a URL", parameters: `{"region":"r","endpointURL":"s3.amazonaws.com"}`, wantErr: backupstorageclasses.ErrInvalidParameters},
		{name: "bucket too short", location: &backupv1alpha1.BackupStorageLocation{ObjectStore: &backupv1alpha1.ObjectStoreLocation{Bucket: "ab"}}, parameters: `{"region":"r","endpointURL":"https://s3"}`, wantErr: backupstorageclasses.ErrInvalidLocation},
		{name: "bucket with uppercase", location: &backupv1alpha1.BackupStorageLocation{ObjectStore: &backupv1alpha1.ObjectStoreLocation{Bucket: "Backups"}}, parameters: `{"region":"r","endpointURL":"https://s3"}`, wantErr: backupstorageclasses.ErrInvalidLocation},
		{name: "bucket with double dot", location: &backupv1alpha1.BackupStorageLocation{ObjectStore: &backupv1alpha1.ObjectStoreLocation{Bucket: "a..b"}}, parameters: `{"region":"r","endpointURL":"https://s3"}`, wantErr: backupstorageclasses.ErrInvalidLocation},
		{name: "no object store", location: &backupv1alpha1.BackupStorageLocation{}, parameters: `{"region":"r","endpointURL":"https://s3"}`, wantErr: backupstorageclasses.ErrInvalidLocation},
		{name: "unknown mode", parameters: `{"region":"r","endpointURL":"https://s3"}`, mode: "SessionToken", wantErr: backupstorageclasses.ErrAuthModeUnsupported},
		{name: "missing secret key", parameters: `{"region":"r","endpointURL":"https://s3"}`, data: map[string][]byte{backupstorageclasses.KeyAWSAccessKeyID: []byte("AKIA")}, wantErr: backupstorageclasses.ErrCredentialsMissing},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			loc := location
			if tt.location != nil {
				loc = *tt.location
			}
			mode := tt.mode
			if mode == "" {
				mode = backupstorageclasses.AuthModeAccessKey
			}
			data := tt.data
			if data == nil {
				data = keys
			}
			_, _, err := backupstorageclasses.S3From(loc, raw(tt.parameters), mode, data)
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestEffectiveImplementation(t *testing.T) {
	t.Parallel()

	for in, want := range map[backupstorageclasses.S3Implementation]backupstorageclasses.S3Implementation{
		"":       backupstorageclasses.S3ImplementationAWS,
		"aws":    backupstorageclasses.S3ImplementationAWS,
		"minio":  backupstorageclasses.S3ImplementationMinIO,
		"ceph":   backupstorageclasses.S3ImplementationCeph,
		"gcs":    backupstorageclasses.S3ImplementationGCS,
		"other":  backupstorageclasses.S3ImplementationOther,
		"wasabi": backupstorageclasses.S3ImplementationOther,
	} {
		assert.Equal(t, want, backupstorageclasses.S3{Implementation: in}.EffectiveImplementation(), "implementation %q", in)
	}
}
