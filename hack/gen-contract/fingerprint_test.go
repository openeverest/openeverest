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
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	crdPath      = "config/crd/bases/test.openeverest.io_widgets.yaml"
	httpAPIPath  = "api/openapi/http-api.yaml"
	crdsGenPath  = "api/openapi/crds.gen.yaml"
	classPath    = "config/backupstorageclasses/s3.yaml"
	classGoPath  = "backupstorageclasses/s3/s3.go"
	contractPath = "contract/keys.go"
	vectorsPath  = "contract/vectors.json"
)

func testFingerprints(t *testing.T, files map[string]string) map[string]string {
	t.Helper()

	fps, err := fingerprints(writeRepo(t, files))
	require.NoError(t, err)
	return fps
}

func TestFingerprintsAreStable(t *testing.T) {
	t.Parallel()

	fps := testFingerprints(t, testRepo())
	assert.Len(t, fps, len(inputs))
	for name, d := range fps {
		assert.Regexp(t, digestPattern, d, name)
	}
	assert.Equal(t, fps, testFingerprints(t, testRepo()))
}

func TestFingerprintNormalisation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(t *testing.T, files map[string]string)
		// input is the input whose digest the mutation changes, if any.
		input string
	}{
		// CRDs.
		{name: "CRD description", mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, crdPath, "Size of the widget.", "How big the widget is.")
		}},
		{name: "CRD example", mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, crdPath, "example: 3", "example: 4")
		}},
		{name: "CRD printer column", mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, crdPath, "      name: Size\n", "      name: Bigness\n")
		}},
		{name: "CRD validation message", mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, crdPath, "message: size must be set", "messageExpression: \"'size is ' + 'required'\"")
		}},
		{name: "CRD metadata", mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, crdPath, "controller-gen.kubebuilder.io/version: v0.18.0",
				"controller-gen.kubebuilder.io/version: v0.19.0\n    openeverest.io/contract-version: 2.0.0",
			)
		}},
		{name: "CRD file name", mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			f["config/crd/bases/widgets.yaml"] = f[crdPath]
			delete(f, crdPath)
		}},
		{name: "CRD layout and line endings", mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, crdPath, "      status: {}\n", "      status:\n        {}\n")
			replace(t, f, crdPath, "  scope: Namespaced\n", "")
			replace(t, f, crdPath, "  group: test.openeverest.io\n", "  scope: \"Namespaced\"\n  group: test.openeverest.io\n")
			f[crdPath] = strings.ReplaceAll(f[crdPath], "\n", "\r\n")
		}},
		{name: "CRD property", input: "crds", mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, crdPath, "              size:\n", "              color:\n                type: string\n              size:\n")
		}},
		{name: "CRD property named description", input: "crds", mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, crdPath, "              description:\n                type: string\n                description: Description is free text.\n", "")
		}},
		{name: "CRD type", input: "crds", mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, crdPath, "                type: integer\n", "                type: number\n")
		}},
		{name: "CRD maximum", input: "crds", mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, crdPath, "maximum: 10", "maximum: 9")
		}},
		{name: "CRD default", input: "crds", mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, crdPath, "default: 1", "default: 2")
		}},
		{name: "CRD validation rule", input: "crds", mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, crdPath, "rule: has(self.size)", "rule: self.size > 0")
		}},
		{name: "CRD served", input: "crds", mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, crdPath, "served: true", "served: false")
		}},
		{name: "CRD subresource", input: "crds", mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, crdPath, "      status: {}\n", "      scale:\n        specReplicasPath: .spec.size\n        statusReplicasPath: .status.size\n      status: {}\n")
		}},
		{name: "CRD scope", input: "crds", mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, crdPath, "scope: Namespaced", "scope: Cluster")
		}},
		{name: "CRD added", input: "crds", mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			f["config/crd/bases/test.openeverest.io_gizmos.yaml"] = strings.NewReplacer("widget", "gizmo", "Widget", "Gizmo").Replace(f[crdPath])
		}},

		// The HTTP API.
		{name: "HTTP API documentation", mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, httpAPIPath, "summary: List widgets", "summary: Widgets")
			replace(t, f, httpAPIPath, "description: Lists every widget.", "description: Lists all widgets.")
			replace(t, f, httpAPIPath, "description: The test API.", "description: An API.")
			replace(t, f, httpAPIPath, "description: Successful operation", "description: OK")
			replace(t, f, httpAPIPath, "example: 10", "example: 20")
			replace(t, f, httpAPIPath, "value: {name: a}", "value: {name: b}")
			replace(t, f, httpAPIPath, "description: A summary of the widget.", "description: What the widget is.")
		}},
		{name: "HTTP API operation", input: "http-api", mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, httpAPIPath, "operationId: listWidgets", "operationId: getWidgets")
		}},
		{name: "HTTP API parameter", input: "http-api", mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, httpAPIPath, "required: false", "required: true")
		}},
		{name: "HTTP API path", input: "http-api", mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, httpAPIPath, "  /widgets:\n", "  /gadgets:\n")
		}},
		{name: "HTTP API property named summary", input: "http-api", mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, httpAPIPath, "        summary:\n          type: string\n          description: A summary of the widget.\n", "")
		}},
		{name: "CRD schemas of the HTTP API", input: "http-api", mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, crdsGenPath, "type: integer", "type: string")
		}},

		// A storage class this repository ships is an extension, which no
		// input covers.
		{name: "storage class", mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, classPath, "          type: string\n", "          type: string\n          maxLength: 63\n")
			replace(t, f, classPath, "expression: has(self.parameters)", "expression: has(self.parameters.region)")
			replace(t, f, classPath, "      - AWS_ACCESS_KEY_ID\n", "      - AWS_ACCESS_KEY_ID\n      - AWS_SESSION_TOKEN\n")
			replace(t, f, classGoPath, `"s3"`, `"s3v2"`)
		}},

		// Go identifiers.
		{name: "Go documentation", mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, contractPath, "// Check checks.", "// Check reports whether an extension is supported.")
			replace(t, f, contractPath, "\tReason string\n", "\t// Reason says why.\n\tReason string // not Status\n")
		}},
		{name: "Go layout, parameter names and quoting", mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, contractPath, "func Check(ext, minimum, current string) Result {\n\treturn Result{Reason: fmt.Sprint(ext, minimum, current, internalKey)}",
				"func Check(\n\tstamp string,\n\tlow, high string,\n) Result {\n\treturn Result{Reason: fmt.Sprint(stamp, low, high, internalKey)}",
			)
			replace(t, f, contractPath, `"openeverest.io/contract-version"`, "`openeverest.io/contract-version`")
			f[contractPath] = strings.ReplaceAll(f[contractPath], "\n", "\r\n")
		}},
		{name: "Go unexported identifiers and bodies", mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, contractPath, "func helper() {}", "func helper() int { return 1 }\n\nfunc other() {}")
			replace(t, f, contractPath, "\tcache  string\n", "\tcache  int\n\tmore   bool\n")
			replace(t, f, contractPath, `const internalKey = "x"`, `const internalKey = "y"`)
			f["contract/more_test.go"] = "package contract\n\nfunc TestMore() {}\n"
			f["contract/internal/x.go"] = "package internal\n\nfunc unexported() {}\n"
		}},
		{name: "Go generated window", mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			f["contract/"+outputFile] = "package contract\n\nconst Version = \"9.9.9\"\n"
		}},
		{name: "Go test data", mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			f["contract/testdata/x.go"] = "package x\n\nfunc Exported() {}\n"
		}},
		{name: "Go signature", input: "go-api", mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, contractPath, "func Check(ext, minimum, current string) Result {", "func Check(ext, minimum, current string, strict bool) Result {\n\t_ = strict")
		}},
		{name: "Go constant value", input: "go-api", mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, contractPath, `"openeverest.io/contract-version"`, `"openeverest.io/contract"`)
		}},
		{name: "Go iota order", input: "go-api", mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, contractPath, "\tKindA Kind = iota\n\tKindB\n", "\tKindB Kind = iota\n\tKindA\n")
		}},
		{name: "Go exported field", input: "go-api", mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, contractPath, "\tReason string\n", "\tReason string\n\tMessage string\n")
		}},
		{name: "Go struct tag", input: "go-api", mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, contractPath, `json:"status"`, `json:"state"`)
		}},
		{name: "Go exported function", input: "go-api", mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, contractPath, "func helper() {}", "func Helper() {}")
		}},
		{name: "Go module path", input: "go-api", mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, goModFile, "openeverest/v2", "openeverest/v3")
		}},
		{name: "Go module requirements", mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			f[goModFile] += "\nrequire github.com/blang/semver/v4 v4.0.0\n"
		}},
		{name: "Go subpackage", input: "go-api", mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			f["contract/references/errors.go"] = "package references\n\nimport \"errors\"\n\nvar ErrNotFound = errors.New(\"not found\")\n"
		}},

		// The conformance vectors test the rules, which no artifact shows:
		// a case added or changed changes no digest.
		{name: "vectors", mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, vectorsPath, `"reason": "Supported"`, `"reason": "CoreTooOld"`)
			replace(t, f, vectorsPath, `["2.0.0-alpha.1", "2.0.0"]`, `["2.0.0-alpha.1", "2.0.0", "2.1.0"]`)
		}},

		// The log of contract changes.
		{name: "CHANGES spacing", mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			f[changesFile] = "\n" + strings.ReplaceAll(f[changesFile], "\n", "  \r\n\n")
		}},
		{name: "CHANGES comments", mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, changesFile, "# Changes no artifact shows.", "# Contract changes no artifact shows.\n#")
		}},
		{name: "CHANGES missing", mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			delete(f, changesFile)
		}},
		{name: "CHANGES entry", input: "changes", mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			f[changesFile] += "- #1234: provider webhooks must declare sideEffects None or NoneOnDryRun.\n"
		}},
	}

	base := testFingerprints(t, testRepo())
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			files := cloneRepo(testRepo())
			tt.mutate(t, files)
			got := testFingerprints(t, files)
			for _, name := range inputNames() {
				if name == tt.input {
					assert.NotEqual(t, base[name], got[name], "a change to %s must change its digest", name)
				} else {
					assert.Equal(t, base[name], got[name], "a change outside %s must not change its digest", name)
				}
			}
		})
	}
}

func TestFingerprintErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(files map[string]string)
		wantErr string
	}{
		{name: "not a CRD", wantErr: "crds: config/crd/bases/*.yaml: config/crd/bases/cm.yaml: kind ConfigMap is not CustomResourceDefinition", mutate: func(f map[string]string) {
			f["config/crd/bases/cm.yaml"] = "apiVersion: v1\nkind: ConfigMap\n"
		}},
		{name: "two documents", wantErr: "holds more than one YAML document", mutate: func(f map[string]string) {
			f[crdPath] += "---\n" + testCRD
		}},
		{name: "empty file", wantErr: "holds no YAML document", mutate: func(f map[string]string) {
			f[crdPath] = "---\n"
		}},
		{name: "same CRD twice", wantErr: "crds: two artifacts are named widgets.test.openeverest.io", mutate: func(f map[string]string) {
			f["config/crd/bases/copy.yaml"] = testCRD
		}},
		{name: "not OpenAPI", wantErr: "not an OpenAPI document", mutate: func(f map[string]string) {
			f[httpAPIPath] = "swagger: '2.0'\n"
		}},
		{name: "invalid Go", wantErr: "contract/keys.go:3:6: expected 'IDENT'", mutate: func(f map[string]string) {
			f[contractPath] = "package contract\n\nfunc {\n"
		}},
		{name: "CHANGES entry without its bullet", wantErr: `changes: contract/CHANGES: line 4 is neither a comment`, mutate: func(f map[string]string) {
			f[changesFile] += "#1234: a change.\n"
		}},
		{name: "Go packages without a module", wantErr: "go-api: contract/...: the import path of contract: open ", mutate: func(f map[string]string) {
			delete(f, goModFile)
		}},
		{name: "go.mod without a module", wantErr: "the import path of contract: go.mod declares no module", mutate: func(f map[string]string) {
			f[goModFile] = "go 1.27\n"
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			files := cloneRepo(testRepo())
			tt.mutate(files)
			_, err := fingerprints(writeRepo(t, files))
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

// TestEntryKeys checks that every entry is keyed by what it is, not by the
// path of its file, so moving a file changes no digest once its part names
// the new path; only a Go package is keyed by where it is, its import
// path, which is part of its API.
func TestEntryKeys(t *testing.T) {
	t.Parallel()

	root := writeRepo(t, testRepo())
	want := map[string][]string{
		"crds":     {"widgets.test.openeverest.io"},
		"http-api": {"crd-schemas", "http-api"},
		"go-api":   {"github.com/openeverest/openeverest/v2/contract"},
		"changes":  {changesKey},
	}
	for _, in := range inputs {
		entries, err := in.entries(root)
		require.NoError(t, err)
		var keys []string
		for _, e := range entries {
			keys = append(keys, e.key)
		}
		assert.Equal(t, want[in.name], keys, in.name)
	}

	// The same artifacts at other paths.
	moved := map[string]string{goModFile: testGoMod}
	for from, to := range map[string]string{
		crdPath:     "config/crd/test.yaml",
		httpAPIPath: "api/http-api.yaml",
		crdsGenPath: "api/crds.gen.yaml",
		changesFile: "CHANGES",
	} {
		moved[to] = testRepo()[from]
	}
	movedRoot := writeRepo(t, moved)
	for _, tt := range []struct {
		entries  func(root, pattern string) ([]entry, error)
		from, to string
	}{
		{entries: yamlEntries(canonicalCRD), from: crdPath, to: "config/crd/*.yaml"},
		{entries: yamlEntries(canonicalOpenAPI("http-api")), from: httpAPIPath, to: "api/http-api.yaml"},
		{entries: yamlEntries(canonicalOpenAPI("crd-schemas")), from: crdsGenPath, to: "api/crds.gen.yaml"},
		{entries: changesEntries, from: changesFile, to: "CHANGES"},
	} {
		before, err := tt.entries(root, tt.from)
		require.NoError(t, err)
		after, err := tt.entries(movedRoot, tt.to)
		require.NoError(t, err)
		assert.Equal(t, stream(before), stream(after), "%s moved to %s", tt.from, tt.to)
	}
}

// TestAbsentInputs checks the inputs a tree can lack, as a tag that
// predates them does: an input none of whose artifacts exist has no
// digest, and a missing CHANGES is the empty log, never absent.
func TestAbsentInputs(t *testing.T) {
	t.Parallel()

	base := testFingerprints(t, testRepo())
	tests := []struct {
		name       string
		delete     []string
		wantAbsent string
	}{
		{name: "CRDs", delete: []string{crdPath}, wantAbsent: "crds"},
		{name: "HTTP API", delete: []string{httpAPIPath, crdsGenPath}, wantAbsent: "http-api"},
		{name: "Go API", delete: []string{contractPath}, wantAbsent: "go-api"},
		{name: "CHANGES", delete: []string{changesFile}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			files := cloneRepo(testRepo())
			for _, f := range tt.delete {
				require.Contains(t, files, f)
				delete(files, f)
			}
			got := testFingerprints(t, files)
			for _, name := range inputNames() {
				if name == tt.wantAbsent {
					assert.NotContains(t, got, name)
				} else {
					assert.Equal(t, base[name], got[name], name)
				}
			}
		})
	}

	// One part of an input is enough for it to be present.
	files := cloneRepo(testRepo())
	delete(files, httpAPIPath)
	got := testFingerprints(t, files)
	assert.Contains(t, got, "http-api")
	assert.NotEqual(t, base["http-api"], got["http-api"])
}

func TestParseChanges(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		content string
		want    []string
		wantErr string
	}{
		{name: "empty", content: ""},
		{name: "comments only", content: "# A comment.\n#\n\n# - #1: commented out.\n"},
		{
			name:    "entries",
			content: "# Header.\n- #12: first.  \r\n\n- #3: second, with # inside.\n",
			want:    []string{"#12: first.", "#3: second, with # inside."},
		},
		{name: "prose", content: "# Header.\nA change.\n", wantErr: `line 2 is neither a comment ("# ...") nor an entry`},
		{name: "entry without bullet", content: "#12: a change.\n", wantErr: "line 1"},
		{name: "entry without pull request", content: "- a change.\n", wantErr: "line 1"},
		{name: "pull request zero", content: "- #0: a change.\n", wantErr: "line 1"},
		{name: "empty entry", content: "- #12: \n", wantErr: "line 1"},
		{name: "indented entry", content: "  - #12: a change.\n", wantErr: "line 1"},
		{name: "comment without space", content: "#comment\n", wantErr: "line 1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseChanges([]byte(tt.content))
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestStream checks the canonical stream of an input, which a digest
// hashes and diff -raw prints.
func TestStream(t *testing.T) {
	t.Parallel()

	got := stream([]entry{{key: "a", data: []byte("x\ny")}, {key: "b"}})
	assert.Equal(t, "a 3\nx\ny\nb 0\n\n", string(got))
	assert.Equal(t, "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", digest(nil))
}

// TestFingerprintOfThisRepository checks that every input is present in
// this repository, so a pattern that names nothing cannot drop an input.
func TestFingerprintOfThisRepository(t *testing.T) {
	t.Parallel()

	fps, err := fingerprints(filepath.Join("..", ".."))
	require.NoError(t, err)
	assert.Equal(t, inputNames(), sortedInputs(fps))
	_, err = os.Stat(filepath.Join("..", "..", filepath.FromSlash(changesFile)))
	require.NoError(t, err)
}

// contractAdjacentPaths returns the paths of the contract-adjacent block of
// this repository's CODEOWNERS, relative to the root, without a trailing
// slash.
func contractAdjacentPaths(t *testing.T) []string {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("..", "..", ".github", "CODEOWNERS"))
	require.NoError(t, err)
	block := string(data)
	start := strings.Index(block, "# BEGIN contract-adjacent")
	end := strings.Index(block, "# END contract-adjacent")
	require.True(t, start >= 0 && end > start, "CODEOWNERS has no contract-adjacent block")
	var paths []string
	for line := range strings.Lines(block[start:end]) {
		if line = strings.TrimSpace(line); line != "" && !strings.HasPrefix(line, "#") {
			paths = append(paths, strings.Trim(strings.Fields(line)[0], "/"))
		}
	}
	return paths
}

// TestInputsAreContractAdjacent checks that the paths of every input are
// in the contract-adjacent block of CODEOWNERS, so a pull request that
// changes them needs a contract decision and the owners of the contract.
func TestInputsAreContractAdjacent(t *testing.T) {
	t.Parallel()

	paths := contractAdjacentPaths(t)
	for _, prefix := range inputPrefixes() {
		covered := false
		for _, p := range paths {
			covered = covered || prefix == p || strings.HasPrefix(prefix, p+"/")
		}
		assert.True(t, covered, "%s is an input of the fingerprint but not in the contract-adjacent block of .github/CODEOWNERS", prefix)
	}
}

// markdownNames are names of Markdown files a glob must not match.
//
//nolint:gochecknoglobals // immutable test table
var markdownNames = []string{"x.md", "README.md", "MAINTAINING.md", "CHANGES.md", "X.MD"}

// couldMatchMarkdown reports whether a glob, whose last element names
// files, could match a Markdown file.
func couldMatchMarkdown(glob string) bool {
	if strings.HasSuffix(strings.ToLower(glob), ".md") {
		return true
	}
	for _, name := range markdownNames {
		if ok, _ := path.Match(path.Base(glob), name); ok {
			return true
		}
	}
	return false
}

// TestNoMarkdownIsContract keeps documentation out of the contract, which
// lets contract-decision drop Markdown files from what a pull request
// touches: no fingerprint input reads a Markdown file, and no package
// under a contract-adjacent path embeds one, so a change to one cannot
// change the contract.
func TestNoMarkdownIsContract(t *testing.T) {
	t.Parallel()

	// No pattern of a file names a Markdown file, and one next to, or
	// below, every path an input reads, a directory of Go packages
	// included, changes no digest.
	for _, in := range inputs {
		for _, p := range in.parts {
			if !strings.HasSuffix(p.pattern, "/...") {
				assert.False(t, couldMatchMarkdown(p.pattern), "%s: %s could match a Markdown file", in.name, p.pattern)
			}
		}
	}
	files := testRepo()
	for _, prefix := range inputPrefixes() {
		dir := prefix
		if _, isFile := files[prefix]; isFile {
			dir = path.Dir(prefix)
		}
		for _, name := range markdownNames {
			files[path.Join(dir, name)] = "# Not part of the contract.\n"
			files[path.Join(dir, "docs", name)] = "# Not part of the contract.\n"
		}
	}
	assert.Equal(t, testFingerprints(t, testRepo()), testFingerprints(t, files), "a Markdown file changed a digest")

	// No package under a contract-adjacent path embeds a Markdown file, or
	// a directory, which could hold one.
	root := filepath.Join("..", "..")
	for _, adjacent := range contractAdjacentPaths(t) {
		err := filepath.WalkDir(filepath.Join(root, filepath.FromSlash(adjacent)), func(file string, d fs.DirEntry, err error) error {
			switch {
			case err != nil:
				return err
			case d.IsDir() && (d.Name() == "testdata" || d.Name() == "vendor" || d.Name() == "node_modules"):
				return filepath.SkipDir
			case d.IsDir() || !strings.HasSuffix(file, ".go"):
				return nil
			}
			for _, pattern := range embedPatterns(t, file) {
				assert.False(t, couldMatchMarkdown(pattern), "%s: //go:embed %s could match a Markdown file", file, pattern)
				matches, err := filepath.Glob(filepath.Join(filepath.Dir(file), filepath.FromSlash(pattern)))
				require.NoError(t, err)
				for _, m := range matches {
					info, err := os.Stat(m)
					require.NoError(t, err)
					assert.False(t, info.IsDir(), "%s: //go:embed %s embeds the directory %s, which could hold a Markdown file", file, pattern, m)
				}
			}
			return nil
		})
		if !errors.Is(err, fs.ErrNotExist) {
			require.NoError(t, err, adjacent)
		}
	}
}

// embedPatterns returns the patterns of the //go:embed directives of a Go
// file, without the all: prefix.
func embedPatterns(t *testing.T, file string) []string {
	t.Helper()

	data, err := os.ReadFile(file) //nolint:gosec // a Go file of this repository
	require.NoError(t, err)
	var patterns []string
	for line := range strings.Lines(string(data)) {
		args, ok := strings.CutPrefix(strings.TrimSpace(line), "//go:embed")
		if !ok || (args != "" && args[0] != ' ' && args[0] != '\t') {
			continue
		}
		for f := range strings.FieldsSeq(args) {
			if f[0] == '"' || f[0] == '`' {
				unquoted, err := strconv.Unquote(f)
				require.NoError(t, err, "%s: //go:embed %s", file, args)
				f = unquoted
			}
			patterns = append(patterns, strings.TrimPrefix(f, "all:"))
		}
	}
	return patterns
}
