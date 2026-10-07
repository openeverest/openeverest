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

// ClusterInstancePresetSpec defines the desired state of ClusterInstancePreset.
type ClusterInstancePresetSpec struct {
	InstanceSpec `json:",inline"`
}

// ClusterInstancePresetStatus defines the observed state of ClusterInstancePreset.
type ClusterInstancePresetStatus struct {
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,shortName=cpre;cpreset

// ClusterInstancePreset is a cluster-scoped, template of portable provider
// defaults for creating Instances.
//
// Because it is cluster-scoped it is available in every namespace, therefore
// must not embed any namespace-scoped reference.
//
// It is similar to an InstancePreset, however it is cluster-scoped and
// therefore can be used to create Instances in any namespace, not just the
// namespace of the referent.
type ClusterInstancePreset struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of ClusterInstancePreset
	// +required
	Spec ClusterInstancePresetSpec `json:"spec"`

	// status defines the observed state of ClusterInstancePreset
	// +optional
	Status ClusterInstancePresetStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// ClusterInstancePresetList contains a list of ClusterInstancePreset
type ClusterInstancePresetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []ClusterInstancePreset `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterInstancePreset{}, &ClusterInstancePresetList{})
}
