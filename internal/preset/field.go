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

package preset

// FieldRef is a handle to a resolvable resource reference inside a spec.
type FieldRef interface {
	// Component is the name of the component that holds the reference.
	Component() string
	// FieldKind is the kind of referenced object (e.g. Secret, ConfigMap).
	FieldKind() Kind
	// Scope reports whether the resource is namespace- or cluster-scoped.
	Scope() Scope
	// Path is a human-readable location used in error messages.
	Path() string
	// IsEmpty reports whether the reference is unset.
	IsEmpty() bool
	// Set writes the referenced resource namespace and name.
	// If the namespace key is not set in the underlying object
	// or its cluster-scoped object, the namespace argument is ignored.
	Set(namespace, name string)
}

// meta holds the metadata shared by every FieldRef implementation.
type meta struct {
	component string
	kind      Kind
	scope     Scope
	path      string
}

func (m meta) Component() string { return m.component }
func (m meta) FieldKind() Kind   { return m.kind }
func (m meta) Scope() Scope      { return m.scope }
func (m meta) Path() string      { return m.path }

// newMeta builds a meta, deriving the scope from the registry so scope is defined
// in exactly one place.
func newMeta(component string, kind Kind, path string) meta {
	return meta{component: component, kind: kind, scope: scopeOf(kind), path: path}
}
