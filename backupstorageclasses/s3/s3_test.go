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

package s3_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openeverest/openeverest/v2/backupstorageclasses"
	"github.com/openeverest/openeverest/v2/backupstorageclasses/s3"
)

func TestDecode(t *testing.T) {
	t.Parallel()

	location := s3.Location{Bucket: "team-a-backups", Prefix: "everest"}

	t.Run("decodes parameters", func(t *testing.T) {
		t.Parallel()

		params := []byte(`{"region":"us-east-1","endpointURL":"https://minio.example.com:9000","forcePathStyle":true,"implementation":"minio"}`)
		got, err := s3.Decode(location, params)
		require.NoError(t, err)
		want := s3.Parameters{
			Region:         "us-east-1",
			EndpointURL:    "https://minio.example.com:9000",
			ForcePathStyle: true,
			Implementation: s3.ImplementationMinIO,
		}
		assert.Equal(t, want, got)
		ep, err := got.Endpoint()
		require.NoError(t, err)
		assert.Equal(t, "minio.example.com:9000", ep.HostPort())
	})

	tests := []struct {
		name       string
		bucket     string
		parameters string
		wantErr    error
	}{
		{name: "unknown parameter", parameters: `{"region":"r","endpointURL":"https://s3","checksumAlgorithm":"crc32"}`, wantErr: backupstorageclasses.ErrUnsupportedParameter},
		{name: "wrong type", parameters: `{"region":"r","endpointURL":"https://s3","forcePathStyle":"yes"}`, wantErr: backupstorageclasses.ErrInvalidParameters},
		{name: "no parameters", parameters: ``, wantErr: backupstorageclasses.ErrInvalidParameters},
		{name: "missing region", parameters: `{"endpointURL":"https://s3"}`, wantErr: backupstorageclasses.ErrInvalidParameters},
		{name: "missing endpoint", parameters: `{"region":"r"}`, wantErr: backupstorageclasses.ErrInvalidParameters},
		{name: "endpoint not a URL", parameters: `{"region":"r","endpointURL":"s3.amazonaws.com"}`, wantErr: backupstorageclasses.ErrInvalidParameters},
		{name: "bucket too short", bucket: "ab", parameters: `{"region":"r","endpointURL":"https://s3"}`, wantErr: backupstorageclasses.ErrInvalidLocation},
		{name: "bucket with uppercase", bucket: "Backups", parameters: `{"region":"r","endpointURL":"https://s3"}`, wantErr: backupstorageclasses.ErrInvalidLocation},
		{name: "bucket with double dot", bucket: "a..b", parameters: `{"region":"r","endpointURL":"https://s3"}`, wantErr: backupstorageclasses.ErrInvalidLocation},
		{name: "no bucket", bucket: "-", parameters: `{"region":"r","endpointURL":"https://s3"}`, wantErr: backupstorageclasses.ErrInvalidLocation},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			loc := location
			switch tt.bucket {
			case "":
			case "-":
				loc = s3.Location{}
			default:
				loc.Bucket = tt.bucket
			}
			_, err := s3.Decode(loc, []byte(tt.parameters))
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestDecodeCredentials(t *testing.T) {
	t.Parallel()

	keys := map[string][]byte{
		s3.KeyAWSAccessKeyID:     []byte("AKIA"),
		s3.KeyAWSSecretAccessKey: []byte("secret"),
		"unrelated":              []byte("ignored"),
	}
	creds, err := s3.DecodeCredentials(s3.AuthModeAccessKey, keys)
	require.NoError(t, err)
	assert.Equal(t, s3.Credentials{AccessKeyID: "AKIA", SecretAccessKey: "secret"}, creds)

	_, err = s3.DecodeCredentials("SessionToken", keys)
	require.ErrorIs(t, err, backupstorageclasses.ErrAuthModeUnsupported)
	_, err = s3.DecodeCredentials(s3.AuthModeAccessKey, map[string][]byte{s3.KeyAWSAccessKeyID: []byte("AKIA")})
	require.ErrorIs(t, err, backupstorageclasses.ErrCredentialsMissing)
}

func TestEffectiveImplementation(t *testing.T) {
	t.Parallel()

	for in, want := range map[s3.Implementation]s3.Implementation{
		"":       s3.ImplementationAWS,
		"aws":    s3.ImplementationAWS,
		"minio":  s3.ImplementationMinIO,
		"ceph":   s3.ImplementationCeph,
		"gcs":    s3.ImplementationGCS,
		"other":  s3.ImplementationOther,
		"wasabi": s3.ImplementationOther,
	} {
		assert.Equal(t, want, s3.Parameters{Implementation: in}.EffectiveImplementation(), "implementation %q", in)
	}
}
