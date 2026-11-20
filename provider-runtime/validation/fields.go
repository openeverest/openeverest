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

package validation

import (
	"context"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"

	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

// checkComponentFields checks a component's core fields against the
// supportedFields its topology declares. No declaration accepts every field.
// A declared property without properties of its own accepts everything
// beneath it. The type is checked against the catalog and the parameters
// against their own schema instead.
func checkComponentFields(
	ctx context.Context,
	path *field.Path,
	component v1alpha1.ComponentSpec,
	declared *common.ParametersSchema,
	topology string,
) field.ErrorList {
	if declared == nil || declared.OpenAPIV3Schema == nil {
		return nil
	}
	component.Type = ""
	component.Parameters = nil
	value, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&component)
	if err != nil {
		return field.ErrorList{field.InternalError(path, err)}
	}

	schema := declared.OpenAPIV3Schema.DeepCopy()
	withTypes(schema, reflect.TypeFor[v1alpha1.ComponentSpec]())
	s, err := compile(schema)
	if err != nil {
		return field.ErrorList{field.InternalError(path, err)}
	}

	detail := fmt.Sprintf("the provider does not support this field in the %s topology", topology)
	errs := unsupportedFields(path, value, schema, detail)
	return append(errs, s.validate(ctx, path, value)...)
}

// unsupportedFields reports every field of value the schema does not declare,
// descending only into properties that declare properties of their own.
func unsupportedFields(
	path *field.Path,
	value map[string]any,
	schema *apiextensionsv1.JSONSchemaProps,
	detail string,
) field.ErrorList {
	var errs field.ErrorList
	for _, name := range slices.Sorted(maps.Keys(value)) {
		property, ok := schema.Properties[name]
		if !ok {
			errs = append(errs, undeclared(path.Child(name), detail))
			continue
		}
		if nested, ok := value[name].(map[string]any); ok && len(property.Properties) > 0 {
			errs = append(errs, unsupportedFields(path.Child(name), nested, &property, detail)...)
		}
	}
	return errs
}

// withTypes sets the type of every declared property that omits it from the
// Go field it selects, so that CEL rules on the declaration compile.
func withTypes(schema *apiextensionsv1.JSONSchemaProps, t reflect.Type) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if schema.Type == "" {
		schema.Type = jsonType(t)
	}
	if t.Kind() != reflect.Struct {
		return
	}
	for name, property := range schema.Properties {
		if f, ok := fieldByJSONName(t, name); ok {
			withTypes(&property, f.Type)
			schema.Properties[name] = property
		}
	}
}

// jsonType is the OpenAPI type t serializes as. A Quantity is a struct in Go
// and a string on the wire.
func jsonType(t reflect.Type) string {
	if t == reflect.TypeFor[resource.Quantity]() {
		return "string"
	}
	switch k := t.Kind(); {
	case k == reflect.Bool:
		return "boolean"
	case k >= reflect.Int && k <= reflect.Uint64:
		return "integer"
	case k == reflect.Float32 || k == reflect.Float64:
		return "number"
	case k == reflect.String:
		return "string"
	case k == reflect.Slice || k == reflect.Array:
		return "array"
	case k == reflect.Struct || k == reflect.Map:
		return "object"
	default:
		return ""
	}
}

// fieldByJSONName finds the field of struct t serialized as name, looking
// into inlined embedded structs.
func fieldByJSONName(t reflect.Type, name string) (reflect.StructField, bool) {
	for f := range t.Fields() {
		tag, opts, _ := strings.Cut(f.Tag.Get("json"), ",")
		if f.Anonymous && tag == "" && strings.Contains(opts, "inline") {
			if inner, ok := fieldByJSONName(f.Type, name); ok {
				return inner, true
			}
			continue
		}
		if tag == name {
			return f, true
		}
	}
	return reflect.StructField{}, false
}
