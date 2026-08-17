// everest
// Copyright (C) 2023 Percona LLC
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

package oidc_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/percona/everest/pkg/oidc"
)

func TestValidateURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{name: "empty", url: "", wantErr: true},
		// url.ParseRequestURI accepts scheme-relative and relative-looking
		// strings that ValidateURL's explicit scheme/host checks are there
		// to reject (this is the behavior the maintainer specifically
		// flagged as worth covering).
		{name: "missing scheme", url: "example.com/issuer", wantErr: true},
		{name: "scheme only, no host", url: "https://", wantErr: true},
		{name: "path only, no scheme or host", url: "/just/a/path", wantErr: true},
		{name: "valid https URL", url: "https://accounts.example.com", wantErr: false},
		{name: "valid https URL with path", url: "https://login.microsoftonline.com/tenant/v2.0", wantErr: false},
		{name: "valid http URL", url: "http://localhost:8080", wantErr: false},
		{name: "not a URL at all", url: "not a url", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := oidc.ValidateURL(tt.url)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
