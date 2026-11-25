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

package contract_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openeverest/openeverest/v2/contract"
)

func TestParseAcknowledgedExtensions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		v       string
		want    map[string]string
		wantErr string
	}{
		{
			name: "empty",
			v:    "",
			want: make(map[string]string),
		},
		{
			name: "only spaces",
			v:    "  ",
			want: make(map[string]string),
		},
		{
			name: "one entry",
			v:    "percona-server-mongodb@2.1.0",
			want: map[string]string{"percona-server-mongodb": "2.1.0"},
		},
		{
			name: "several entries with spaces",
			v:    " percona-server-mongodb @ 2.1.0 , pmm@2.1.1-rc.1,audit@2.1.0+build.1 ",
			want: map[string]string{
				"percona-server-mongodb": "2.1.0",
				"pmm":                    "2.1.1-rc.1",
				"audit":                  "2.1.0+build.1",
			},
		},
		{
			name:    "no release",
			v:       "percona-server-mongodb",
			wantErr: `acknowledged extension "percona-server-mongodb" is not <name>@<release>`,
		},
		{
			name:    "empty entry",
			v:       "percona-server-mongodb@2.1.0,,pmm@2.1.0",
			wantErr: `acknowledged extension "" is not <name>@<release>`,
		},
		{
			name:    "trailing comma",
			v:       "percona-server-mongodb@2.1.0,",
			wantErr: `acknowledged extension "" is not <name>@<release>`,
		},
		{
			name:    "empty name",
			v:       " @2.1.0",
			wantErr: `acknowledged extension " @2.1.0" has no name`,
		},
		{
			name:    "empty release",
			v:       "pmm@",
			wantErr: `acknowledged extension "pmm": "" is not a SemVer 2.0.0 version`,
		},
		{
			name:    "invalid release",
			v:       "pmm@v2.1.0",
			wantErr: `acknowledged extension "pmm": "v2.1.0" is not a SemVer 2.0.0 version`,
		},
		{
			name:    "short release",
			v:       "pmm@2.1",
			wantErr: `acknowledged extension "pmm": "2.1" is not a SemVer 2.0.0 version`,
		},
		{
			name:    "two separators",
			v:       "pmm@2.1.0@2.2.0",
			wantErr: `acknowledged extension "pmm": "2.1.0@2.2.0" is not a SemVer 2.0.0 version`,
		},
		{
			name:    "duplicate with the same release",
			v:       "pmm@2.1.0,pmm@2.1.0",
			wantErr: `acknowledged extension "pmm" is listed more than once`,
		},
		{
			name:    "duplicate with another release",
			v:       "pmm@2.1.0, pmm @2.2.0",
			wantErr: `acknowledged extension "pmm" is listed more than once`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := contract.ParseAcknowledgedExtensions(tt.v)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestAcknowledgedExtensionsHold reads a published list the way readers do:
// parse it, then decide each entry by AcknowledgementHolds against the
// installed core.
func TestAcknowledgedExtensionsHold(t *testing.T) {
	t.Parallel()

	acks, err := contract.ParseAcknowledgedExtensions("percona-server-mongodb@2.1.0,pmm@2.0.3,audit@2.2.0")
	require.NoError(t, err)

	const contractVersion, coreVersion = "2.1.0", "2.1.1"
	holds := make(map[string]bool, len(acks))
	for name, release := range acks {
		holds[name] = contract.AcknowledgementHolds(release, contractVersion, coreVersion)
	}
	want := map[string]bool{
		"percona-server-mongodb": true,
		"pmm":                    false,
		"audit":                  false,
	}
	assert.Equal(t, want, holds)
}
