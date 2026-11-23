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

package cli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncode(t *testing.T) {
	t.Parallel()

	type item struct {
		ProjectName   string  `json:"projectName"`
		ServerVersion *string `json:"serverVersion,omitempty"`
	}
	v := item{ProjectName: "db"}

	tests := []struct {
		name   string
		asYAML bool
		want   string
	}{
		{name: "json", asYAML: false, want: "{\"projectName\":\"db\"}\n"},
		{name: "yaml", asYAML: true, want: "projectName: db\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			require.NoError(t, Encode(&buf, v, tt.asYAML))
			assert.Equal(t, tt.want, buf.String())
		})
	}
}
