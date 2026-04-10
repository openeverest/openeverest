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
	"k8s.io/apimachinery/pkg/runtime"
)

// ProviderSpec defines the desired state of Provider
type ProviderSpec struct {
	ComponentTypes map[string]ComponentType `json:"componentTypes,omitempty"`
	Components     map[string]Component     `json:"components,omitempty"`
	Topologies     map[string]Topology      `json:"topologies,omitempty"`

	// Versions defines curated version bundles — named sets of component
	// versions that are known to be mutually compatible. Users reference
	// a bundle via Instance.Spec.Version. If the user does not set a version,
	// the bundle whose Default field is true is used automatically.
	Versions []VersionBundle `json:"versions,omitempty"`

	// GlobalConfigSchema holds the OpenAPI v3 schema for the global configuration.
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	GlobalConfigSchema *runtime.RawExtension `json:"globalConfigSchema,omitempty"`

	// UISchema holds the UI rendering hints for each topology.
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	UISchema *runtime.RawExtension `json:"uiSchema,omitempty"`
}

// VersionBundle is a curated set of component versions known to be mutually
// compatible. Provider developers define bundles in definition/versions.yaml.
type VersionBundle struct {
	// Name is the unique identifier for this bundle (e.g. "8.0.12").
	// Users set Instance.Spec.Version to this value to select the bundle.
	Name string `json:"name"`

	// Components maps component names to their version strings for this bundle.
	// Keys must match component names defined in ProviderSpec.Components.
	Components map[string]string `json:"components,omitempty"`

	// Default marks this bundle as the implicit choice when an Instance omits
	// Spec.Version entirely. Exactly one bundle should have Default: true.
	Default bool `json:"default,omitempty"`
}

type ComponentType struct {
	Versions []ComponentVersion `json:"versions,omitempty"`
}

type ComponentVersion struct {
	Version string `json:"version,omitempty"`
	Image   string `json:"image,omitempty"`
	Default bool   `json:"default,omitempty"`
}

type Component struct {
	Type string `json:"type,omitempty"`

	// CustomSpecSchema holds the OpenAPI v3 schema for this component's CustomSpec.
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	CustomSpecSchema *runtime.RawExtension `json:"customSpecSchema,omitempty"`
}

type Topology struct {
	Components map[string]TopologyComponent `json:"components,omitempty"`

	// ConfigSchema holds the OpenAPI v3 schema for topology-specific configuration.
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	ConfigSchema *runtime.RawExtension `json:"configSchema,omitempty"`
}

type TopologyComponent struct {
	Optional bool `json:"optional,omitempty"`
	// TODO: Do we need defaults?
	// Defaults map[string]interface{} `json:"defaults,omitempty"`
}

// ProviderStatus defines the observed state of Provider.
type ProviderStatus struct {
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=pr;prv;prov
// +kubebuilder:resource:scope=Cluster

// Provider is the Schema for the providers API
type Provider struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// +required
	Spec ProviderSpec `json:"spec"`
	// +optional
	Status ProviderStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// ProviderList contains a list of Provider
type ProviderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []Provider `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Provider{}, &ProviderList{})
}
