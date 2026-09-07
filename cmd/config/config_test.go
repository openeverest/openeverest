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

package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseConfig_TelemetryInterval(t *testing.T) {
	testCases := []struct {
		name        string
		interval    string
		wantErr     bool
		expectedErr string
	}{
		{
			name:     "valid duration 10s",
			interval: "10s",
			wantErr:  false,
		},
		{
			name:     "valid duration 1m",
			interval: "1m",
			wantErr:  false,
		},
		{
			name:     "valid duration 24h",
			interval: "24h",
			wantErr:  false,
		},
		{
			name:        "zero duration 0s",
			interval:    "0s",
			wantErr:     true,
			expectedErr: "TELEMETRY_INTERVAL must be a positive duration",
		},
		{
			name:        "zero duration 0m",
			interval:    "0m",
			wantErr:     true,
			expectedErr: "TELEMETRY_INTERVAL must be a positive duration",
		},
		{
			name:        "zero duration 0",
			interval:    "0",
			wantErr:     true,
			expectedErr: "TELEMETRY_INTERVAL must be a positive duration",
		},
		{
			name:        "negative duration -1m",
			interval:    "-1m",
			wantErr:     true,
			expectedErr: "TELEMETRY_INTERVAL must be a positive duration",
		},
		{
			name:        "negative duration -5s",
			interval:    "-5s",
			wantErr:     true,
			expectedErr: "TELEMETRY_INTERVAL must be a positive duration",
		},
		{
			name:        "invalid duration typo 5mn",
			interval:    "5mn",
			wantErr:     true,
			expectedErr: "invalid TELEMETRY_INTERVAL",
		},
		{
			name:        "invalid duration string abc",
			interval:    "abc",
			wantErr:     true,
			expectedErr: "invalid TELEMETRY_INTERVAL",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("NAMESPACE", "everest-system")
			t.Setenv("TELEMETRY_INTERVAL", tc.interval)

			cfg, err := ParseConfig()
			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectedErr)
				assert.Nil(t, cfg)
			} else {
				require.NoError(t, err)
				require.NotNil(t, cfg)
				assert.Equal(t, tc.interval, cfg.TelemetryInterval)
			}
		})
	}
}

func TestParseConfig_EmptyTelemetryInterval(t *testing.T) {
	t.Setenv("NAMESPACE", "everest-system")
	t.Setenv("TELEMETRY_INTERVAL", "")

	cfg, err := ParseConfig()
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, TelemetryInterval, cfg.TelemetryInterval)
}

func TestParseConfig_MissingNamespace(t *testing.T) {
	t.Setenv("NAMESPACE", "dummy")
	// Unset to verify required:"true" behavior
	os.Unsetenv("NAMESPACE")
	defer func() {
		if val, ok := os.LookupEnv("NAMESPACE"); ok {
			t.Setenv("NAMESPACE", val)
		}
	}()

	cfg, err := ParseConfig()
	require.Error(t, err)
	assert.Nil(t, cfg)
}

func TestParseConfig_TLSCertsPath(t *testing.T) {
	t.Setenv("NAMESPACE", "everest-system")
	t.Setenv("TLS_CERTS_PATH", "/path/to/certs/")

	cfg, err := ParseConfig()
	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, "/path/to/certs", cfg.TLSCertsPath)
}
