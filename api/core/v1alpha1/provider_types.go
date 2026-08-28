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

	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
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

	// ParametersSchema declares the OpenAPI v3 schema for the instance-wide
	// parameters payload (Instance.spec.parameters).
	// +optional
	ParametersSchema *common.ParametersSchema `json:"parametersSchema,omitempty"`

	// UISchema holds the UI rendering hints for each topology.
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	UISchema *runtime.RawExtension `json:"uiSchema,omitempty"`

	// Secrets defines Secret types this provider supports.
	// +optional
	Secrets map[string]SecretDefinition `json:"secrets,omitempty"`

	// ConfigMaps defines ConfigMap types this provider supports.
	// +optional
	ConfigMaps map[string]ConfigMapDefinition `json:"configMaps,omitempty"`

	// Release identifies this provider release — the shipped unit of
	// controller, bundled operator, and version catalog — and its
	// upgrade-path constraints. It is read by the pre-upgrade preflight.
	// +optional
	Release *Release `json:"release,omitempty"`
}

// Release identifies a provider release and its upgrade-path constraints.
type Release struct {
	// Version is the provider release version (P), populated from the chart
	// appVersion (e.g. "0.3"). Named distinctly from ProviderSpec.Versions
	// (the engine bundle catalog).
	// +optional
	Version string `json:"version,omitempty"`

	// MinUpgradableFrom is the lowest provider release version from which a
	// single-step upgrade to this release is permitted; a lower installed
	// version is blocked and must step through intermediate releases.
	// Empty means no floor.
	// +optional
	MinUpgradableFrom string `json:"minUpgradableFrom,omitempty"`
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

	// Deprecated marks a version as still supported but scheduled for
	// removal. Instances running on it get a proactive warning with a
	// remediation runway instead of a blocked upgrade.
	// +optional
	Deprecated bool `json:"deprecated,omitempty"`

	// RemovedInVersion is the provider version (P) in which this engine
	// version is dropped. Upgrading the provider to >= this version while
	// an Instance still uses this engine version is a blocking error.
	// +optional
	RemovedInVersion string `json:"removedInVersion,omitempty"`
}

type Component struct {
	Type string `json:"type,omitempty"`

	// ParametersSchema declares the OpenAPI v3 schema for this component's
	// parameters payload (Instance.spec.components[].parameters).
	// +optional
	ParametersSchema *common.ParametersSchema `json:"parametersSchema,omitempty"`
}

type Topology struct {
	Components map[string]TopologyComponent `json:"components,omitempty"`

	// ParametersSchema declares the OpenAPI v3 schema for topology-specific
	// parameters (Instance.spec.topology.parameters).
	// +optional
	ParametersSchema *common.ParametersSchema `json:"parametersSchema,omitempty"`
}

type TopologyComponent struct {
	Optional bool `json:"optional,omitempty"`
	// TODO: Do we need defaults?
	// Defaults map[string]interface{} `json:"defaults,omitempty"`

	// SupportedFields declares which ComponentSpec fields this component
	// honours in this topology, as an OpenAPI v3 schema over ComponentSpec's
	// own properties. A property is declared if and only if the provider
	// reads it, at any depth, so a component may honour part of a grouped
	// field such as schedulingPolicy. Constraints the provider enforces
	// (required, bounds) are expressed with the same schema vocabulary.
	//
	// Applicability is declared per topology because core fields configure
	// the deployment, and the deployment shape is what a topology chooses.
	//
	// When unset, no constraint is placed on the component and every field
	// is accepted, which is the behaviour of providers that do not declare.
	// +optional
	SupportedFields *common.ParametersSchema `json:"supportedFields,omitempty"`
}

// SecretDefinition describes a Secret type that a provider supports.
type SecretDefinition struct {
	// UISchema holds UI rendering hints for the secret creation form.
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	UISchema *runtime.RawExtension `json:"uiSchema,omitempty"`

	// ParametersSchema declares the OpenAPI v3 schema for validating secret data/stringData.
	// +optional
	ParametersSchema *common.ParametersSchema `json:"parametersSchema,omitempty"`
}

// ConfigMapDefinition describes a ConfigMap type that a provider supports.
type ConfigMapDefinition struct {
	// UISchema holds UI rendering hints for the configmap creation form.
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	UISchema *runtime.RawExtension `json:"uiSchema,omitempty"`

	// ParametersSchema declares the OpenAPI v3 schema for validating configmap data/binaryData.
	// +optional
	ParametersSchema *common.ParametersSchema `json:"parametersSchema,omitempty"`
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
// +kubebuilder:resource:scope=Cluster,shortName=pr;prv;prov

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
