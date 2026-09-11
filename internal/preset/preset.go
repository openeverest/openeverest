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

// Package preset provides shared logic for manipulating the resolvable resource
// references contained in an Instance (or InstancePreset) spec.
//
// An InstancePreset is a cluster-scoped template mirroring an Instance spec.
// Several independent operations need to walk the same set of resource-reference
// fields in a spec and act on them:
//
//   - validation ensures namespace-scoped references are empty in a preset,
//   - draft-from clears namespace-scoped references when drafting a
//     preset from an Instance,
//   - resolve fills empty references with the namespace/cluster default.
//
// These references live in two representations: typed struct fields
// (Storage.StorageClass) and free-form Parameters JSON. This package
// abstracts both behind FieldRef and a single WalkRefs traversal so
// the operations above share one implementation, and adding a new
// reference type is a single registry entry.
package preset

import "slices"

// Scope describes whether a referenced resource is namespace- or cluster-scoped.
type Scope int

const (
	// ScopeNamespace marks resources that live inside a namespace (Secret,
	// ConfigMap, MonitoringConfig). Namespace scoped fields must be empty
	// in a preset.
	ScopeNamespace Scope = iota
	// ScopeCluster marks cluster-scoped resources (StorageClass). These may hold
	// a value in a preset and are only filled when empty during resolution.
	ScopeCluster
)

// Kind identifies the kind of referenced object (e.g. Secret, ConfigMap).
type Kind string

// Supported kind of objects referenced from a spec.
const (
	KindSecret           Kind = "Secret"
	KindConfigMap        Kind = "ConfigMap"
	KindMonitoringConfig Kind = "MonitoringConfig"
	KindStorageClass     Kind = "StorageClass"
)

// refObjectType couples a referenced object with its scope and the field names
// that reference it.
type refObjectType struct {
	kind       Kind
	scope      Scope
	fieldNames []string
}

// registry is the single source of truth for resolvable references.
// To support a new resource type, add one entry here. Adding a
// resource that also appears as a typed struct field additionally requires
// emitting it from walkComponent.
var registry = []refObjectType{ //nolint:gochecknoglobals // this is a static registry
	// TODO: add support for additional kind of resources if needed.
	{
		kind:       KindSecret,
		scope:      ScopeNamespace,
		fieldNames: []string{"secret", "secretRef", "secretName"}, //nolint:goconst // these are the valid field names
	},
	{
		kind:       KindConfigMap,
		scope:      ScopeNamespace,
		fieldNames: []string{"configMap", "configMapRef", "configMapName"}, //nolint:goconst // these are the valid field names
	},
	{
		kind:       KindMonitoringConfig,
		scope:      ScopeNamespace,
		fieldNames: []string{"monitoringConfig", "monitoringConfigRef", "monitoringConfigName"}, //nolint:goconst // these are the valid field names
	},
	{
		kind:       KindStorageClass,
		scope:      ScopeCluster,
		fieldNames: []string{"storageClass", "storageClassName"},
	},
}

// scopeOf returns whether the resource is cluster or namespace scope.
func scopeOf(resourceType Kind) Scope {
	for _, r := range registry {
		if r.kind == resourceType {
			return r.scope
		}
	}

	panic("unreachable: unknown resource type")
}

// refField returns the referenced resource for the given field name.
// Returns false if the field name is not a known reference.
func refField(field string) (refObjectType, bool) {
	for _, r := range registry {
		if slices.Contains(r.fieldNames, field) {
			return r, true
		}
	}
	return refObjectType{}, false
}
