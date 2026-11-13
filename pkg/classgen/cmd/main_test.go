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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/yaml"
)

const paramsPackage = "../testdata/params"

func TestRunPrintsSchema(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	require.NoError(t, run([]string{"--package", paramsPackage, "--type", "Tag"}, &stdout, &bytes.Buffer{}))

	out, ok := strings.CutPrefix(stdout.String(), header)
	require.True(t, ok, "output starts with the generated-code header")
	want := `
type: object
required: [name]
properties:
  name:
    type: string
    description: Name of the tag.
    pattern: ^[a-z]+$
`
	assert.YAMLEq(t, want, out)
}

func TestRunWritesOutput(t *testing.T) {
	t.Parallel()

	file := filepath.Join(t.TempDir(), "schema.yaml")
	var stdout bytes.Buffer
	require.NoError(t, run([]string{"-package", paramsPackage, "-type", "Parameters", "-output", file}, &stdout, &bytes.Buffer{}))
	assert.Empty(t, stdout.String())

	data, err := os.ReadFile(file) //nolint:gosec // the test's own file
	require.NoError(t, err)
	require.True(t, bytes.HasPrefix(data, []byte(header)))
	schema := &apiextensionsv1.JSONSchemaProps{}
	require.NoError(t, yaml.UnmarshalStrict(data, schema))
	assert.Equal(t, []string{"url"}, schema.Required)
}

func TestRunErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{name: "no package", args: []string{"--type", "Tag"}, wantErr: "--package and --type are required"},
		{name: "no type", args: []string{"--package", paramsPackage}, wantErr: "--package and --type are required"},
		{name: "extra argument", args: []string{"--package", paramsPackage, "--type", "Tag", "extra"}, wantErr: "unexpected arguments: extra"},
		{name: "unknown flag", args: []string{"--packages", paramsPackage}, wantErr: "flag provided but not defined"},
		{name: "unknown type", args: []string{"--package", paramsPackage, "--type", "Missing"}, wantErr: "type Missing not found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := run(tt.args, &bytes.Buffer{}, &bytes.Buffer{})
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}
