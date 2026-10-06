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

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
)

// NamespaceDefaultsSpec declares the per-namespace default resource for each
// namespace-scoped reference an Instance can carry.
type NamespaceDefaultsSpec struct {
	// Defaults lists the default resource for each referenced path, optionally
	// scoped to a provider.
	// +optional
	// +listType=atomic
	// +kubebuilder:validation:MaxItems=32
	// +kubebuilder:validation:XValidation:rule="self.all(x, self.exists_one(y, (has(x.providerRef) ? x.providerRef.name : '') == (has(y.providerRef) ? y.providerRef.name : '') && x.path == y.path))",message="each (providerRef, path) pair must be unique"
	Defaults []NamespaceDefault `json:"defaults,omitempty"`
}

// NamespaceDefault declares the default resource for a single reference,
// identified by its path within the Instance spec.
type NamespaceDefault struct {
	// ProviderRef scopes this entry to one provider. Omit for any provider;
	// a provider-scoped entry takes precedence over an agnostic one for the
	// same path.
	// +optional
	ProviderRef *common.ObjectRef `json:"providerRef,omitempty"`

	// Path is the reference field's dot-separated location under Instance.spec.
	// The component references are "components.<name>.<field>" (e.g.
	// "components.monitoring.monitoringConfigRef.name"), and top-level references
	// are the bare field name (e.g. "userSecretRef").
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=1024
	Path string `json:"path"`

	// Name of the default resource in this namespace.
	// +kubebuilder:validation:Required
	DefaultRef common.ObjectRef `json:"defaultRef"`
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
// +kubebuilder:validation:XValidation:rule="self.metadata.name == 'defaults'",message="only one NamespaceDefaults per namespace and it must be named 'defaults'"

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
