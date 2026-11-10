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
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseParameterValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		raw  string
		want any
	}{
		{raw: "us-east-1", want: "us-east-1"},
		{raw: "https://s3.amazonaws.com", want: "https://s3.amazonaws.com"},
		{raw: "true", want: true},
		{raw: "false", want: false},
		{raw: "3", want: json.Number("3")},
		{raw: "-1.5", want: json.Number("-1.5")},
		{raw: "1e3", want: json.Number("1e3")},
		{raw: "True", want: "True"},
		{raw: "yes", want: "yes"},
		{raw: "null", want: "null"},
		{raw: `"quoted"`, want: `"quoted"`},
		{raw: "{}", want: "{}"},
		{raw: " 3", want: " 3"},
		{raw: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, parseParameterValue(tt.raw))
		})
	}
}

func TestParameterOptionsBuild(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		opts    ParameterOptions
		want    map[string]any
		wantErr string
	}{
		{
			name: "nothing set",
			want: make(map[string]any),
		},
		{
			name: "parameters keep their JSON type",
			opts: ParameterOptions{Parameters: []string{"region=us-east-1", "forcePathStyle=true", "retries=3"}},
			want: map[string]any{"region": "us-east-1", "forcePathStyle": true, "retries": json.Number("3")},
		},
		{
			name: "value split at the first =",
			opts: ParameterOptions{Parameters: []string{"endpointURL=https://s3.example.com/?a=b"}},
			want: map[string]any{"endpointURL": "https://s3.example.com/?a=b"},
		},
		{
			name: "shorthands",
			opts: ParameterOptions{
				Region:         new("us-east-1"),
				EndpointURL:    new("https://s3.amazonaws.com"),
				ForcePathStyle: new(false),
				Implementation: new("minio"),
			},
			want: map[string]any{
				"region": "us-east-1", "endpointURL": "https://s3.amazonaws.com",
				"forcePathStyle": false, "implementation": "minio",
			},
		},
		{
			name: "parameters and shorthands for different keys",
			opts: ParameterOptions{Parameters: []string{"implementation=ceph"}, Region: new("us-east-1")},
			want: map[string]any{"implementation": "ceph", "region": "us-east-1"},
		},
		{
			name:    "duplicate parameter",
			opts:    ParameterOptions{Parameters: []string{"region=us-east-1", "region=eu-west-1"}},
			wantErr: "--parameter region is set more than once",
		},
		{
			name:    "missing =",
			opts:    ParameterOptions{Parameters: []string{"region"}},
			wantErr: `invalid --parameter "region": must be in the form key=value`,
		},
		{
			name:    "empty key",
			opts:    ParameterOptions{Parameters: []string{"=us-east-1"}},
			wantErr: `invalid --parameter "=us-east-1": must be in the form key=value`,
		},
		{
			name:    "region shorthand conflicts",
			opts:    ParameterOptions{Parameters: []string{"region=us-east-1"}, Region: new("eu-west-1")},
			wantErr: "--region conflicts with --parameter region",
		},
		{
			name:    "endpoint shorthand conflicts",
			opts:    ParameterOptions{Parameters: []string{"endpointURL=https://a"}, EndpointURL: new("https://b")},
			wantErr: "--endpoint-url conflicts with --parameter endpointURL",
		},
		{
			name:    "force path style shorthand conflicts",
			opts:    ParameterOptions{Parameters: []string{"forcePathStyle=true"}, ForcePathStyle: new(true)},
			wantErr: "--force-path-style conflicts with --parameter forcePathStyle",
		},
		{
			name:    "implementation shorthand conflicts",
			opts:    ParameterOptions{Parameters: []string{"implementation=aws"}, Implementation: new("minio")},
			wantErr: "--implementation conflicts with --parameter implementation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tt.opts.build()

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseCASecret(t *testing.T) {
	t.Parallel()

	tests := []struct {
		ref      string
		wantName string
		wantKey  string
		wantErr  bool
	}{
		{ref: "my-ca", wantName: "my-ca", wantKey: "ca.crt"},
		{ref: "my-ca:bundle.pem", wantName: "my-ca", wantKey: "bundle.pem"},
		{ref: "my-ca:", wantErr: true},
		{ref: ":ca.crt", wantErr: true},
		{ref: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.ref, func(t *testing.T) {
			t.Parallel()

			name, key, err := parseCASecret(tt.ref)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "must be NAME or NAME:KEY")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantName, name)
			assert.Equal(t, tt.wantKey, key)
		})
	}
}
