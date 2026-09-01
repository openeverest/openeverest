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

// Package v1alpha1 contains the shared reference types used by every
// OpenEverest API group to point at other objects (custom resources,
// Secrets, ConfigMaps, Services, and operator-native resources).
//
// Conventions:
//
//   - Every cross-object reference is a struct field named "{something}Ref"
//     using one of the types below. Bare string fields are reserved for
//     intra-object logical keys (e.g. a schedule name defined elsewhere in
//     the same object) and are never object references.
//   - References never cross namespaces unless the referrer is
//     cluster-scoped, in which case NamespacedObjectRef is used.
//
// Fork-on-divergence policy: these types are intentionally generic and
// shared. When a specific usage needs extra fields (e.g. a key selector, a
// namespace, a uid pin), fork that usage into its own purpose-built type
// instead of widening the shared one. Swapping the Go type of a field is
// invisible on the wire as long as the serialized shape stays a superset,
// so this is always a non-breaking move.
//
// +kubebuilder:object:generate=true
package v1alpha1

// ObjectRef references an object by name. The target is either in the same
// namespace as the referrer or cluster-scoped; which one applies is fixed by
// the field using this type and documented there.
//
// +structType=atomic
type ObjectRef struct {
	// Name of the referenced object.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`
	Name string `json:"name"`
}

// SecretRef references a whole Secret in the same namespace as the referrer.
//
// +structType=atomic
type SecretRef struct {
	// Name of the referenced Secret.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`
	Name string `json:"name"`
}

// NamespacedObjectRef references a namespaced object from a cluster-scoped
// referrer, so the namespace must be spelled out. Namespace-scoped referrers
// must use ObjectRef instead; cross-namespace references between namespaced
// objects are not permitted.
//
// +structType=atomic
type NamespacedObjectRef struct {
	// Name of the referenced object.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`
	Name string `json:"name"`
	// Namespace of the referenced object.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`
	Namespace string `json:"namespace"`
}

// TypedObjectRef references an object of an arbitrary kind in the same
// namespace as the referrer. It is used in status fields to point at
// operator-native resources (e.g. PerconaServerMongoDBBackup) whose kind is
// only known to the provider that created them.
//
// +structType=atomic
type TypedObjectRef struct {
	// Group is the API group of the referenced object. Empty for objects in
	// the core API group.
	// +optional
	// +kubebuilder:validation:MaxLength=253
	Group string `json:"group,omitempty"`
	// Kind of the referenced object.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	Kind string `json:"kind"`
	// Name of the referenced object.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	Name string `json:"name"`
}
