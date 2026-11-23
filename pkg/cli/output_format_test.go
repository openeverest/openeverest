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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOutputFormat_Set(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		in      string
		want    OutputFormat
		wantErr string
	}{
		{name: "json", in: "json", want: OutputJSON},
		{name: "yaml", in: "yaml", want: OutputYAML},
		{name: "rejects empty", in: "", wantErr: `invalid output format "": must be "json" or "yaml"`},
		{name: "rejects unknown", in: "xml", wantErr: `invalid output format "xml": must be "json" or "yaml"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var o OutputFormat
			err := o.Set(tt.in)
			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, o)
			assert.Equal(t, string(tt.want), o.String())
		})
	}
}
