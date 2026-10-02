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

package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// NamespaceDefaultsSpec declares the per-namespace default resource for each
// kind of namespace-scoped resource an Instance can reference.
//
// During preset resolution the server fills each empty, namespace-scoped
// reference in the resolved InstancePreset from the matching entry here.
type NamespaceDefaultsSpec struct {
	// Defaults lists the default resource for each referenced kind. At most
	// one entry may exist per matching key: kind alone for managed CRs such as
	// MonitoringConfig, or (kind, definition) for Secret and ConfigMap, whose
	// generic kind is disambiguated by the openeverest.io/definition the
	// OpenEverest API stamps on resources it creates.
	// +optional
	// +listType=atomic
	Defaults []NamespaceDefault `json:"defaults,omitempty"`
}

// NamespaceDefault declares the default resource of a single referenced kind
// within the namespace.
type NamespaceDefault struct {
	// Name is the name of the default resource in this namespace.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// Kind is the referenced resource kind, e.g. "MonitoringConfig",
	// "Secret", or "ConfigMap".
	// +kubebuilder:validation:Required
	Kind string `json:"kind"`

	// Definition disambiguates generic kinds (Secret, ConfigMap) by the
	// provider definition the resource was created from, matching the
	// openeverest.io/definition label the OpenEverest API stamps on those
	// resources. It is required for Secret and ConfigMap and omitted for
	// CRs.
	// +optional
	Definition string `json:"definition,omitempty"`
}

// NamespaceDefaultsStatus defines the observed state of NamespaceDefaults.
type NamespaceDefaultsStatus struct {
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=nsdef;nsdefaults
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// NamespaceDefaults is the Schema for the namespacedefaults API.
type NamespaceDefaults struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of NamespaceDefaults
	// +required
	Spec NamespaceDefaultsSpec `json:"spec"`

	// status defines the observed state of NamespaceDefaults
	// +optional
	Status NamespaceDefaultsStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// NamespaceDefaultsList contains a list of NamespaceDefaults.
type NamespaceDefaultsList struct {
	metav1.TypeMeta `json:",inline"`

	metav1.ListMeta `json:"metadata,omitzero"`

	Items []NamespaceDefaults `json:"items"`
}

//nolint:gochecknoinits
func init() {
	SchemeBuilder.Register(&NamespaceDefaults{}, &NamespaceDefaultsList{})
}
