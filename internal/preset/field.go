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

import (
	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

// FieldRef is a handle to a single resolvable resource reference inside a spec.
// It hides whether the reference is a typed struct field or a dynamic customSpec
// entry so callers can read, test and write it uniformly.
type FieldRef interface {
	// Component is the name of the component that owns the reference.
	Component() string
	// Kind is the Kubernetes resource the reference points at.
	Kind() ResourceKind
	// Scope reports whether the resource is namespace- or cluster-scoped.
	Scope() Scope
	// Path is a human-readable location used in error messages.
	Path() string
	// IsEmpty reports whether the reference is unset.
	IsEmpty() bool
	// Set writes the referenced resource name.
	Set(name string)
}

// meta holds the metadata shared by every FieldRef implementation.
type meta struct {
	component string
	kind      ResourceKind
	scope     Scope
	path      string
}

func (m meta) Component() string  { return m.component }
func (m meta) Kind() ResourceKind { return m.kind }
func (m meta) Scope() Scope       { return m.scope }
func (m meta) Path() string       { return m.path }

// newMeta builds a meta, deriving the scope from the registry so scope is defined
// in exactly one place.
func newMeta(component string, kind ResourceKind, path string) meta {
	return meta{component: component, kind: kind, scope: scopeOf(kind), path: path}
}

// nameRef references a required string field such as LocalObjectReference.Name.
type nameRef struct {
	meta

	value *string
}

func (r nameRef) IsEmpty() bool { return r.value == nil || *r.value == "" }

func (r nameRef) Set(name string) {
	if r.value != nil {
		*r.value = name
	}
}

// storageClassRef references the optional Storage.StorageClass pointer, which may
// be nil and therefore needs allocation on write.
type storageClassRef struct {
	meta

	storage *corev1alpha1.Storage
}

func (r storageClassRef) IsEmpty() bool {
	return r.storage.StorageClass == nil || *r.storage.StorageClass == ""
}

func (r storageClassRef) Set(name string) {
	r.storage.StorageClass = &name
}

// objectRef references a common.ObjectRef field.
type objectRef struct {
	meta

	ref *common.ObjectRef
}

func (r objectRef) IsEmpty() bool {
	return r.ref == nil || r.ref.Name == ""
}

func (r objectRef) Set(name string) {
	if r.ref != nil {
		r.ref.Name = name
	}
}

// secretRef references a common.SecretRef field.
type secretRef struct {
	meta

	ref *common.SecretRef
}

func (r secretRef) IsEmpty() bool {
	return r.ref == nil || r.ref.Name == ""
}

func (r secretRef) Set(name string) {
	if r.ref != nil {
		r.ref.Name = name
	}
}

// parametersRef references a customSpec entry. The value may be a plain string or a
// ref-like object such as {"name": ""}; both shapes are handled transparently.
// Writes flip the shared dirty flag so the traversal knows to re-marshal.
type parametersRef struct {
	meta
	parent map[string]any
	key    string
	dirty  *bool
}

func (r parametersRef) IsEmpty() bool {
	switch value := r.parent[r.key].(type) {
	case string:
		return value == ""
	case map[string]any:
		if len(value) == 0 {
			return true
		}
		if len(value) == 1 {
			if name, ok := value["name"].(string); ok {
				return name == ""
			}
		}
	}
	return false
}

func (r parametersRef) Set(name string) {
	if obj, ok := r.parent[r.key].(map[string]any); ok {
		obj["name"] = name
	} else {
		r.parent[r.key] = name
	}
	*r.dirty = true
}
