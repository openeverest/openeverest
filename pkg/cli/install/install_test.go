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

package install

import (
	"testing"

	versionpb "github.com/Percona-Lab/percona-version-service/versionpb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstall_latestVersion(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		providedVersion string
		meta            *versionpb.MetadataResponse
		expectVersion   string
		expectError     bool
	}{
		{
			name: "Installs the latest version",
			meta: &versionpb.MetadataResponse{
				Versions: []*versionpb.MetadataVersion{
					{Version: "0.8.0"},
					{Version: "0.10.0"},
					{Version: "0.9.0"},
				},
			},
			expectVersion: "0.10.0",
		},
		{
			name: "Installs the latest version if only one version",
			meta: &versionpb.MetadataResponse{
				Versions: []*versionpb.MetadataVersion{
					{Version: "0.10.0"},
				},
			},
			expectVersion: "0.10.0",
		},
		{
			name: "Installs the target version",
			meta: &versionpb.MetadataResponse{
				Versions: []*versionpb.MetadataVersion{
					{Version: "0.8.0"},
					{Version: "0.11.0"},
					{Version: "0.10.0"},
					{Version: "0.9.0"},
				},
			},
			providedVersion: "0.10.0",
			expectVersion:   "0.10.0",
		},
		{
			name: "Fails if target version is not available",
			meta: &versionpb.MetadataResponse{
				Versions: []*versionpb.MetadataVersion{
					{Version: "0.8.0"},
					{Version: "0.11.0"},
					{Version: "0.10.0"},
					{Version: "0.9.0"},
				},
			},
			providedVersion: "0.12.0",
			expectError:     true,
		},
		{
			name: "Fails on invalid version",
			meta: &versionpb.MetadataResponse{
				Versions: []*versionpb.MetadataVersion{
					{Version: "0.8.0"},
					{Version: "0.11.0"},
					{Version: "0.10.0"},
					{Version: "0.9.0"},
				},
			},
			providedVersion: "abcd",
			expectError:     true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			i := &Installer{
				cfg: Config{
					Version: tc.providedVersion,
				},
			}

			v, _, err := i.latestVersion(tc.meta)
			if tc.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, v.String(), tc.expectVersion)
		})
	}
}
