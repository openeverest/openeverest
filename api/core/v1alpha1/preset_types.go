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

// PresetSpec defines the desired state of Preset
type PresetSpec struct {
	// Template holds the configuration values this preset provides.
	// When an Instance is created from this preset, the template is used
	// as the starting point for the Instance's spec.
	// The provider field in template is required and determines which
	// provider this preset targets.
	// +kubebuilder:validation:Required
	Template InstanceSpec `json:"template"`
}

// PresetStatus defines the observed state of Preset.
type PresetStatus struct {
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=ps;preset

// Preset is the Schema for the presets API. A Preset provides a reusable
// set of default values for creating Instances, enabling one-click
// deployment with sensible defaults.
type Preset struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of Preset
	// +required
	Spec PresetSpec `json:"spec"`

	// status defines the observed state of Preset
	// +optional
	Status PresetStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// PresetList contains a list of Preset
type PresetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []Preset `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Preset{}, &PresetList{})
}
