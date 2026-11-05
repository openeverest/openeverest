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
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/yaml"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
	"github.com/openeverest/openeverest/v2/provider-runtime/validation"
)

const uiTypeNumber = "number"

// UISchemaIsAccepted asserts that every value the UI schema lets a user submit
// passes the validation the runtime runs before Sync. The API may accept more
// than the UI offers, but a form that offers a value the API rejects fails
// only once the user submits it.
//
// For each topology it builds the Instance the form submits with its
// defaults, sets one field at a time to each value the form offers for it,
// and fails when validation reports an error on that field, beneath it, or on
// a field containing it. The values tried for a field bound to a spec path
// are its fieldParams.options values, its fieldParams.defaultValue and, for a
// number field without options, its validation.min and validation.max. They
// are submitted as the form submits them: a number field turns numeric
// strings into numbers, and with fieldParams.badgeToApi the badge is
// appended, so 25 with badge Gi becomes "25Gi". Empty and non-scalar values,
// multiple selects and options loaded at runtime are not tried.
func UISchemaIsAccepted(t *testing.T, cfg Config) {
	t.Helper()

	spec, err := loadProviderSpec(cfg.ProviderSpecPath)
	if err != nil {
		t.Fatalf("conformance: %v", err)
	}

	for _, topology := range sortedKeys(spec.Topologies) {
		fields, err := uiFields(spec, topology)
		if err != nil {
			t.Fatalf("conformance: reading UI schema for topology %q: %v", topology, err)
		}

		got := checkAccepted(t.Context(), cfg, spec, topology, fields)
		if len(got.base) > 0 {
			t.Errorf("UI schema cannot be checked, because validation rejects the Instance its form submits by default:\n  topology %s\n%s",
				topology, formatErrors(got.base))
			continue
		}
		for _, r := range got.rejected {
			t.Errorf("UI schema offers a value this provider rejects:\n  topology %s\n    %s = %s\n%s",
				topology, r.path, serialize(r.value), formatErrors(r.errs))
		}
	}
}

// uiField is a form control bound to an Instance path, reduced to what decides
// the values it can submit.
type uiField struct {
	Path        string `json:"path"`
	UIType      string `json:"uiType"`
	FieldParams struct {
		Options []struct {
			Value any `json:"value"`
		} `json:"options"`
		DefaultValue any    `json:"defaultValue"`
		Multiple     bool   `json:"multiple"`
		Badge        string `json:"badge"`
		BadgeToAPI   bool   `json:"badgeToApi"`
	} `json:"fieldParams"`
	Validation struct {
		Min *float64 `json:"min"`
		Max *float64 `json:"max"`
	} `json:"validation"`
}

// values returns what the form can submit for the field, each once.
func (f uiField) values() []any {
	// A multiple select submits a list of options, not one of them.
	if f.FieldParams.Multiple {
		return nil
	}

	candidates := make([]any, 0, len(f.FieldParams.Options)+3)
	for _, option := range f.FieldParams.Options {
		candidates = append(candidates, option.Value)
	}
	candidates = append(candidates, f.FieldParams.DefaultValue)
	if f.UIType == uiTypeNumber && len(f.FieldParams.Options) == 0 {
		for _, bound := range []*float64{f.Validation.Min, f.Validation.Max} {
			if bound != nil {
				candidates = append(candidates, *bound)
			}
		}
	}

	var out []any
	for _, candidate := range candidates {
		if value, ok := f.submitted(candidate); ok && !slices.Contains(out, value) {
			out = append(out, value)
		}
	}
	return out
}

// defaultValue returns what the form submits for the field untouched.
func (f uiField) defaultValue() (any, bool) {
	if f.FieldParams.Multiple {
		return nil, false
	}
	return f.submitted(f.FieldParams.DefaultValue)
}

// submitted returns value as the form sends it, and false when the form sends
// nothing or something other than a scalar.
func (f uiField) submitted(value any) (any, bool) {
	if s, ok := value.(string); ok && f.UIType == uiTypeNumber {
		if n, err := strconv.ParseFloat(strings.TrimSpace(s), 64); err == nil {
			value = n
		}
	}

	var text string
	switch v := value.(type) {
	case string:
		if v == "" {
			return nil, false
		}
		text = strings.TrimSpace(v)
	case float64:
		text = strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		text = strconv.FormatBool(v)
	default:
		return nil, false
	}

	if f.FieldParams.BadgeToAPI && f.FieldParams.Badge != "" {
		return text + f.FieldParams.Badge, true
	}
	return value, true
}

// uiFields returns the form controls the topology's UI schema binds to a
// single Instance path.
func uiFields(spec *corev1alpha1.ProviderSpec, topology string) ([]uiField, error) {
	if spec.UISchema == nil || len(spec.UISchema.Raw) == 0 {
		return nil, nil
	}

	var schemas map[string]any
	if err := yaml.Unmarshal(spec.UISchema.Raw, &schemas); err != nil {
		return nil, err
	}

	var nodes []map[string]any
	walkFields(schemas[topology], func(_ string, node map[string]any) {
		nodes = append(nodes, node)
	})

	fields := make([]uiField, 0, len(nodes))
	for _, node := range nodes {
		data, err := json.Marshal(node)
		if err != nil {
			return nil, err
		}
		var f uiField
		if err := json.Unmarshal(data, &f); err != nil {
			return nil, fmt.Errorf("field %v: %w", node["path"], err)
		}
		fields = append(fields, f)
	}
	return fields, nil
}

// acceptance is what validation made of one topology's form.
type acceptance struct {
	// base holds the errors on the Instance the form submits with its
	// defaults that no default explains, since no field can be judged
	// against that Instance then.
	base     field.ErrorList
	rejected []rejection
}

// rejection is a value the form can submit and validation refuses.
type rejection struct {
	path  string
	value any
	errs  field.ErrorList
}

func checkAccepted(
	ctx context.Context,
	cfg Config,
	spec *corev1alpha1.ProviderSpec,
	topology string,
	fields []uiField,
) acceptance {
	values, defaults := submittedValues(fields, cfg.Unverifiable)
	if len(values) == 0 {
		return acceptance{}
	}

	name := cfg.Provider.Name()
	provider := providerObject(name, spec)
	validate := func(form map[string]any) (field.ErrorList, error) {
		instance, err := buildInstance(spec, name, topology, form)
		if err != nil {
			return nil, err
		}
		in := controller.ValidationInput{Instance: instance, Provider: provider}
		return validation.Run(ctx, cfg.Provider, in).Errors, nil
	}

	// A default the API cannot decode would spoil every form carrying it, so
	// it is left out here and reported when its own field is checked.
	for _, path := range sortedKeys(defaults) {
		if _, err := buildInstance(spec, name, topology, map[string]any{path: defaults[path]}); err != nil {
			delete(defaults, path)
		}
	}

	errs, err := validate(defaults)
	if err != nil {
		return acceptance{base: field.ErrorList{field.InternalError(field.NewPath("spec"), err)}}
	}
	if unexplained := errorsOff(errs, sortedKeys(defaults)); len(unexplained) > 0 {
		return acceptance{base: unexplained}
	}

	var out acceptance
	for _, path := range sortedKeys(values) {
		for _, value := range values[path] {
			form := maps.Clone(defaults)
			form[path] = value
			errs, err = validate(form)
			if err != nil {
				// The API cannot decode the value into an Instance either.
				errs = field.ErrorList{field.Invalid(fieldPath(path), value, err.Error())}
			}
			if errs = errorsOn(errs, path); len(errs) > 0 {
				out.rejected = append(out.rejected, rejection{path: path, value: value, errs: errs})
			}
		}
	}
	return out
}

// submittedValues returns, per spec path, the values the form can write there
// and the one it writes by default.
func submittedValues(fields []uiField, unverifiable map[string]string) (map[string][]any, map[string]any) {
	values := map[string][]any{}
	defaults := map[string]any{}
	for _, f := range fields {
		if _, skip := unverifiable[f.Path]; skip || !strings.HasPrefix(f.Path, "spec.") {
			continue
		}
		for _, value := range f.values() {
			if !slices.Contains(values[f.Path], value) {
				values[f.Path] = append(values[f.Path], value)
			}
		}
		if value, ok := f.defaultValue(); ok {
			if _, set := defaults[f.Path]; !set {
				defaults[f.Path] = value
			}
		}
	}
	return values, defaults
}

// fieldPath spells a dotted UI path the way validation errors do, with the
// component name as a map key: spec.components[proxy].type.
func fieldPath(path string) *field.Path {
	segments := strings.Split(path, ".")
	p := field.NewPath(segments[0])
	for i, segment := range segments[1:] {
		if i == 1 && segments[1] == componentsSegment {
			p = p.Key(segment)
			continue
		}
		p = p.Child(segment)
	}
	return p
}

// errorsOn returns the errors on the field at path, beneath it, or on a field
// containing it, which is where an undeclared resources lands when path is
// spec.components.engine.resources.limits.cpu.
func errorsOn(errs field.ErrorList, path string) field.ErrorList {
	var out field.ErrorList
	for _, err := range errs {
		if related(err.Field, path) {
			out = append(out, err)
		}
	}
	return out
}

// errorsOff returns the errors on none of the fields at paths.
func errorsOff(errs field.ErrorList, paths []string) field.ErrorList {
	var out field.ErrorList
	for _, err := range errs {
		if !slices.ContainsFunc(paths, func(path string) bool { return related(err.Field, path) }) {
			out = append(out, err)
		}
	}
	return out
}

// related reports whether the error field and the dotted UI path are the same
// field or one contains the other.
func related(errField, path string) bool {
	target := fieldPath(path).String()
	return within(errField, target) || within(target, errField)
}

func within(inner, outer string) bool {
	return inner == outer || strings.HasPrefix(inner, outer+".") || strings.HasPrefix(inner, outer+"[")
}

func formatErrors(errs field.ErrorList) string {
	lines := make([]string, 0, len(errs))
	for _, err := range errs {
		lines = append(lines, "      "+err.Error())
	}
	return strings.Join(lines, "\n")
}
