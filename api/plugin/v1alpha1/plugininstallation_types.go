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

// PluginInstallationSpec defines the desired state of PluginInstallation
type PluginInstallationSpec struct {
	// PluginName references the cluster-scoped Plugin CR by name.
	// +required
	PluginName string `json:"pluginName"`

	// Enabled controls whether the plugin is active in this namespace.
	// +optional
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`

	// ConfigSecretRef is an optional reference to a Secret in the same namespace
	// that holds plugin-specific configuration (mounted as env vars in the backend).
	// +optional
	ConfigSecretRef string `json:"configSecretRef,omitempty"`
}

// PluginInstallationStatus defines the observed state of PluginInstallation.
type PluginInstallationStatus struct {
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=pi;plugininstall
// +kubebuilder:printcolumn:name="Plugin",type="string",JSONPath=".spec.pluginName"
// +kubebuilder:printcolumn:name="Enabled",type="boolean",JSONPath=".spec.enabled"

// PluginInstallation is the Schema for the plugininstallations API
type PluginInstallation struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// +required
	Spec PluginInstallationSpec `json:"spec"`
	// +optional
	Status PluginInstallationStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// PluginInstallationList contains a list of PluginInstallation
type PluginInstallationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []PluginInstallation `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PluginInstallation{}, &PluginInstallationList{})
}
