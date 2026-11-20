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

package classgen_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/yaml"

	"github.com/openeverest/openeverest/v2/pkg/classgen"
)

const (
	paramsPackage  = "./testdata/params"
	viewPackage    = "./testdata/view"
	invalidPackage = "./testdata/invalid"
	brokenPackage  = "./testdata/broken"
)

// wantParameters is the schema of params.Parameters: markers, enum, default,
// required, nested and closed objects, maps of quantities and lists, and no
// root description.
const wantParameters = `
type: object
required: [url]
properties:
  url:
    type: string
    description: URL of the server.
    maxLength: 2048
    pattern: ^https?://
  mode:
    type: string
    description: Mode of the connection.
    default: tls
    enum: [plain, tls]
  retries:
    type: integer
    format: int32
    description: Retries bounds the connection attempts.
    minimum: 0
    maximum: 10
  resources:
    type: object
    description: Resources of the agent.
    additionalProperties: false
    properties:
      limits:
        type: object
        description: Limits of the agent.
        additionalProperties: &quantity
          anyOf: [{type: integer}, {type: string}]
          pattern: ^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$
          x-kubernetes-int-or-string: true
      requests:
        type: object
        description: Requests of the agent.
        additionalProperties: *quantity
  tags:
    type: array
    description: Tags label the server.
    maxItems: 4
    items:
      type: object
      description: Tag is a label.
      required: [name]
      properties:
        name:
          type: string
          description: Name of the tag.
          pattern: ^[a-z]+$
`

func TestSchema(t *testing.T) {
	t.Parallel()

	got, err := classgen.Schema([]string{paramsPackage}, "Parameters")
	require.NoError(t, err)
	want := &apiextensionsv1.JSONSchemaProps{}
	require.NoError(t, yaml.UnmarshalStrict([]byte(wantParameters), want))
	assert.Equal(t, want, got)
}

func TestSchemaReturnsACopy(t *testing.T) {
	t.Parallel()

	p, err := classgen.Load(paramsPackage)
	require.NoError(t, err)
	first, err := p.Schema("Parameters")
	require.NoError(t, err)
	first.Properties = nil
	second, err := p.Schema("Parameters")
	require.NoError(t, err)
	assert.Contains(t, second.Properties, "url")
}

func TestSchemaErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		packages []string
		typeName string
		wantErr  string
	}{
		{
			name:     "CEL on a field",
			packages: []string{invalidPackage},
			typeName: "NestedCEL",
			wantErr:  "NestedCEL.name: x-kubernetes-validations are not evaluated on parameters",
		},
		{
			name:     "CEL on the root",
			packages: []string{invalidPackage},
			typeName: "RootCEL",
			wantErr:  "RootCEL: x-kubernetes-validations are not evaluated on parameters",
		},
		{
			name:     "closed map",
			packages: []string{invalidPackage},
			typeName: "ClosedMap",
			wantErr:  "+openeverest:validation:Closed applies to struct types",
		},
		{
			name:     "not an object",
			packages: []string{invalidPackage},
			typeName: "NotAnObject",
			wantErr:  `NotAnObject renders to type "string": parameters are an object`,
		},
		{
			name:     "unknown type",
			packages: []string{paramsPackage},
			typeName: "Missing",
			wantErr:  "type Missing not found in ./testdata/params",
		},
		{
			name:     "ambiguous type",
			packages: []string{paramsPackage, invalidPackage},
			typeName: "Tag",
			wantErr:  "type Tag is declared in several packages",
		},
		{
			name:     "package that does not compile",
			packages: []string{brokenPackage},
			typeName: "Parameters",
			wantErr:  "undefined: Undefined",
		},
		{
			name:     "no package",
			packages: []string{"./testdata/missing"},
			typeName: "Parameters",
			wantErr:  "testdata/missing",
		},
		{
			name:     "no pattern",
			typeName: "Parameters",
			wantErr:  "no package to load",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := classgen.Schema(tt.packages, tt.typeName)
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestViewValidations(t *testing.T) {
	t.Parallel()

	p, err := classgen.Load(paramsPackage, viewPackage)
	require.NoError(t, err)
	_, err = p.Schema("StorageView")
	require.ErrorContains(t, err, "StorageView: x-kubernetes-validations are not evaluated on parameters")

	view, err := p.View("StorageView")
	require.NoError(t, err)
	reasons := map[string]string{".location.bucket": "InvalidLocation", ".tls": "InvalidParameters"}
	got, err := classgen.Validations(view, func(fieldPath string) (string, error) {
		return reasons[fieldPath], nil
	})
	require.NoError(t, err)
	want := []classgen.Validation{
		{
			Expression: "!self.location.bucket.contains('..')",
			Message:    "bucket must not contain ..",
			Reason:     "InvalidLocation",
		},
		{
			Expression: "!has(self.tls) || self.parameters.url.startsWith('https://')",
			Message:    "tls needs an https url",
			Reason:     "InvalidParameters",
		},
	}
	assert.Equal(t, want, got)
}
