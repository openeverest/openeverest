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

package backupclass

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"
	"sync"

	goversion "github.com/hashicorp/go-version"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utiljson "k8s.io/apimachinery/pkg/util/json"
	"sigs.k8s.io/yaml"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	"github.com/openeverest/openeverest/v2/config/crd/bases"
)

// backupClassCRD is the file of the BackupClass CRD in bases.CRDs.
const backupClassCRD = "backup.openeverest.io_backupclasses.yaml"

// Claim returns the Accepted condition provider sets on class, a class that
// names provider as its executor, given the classes the provider ships:
//   - Unknown with NewerDefinition or OlderDefinition when class is stamped
//     with another provider release than the shipped class of its name;
//   - otherwise True when the capabilities of class equal those of the
//     shipped class, and False with SpecDrift, naming the fields that
//     differ, when they do not;
//   - False with UnknownClass when the provider ships no class of its name.
//
// The capabilities are the whole spec except displayName, description and
// uiSchema, compared in a canonical form (see Capabilities).
func Claim(class *backupv1alpha1.BackupClass, shipped []backupv1alpha1.BackupClass, provider string) (metav1.Condition, error) {
	cond := metav1.Condition{
		Type:               backupv1alpha1.BackupClassConditionAccepted,
		ObservedGeneration: class.Generation,
	}
	i := slices.IndexFunc(shipped, func(s backupv1alpha1.BackupClass) bool { return s.Name == class.Name })
	if i < 0 {
		cond.Status = metav1.ConditionFalse
		cond.Reason = backupv1alpha1.ReasonUnknownClass
		cond.Message = fmt.Sprintf("provider %s does not ship a BackupClass named %s", provider, class.Name)
		return cond, nil
	}
	own := &shipped[i]

	stamp, release := class.Annotations[backupv1alpha1.ProviderVersionAnnotation], own.Annotations[backupv1alpha1.ProviderVersionAnnotation]
	if stamp != "" {
		switch compareReleases(stamp, release) {
		case 1:
			cond.Status = metav1.ConditionUnknown
			cond.Reason = backupv1alpha1.ReasonNewerDefinition
			cond.Message = fmt.Sprintf(
				"BackupClass %s is the definition of provider %s release %s, newer than %s, which this provider runs: "+
					"waiting for the provider to be upgraded", class.Name, provider, stamp, releaseName(release),
			)
			return cond, nil
		case -1:
			cond.Status = metav1.ConditionUnknown
			cond.Reason = backupv1alpha1.ReasonOlderDefinition
			cond.Message = fmt.Sprintf(
				"BackupClass %s is the definition of provider %s release %s, older than %s, which this provider runs: "+
					"waiting for the class to be upgraded", class.Name, provider, stamp, releaseName(release),
			)
			return cond, nil
		}
	}

	drift, err := CapabilityDiff(&class.Spec, &own.Spec)
	if err != nil {
		return metav1.Condition{}, err
	}
	if len(drift) > 0 {
		cond.Status = metav1.ConditionFalse
		cond.Reason = backupv1alpha1.ReasonSpecDrift
		cond.Message = fmt.Sprintf("the capabilities of BackupClass %s differ from those of the class provider %s ships: %s",
			class.Name, provider, strings.Join(drift, ", "),
		)
		return cond, nil
	}
	cond.Status = metav1.ConditionTrue
	cond.Reason = backupv1alpha1.ReasonAccepted
	cond.Message = fmt.Sprintf("provider %s executes BackupClass %s", provider, class.Name)
	return cond, nil
}

// compareReleases returns 0 when the class stamped with stamp belongs to
// release, 1 when it belongs to a newer release and -1 when to an older one.
// A stamp that differs from release where either is not a version counts as
// newer.
func compareReleases(stamp, release string) int {
	if stamp == release {
		return 0
	}
	s, err := goversion.NewVersion(stamp)
	if err != nil {
		return 1
	}
	r, err := goversion.NewVersion(release)
	if err != nil {
		return 1
	}
	if c := s.Compare(r); c != 0 {
		return c
	}
	// Equal versions spelled differently, such as v1.0.0 and 1.0.0, are one
	// release.
	return 0
}

func releaseName(release string) string {
	if release == "" {
		return "an unstamped release"
	}
	return "release " + release
}

// CapabilityDiff returns the paths of the capability fields where a and b
// differ, such as spec.backupStorageClasses[s3].tls, in order. Parameter
// schemas are reported as a whole.
func CapabilityDiff(a, b *backupv1alpha1.BackupClassSpec) ([]string, error) {
	ca, err := Capabilities(a)
	if err != nil {
		return nil, err
	}
	cb, err := Capabilities(b)
	if err != nil {
		return nil, err
	}
	var paths []string
	diff("spec", ca, cb, &paths)
	return paths, nil
}

// Capabilities returns the capability fields of spec in a canonical form
// that compares equal for specs the API server stores alike: every field
// except displayName, description and uiSchema, as JSON values, without the
// fields that hold the default the BackupClass CRD gives them, and without
// the typed objects and lists that hold nothing. A spec as written in a
// manifest and the same spec read back from the API server, which applied
// the CRD's defaults and reordered schema keys, have the same capabilities.
func Capabilities(spec *backupv1alpha1.BackupClassSpec) (map[string]any, error) {
	s := spec.DeepCopy()
	s.DisplayName, s.Description, s.UISchema = "", "", nil
	raw, err := json.Marshal(s)
	if err != nil {
		return nil, fmt.Errorf("encode BackupClass spec: %w", err)
	}
	var v map[string]any
	if err := utiljson.Unmarshal(raw, &v); err != nil {
		return nil, fmt.Errorf("decode BackupClass spec: %w", err)
	}
	schema, err := specSchema()
	if err != nil {
		return nil, err
	}
	normalize(v, schema)
	return v, nil
}

// specSchema returns the schema of BackupClass.spec in the CRD this binary
// was built with.
var specSchema = sync.OnceValues(func() (*apiextensionsv1.JSONSchemaProps, error) { //nolint:gochecknoglobals // parsed once, read-only
	data, err := bases.CRDs.ReadFile(backupClassCRD)
	if err != nil {
		return nil, err
	}
	crd := &apiextensionsv1.CustomResourceDefinition{}
	if err := yaml.Unmarshal(data, crd); err != nil {
		return nil, fmt.Errorf("%s: %w", backupClassCRD, err)
	}
	for _, v := range crd.Spec.Versions {
		if v.Name != backupv1alpha1.GroupVersion.Version || v.Schema == nil || v.Schema.OpenAPIV3Schema == nil {
			continue
		}
		if spec, ok := v.Schema.OpenAPIV3Schema.Properties["spec"]; ok {
			return &spec, nil
		}
	}
	return nil, errors.New(backupClassCRD + ": no spec schema for " + backupv1alpha1.GroupVersion.Version)
})

// normalize removes from v, a value of schema, every field that holds the
// default schema declares for it, and every object or list field of a typed
// schema that holds nothing: for an optional field, the API server stores an
// absent value and a default, or an empty one, alike. Only the structure
// schema declares is walked: the contents of schemaless fields, such as
// parameter schemas, are kept as they are, since an empty schema is not an
// absent one.
func normalize(v any, schema *apiextensionsv1.JSONSchemaProps) {
	switch x := v.(type) {
	case map[string]any:
		for k, child := range x {
			s, ok := fieldSchema(schema, k)
			if !ok {
				continue
			}
			if s.Default != nil {
				var def any
				if err := utiljson.Unmarshal(s.Default.Raw, &def); err == nil && reflect.DeepEqual(child, def) {
					delete(x, k)
					continue
				}
			}
			normalize(child, s)
			if typed(s) && empty(child) {
				delete(x, k)
			}
		}
	case []any:
		if schema.Items == nil || schema.Items.Schema == nil {
			return
		}
		for _, e := range x {
			normalize(e, schema.Items.Schema)
		}
	}
}

// fieldSchema returns the schema of the field k of an object of schema.
func fieldSchema(schema *apiextensionsv1.JSONSchemaProps, k string) (*apiextensionsv1.JSONSchemaProps, bool) {
	if s, ok := schema.Properties[k]; ok {
		return &s, true
	}
	if schema.AdditionalProperties != nil && schema.AdditionalProperties.Schema != nil {
		return schema.AdditionalProperties.Schema, true
	}
	return nil, false
}

// typed reports whether schema declares the structure of its values: an
// object with properties or a list of items.
func typed(schema *apiextensionsv1.JSONSchemaProps) bool {
	return len(schema.Properties) > 0 || (schema.Items != nil && schema.Items.Schema != nil)
}

// empty reports whether v is null, or an object or list without elements.
func empty(v any) bool {
	switch x := v.(type) {
	case nil:
		return true
	case map[string]any:
		return len(x) == 0
	case []any:
		return len(x) == 0
	}
	return false
}

// diff appends to paths the path of every value where a and b differ,
// descending into objects, and into lists whose elements are objects keyed
// by name. Parameter schemas are compared as a whole.
func diff(path string, a, b any, paths *[]string) {
	if reflect.DeepEqual(a, b) {
		return
	}
	if am, ok := a.(map[string]any); ok && !strings.HasSuffix(strings.ToLower(path), "parametersschema") {
		if bm, ok := b.(map[string]any); ok {
			for _, k := range slices.Sorted(maps.Keys(union(am, bm))) {
				diff(path+"."+k, am[k], bm[k], paths)
			}
			return
		}
	}
	if al, ok := byName(a); ok {
		if bl, ok := byName(b); ok {
			for _, k := range slices.Sorted(maps.Keys(union(al, bl))) {
				diff(path+"["+k+"]", al[k], bl[k], paths)
			}
			return
		}
	}
	*paths = append(*paths, path)
}

func union(a, b map[string]any) map[string]any {
	u := make(map[string]any, len(a)+len(b))
	maps.Copy(u, a)
	maps.Copy(u, b)
	return u
}

// byName returns the elements of a list of objects that each have a
// distinct string name, keyed by it.
func byName(v any) (map[string]any, bool) {
	list, ok := v.([]any)
	if !ok {
		return nil, false
	}
	out := make(map[string]any, len(list))
	for _, e := range list {
		m, ok := e.(map[string]any)
		if !ok {
			return nil, false
		}
		name, ok := m["name"].(string)
		if !ok {
			return nil, false
		}
		if _, dup := out[name]; dup {
			return nil, false
		}
		out[name] = e
	}
	return out, true
}
