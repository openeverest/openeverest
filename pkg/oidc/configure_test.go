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

package oidc

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseScopes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		scopesStr  string
		wantScopes []string
		wantErr    bool
	}{
		{
			name:       "valid scopes",
			scopesStr:  "openid,profile,email",
			wantScopes: []string{"openid", "profile", "email"},
		},
		{
			name:       "valid scopes with spaces",
			scopesStr:  "openid, profile, email ",
			wantScopes: []string{"openid", "profile", "email"},
		},
		{
			name:       "openid after a comma and a space",
			scopesStr:  "email, openid",
			wantScopes: []string{"email", "openid"},
		},
		{
			name:       "valid scopes with empty items",
			scopesStr:  "openid,, profile, , email",
			wantScopes: []string{"openid", "profile", "email"},
		},
		{
			name:      "missing openid",
			scopesStr: "profile,email",
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseScopes(tt.scopesStr)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantScopes, got)
		})
	}
}
