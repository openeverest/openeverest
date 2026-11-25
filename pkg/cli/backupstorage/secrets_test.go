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
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCredentialsFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		data    string
		want    map[string]string
		wantErr string
	}{
		{
			name: "key value lines",
			data: "AWS_ACCESS_KEY_ID=AKIA123\nAWS_SECRET_ACCESS_KEY=shh\n",
			want: map[string]string{"AWS_ACCESS_KEY_ID": "AKIA123", "AWS_SECRET_ACCESS_KEY": "shh"},
		},
		{
			name: "comments, blank lines, CRLF and surrounding spaces",
			data: "# S3 keys\r\n\r\n  AWS_ACCESS_KEY_ID = AKIA123  \r\n\t# rotated 2026-10\r\nAWS_SECRET_ACCESS_KEY=shh\r\n",
			want: map[string]string{"AWS_ACCESS_KEY_ID": "AKIA123", "AWS_SECRET_ACCESS_KEY": "shh"},
		},
		{
			name: "no trailing newline",
			data: "AWS_ACCESS_KEY_ID=AKIA123",
			want: map[string]string{"AWS_ACCESS_KEY_ID": "AKIA123"},
		},
		{
			name: "quoted values",
			data: "A=\"with spaces\"\nB='single'\nC=\"unbalanced'\n",
			want: map[string]string{"A": "with spaces", "B": "single", "C": "\"unbalanced'"},
		},
		{
			name: "value keeps every = after the first",
			data: "AWS_SECRET_ACCESS_KEY=abc=def==\n",
			want: map[string]string{"AWS_SECRET_ACCESS_KEY": "abc=def=="},
		},
		{
			name: "empty value",
			data: "AWS_SESSION_TOKEN=\n",
			want: map[string]string{"AWS_SESSION_TOKEN": ""},
		},
		{
			name:    "line without =",
			data:    "AWS_ACCESS_KEY_ID=AKIA123\n\nsupersecretvalue\n",
			wantErr: "line 3: expected KEY=VALUE",
		},
		{
			name:    "empty key",
			data:    "=supersecretvalue\n",
			wantErr: "line 1: expected KEY=VALUE",
		},
		{
			name:    "duplicate key",
			data:    "AWS_ACCESS_KEY_ID=supersecretvalue\nAWS_ACCESS_KEY_ID=other\n",
			wantErr: "line 2: AWS_ACCESS_KEY_ID is set more than once",
		},
		{
			name:    "only comments",
			data:    "# nothing here\n\n",
			wantErr: "no KEY=VALUE entries",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseCredentialsFile([]byte(tt.data))

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				assert.NotContains(t, err.Error(), "supersecretvalue", "errors must never echo a credential")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestReadCAFile(t *testing.T) {
	t.Parallel()

	got, err := readCAFile(writeTempFile(t, "ca.pem", testPEM))
	require.NoError(t, err)
	assert.Equal(t, testPEM, got)

	_, err = readCAFile(writeTempFile(t, "ca.pem", "not a certificate"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "holds no PEM data")

	_, err = readCAFile(filepath.Join(t.TempDir(), "missing.pem"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read CA file")
}
