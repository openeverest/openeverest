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
	"errors"
	"fmt"
	"reflect"
	"strings"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

// ProviderUnderTest is the contract a conformance run exercises.
type ProviderUnderTest = controller.ProviderInterface

// leafKind is the JSON shape of the value a UI control writes.
type leafKind string

const (
	leafString   leafKind = "string"
	leafInteger  leafKind = "integer"
	leafNumber   leafKind = "number"
	leafBool     leafKind = "boolean"
	leafQuantity leafKind = "quantity"
	leafUnknown  leafKind = ""
)

// quantityTypeName is a struct in Go but a string on the wire.
const quantityTypeName = "Quantity"

// componentsSegment is the map field every per-component path goes through.
const componentsSegment = "components"

// resolveLeaf reports the shape of the value a path addresses. Paths above a
// parameters payload are resolved against the Instance types; below one they
// are resolved against the schema the provider declared, since those types live
// in the provider's own module.
func resolveLeaf(spec *corev1alpha1.ProviderSpec, topology, path string) (leafKind, error) {
	rest, ok := strings.CutPrefix(path, "spec.")
	if !ok {
		return leafUnknown, fmt.Errorf("path %q does not address the Instance spec", path)
	}

	current := reflect.TypeFor[corev1alpha1.InstanceSpec]()
	var walked []string

	for i, segment := range strings.Split(rest, ".") {
		current = deref(current)

		if current.Name() == "RawExtension" {
			schema := parametersSchema(spec, topology, walked)
			if schema == nil {
				return leafUnknown, fmt.Errorf("no parametersSchema declared for %q", strings.Join(walked, "."))
			}
			return schemaLeaf(schema, strings.Split(rest, ".")[i:])
		}
		if current.Name() == quantityTypeName {
			return leafUnknown, fmt.Errorf("%q is a quantity", strings.Join(walked, "."))
		}

		switch current.Kind() { //nolint:exhaustive // the default branch covers every other kind
		case reflect.Map:
			current = current.Elem()
		case reflect.Struct:
			field, ok := fieldByJSONTag(current, segment)
			if !ok {
				return leafUnknown, fmt.Errorf("no field %q on %s", segment, current.Name())
			}
			current = field.Type
		default:
			return leafUnknown, fmt.Errorf("%q is not addressable", strings.Join(walked, "."))
		}
		walked = append(walked, segment)
	}

	return goLeaf(deref(current))
}

func goLeaf(t reflect.Type) (leafKind, error) {
	if t.Name() == quantityTypeName {
		return leafQuantity, nil
	}
	switch t.Kind() { //nolint:exhaustive // the default branch covers every other kind
	case reflect.String:
		return leafString, nil
	case reflect.Int, reflect.Int32, reflect.Int64:
		return leafInteger, nil
	case reflect.Float32, reflect.Float64:
		return leafNumber, nil
	case reflect.Bool:
		return leafBool, nil
	default:
		return leafUnknown, fmt.Errorf("cannot generate a value for %s", t.Kind())
	}
}

// schemaLeaf walks a declared parameters schema to the addressed property.
func schemaLeaf(schema *apiextensionsv1.JSONSchemaProps, segments []string) (leafKind, error) {
	current := schema
	for _, segment := range segments {
		next, ok := current.Properties[segment]
		if !ok {
			return leafUnknown, fmt.Errorf("no property %q in the declared parameters schema", segment)
		}
		current = &next
	}
	switch current.Type {
	case "string", "integer", "number", "boolean":
		return leafKind(current.Type), nil
	default:
		return leafUnknown, fmt.Errorf("cannot generate a value for schema type %q", current.Type)
	}
}

// parametersSchema returns the schema governing the payload at this position.
func parametersSchema(spec *corev1alpha1.ProviderSpec, topology string, walked []string) *apiextensionsv1.JSONSchemaProps {
	switch {
	case len(walked) == 3 && walked[0] == componentsSegment:
		if c, ok := spec.Components[walked[1]]; ok && c.ParametersSchema != nil {
			return c.ParametersSchema.OpenAPIV3Schema
		}
	case len(walked) == 2 && walked[0] == "topology":
		if t, ok := spec.Topologies[topology]; ok && t.ParametersSchema != nil {
			return t.ParametersSchema.OpenAPIV3Schema
		}
	case len(walked) == 1:
		if spec.ParametersSchema != nil {
			return spec.ParametersSchema.OpenAPIV3Schema
		}
	}
	return nil
}

// versionPath selects a version bundle, whose legal values the Provider CR
// enumerates. An invented token would be rejected, so the probe picks a real
// alternative and looks for the component versions it pulls in.
const versionPath = "spec.version"

// sentinel returns a value recognisable wherever it surfaces, and the token to
// search for. Integers get a distinctive prime so a bare number is unlikely to
// collide with an unrelated default.
func sentinel(spec *corev1alpha1.ProviderSpec, path string, kind leafKind) (any, string, error) {
	if path == versionPath {
		alternative := nonDefaultBundle(spec)
		if alternative == "" {
			return nil, "", errors.New("the provider declares fewer than two version bundles, so selecting one cannot be observed")
		}
		return alternative, alternative, nil
	}

	switch kind {
	case leafString:
		return "everest-conformance-7919", "everest-conformance-7919", nil
	case leafQuantity:
		// Must parse as a quantity, so the magnitude carries the signal.
		return "7919", "7919", nil
	case leafInteger:
		return int64(7919), "7919", nil
	case leafNumber:
		return 79.19, "79.19", nil
	// Bools have no usable token: "true" matches any boolean in the output, so
	// probeBool compares renders instead.
	case leafBool, leafUnknown:
	}
	return nil, "", fmt.Errorf("cannot generate a probe value for %q", path)
}

// nonDefaultBundle returns a bundle the baseline render would not have used.
func nonDefaultBundle(spec *corev1alpha1.ProviderSpec) string {
	for _, bundle := range spec.Versions {
		if !bundle.Default {
			return bundle.Name
		}
	}
	return ""
}

func deref(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}

func fieldByJSONTag(t reflect.Type, name string) (reflect.StructField, bool) {
	for field := range t.Fields() {
		tag, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		if tag == name {
			return field, true
		}
		if tag != "" || !field.Anonymous {
			continue
		}
		if embedded := deref(field.Type); embedded.Kind() == reflect.Struct {
			if promoted, ok := fieldByJSONTag(embedded, name); ok {
				return promoted, true
			}
		}
	}
	return reflect.StructField{}, false
}
