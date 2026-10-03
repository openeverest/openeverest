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

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateTelemetryInterval(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		input   string
		wantErr bool
		errMsg  string
	}{
		{
			name:  "valid: 24h",
			input: "24h",
		},
		{
			name:  "valid: 1h",
			input: "1h",
		},
		{
			name:  "valid: 10s",
			input: "10s",
		},
		{
			name:  "valid: empty string",
			input: "",
		},
		{
			name:    "invalid: zero duration",
			input:   "0s",
			wantErr: true,
			errMsg:  "must be greater than 0",
		},
		{
			name:    "invalid: negative duration",
			input:   "-1m",
			wantErr: true,
			errMsg:  "must be greater than 0",
		},
		{
			name:    "invalid: negative hours",
			input:   "-24h",
			wantErr: true,
			errMsg:  "must be greater than 0",
		},
		{
			name:    "invalid: typo",
			input:   "5mn",
			wantErr: true,
			errMsg:  "invalid TELEMETRY_INTERVAL",
		},
		{
			name:    "invalid: garbage",
			input:   "abc",
			wantErr: true,
			errMsg:  "invalid TELEMETRY_INTERVAL",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := validateTelemetryInterval(tc.input)
			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
