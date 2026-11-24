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

package rbac

import (
	"encoding/json"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"strings"

	"github.com/google/go-cmp/cmp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

// PresetMismatchError refuses an instance created with
// instances:create-from-preset that is not an exact copy of the preset it
// names. It is an ErrInsufficientPermissions, so it answers 403, but it names
// what differs, so the user can tell a preset that changed under them from a
// permission they lack.
type PresetMismatchError struct {
	// PresetKind is InstancePreset or ClusterInstancePreset.
	PresetKind string
	PresetName string
	// Fields are the paths at which the instance spec differs from the preset.
	Fields []string
	// Metadata are the paths of the metadata the instance sets that a copy of
	// the preset may not.
	Metadata []string
}

func (e *PresetMismatchError) Error() string {
	var differences []string
	if len(e.Fields) > 0 {
		differences = append(differences, "changes "+strings.Join(e.Fields, ", "))
	}
	if len(e.Metadata) > 0 {
		differences = append(differences, "sets "+strings.Join(e.Metadata, ", "))
	}
	return fmt.Sprintf(
		"instance does not match %s %q: it %s. instances:%s allows only an exact copy of the preset; changes need instances:%s",
		e.PresetKind, e.PresetName, strings.Join(differences, " and "), rbac.ActionCreateFromPreset, rbac.ActionCreate,
	)
}

// Is makes a mismatch an ErrInsufficientPermissions.
func (e *PresetMismatchError) Is(target error) bool {
	return target == ErrInsufficientPermissions
}

// Causes returns one cause per field that differs from the preset.
func (e *PresetMismatchError) Causes() []metav1.StatusCause {
	causes := make([]metav1.StatusCause, 0, len(e.Fields)+len(e.Metadata))
	for _, field := range e.Fields {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeForbidden,
			Message: fmt.Sprintf("differs from %s %q", e.PresetKind, e.PresetName),
			Field:   field,
		})
	}
	for _, field := range e.Metadata {
		causes = append(causes, metav1.StatusCause{
			Type:    metav1.CauseTypeForbidden,
			Message: "not allowed on an instance created from a preset",
			Field:   field,
		})
	}
	return causes
}

// presetMismatch returns a PresetMismatchError if instance is not an exact
// copy of the preset: the same spec, and no metadata but what
// metadataOutsideAllowlist allows. annotation is the key of the annotation
// naming the preset.
func presetMismatch(instance *corev1alpha1.Instance, preset corev1alpha1.InstanceSpec, kind, name, annotation string) error {
	fields := specDifferences(preset, instance.Spec)
	metadata := metadataOutsideAllowlist(instance.ObjectMeta, annotation)
	if len(fields) == 0 && len(metadata) == 0 {
		return nil
	}
	return &PresetMismatchError{PresetKind: kind, PresetName: name, Fields: fields, Metadata: metadata}
}

// specDifferences returns the JSON path of every field at which spec differs
// from preset, in the order the fields are compared.
func specDifferences(preset, spec corev1alpha1.InstanceSpec) []string {
	r := &fieldPathReporter{}
	cmp.Equal(preset, spec, cmp.Reporter(r))
	return r.paths
}

// fieldPathReporter records the JSON path of every value at which two
// objects differ.
type fieldPathReporter struct {
	steps cmp.Path
	paths []string
}

func (r *fieldPathReporter) PushStep(step cmp.PathStep) {
	r.steps = append(r.steps, step)
}

func (r *fieldPathReporter) PopStep() {
	r.steps = r.steps[:len(r.steps)-1]
}

func (r *fieldPathReporter) Report(result cmp.Result) {
	if result.Equal() {
		return
	}
	if path := jsonPath(r.steps); !slices.Contains(r.paths, path) {
		r.paths = append(r.paths, path)
	}
}

// jsonPath renders steps, which start at an InstanceSpec, as the path of the
// field in the Instance's JSON, e.g. spec.components[engine].replicas.
func jsonPath(steps cmp.Path) string {
	path := "spec"
	for i, step := range steps {
		// Below a type that encodes itself, such as the RawExtension holding
		// parameters, Go fields are not JSON fields: name the whole value.
		if i > 0 && encodesItself(steps[i-1].Type()) {
			break
		}
		switch s := step.(type) {
		case cmp.StructField:
			if name := jsonFieldName(steps[i-1].Type(), s.Index()); name != "" {
				path += "." + name
			}
		case cmp.MapIndex:
			path += fmt.Sprintf("[%v]", s.Key())
		case cmp.SliceIndex:
			// An element only one side has: name the list.
			if s.Key() < 0 {
				return path
			}
			path += fmt.Sprintf("[%d]", s.Key())
		}
	}
	return path
}

//nolint:gochecknoglobals // immutable lookup value
var jsonMarshaler = reflect.TypeFor[json.Marshaler]()

func encodesItself(t reflect.Type) bool {
	return t.Implements(jsonMarshaler) || reflect.PointerTo(t).Implements(jsonMarshaler)
}

// jsonFieldName returns the JSON name of the index-th field of the struct
// parent, or "" for a field inlined into its parent.
func jsonFieldName(parent reflect.Type, index int) string {
	for parent.Kind() == reflect.Pointer {
		parent = parent.Elem()
	}
	field := parent.Field(index)
	name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
	switch {
	case name == "" && field.Anonymous:
		return ""
	case name == "":
		return field.Name
	}
	return name
}

// presetAddressFields are the metadata fields a copy of a preset may set:
// those saying where the instance goes and what it is called.
//
//nolint:gochecknoglobals // immutable lookup table
var presetAddressFields = []string{"name", "generateName", "namespace"}

// metadataOutsideAllowlist returns the path of every metadata field or key
// meta sets that a copy of a preset may not. The allowlist is the instance's
// address and the annotation naming the preset. Any other label or
// annotation, finalizer or owner reference can steer whatever controller,
// scheduler or policy reads it, so it is refused rather than vetted key by
// key, and a field ObjectMeta gains is refused until it is allowed here.
func metadataOutsideAllowlist(meta metav1.ObjectMeta, presetAnnotation string) []string {
	var paths []string
	for _, key := range slices.Sorted(maps.Keys(meta.Labels)) {
		paths = append(paths, fmt.Sprintf("metadata.labels[%s]", key))
	}
	for _, key := range slices.Sorted(maps.Keys(meta.Annotations)) {
		if key != presetAnnotation {
			paths = append(paths, fmt.Sprintf("metadata.annotations[%s]", key))
		}
	}

	value := reflect.ValueOf(meta)
	for i := range value.NumField() {
		name := jsonFieldName(value.Type(), i)
		if name == "labels" || name == "annotations" || slices.Contains(presetAddressFields, name) || isEmpty(value.Field(i)) {
			continue
		}
		paths = append(paths, "metadata."+name)
	}
	return paths
}

// isEmpty reports whether v is the zero value, or a slice or map with no
// elements, which JSON cannot tell apart from an omitted one.
func isEmpty(v reflect.Value) bool {
	if v.Kind() == reflect.Slice || v.Kind() == reflect.Map {
		return v.Len() == 0
	}
	return v.IsZero()
}
