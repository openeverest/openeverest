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
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openeverest/openeverest/v2/contract"
)

const fixtureCodeowners = "testdata/codeowners"

func TestParseBlock(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		want    []string
		wantErr string
	}{
		{
			name:    "block",
			content: "* @all\n# BEGIN contract-adjacent\n# A comment.\n\n/api/ @a @b\n  /contract/\t@a\n# END contract-adjacent\n/docs/ @docs\n",
			want:    []string{"/api/", "/contract/"},
		},
		{name: "no block", content: "* @all\n", wantErr: "no \"# BEGIN contract-adjacent\" block"},
		{name: "unterminated", content: "# BEGIN contract-adjacent\n/api/ @a\n", wantErr: "without \"# END contract-adjacent\""},
		{name: "end first", content: "# END contract-adjacent\n# BEGIN contract-adjacent\n", wantErr: "before"},
		{name: "two blocks", content: "# BEGIN contract-adjacent\n/a/ @a\n# END contract-adjacent\n# BEGIN contract-adjacent\n", wantErr: "more than one"},
		{name: "empty", content: "# BEGIN contract-adjacent\n# Nothing.\n# END contract-adjacent\n", wantErr: "lists no path"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseBlock(strings.NewReader(tt.content))
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMatcher(t *testing.T) {
	t.Parallel()

	tests := []struct {
		pattern string
		match   []string
		noMatch []string
	}{
		{
			pattern: "/contract/",
			match:   []string{"contract/VERSION", "contract/releases/2.0.0.yaml"},
			noMatch: []string{"contract", "contracts/x", "pkg/contract/x.go", "hack/gen-contract/main.go"},
		},
		{
			pattern: "/internal/server/middlewares.go",
			match:   []string{"internal/server/middlewares.go"},
			noMatch: []string{"internal/server/middlewares_test.go", "internal/server/middlewares.gox", "x/internal/server/middlewares.go"},
		},
		{
			pattern: "/api",
			match:   []string{"api", "api/openapi/http-api.yaml"},
			noMatch: []string{"api-tests/x.ts", "ui/api/index.ts"},
		},
		{
			pattern: "docs/",
			match:   []string{"docs/a.md", "ui/docs/b.md"},
			noMatch: []string{"docs", "mydocs/a.md"},
		},
		{
			pattern: "*.md",
			match:   []string{"README.md", "contract/README.md"},
			noMatch: []string{"README.mdx"},
		},
		{
			pattern: "/config/crd/bases/*.yaml",
			match:   []string{"config/crd/bases/a.yaml"},
			noMatch: []string{"config/crd/bases/a.go", "config/crd/a.yaml"},
		},
		{
			pattern: "/api/**/zz_generated.*",
			match:   []string{"api/zz_generated.deepcopy.go", "api/core/v1alpha1/zz_generated.deepcopy.go"},
			noMatch: []string{"api/core/types.go"},
		},
		{
			pattern: "/a.b?",
			match:   []string{"a.bc", "a.bc/d"},
			noMatch: []string{"axbc", "a.b", "a.bcd"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			t.Parallel()

			m, err := matcher(tt.pattern)
			require.NoError(t, err)
			for _, p := range tt.match {
				assert.True(t, m.MatchString(p), "%s should match %s", tt.pattern, p)
			}
			for _, p := range tt.noMatch {
				assert.False(t, m.MatchString(p), "%s should not match %s", tt.pattern, p)
			}
		})
	}

	for _, p := range []string{"/", "*", "/**"} {
		_, err := matcher(p)
		require.ErrorContains(t, err, "matches every path", p)
	}
}

func TestDecide(t *testing.T) {
	t.Parallel()

	patterns := []string{"/contract/", "/api/", "/internal/server/middlewares.go"}
	adjacent := []string{"README.md", "internal/server/middlewares.go"}
	tests := []struct {
		name        string
		baseVersion string
		headVersion string
		files       []string
		allAdjacent bool
		labels      []string
		wantFailure string
		wantNotice  string
	}{
		{
			name:        "base without a contract",
			headVersion: "2.0.0-alpha.1.next",
			files:       adjacent,
			wantNotice:  "no contract/VERSION",
		},
		{
			name:        "contract changed since the last pre-release",
			baseVersion: "2.0.0-alpha.1.next",
			headVersion: "2.0.0-alpha.1.next",
			files:       adjacent,
			wantNotice:  "has changed since the last release already",
		},
		{
			name:        "contract changed since the last release",
			baseVersion: "2.1.0-alpha.0",
			headVersion: "2.1.0-alpha.0",
			files:       adjacent,
			wantNotice:  "no decision is needed",
		},
		{
			name:        "nothing adjacent",
			baseVersion: "2.0.0",
			headVersion: "2.0.0",
			files:       []string{"README.md", "ui/api/index.ts", "pkg/cli/x.go"},
			wantNotice:  "touches no contract-adjacent path.",
		},
		{
			name:        "only Markdown on adjacent paths",
			baseVersion: "2.0.0",
			headVersion: "2.0.0",
			files:       []string{"contract/MAINTAINING.md", "api/README.MD", "pkg/cli/x.go"},
			wantNotice:  "touches no contract-adjacent path but Markdown files (contract/MAINTAINING.md, api/README.MD), which are never part of the contract.",
		},
		{
			name:        "Markdown and code on adjacent paths",
			baseVersion: "2.0.0",
			headVersion: "2.0.0",
			files:       []string{"contract/MAINTAINING.md", "internal/server/middlewares.go"},
			wantFailure: "touches contract-adjacent paths (internal/server/middlewares.go), and the contract is unchanged",
		},
		{
			name:        "Markdown-like names that are not Markdown",
			baseVersion: "2.0.0",
			headVersion: "2.0.0",
			files:       []string{"contract/vectors.md.json", "api/md"},
			wantFailure: "touches contract-adjacent paths (contract/vectors.md.json, api/md)",
		},
		{
			name:        "no label",
			baseVersion: "2.0.0",
			headVersion: "2.0.0",
			files:       adjacent,
			labels:      []string{"kind/bug"},
			wantFailure: "touches contract-adjacent paths (internal/server/middlewares.go), and the contract is unchanged since the last release (contract/VERSION is 2.0.0 on the base branch). Add exactly one label",
		},
		{
			name:        "both labels",
			baseVersion: "2.0.0",
			headVersion: "2.1.0-alpha.0",
			files:       adjacent,
			labels:      []string{labelChange, labelNoChange},
			wantFailure: "Add exactly one label",
		},
		{
			name:        "no contract change",
			baseVersion: "2.0.0",
			headVersion: "2.0.0",
			files:       adjacent,
			labels:      []string{labelNoChange},
			wantNotice:  "Labelled no-contract-change; touches internal/server/middlewares.go.",
		},
		{
			name:        "contract change recorded",
			baseVersion: "2.0.0-alpha.2",
			headVersion: "2.0.0-alpha.2.next",
			files:       adjacent,
			labels:      []string{labelChange},
			wantNotice:  "Labelled contract-change",
		},
		{
			name:        "contract change not recorded",
			baseVersion: "2.0.0",
			headVersion: "2.0.0",
			files:       adjacent,
			labels:      []string{labelChange},
			wantFailure: "labelled contract-change, but contract/VERSION is still 2.0.0. Record the change in contract/CHANGES",
		},
		{
			name:        "contract change denied",
			baseVersion: "2.0.0",
			headVersion: "2.1.0-alpha.0",
			files:       []string{"api/openapi/http-api.yaml"},
			labels:      []string{labelNoChange},
			wantFailure: "labelled no-contract-change, but it changes the contract: a contract artifact or contract/CHANGES changed, so contract/VERSION moves to 2.1.0-alpha.0. Label it contract-change instead.",
		},
		{
			name:        "too many paths to list",
			baseVersion: "2.0.0",
			headVersion: "2.0.0",
			allAdjacent: true,
			wantFailure: "touches contract-adjacent paths (every path, since they could not be listed)",
		},
		{
			name:        "many adjacent paths",
			baseVersion: "2.0.0",
			headVersion: "2.0.0",
			files:       []string{"api/1", "api/2", "api/3", "api/4", "api/5", "api/6", "api/7", "api/8", "api/9", "api/10", "api/11", "api/12"},
			wantFailure: "(api/1, api/2, api/3, api/4, api/5, api/6, api/7, api/8, api/9, api/10 and 2 more)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			d := decide(input{
				baseVersion: tt.baseVersion,
				headVersion: tt.headVersion,
				patterns:    patterns,
				files:       tt.files,
				allAdjacent: tt.allAdjacent,
				labels:      tt.labels,
			})
			if tt.wantFailure != "" {
				assert.Contains(t, d.failure, tt.wantFailure)
				return
			}
			assert.Empty(t, d.failure)
			require.Len(t, d.notices, 1)
			assert.Contains(t, d.notices[0], tt.wantNotice)
		})
	}
}

// TestRun runs the command on the fixtures, as the workflow does.
func TestRun(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		args     []string
		stdin    string
		actions  bool
		wantCode int
		wantOut  string
	}{
		{
			name:     "unrelated",
			args:     []string{"-base-version", "2.0.0\n", "-head-version", "2.0.0\n", "-files", "testdata/files-unrelated.txt"},
			wantCode: 0,
			wantOut:  "notice: This pull request touches no contract-adjacent path.\n",
		},
		{
			name:     "middleware without a label",
			args:     []string{"-base-version", "2.0.0", "-head-version", "2.0.0", "-files", "testdata/files-middleware.txt", "-label", "kind/bug"},
			actions:  true,
			wantCode: 1,
			wantOut:  "::error title=Contract decision::This pull request touches contract-adjacent paths (internal/server/middlewares.go)",
		},
		{
			name:     "middleware without a contract change",
			args:     []string{"-base-version", "2.0.0", "-head-version", "2.0.0", "-files", "testdata/files-middleware.txt", "-label", "kind/bug", "-label", labelNoChange},
			wantCode: 0,
			wantOut:  "notice: Labelled no-contract-change; touches internal/server/middlewares.go.\n",
		},
		{
			name:     "API change from stdin",
			args:     []string{"-base-version", "2.0.0", "-head-version", "2.1.0-alpha.0", "-label", labelChange},
			stdin:    "api/core/v1alpha1/instance_types.go\ncontract/VERSION\n",
			wantCode: 0,
			wantOut:  "notice: Labelled contract-change; touches api/core/v1alpha1/instance_types.go, contract/VERSION.\n",
		},
		{
			name:     "API change denied",
			args:     []string{"-base-version", "2.0.0", "-head-version", "2.1.0-alpha.0", "-files", "testdata/files-api.txt", "-label", labelNoChange},
			wantCode: 1,
			wantOut:  "error: This pull request is labelled no-contract-change, but it changes the contract",
		},
		{
			name:     "missing CODEOWNERS",
			args:     []string{"-codeowners", "testdata/missing", "-base-version", "2.0.0"},
			wantCode: 1,
			wantOut:  "error: open testdata/missing",
		},
		{
			name:     "missing list of paths",
			args:     []string{"-base-version", "2.0.0", "-files", "testdata/missing"},
			wantCode: 1,
			wantOut:  "error: open testdata/missing",
		},
		{
			name:     "unknown flag",
			args:     []string{"-base", "2.0.0"},
			wantCode: 2,
			wantOut:  "flag provided but not defined: -base",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var out bytes.Buffer
			args := tt.args
			if !strings.HasPrefix(strings.Join(args, " "), "-codeowners") {
				args = append([]string{"-codeowners", fixtureCodeowners}, args...)
			}
			code := run(args, strings.NewReader(tt.stdin), &out, tt.actions)
			assert.Equal(t, tt.wantCode, code, out.String())
			assert.Contains(t, out.String(), tt.wantOut)
		})
	}
}

// TestIsDevelopment checks the rule against the contract package's, which
// this command does not import to stay quick to build.
func TestIsDevelopment(t *testing.T) {
	t.Parallel()

	var vectors struct {
		Order       []string `json:"order"`
		Development []struct {
			Version string `json:"version"`
		} `json:"development"`
	}
	require.NoError(t, json.Unmarshal(contract.OrderVectors, &vectors))
	versions := slices.Clone(vectors.Order)
	for _, d := range vectors.Development {
		versions = append(versions, d.Version)
	}
	for _, v := range versions {
		// It reads contract/VERSION, which gen-contract writes valid.
		if contract.Valid(v) {
			assert.Equal(t, contract.IsDevelopment(v), isDevelopment(v), v)
		}
	}
	assert.True(t, isDevelopment("2.0.0-alpha.1.next"))
	assert.True(t, isDevelopment("2.1.0-alpha.0"))
	assert.False(t, isDevelopment("2.0.0-dev.4"))
	assert.False(t, isDevelopment("2.0.0"))
}

// TestRepositoryBlock checks the contract-adjacent block of this
// repository: every path it lists exists, and it covers every artifact the
// contract's fingerprint covers, so no change to the contract can skip the
// decision.
func TestRepositoryBlock(t *testing.T) {
	t.Parallel()

	root := filepath.Join("..", "..")
	patterns, err := readBlock(filepath.Join(root, ".github", "CODEOWNERS"))
	require.NoError(t, err)
	for _, p := range patterns {
		require.True(t, strings.HasPrefix(p, "/"), "%s: anchor contract-adjacent paths at the root", p)
		require.NotContains(t, p, "*", "%s: list contract-adjacent paths without wildcards", p)
		_, err := os.Stat(filepath.Join(root, filepath.FromSlash(p)))
		require.NoError(t, err, "%s is listed as contract-adjacent", p)
	}

	d := decide(input{
		baseVersion: "2.0.0",
		headVersion: "2.0.0",
		patterns:    patterns,
		files: []string{
			"contract/CHANGES",
			"contract/vectors.json",
			"contract/releases/2.0.0.yaml",
			"config/crd/bases/core.openeverest.io_instances.yaml",
			"api/openapi/http-api.yaml",
			"api/openapi/crds.gen.yaml",
			"hack/gen-contract/fingerprint.go",
		},
	})
	assert.Contains(t, d.failure, "(contract/CHANGES, contract/vectors.json, contract/releases/2.0.0.yaml, config/crd/bases/core.openeverest.io_instances.yaml, "+
		"api/openapi/http-api.yaml, api/openapi/crds.gen.yaml, hack/gen-contract/fingerprint.go)")
}
