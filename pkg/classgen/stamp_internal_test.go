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

package classgen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/openeverest/openeverest/v2/contract"
)

func TestSetChartAnnotation(t *testing.T) {
	t.Parallel()

	const key = contract.VersionAnnotation
	tests := map[string]struct {
		in   string
		want string
		// wantParsed is checked instead of want when re-encoding may change
		// the layout.
		wantParsed map[string]any
		wantErr    string
	}{
		"no annotations": {
			in: `apiVersion: v2
name: monitoring-x
version: 0.1.0
keywords:
  - openeverest

# OpenEverest must be installed before this chart.
`,
			want: `apiVersion: v2
name: monitoring-x
version: 0.1.0
keywords:
  - openeverest

# OpenEverest must be installed before this chart.
annotations:
  openeverest.io/contract-version: "2.0.0"
`,
		},
		"no trailing newline": {
			in:   "apiVersion: v2\nname: monitoring-x\nversion: 0.1.0",
			want: "apiVersion: v2\nname: monitoring-x\nversion: 0.1.0\nannotations:\n  openeverest.io/contract-version: \"2.0.0\"\n",
		},
		"other annotations, comments and blank lines": {
			in: `apiVersion: v2
name: monitoring-x
annotations: # chart metadata

  # The kind of extension.
  openeverest.io/extension-type: monitoring
  example.com/team: "dbs"
version: 0.1.0
`,
			want: `apiVersion: v2
name: monitoring-x
annotations: # chart metadata
  openeverest.io/contract-version: "2.0.0"

  # The kind of extension.
  openeverest.io/extension-type: monitoring
  example.com/team: "dbs"
version: 0.1.0
`,
		},
		"four-space indent": {
			in:   "name: monitoring-x\nannotations:\n    example.com/team: dbs\n",
			want: "name: monitoring-x\nannotations:\n    openeverest.io/contract-version: \"2.0.0\"\n    example.com/team: dbs\n",
		},
		"plain stamp with a comment": {
			in:   "name: monitoring-x\nannotations:\n  a: b\n  openeverest.io/contract-version: 2.0.0-alpha.1 # stamped\n  c: d\n",
			want: "name: monitoring-x\nannotations:\n  a: b\n  openeverest.io/contract-version: \"2.0.0\" # stamped\n  c: d\n",
		},
		"double-quoted stamp": {
			in:   "name: monitoring-x\nannotations:\n  openeverest.io/contract-version: \"2.0.0-alpha.1\"\nversion: 0.1.0\n",
			want: "name: monitoring-x\nannotations:\n  openeverest.io/contract-version: \"2.0.0\"\nversion: 0.1.0\n",
		},
		"single-quoted stamp": {
			in:   "name: monitoring-x\nannotations:\n  'openeverest.io/contract-version': 'it''s 1'\n",
			want: "name: monitoring-x\nannotations:\n  'openeverest.io/contract-version': \"2.0.0\"\n",
		},
		"stamp unchanged": {
			in:   "name: monitoring-x\nannotations:\n  openeverest.io/contract-version: 2.0.0 # kept as written\n",
			want: "name: monitoring-x\nannotations:\n  openeverest.io/contract-version: 2.0.0 # kept as written\n",
		},
		"flow annotations": {
			in: "name: monitoring-x\nannotations: {example.com/team: dbs}\nversion: 0.1.0\n",
			wantParsed: map[string]any{
				"name":        "monitoring-x",
				"annotations": map[string]any{"example.com/team": "dbs", key: "2.0.0"},
				"version":     "0.1.0",
			},
		},
		"empty annotations": {
			in: "name: monitoring-x\nannotations:\nversion: 0.1.0\n",
			wantParsed: map[string]any{
				"name":        "monitoring-x",
				"annotations": map[string]any{key: "2.0.0"},
				"version":     "0.1.0",
			},
		},
		"multi-line stamp": {
			in: "name: monitoring-x\nannotations:\n  openeverest.io/contract-version: >-\n    2.0.0-alpha.1\n",
			wantParsed: map[string]any{
				"name":        "monitoring-x",
				"annotations": map[string]any{key: "2.0.0"},
			},
		},
		"not a mapping": {
			in:      "- name: monitoring-x\n",
			wantErr: "not a YAML mapping",
		},
		"annotations not a mapping": {
			in:      "name: monitoring-x\nannotations: [a, b]\n",
			wantErr: "annotations is not a mapping",
		},
		"invalid YAML": {
			in:      "name: [monitoring-x\n",
			wantErr: "parsing",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, err := setChartAnnotation([]byte(tt.in), key, "2.0.0")

			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			if tt.wantParsed != nil {
				var parsed map[string]any
				require.NoError(t, yaml.Unmarshal(got, &parsed))
				assert.Equal(t, tt.wantParsed, parsed)
				return
			}
			assert.Equal(t, tt.want, string(got))
		})
	}
}

func TestChartStamp(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		in   string
		want string
	}{
		"stamped":                   {in: "annotations:\n  openeverest.io/contract-version: \"2.0.0\"\n", want: "2.0.0"},
		"no stamp":                  {in: "annotations:\n  a: b\n", want: ""},
		"no annotations":            {in: "name: monitoring-x\n", want: ""},
		"annotations not a mapping": {in: "annotations: [a]\n", want: ""},
		"invalid YAML":              {in: "name: [x\n", want: ""},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, chartStamp([]byte(tt.in)))
		})
	}
}
