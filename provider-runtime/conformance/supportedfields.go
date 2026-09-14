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
	"reflect"
	"strings"
	"testing"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

// maxDeclarationDepth bounds the walk into a declared field. Nothing in
// ComponentSpec needs more, and a cycle would otherwise not terminate.
const maxDeclarationDepth = 4

// SupportedFieldsAreReconciled asserts that every field a component declares
// in supportedFields is one the provider actually consumes.
//
// The declaration is a promise to the API: an undeclared field is rejected, so
// a declared one that changes nothing is the silent no-op the declaration
// exists to remove, now published as a contract.
//
// A property declared without nested properties promises the whole struct
// beneath it, so it is probed at every scalar that struct reaches. Declaring
// schedulingPolicy as a whole therefore has to hold for schedulerName too, not
// only for the affinity the provider happens to read.
func SupportedFieldsAreReconciled(t *testing.T, cfg Config) {
	t.Helper()

	spec, err := loadProviderSpec(cfg.ProviderSpecPath)
	if err != nil {
		t.Fatalf("conformance: %v", err)
	}

	var results []result
	for _, topology := range sortedKeys(spec.Topologies) {
		paths := declaredPaths(spec, topology)
		if len(paths) == 0 {
			continue
		}
		results = append(results, probeTopology(t, cfg, spec, topology, paths)...)
	}

	report(t, cfg, results, supportedFieldsSource())
}

func supportedFieldsSource() source {
	return source{
		subject: "supportedFields declares",
		ignored: "declared in supportedFields, but setting it changes nothing the provider applies or requests",
	}
}

// declaredPaths returns the scalar Instance fields the topology's components
// declare they honour.
func declaredPaths(spec *corev1alpha1.ProviderSpec, topology string) []string {
	componentSpec := reflect.TypeFor[corev1alpha1.ComponentSpec]()

	var paths []string
	for _, name := range sortedKeys(spec.Topologies[topology].Components) {
		component := spec.Topologies[topology].Components[name]
		if component.SupportedFields == nil || component.SupportedFields.OpenAPIV3Schema == nil {
			continue
		}
		expandDeclaration(
			componentSpec,
			component.SupportedFields.OpenAPIV3Schema,
			"spec."+componentsSegment+"."+name,
			&paths,
		)
	}
	return dedupe(paths)
}

// expandDeclaration walks a declared schema against the type it selects from,
// descending where the declaration does and expanding the Go type where it
// stops.
func expandDeclaration(t reflect.Type, schema *apiextensionsv1.JSONSchemaProps, prefix string, out *[]string) {
	t = deref(t)
	if t.Kind() != reflect.Struct {
		return
	}
	for _, name := range sortedKeys(schema.Properties) {
		property := schema.Properties[name]
		field, ok := fieldByJSONTag(t, name)
		if !ok {
			// The generator rejects a property that names no field, so by the
			// time a spec is published there is nothing to report here.
			continue
		}
		path := prefix + "." + name
		if len(property.Properties) > 0 {
			expandDeclaration(field.Type, &property, path, out)
			continue
		}
		scalarLeaves(field.Type, path, out, 0)
	}
}

// scalarLeaves appends every scalar path reachable from t.
//
// It stops at maps, slices and parameters payloads: those have no fixed member
// to address, so no probe can name one. A field that reaches no scalar is
// simply not probed — the harness reports what it proved, not what it wished.
func scalarLeaves(t reflect.Type, prefix string, out *[]string, depth int) {
	t = deref(t)
	if _, err := goLeaf(t); err == nil {
		*out = append(*out, prefix)
		return
	}
	if depth >= maxDeclarationDepth || t.Kind() != reflect.Struct || t.Name() == "RawExtension" {
		return
	}

	for field := range t.Fields() {
		tag, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		if tag == "" {
			if field.Anonymous {
				// Inlined by core, so its fields are addressed as if declared here.
				scalarLeaves(field.Type, prefix, out, depth)
			}
			continue
		}
		scalarLeaves(field.Type, prefix+"."+tag, out, depth+1)
	}
}
