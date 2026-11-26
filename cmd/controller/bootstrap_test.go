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

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestControllerNamespace(t *testing.T) {
	dir := t.TempDir()
	saFile := filepath.Join(dir, "namespace")
	require.NoError(t, os.WriteFile(saFile, []byte("everest-system\n"), 0o600))
	emptyFile := filepath.Join(dir, "empty")
	require.NoError(t, os.WriteFile(emptyFile, nil, 0o600))

	tests := []struct {
		name         string
		podNamespace string
		file         string
		want         string
		wantErr      bool
	}{
		{name: "POD_NAMESPACE wins", podNamespace: "from-env", file: saFile, want: "from-env"},
		{name: "the ServiceAccount's namespace", file: saFile, want: "everest-system"},
		{name: "outside a cluster", file: filepath.Join(dir, "missing"), wantErr: true},
		{name: "an empty namespace file", file: emptyFile, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("POD_NAMESPACE", tt.podNamespace)
			got, err := controllerNamespace(tt.file)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
