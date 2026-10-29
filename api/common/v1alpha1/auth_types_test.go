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

func TestAuthModeValidateCredentials(t *testing.T) {
	t.Parallel()

	accessKey := &common.AuthMode{
		Name: "AccessKey",
		Credentials: &common.CredentialKeys{
			Required: []string{"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY"},
			Optional: []string{"AWS_SESSION_TOKEN"},
		},
	}
	tests := []struct {
		name    string
		mode    *common.AuthMode
		data    map[string][]byte
		wantErr string
	}{
		{
			name: "every required key present, extra keys ignored",
			mode: accessKey,
			data: map[string][]byte{"AWS_ACCESS_KEY_ID": []byte("id"), "AWS_SECRET_ACCESS_KEY": []byte("secret"), "other": []byte("x")},
		},
		{
			name:    "required key missing",
			mode:    accessKey,
			data:    map[string][]byte{"AWS_ACCESS_KEY_ID": []byte("id")},
			wantErr: `missing key "AWS_SECRET_ACCESS_KEY" required by authentication mode "AccessKey"`,
		},
		{
			name:    "required key empty",
			mode:    accessKey,
			data:    map[string][]byte{"AWS_ACCESS_KEY_ID": []byte("id"), "AWS_SECRET_ACCESS_KEY": {}},
			wantErr: `missing key "AWS_SECRET_ACCESS_KEY" required by authentication mode "AccessKey"`,
		},
		{
			name: "reserved mode needs nothing",
			mode: &common.AuthMode{Name: common.AuthModeAnonymous},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.mode.ValidateCredentials(tt.data)
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.EqualError(t, err, tt.wantErr)
		})
	}
}

func TestIsReservedAuthMode(t *testing.T) {
	t.Parallel()

	assert.True(t, common.IsReservedAuthMode("WorkloadIdentity"))
	assert.True(t, common.IsReservedAuthMode("Anonymous"))
	assert.False(t, common.IsReservedAuthMode("AccessKey"))
	assert.False(t, common.IsReservedAuthMode("anonymous"))
}
