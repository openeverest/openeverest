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

package v1alpha1_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
)

func TestParseEndpoint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		raw      string
		want     common.Endpoint
		hostPort string
		wantErr  bool
	}{
		{name: "https default port", raw: "https://s3.us-east-1.amazonaws.com", want: common.Endpoint{Host: "s3.us-east-1.amazonaws.com", Port: 443, Secure: true}, hostPort: "s3.us-east-1.amazonaws.com:443"},
		{name: "http default port", raw: "http://minio.minio.svc", want: common.Endpoint{Host: "minio.minio.svc", Port: 80}, hostPort: "minio.minio.svc:80"},
		{name: "explicit port and path", raw: "https://minio.example.com:9000/ignored", want: common.Endpoint{Host: "minio.example.com", Port: 9000, Secure: true}, hostPort: "minio.example.com:9000"},
		{name: "IPv6", raw: "http://[fd00::1]:9000", want: common.Endpoint{Host: "fd00::1", Port: 9000}, hostPort: "[fd00::1]:9000"},
		{name: "other scheme", raw: "s3://bucket", wantErr: true},
		{name: "relative", raw: "minio:9000", wantErr: true},
		{name: "no host", raw: "https://", wantErr: true},
		{name: "user information", raw: "https://user:pass@minio.example.com", wantErr: true}, //nolint:gosec // not a real credential
		{name: "port out of range", raw: "https://minio.example.com:70000", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := common.ParseEndpoint(tt.raw)
			if tt.wantErr {
				require.ErrorIs(t, err, common.ErrInvalidEndpoint)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.hostPort, got.HostPort())
		})
	}
}
