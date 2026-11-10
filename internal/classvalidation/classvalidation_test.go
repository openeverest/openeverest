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

package classvalidation_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"sigs.k8s.io/yaml"

	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	"github.com/openeverest/openeverest/v2/internal/classvalidation"
)

const specSchema = `
type: object
x-kubernetes-validations:
- rule: self.name != 'forbidden-by-the-crd'
properties:
  name:
    type: string
    maxLength: 63
  parameters:
    type: object
    properties:
      url:
        type: string
        maxLength: 2048
  items:
    type: array
    items:
      type: string
`

func schema(t *testing.T, s string) *apiextensionsv1.JSONSchemaProps {
	t.Helper()
	out := &apiextensionsv1.JSONSchemaProps{}
	require.NoError(t, yaml.Unmarshal([]byte(s), out))
	return out
}

func TestCompile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		schema  string
		rule    string
		wantErr string
	}{
		{name: "valid", schema: specSchema, rule: "self.name.startsWith('a')"},
		{name: "syntax error", schema: specSchema, rule: "self.name.startsWith(", wantErr: "validations[0]: compilation failed"},
		{name: "unknown field", schema: specSchema, rule: "self.region == 'x'", wantErr: "undefined field 'region'"},
		{name: "not a boolean", schema: specSchema, rule: "self.name", wantErr: "must evaluate to a bool"},
		{name: "oldSelf", schema: specSchema, rule: "self.name == oldSelf.name", wantErr: "oldSelf is not available"},
		{
			name: "too expensive", schema: specSchema,
			rule:    "self.items.all(a, self.items.all(b, self.items.all(c, a + b + c != '')))",
			wantErr: "exceeds the limit",
		},
		{name: "not structural", schema: "type: object\nproperties:\n  name: {}\n", rule: "true", wantErr: "not structural"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := classvalidation.Compile(schema(t, tt.schema), []common.ClassValidation{{Expression: tt.rule, Message: "m", Reason: "Invalid"}})
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestEvaluate(t *testing.T) {
	t.Parallel()

	rules, err := classvalidation.Compile(schema(t, specSchema), []common.ClassValidation{
		{Expression: "self.name.startsWith('a')", Message: "name must start with a", Reason: "InvalidName"},
		{Expression: "!has(self.parameters) || !has(self.parameters.url) || self.parameters.url.startsWith('https://')", Message: "url must be https", Reason: "InvalidParameters"},
		{Expression: "self.parameters.url != ''", Message: "url must not be empty", Reason: "InvalidParameters"},
	})
	require.NoError(t, err)

	tests := []struct {
		name string
		spec map[string]any
		want []classvalidation.Violation
	}{
		{
			name: "valid",
			spec: map[string]any{"name": "abc", "parameters": map[string]any{"url": "https://x"}},
		},
		{
			name: "the schema's own rules are not evaluated",
			spec: map[string]any{"name": "forbidden-by-the-crd", "parameters": map[string]any{"url": "https://x"}},
			want: []classvalidation.Violation{{Reason: "InvalidName", Message: "name must start with a"}},
		},
		{
			name: "several broken, in order",
			spec: map[string]any{"name": "zzz", "parameters": map[string]any{"url": "http://x"}},
			want: []classvalidation.Violation{
				{Reason: "InvalidName", Message: "name must start with a"},
				{Reason: "InvalidParameters", Message: "url must be https"},
			},
		},
		{
			name: "a rule that cannot be evaluated is broken",
			spec: map[string]any{"name": "abc"},
			want: []classvalidation.Violation{{Reason: "InvalidParameters", Message: "no such key: parameters evaluating rule: url must not be empty"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, rules.Evaluate(t.Context(), tt.spec))
		})
	}
}

func TestCompileWithoutValidations(t *testing.T) {
	t.Parallel()

	rules, err := classvalidation.Compile(nil, nil)
	require.NoError(t, err)
	assert.Empty(t, rules.Evaluate(t.Context(), make(map[string]any)))
}
