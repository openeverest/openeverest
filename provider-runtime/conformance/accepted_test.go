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

package conformance

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/yaml"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

// rulesProvider applies rules of its own, as a provider's Validate does.
type rulesProvider struct {
	rules func(*corev1alpha1.Instance) field.ErrorList
}

func (p *rulesProvider) Name() string                       { return "rules" }
func (p *rulesProvider) Types() func(*runtime.Scheme) error { return nil }
func (p *rulesProvider) Sync(*controller.Context) error     { return nil }
func (p *rulesProvider) Cleanup(*controller.Context) error  { return nil }

func (p *rulesProvider) Status(*controller.Context) (controller.Status, error) {
	return controller.Status{}, nil
}

func (p *rulesProvider) Validate(in controller.ValidationInput) controller.ValidationResult {
	if p.rules == nil {
		return controller.ValidationResult{}
	}
	return controller.ValidationResult{Errors: p.rules(in.Instance)}
}

func enginePath() *field.Path { return field.NewPath("spec", "components").Key("engine") }

// specOffering builds a spec whose "simple" topology renders fields in one
// section and, unless supported is empty, gives engine those supportedFields.
func specOffering(t *testing.T, supported, fields string) *corev1alpha1.ProviderSpec {
	t.Helper()

	spec := testSpec()
	if supported != "" {
		spec = specDeclaring(t, supported)
	}

	components := map[string]any{}
	require.NoError(t, yaml.Unmarshal([]byte(fields), &components))
	raw, err := json.Marshal(map[string]any{
		"simple": map[string]any{
			"sections": map[string]any{
				"main": map[string]any{"components": components},
			},
		},
	})
	require.NoError(t, err)
	spec.UISchema = &runtime.RawExtension{Raw: raw}
	return spec
}

func checkOffering(t *testing.T, cfg Config, spec *corev1alpha1.ProviderSpec) acceptance {
	t.Helper()

	fields, err := uiFields(spec, "simple")
	require.NoError(t, err)
	return checkAccepted(t.Context(), cfg, spec, "simple", fields)
}

const typeSelect = `
type:
  uiType: select
  path: spec.components.engine.type
  fieldParams:
    defaultValue: db
    options:
      - {label: DB, value: db}
      - {label: Alt, value: db-alt}
      - {label: Other, value: db-other}
`

func TestCheckAccepted(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		supported    string
		fields       string
		rules        func(*corev1alpha1.Instance) field.ErrorList
		unverifiable map[string]string
		want         []string
	}{
		"only the option outside the declared type enum is rejected": {
			supported: "properties:\n  type: {enum: [db, db-alt]}",
			fields:    typeSelect,
			want:      []string{`spec.components.engine.type = "db-other"`},
		},
		"a field supportedFields does not declare": {
			supported: "properties:\n  replicas: {}",
			fields: `
nodes:
  uiType: number
  path: spec.components.engine.replicas
  fieldParams: {defaultValue: 3}
cpu:
  uiType: number
  path: spec.components.engine.resources.limits.cpu
  validation: {min: 0.5}
`,
			want: []string{`spec.components.engine.resources.limits.cpu = 0.5`},
		},
		"a minimum below the declared minimum": {
			supported: "properties:\n  replicas: {minimum: 1}",
			fields: `
nodes:
  uiType: number
  path: spec.components.engine.replicas
  validation: {min: 0, max: 5}
`,
			want: []string{`spec.components.engine.replicas = 0`},
		},
		"a value the provider's own rules reject": {
			fields: `
nodes:
  uiType: number
  path: spec.components.engine.replicas
  validation: {min: 1, max: 4}
`,
			rules: func(in *corev1alpha1.Instance) field.ErrorList {
				if r := in.Spec.Components["engine"].Replicas; r != nil && *r%2 == 0 {
					return field.ErrorList{field.Invalid(enginePath().Child("replicas"), *r, "must be odd")}
				}
				return nil
			},
			want: []string{`spec.components.engine.replicas = 4`},
		},
		"the badge is part of the submitted value": {
			fields: `
disk:
  uiType: number
  path: spec.components.engine.storage.size
  fieldParams: {badge: Gi, badgeToApi: true, defaultValue: 25}
`,
			rules: func(in *corev1alpha1.Instance) field.ErrorList {
				if s := in.Spec.Components["engine"].Storage; s != nil {
					return field.ErrorList{field.Forbidden(enginePath().Child("storage", "size"), "rejects every size")}
				}
				return nil
			},
			want: []string{`spec.components.engine.storage.size = "25Gi"`},
		},
		"an error on another field is not this field's": {
			fields: `
nodes:
  uiType: number
  path: spec.components.engine.replicas
  validation: {min: 1, max: 5}
`,
			rules: func(in *corev1alpha1.Instance) field.ErrorList {
				if r := in.Spec.Components["engine"].Replicas; r != nil && *r == 5 {
					return field.ErrorList{field.Required(enginePath().Child("storage"), "needed with 5 replicas")}
				}
				return nil
			},
		},
		"every form carries the other fields' defaults": {
			fields: `
disk:
  uiType: number
  path: spec.components.engine.storage.size
  fieldParams: {badge: Gi, badgeToApi: true, defaultValue: 25}
nodes:
  uiType: number
  path: spec.components.engine.replicas
  validation: {min: 1}
`,
			rules: func(in *corev1alpha1.Instance) field.ErrorList {
				if in.Spec.Components["engine"].Storage == nil {
					return field.ErrorList{field.Required(enginePath().Child("storage"), "")}
				}
				return nil
			},
		},
		"a rejected default is reported on its own field": {
			fields: `
nodes:
  uiType: number
  path: spec.components.engine.replicas
  fieldParams: {defaultValue: 2}
  validation: {min: 1}
`,
			rules: func(in *corev1alpha1.Instance) field.ErrorList {
				if r := in.Spec.Components["engine"].Replicas; r != nil && *r%2 == 0 {
					return field.ErrorList{field.Invalid(enginePath().Child("replicas"), *r, "must be odd")}
				}
				return nil
			},
			want: []string{`spec.components.engine.replicas = 2`},
		},
		"a default an Instance cannot hold is reported on its own field": {
			fields: `
nodes:
  uiType: number
  path: spec.components.engine.replicas
  fieldParams: {defaultValue: 1.5}
disk:
  uiType: number
  path: spec.components.engine.storage.size
  fieldParams: {badge: Gi, badgeToApi: true, defaultValue: 25}
`,
			want: []string{`spec.components.engine.replicas = 1.5`},
		},
		"an unverifiable path is not checked": {
			supported:    "properties:\n  type: {enum: [db, db-alt]}",
			fields:       typeSelect,
			unverifiable: map[string]string{"spec.components.engine.type": "reason"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := checkOffering(t, Config{
				Provider:     &rulesProvider{rules: tt.rules},
				Unverifiable: tt.unverifiable,
			}, specOffering(t, tt.supported, tt.fields))
			require.Empty(t, got.base)

			rejected := make([]string, 0, len(got.rejected))
			for _, r := range got.rejected {
				rejected = append(rejected, r.path+" = "+serialize(r.value))
			}
			if len(tt.want) == 0 {
				assert.Empty(t, rejected)
				return
			}
			assert.Equal(t, tt.want, rejected)
		})
	}
}

// An error no default explains leaves no form any field can be judged against,
// so the topology is reported once rather than once per value.
func TestCheckAcceptedReportsARejectedBaseOnce(t *testing.T) {
	t.Parallel()

	got := checkOffering(t, Config{Provider: &rulesProvider{
		rules: func(in *corev1alpha1.Instance) field.ErrorList {
			if in.Spec.Components["engine"].Replicas == nil {
				return field.ErrorList{field.Required(enginePath().Child("replicas"), "")}
			}
			return nil
		},
	}}, specOffering(t, "", `
nodes:
  uiType: number
  path: spec.components.engine.replicas
  validation: {min: 1, max: 3}
`))

	require.Len(t, got.base, 1)
	assert.Equal(t, "spec.components[engine].replicas", got.base[0].Field)
	assert.Empty(t, got.rejected)
}

func TestUIFieldValues(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		field string
		want  []any
	}{
		"options and the default, each once": {
			field: "uiType: select\nfieldParams: {defaultValue: b, options: [{value: a}, {value: b}]}",
			want:  []any{"a", "b"},
		},
		"a number field's default and bounds": {
			field: "uiType: number\nfieldParams: {defaultValue: 3}\nvalidation: {min: 1, max: 5}",
			want:  []any{3.0, 1.0, 5.0},
		},
		"a number field with options ignores its bounds": {
			field: "uiType: number\nfieldParams: {options: [{value: 1}, {value: 3}]}\nvalidation: {min: 0}",
			want:  []any{1.0, 3.0},
		},
		"a text field's length bounds are not values": {
			field: "uiType: text\nvalidation: {min: 1, max: 63}",
			want:  nil,
		},
		"a badge sent to the API is appended": {
			field: "uiType: number\nfieldParams: {badge: Gi, badgeToApi: true, defaultValue: 25}\nvalidation: {min: 0.5}",
			want:  []any{"25Gi", "0.5Gi"},
		},
		"a badge only for display is not": {
			field: "uiType: number\nfieldParams: {badge: Gi, defaultValue: 25}",
			want:  []any{25.0},
		},
		"a number field submits numeric strings as numbers": {
			field: "uiType: number\nfieldParams: {defaultValue: \"3\"}",
			want:  []any{3.0},
		},
		"empty and non-scalar values are not submitted": {
			field: "uiType: select\nfieldParams: {defaultValue: \"\", options: [{value: {a: b}}]}",
			want:  nil,
		},
		"a multiple select submits a list": {
			field: "uiType: select\nfieldParams: {multiple: true, options: [{value: a}]}",
			want:  nil,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var f uiField
			require.NoError(t, yaml.Unmarshal([]byte(tt.field), &f))
			assert.Equal(t, tt.want, f.values())
		})
	}
}

func TestFieldPath(t *testing.T) {
	t.Parallel()

	for path, want := range map[string]string{
		"spec.version":                                    "spec.version",
		"spec.components.proxy.type":                      "spec.components[proxy].type",
		"spec.components.engine.parameters.configuration": "spec.components[engine].parameters.configuration",
		"spec.topology.parameters.shards":                 "spec.topology.parameters.shards",
	} {
		assert.Equal(t, want, fieldPath(path).String(), path)
	}
}

func TestErrorsOn(t *testing.T) {
	t.Parallel()

	errs := field.ErrorList{
		field.Forbidden(enginePath().Child("resources"), "a field containing it"),
		field.Invalid(enginePath().Child("resources", "limits", "cpu"), "", "the field itself"),
		field.Invalid(enginePath().Child("resources", "limits", "cpu").Index(0), "", "beneath it"),
		field.Invalid(enginePath().Child("resources", "limits", "cpuShares"), "", "a sibling sharing its name"),
		field.Invalid(enginePath().Child("replicas"), "", "an unrelated field"),
	}

	got := errorsOn(errs, "spec.components.engine.resources.limits.cpu")

	details := make([]string, 0, len(got))
	for _, err := range got {
		details = append(details, err.Detail)
	}
	assert.Equal(t, []string{"a field containing it", "the field itself", "beneath it"}, details)
}
