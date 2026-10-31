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

package storageclasses_test

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

	"github.com/openeverest/openeverest/v2/api/backup/v1alpha1/storageclasses"
	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
)

// parameterTypes returns the class parameter types, by class.
func parameterTypes() map[string]any {
	return map[string]any{
		storageclasses.ClassS3: storageclasses.S3{},
	}
}

// TestGeneratedClassesMatchTypes guards against a generated class drifting
// from the type its executors decode with. Run make gen after changing a type.
func TestGeneratedClassesMatchTypes(t *testing.T) {
	t.Parallel()

	for name, typ := range parameterTypes() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			schema := builtinClass(t, name).Spec.ParametersSchema.OpenAPIV3Schema
			require.NotNil(t, schema)
			var fields, required []string
			for f := range reflect.TypeOf(typ).Fields() {
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
		})
	}
}

func TestGeneratedS3Class(t *testing.T) {
	t.Parallel()

	class := builtinClass(t, storageclasses.ClassS3)
	values := class.Spec.ParametersSchema.OpenAPIV3Schema.Properties["implementation"].Enum
	enum := make([]storageclasses.S3Implementation, 0, len(values))
	for _, v := range values {
		enum = append(enum, storageclasses.S3Implementation(strings.Trim(string(v.Raw), `"`)))
	}
	wantEnum := []storageclasses.S3Implementation{
		storageclasses.S3ImplementationAWS, storageclasses.S3ImplementationMinIO, storageclasses.S3ImplementationCeph,
		storageclasses.S3ImplementationGCS, storageclasses.S3ImplementationOther,
	}
	assert.Equal(t, wantEnum, enum)
	wantModes := []common.AuthMode{{
		Name: storageclasses.AuthModeAccessKey,
		Credentials: &common.CredentialKeys{
			Required: []string{storageclasses.KeyAWSAccessKeyID, storageclasses.KeyAWSSecretAccessKey},
		},
	}}
	assert.Equal(t, wantModes, class.Spec.AuthModes)
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
	types := make(map[string]*ast.StructType)
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, e.Name(), nil, parser.ParseComments)
		require.NoError(t, err)
		ast.Inspect(file, func(n ast.Node) bool {
			if ts, ok := n.(*ast.TypeSpec); ok {
				if st, ok := ts.Type.(*ast.StructType); ok {
					types[ts.Name.Name] = st
				}
			}
			return true
		})
	}
	for class, typ := range parameterTypes() {
		name := reflect.TypeOf(typ).Name()
		st, ok := types[name]
		require.True(t, ok, "type %s of class %s", name, class)
		for _, f := range st.Fields.List {
			doc := f.Doc.Text()
			field := f.Names[0].Name
			assert.Contains(t, doc, "+openeverest:param=connection", "%s.%s", name, field)
			assert.NotContains(t, doc, "XValidation", "%s.%s", name, field)
			jsonName, _, _ := strings.Cut(reflect.StructTag(strings.Trim(f.Tag.Value, "`")).Get("json"), ",")
			assert.NotContains(t, forbidden, jsonName, "%s.%s", name, field)
		}
	}
}
