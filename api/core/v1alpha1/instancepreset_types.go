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
)

// InstancePresetSpec defines the desired state of InstancePreset
type InstancePresetSpec struct {
	InstanceSpec `json:",inline"`

	// DefaultRefs lists the namespace references this preset requires. Each
	// listed path must be resolved from the target namespace's NamespaceDefaults
	// at resolve time; the preset itself carries no value for it. References
	// not listed here keep their literal value — an empty value stays empty.
	//
	// This field is preset-only: it is stripped during resolve and is never
	// copied onto an Instance.
	// +optional
	// +listType=atomic
	// +kubebuilder:validation:MaxItems=32
	// +kubebuilder:validation:XValidation:rule="self.all(x, self.exists_one(y, x.path == y.path))",message="each path must be unique"
	DefaultRefs []PresetDefaultRef `json:"defaultRefs,omitempty"`
}

// PresetDefaultRef marks a single reference that must be resolved from the
// target namespace's default, identified by its path within the Instance spec.
type PresetDefaultRef struct {
	// Path is the reference field's dot-separated location under Instance.spec.
	// Component references are "components.<name>.parameters.<field>" (e.g.
	// "components.monitoring.parameters.monitoringConfigRef.name"),
	// and top-level references are the field name (e.g. "userSecretRef.name").
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=1024
	Path string `json:"path"`
}

// InstancePresetStatus defines the observed state of InstancePreset.
type InstancePresetStatus struct {
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,shortName=pre;preset

// InstancePreset is the Schema for the instancepresets API
type InstancePreset struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of InstancePreset
	// +required
	Spec InstancePresetSpec `json:"spec"`

	// status defines the observed state of InstancePreset
	// +optional
	Status InstancePresetStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// InstancePresetList contains a list of InstancePreset
type InstancePresetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []InstancePreset `json:"items"`
}

func init() {
	SchemeBuilder.Register(&InstancePreset{}, &InstancePresetList{})
}
