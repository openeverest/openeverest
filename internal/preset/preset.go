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
//   - create-from-instance clears namespace-scoped references when turning an
//     Instance into a preset,
//   - resolve fills empty references with the namespace/cluster default.
//
// These references live in two representations: typed struct fields
// (Config.SecretRef, Config.ConfigMapRef, Storage.StorageClass) and free-form
// customSpec JSON. This package abstracts both behind FieldRef and a single
// WalkRefs traversal so the operations above share one implementation, and adding
// a new reference type is a single registry entry.
package preset

import "slices"

// Scope describes whether a referenced resource is namespace- or cluster-scoped.
type Scope int

const (
	// ScopeNamespace marks resources that live inside a namespace (Secret,
	// ConfigMap, MonitoringConfig). These must be empty in a preset and are
	// resolved from a namespace default.
	ScopeNamespace Scope = iota
	// ScopeCluster marks cluster-scoped resources (StorageClass). These may hold
	// a value in a preset and are only filled when empty during resolution.
	ScopeCluster
)

// ResourceKind identifies the Kubernetes resource a FieldRef points at.
type ResourceKind string

// Supported resource kinds referenced from a spec.
const (
	KindSecret           ResourceKind = "Secret"
	KindConfigMap        ResourceKind = "ConfigMap"
	KindMonitoringConfig ResourceKind = "MonitoringConfig"
	KindStorageClass     ResourceKind = "StorageClass"
)

// refKind couples a resource kind with its scope and the customSpec field names
// (aliases) that reference it.
type refKind struct {
	kind    ResourceKind
	scope   Scope
	aliases []string
}

// registry is the single source of truth for resolvable reference types.
// To support a new customSpec-referenced resource, add one entry here. Adding a
// resource that also appears as a typed struct field additionally requires
// emitting it from walkComponent.
var registry = []refKind{
	{
		kind:    KindSecret,
		scope:   ScopeNamespace,
		aliases: []string{"secret", "secretRef", "secretName"},
	},
	{
		kind:    KindConfigMap,
		scope:   ScopeNamespace,
		aliases: []string{"objectRef"},
	},
	{
		kind:    KindMonitoringConfig,
		scope:   ScopeNamespace,
		aliases: []string{"monitoringConfig", "monitoringConfigRef", "monitoringConfigName"},
	},
	{
		kind:    KindStorageClass,
		scope:   ScopeCluster,
		aliases: []string{"storageClass", "storageClassName"},
	},
}

// scopeOf returns the scope registered for a resource kind.
func scopeOf(kind ResourceKind) Scope {
	for _, rk := range registry {
		if rk.kind == kind {
			return rk.scope
		}
	}
	return ScopeNamespace
}

// aliasKind returns the reference kind registered for a customSpec field name.
func aliasKind(field string) (refKind, bool) {
	for _, rk := range registry {
		if slices.Contains(rk.aliases, field) {
			return rk, true
		}
	}
	return refKind{}, false
}
