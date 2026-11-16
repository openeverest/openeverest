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
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	"github.com/openeverest/openeverest/v2/pkg/classgen"
)

func object(props map[string]apiextensionsv1.JSONSchemaProps) apiextensionsv1.JSONSchemaProps {
	return apiextensionsv1.JSONSchemaProps{Type: "object", Properties: props}
}

// testView is the schema of a view with location.objectStore.bucket,
// parameters.endpointURL, tls and authentication.mode.
func testView(rules ...apiextensionsv1.ValidationRule) *apiextensionsv1.JSONSchemaProps {
	str := apiextensionsv1.JSONSchemaProps{Type: "string"}
	view := object(map[string]apiextensionsv1.JSONSchemaProps{
		"location":       object(map[string]apiextensionsv1.JSONSchemaProps{"objectStore": object(map[string]apiextensionsv1.JSONSchemaProps{"bucket": str})}),
		"parameters":     object(map[string]apiextensionsv1.JSONSchemaProps{"endpointURL": str}),
		"tls":            object(nil),
		"authentication": object(map[string]apiextensionsv1.JSONSchemaProps{"mode": str}),
	})
	view.XValidations = rules
	return &view
}

// reasonFor is the reason mapping of BackupStorageClasses.
func reasonFor(fieldPath string) (string, error) {
	switch first, _, _ := strings.Cut(strings.TrimPrefix(fieldPath, "."), "."); first {
	case "location":
		return "InvalidLocation", nil
	case "parameters", "tls":
		return "InvalidParameters", nil
	default:
		return "", errors.New("a rule judges location, parameters or tls")
	}
}

func TestValidations(t *testing.T) {
	t.Parallel()

	rule := func(fieldPath string) apiextensionsv1.ValidationRule {
		return apiextensionsv1.ValidationRule{Rule: "true", Message: "m", FieldPath: fieldPath}
	}
	reason := "FieldValueInvalid"
	optionalOldSelf := true

	tests := []struct {
		name    string
		rule    apiextensionsv1.ValidationRule
		want    string
		wantErr string
	}{
		{name: "nested location field", rule: rule(".location.objectStore.bucket"), want: "InvalidLocation"},
		{name: "location", rule: rule(".location"), want: "InvalidLocation"},
		{name: "parameters field", rule: rule(".parameters.endpointURL"), want: "InvalidParameters"},
		{name: "tls", rule: rule(".tls"), want: "InvalidParameters"},
		{name: "no fieldPath", rule: rule(""), wantErr: "must name the field"},
		{name: "relative fieldPath", rule: rule("location"), wantErr: "must name the field"},
		{name: "empty segment", rule: rule(".location..bucket"), wantErr: "must name the field"},
		{name: "unknown field", rule: rule(".location.objectStore.region"), wantErr: `no field "region"`},
		{name: "field reasonFor refuses", rule: rule(".authentication.mode"), wantErr: "judges location, parameters or tls"},
		{
			name:    "no message",
			rule:    apiextensionsv1.ValidationRule{Rule: "true", FieldPath: ".tls"},
			wantErr: "message is required",
		},
		{
			name:    "messageExpression",
			rule:    apiextensionsv1.ValidationRule{Rule: "true", Message: "m", MessageExpression: "'m'", FieldPath: ".tls"},
			wantErr: "messageExpression, reason and optionalOldSelf are not supported",
		},
		{
			name:    "reason",
			rule:    apiextensionsv1.ValidationRule{Rule: "true", Message: "m", Reason: (*apiextensionsv1.FieldValueErrorReason)(&reason), FieldPath: ".tls"},
			wantErr: "messageExpression, reason and optionalOldSelf are not supported",
		},
		{
			name:    "optionalOldSelf",
			rule:    apiextensionsv1.ValidationRule{Rule: "true", Message: "m", OptionalOldSelf: &optionalOldSelf, FieldPath: ".tls"},
			wantErr: "messageExpression, reason and optionalOldSelf are not supported",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := classgen.Validations(testView(tt.rule), reasonFor)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, []classgen.Validation{{Expression: "true", Message: "m", Reason: tt.want}}, got)
		})
	}
}

func TestValidationsKeepsOrder(t *testing.T) {
	t.Parallel()

	view := testView(
		apiextensionsv1.ValidationRule{Rule: "b", Message: "second", FieldPath: ".tls"},
		apiextensionsv1.ValidationRule{Rule: "a", Message: "first", FieldPath: ".location"},
	)
	got, err := classgen.Validations(view, reasonFor)
	require.NoError(t, err)
	want := []classgen.Validation{
		{Expression: "b", Message: "second", Reason: "InvalidParameters"},
		{Expression: "a", Message: "first", Reason: "InvalidLocation"},
	}
	assert.Equal(t, want, got)
}

func TestValidationsRejectsNestedRules(t *testing.T) {
	t.Parallel()

	view := testView()
	params := view.Properties["parameters"]
	endpoint := params.Properties["endpointURL"]
	endpoint.XValidations = apiextensionsv1.ValidationRules{{Rule: "true", Message: "m"}}
	params.Properties["endpointURL"] = endpoint
	view.Properties["parameters"] = params

	_, err := classgen.Validations(view, reasonFor)
	require.ErrorContains(t, err, ".parameters.endpointURL: x-kubernetes-validations")
}

func TestValidationsWithoutView(t *testing.T) {
	t.Parallel()

	_, err := classgen.Validations(nil, reasonFor)
	require.ErrorContains(t, err, "no view schema")
}
