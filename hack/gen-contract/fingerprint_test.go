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
	contractPath = "contract/keys.go"
)

func testFingerprint(t *testing.T, files map[string]string) string {
	t.Helper()

	fp, err := fingerprint(writeRepo(t, files), inputs)
	require.NoError(t, err)
	return fp
}

func TestFingerprintIsStable(t *testing.T) {
	t.Parallel()

	fp := testFingerprint(t, testRepo())
	assert.Regexp(t, fingerprintPattern, fp)
	assert.Equal(t, fp, testFingerprint(t, testRepo()))
}

// TestFingerprintNormalisation checks that the fingerprint changes with the
// contract, and only with it.
func TestFingerprintNormalisation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(t *testing.T, files map[string]string)
		// changes is whether the mutation changes the contract.
		changes bool
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
		{name: "CRD property", changes: true, mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, crdPath, "              size:\n", "              color:\n                type: string\n              size:\n")
		}},
		{name: "CRD property named description", changes: true, mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, crdPath, "              description:\n                type: string\n                description: Description is free text.\n", "")
		}},
		{name: "CRD type", changes: true, mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, crdPath, "                type: integer\n", "                type: number\n")
		}},
		{name: "CRD maximum", changes: true, mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, crdPath, "maximum: 10", "maximum: 9")
		}},
		{name: "CRD default", changes: true, mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, crdPath, "default: 1", "default: 2")
		}},
		{name: "CRD validation rule", changes: true, mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, crdPath, "rule: has(self.size)", "rule: self.size > 0")
		}},
		{name: "CRD served", changes: true, mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, crdPath, "served: true", "served: false")
		}},
		{name: "CRD subresource", changes: true, mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, crdPath, "      status: {}\n", "      scale:\n        specReplicasPath: .spec.size\n        statusReplicasPath: .status.size\n      status: {}\n")
		}},
		{name: "CRD scope", changes: true, mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, crdPath, "scope: Namespaced", "scope: Cluster")
		}},
		{name: "CRD added", changes: true, mutate: func(t *testing.T, f map[string]string) {
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
		{name: "HTTP API operation", changes: true, mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, httpAPIPath, "operationId: listWidgets", "operationId: getWidgets")
		}},
		{name: "HTTP API parameter", changes: true, mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, httpAPIPath, "required: false", "required: true")
		}},
		{name: "HTTP API path", changes: true, mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, httpAPIPath, "  /widgets:\n", "  /gadgets:\n")
		}},
		{name: "HTTP API property named summary", changes: true, mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, httpAPIPath, "        summary:\n          type: string\n          description: A summary of the widget.\n", "")
		}},
		{name: "CRD schemas of the HTTP API", changes: true, mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, crdsGenPath, "type: integer", "type: string")
		}},

		// Built-in BackupStorageClasses.
		{name: "class presentation", mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, classPath, "displayName: S3", "displayName: Amazon S3")
			replace(t, f, classPath, "description: Amazon S3.", "description: Object storage.")
			replace(t, f, classPath, "description: Region of the bucket.", "description: The region.")
			replace(t, f, classPath, "components: {}", "components: {region: {uiType: text}}")
			replace(t, f, classPath, "message: parameters are required", "message: set the parameters")
			replace(t, f, classPath, "  name: s3\n", "  name: s3\n  annotations:\n    openeverest.io/contract-version: 2.0.0\n")
		}},
		{name: "class parameter", changes: true, mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, classPath, "          type: string\n", "          type: string\n          maxLength: 63\n")
		}},
		{name: "class validation", changes: true, mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, classPath, "expression: has(self.parameters)", "expression: has(self.parameters.region)")
		}},
		{name: "class credentials", changes: true, mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, classPath, "      - AWS_ACCESS_KEY_ID\n", "      - AWS_ACCESS_KEY_ID\n      - AWS_SESSION_TOKEN\n")
		}},
		{name: "class kustomization", mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			f["config/backupstorageclasses/kustomization.yaml"] = "resources: [s3.yaml]\n"
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
		{name: "Go signature", changes: true, mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, contractPath, "func Check(ext, minimum, current string) Result {", "func Check(ext, minimum, current string, strict bool) Result {\n\t_ = strict")
		}},
		{name: "Go constant value", changes: true, mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, contractPath, `"openeverest.io/contract-version"`, `"openeverest.io/contract"`)
		}},
		{name: "Go iota order", changes: true, mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, contractPath, "\tKindA Kind = iota\n\tKindB\n", "\tKindB Kind = iota\n\tKindA\n")
		}},
		{name: "Go exported field", changes: true, mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, contractPath, "\tReason string\n", "\tReason string\n\tMessage string\n")
		}},
		{name: "Go struct tag", changes: true, mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, contractPath, `json:"status"`, `json:"state"`)
		}},
		{name: "Go exported function", changes: true, mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, contractPath, "func helper() {}", "func Helper() {}")
		}},
		{name: "Go subpackage", changes: true, mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			f["contract/references/errors.go"] = "package references\n\nimport \"errors\"\n\nvar ErrNotFound = errors.New(\"not found\")\n"
		}},
		{name: "Go class package", changes: true, mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, "backupstorageclasses/s3/s3.go", `"s3"`, `"s3v2"`)
		}},

		// The log of contract changes.
		{name: "CHANGES spacing", mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			f[changesFile] = "\n" + strings.ReplaceAll(f[changesFile], "\n", "  \r\n\n")
		}},
		{name: "CHANGES entry", changes: true, mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			f[changesFile] += "#1234: provider webhooks must declare sideEffects None or NoneOnDryRun.\n"
		}},
		{name: "CHANGES preamble", changes: true, mutate: func(t *testing.T, f map[string]string) {
			t.Helper()
			replace(t, f, changesFile, "Changes", "Contract changes")
		}},
	}

	base := testFingerprint(t, testRepo())
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			files := cloneRepo(testRepo())
			tt.mutate(t, files)
			got := testFingerprint(t, files)
			if tt.changes {
				assert.NotEqual(t, base, got, "a change to the contract must change the fingerprint")
			} else {
				assert.Equal(t, base, got, "a change outside the contract must not change the fingerprint")
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
		{name: "no CRDs", wantErr: "config/crd/bases/*.yaml: no contract artifact found", mutate: func(f map[string]string) {
			delete(f, crdPath)
		}},
		{name: "no HTTP API", wantErr: "api/openapi/http-api.yaml: no contract artifact found", mutate: func(f map[string]string) {
			delete(f, httpAPIPath)
		}},
		{name: "no CHANGES", wantErr: "contract/CHANGES:", mutate: func(f map[string]string) {
			delete(f, changesFile)
		}},
		{name: "no Go package", wantErr: "backupstorageclasses/...:", mutate: func(f map[string]string) {
			delete(f, "backupstorageclasses/s3/s3.go")
		}},
		{name: "not a CRD", wantErr: "kind ConfigMap is not CustomResourceDefinition", mutate: func(f map[string]string) {
			f["config/crd/bases/cm.yaml"] = "apiVersion: v1\nkind: ConfigMap\n"
		}},
		{name: "two documents", wantErr: "holds more than one YAML document", mutate: func(f map[string]string) {
			f[crdPath] += "---\n" + testCRD
		}},
		{name: "empty file", wantErr: "holds no YAML document", mutate: func(f map[string]string) {
			f[crdPath] = "---\n"
		}},
		{name: "same CRD twice", wantErr: "two artifacts are named widgets.test.openeverest.io", mutate: func(f map[string]string) {
			f["config/crd/bases/copy.yaml"] = testCRD
		}},
		{name: "not OpenAPI", wantErr: "not an OpenAPI document", mutate: func(f map[string]string) {
			f[httpAPIPath] = "swagger: '2.0'\n"
		}},
		{name: "not a class", wantErr: "kind BackupClass is not BackupStorageClass", mutate: func(f map[string]string) {
			f[classPath] = strings.Replace(testBackupStorageClass, "kind: BackupStorageClass", "kind: BackupClass", 1)
		}},
		{name: "invalid Go", wantErr: "contract/keys.go", mutate: func(f map[string]string) {
			f[contractPath] = "package contract\n\nfunc {\n"
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			files := cloneRepo(testRepo())
			tt.mutate(files)
			_, err := fingerprint(writeRepo(t, files), inputs)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

// TestFingerprintOfThisRepository checks that the fingerprint covers every
// input in this repository, and that every input pattern names something.
func TestFingerprintOfThisRepository(t *testing.T) {
	t.Parallel()

	root := filepath.Join("..", "..")
	for _, in := range inputs {
		entries, err := in.entries(root, in.pattern)
		require.NoError(t, err, in.pattern)
		assert.NotEmpty(t, entries, in.pattern)
	}
	_, err := os.Stat(filepath.Join(root, filepath.FromSlash(changesFile)))
	require.NoError(t, err)
}
