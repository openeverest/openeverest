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

package s3_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/yaml"

	"github.com/openeverest/openeverest/v2/backupstorageclasses/s3"
)

// TestGeneratedClassMatchesTypes guards against the generated class drifting
// from the type executors decode with. Run make gen after changing a type.
func TestGeneratedClassMatchesTypes(t *testing.T) {
	t.Parallel()

	schema := s3.ParametersSchema()
	var fields, required []string
	for f := range reflect.TypeFor[s3.Parameters]().Fields() {
		tag := strings.Split(f.Tag.Get("json"), ",")
		fields = append(fields, tag[0])
		if !slices.Contains(tag[1:], "omitempty") {
			required = append(required, tag[0])
		}
	}
	props := make([]string, 0, len(schema.Properties))
	for p := range schema.Properties {
		props = append(props, p)
	}
	assert.ElementsMatch(t, fields, props, "schema properties")
	assert.ElementsMatch(t, required, schema.Required, "required parameters")

	values := schema.Properties["implementation"].Enum
	enum := make([]s3.Implementation, 0, len(values))
	for _, v := range values {
		enum = append(enum, s3.Implementation(strings.Trim(string(v.Raw), `"`)))
	}
	want := []s3.Implementation{
		s3.ImplementationAWS, s3.ImplementationMinIO, s3.ImplementationCeph, s3.ImplementationGCS, s3.ImplementationOther,
	}
	assert.Equal(t, want, enum)
}

func TestGeneratedClass(t *testing.T) {
	t.Parallel()

	type authMode struct {
		Name        string `json:"name"`
		Credentials struct {
			Required []string `json:"required"`
		} `json:"credentials"`
	}
	var class struct {
		Kind     string `json:"kind"`
		Metadata struct {
			Name string `json:"name"`
		} `json:"metadata"`
		Spec struct {
			AuthModes []authMode `json:"authModes"`
		} `json:"spec"`
	}
	require.NoError(t, yaml.Unmarshal(s3.Manifest(), &class))
	assert.Equal(t, "BackupStorageClass", class.Kind)
	assert.Equal(t, s3.ClassName, class.Metadata.Name)
	want := authMode{Name: s3.AuthModeAccessKey}
	want.Credentials.Required = []string{s3.KeyAWSAccessKeyID, s3.KeyAWSSecretAccessKey}
	assert.Equal(t, []authMode{want}, class.Spec.AuthModes)
}

// TestParameterFieldsAreConnectionParameters checks the rules every class
// parameter type follows: each field carries +openeverest:param=connection,
// none is typed on BackupStorage instead, and none has a CEL rule, which
// parameter validation cannot evaluate.
func TestParameterFieldsAreConnectionParameters(t *testing.T) {
	t.Parallel()

	// Typed on BackupStorage, never class parameters.
	forbidden := []string{"bucket", "prefix", "tls", "ca", "caBundle", "verifyTLS", "insecureSkipVerify", "insecureSkipTLSVerify"}
	fset := token.NewFileSet()
	entries, err := os.ReadDir(".")
	require.NoError(t, err)
	var st *ast.StructType
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, e.Name(), nil, parser.ParseComments)
		require.NoError(t, err)
		ast.Inspect(file, func(n ast.Node) bool {
			if ts, ok := n.(*ast.TypeSpec); ok && ts.Name.Name == "Parameters" {
				st, _ = ts.Type.(*ast.StructType)
			}
			return true
		})
	}
	require.NotNil(t, st, "type Parameters")
	for _, f := range st.Fields.List {
		doc := f.Doc.Text()
		field := f.Names[0].Name
		assert.Contains(t, doc, "+openeverest:param=connection", "Parameters.%s", field)
		assert.NotContains(t, doc, "XValidation", "Parameters.%s", field)
		jsonName, _, _ := strings.Cut(reflect.StructTag(strings.Trim(f.Tag.Value, "`")).Get("json"), ",")
		assert.NotContains(t, forbidden, jsonName, "Parameters.%s", field)
	}
}

func TestManifestIsACopy(t *testing.T) {
	t.Parallel()

	m := s3.Manifest()
	m[0] = 'X'
	assert.NotEqual(t, m[0], s3.Manifest()[0])
	s := s3.ParametersSchema()
	s.Required = nil
	assert.NotEmpty(t, s3.ParametersSchema().Required)
}
