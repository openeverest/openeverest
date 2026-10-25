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
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

// Condition types for MonitoringClass.
const (
	// MonitoringClassConditionAccepted is set by the controller named in
	// spec.controllerName once it has claimed the class. It defaults to
	// Unknown so a class nobody claims is visibly unusable.
	MonitoringClassConditionAccepted = "Accepted"
)

// MonitoringClassSpec defines the desired state of MonitoringClass.
//
// +kubebuilder:validation:XValidation:rule="has(self.providerManaged) || has(self.extensionManaged)",message="a MonitoringClass must declare at least one fulfilment block (providerManaged or extensionManaged)"
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.providerManaged) || has(self.providerManaged)",message="spec.providerManaged cannot be removed once set"
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.extensionManaged) || has(self.extensionManaged)",message="spec.extensionManaged cannot be removed once set"
type MonitoringClassSpec struct {
	// ControllerName is the domain-prefixed name of the controller that
	// fulfils this class (e.g. "openeverest.io/monitoring-pmm"). Only that
	// controller sets the Accepted condition. Immutable.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*/[A-Za-z0-9/\-._~%!$&'()*+,;=:]+$`
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="spec.controllerName is immutable"
	ControllerName string `json:"controllerName"`
	// DisplayName is a human-readable name for the monitoring technology.
	// +optional
	DisplayName string `json:"displayName,omitempty"`
	// Description describes the monitoring technology.
	// +optional
	Description string `json:"description,omitempty"`
	// ProviderManaged declares that providers whose engine operator natively
	// integrates this technology render it themselves inside Sync. Matched
	// against Provider.spec.monitoring.integrations.
	// +optional
	ProviderManaged *ProviderManagedFulfilment `json:"providerManaged,omitempty"`
	// ExtensionManaged declares that the class controller wires the backend
	// from the provider's monitoring contract alone
	// (Instance.status.monitoring.sources).
	// +optional
	ExtensionManaged *ExtensionManagedFulfilment `json:"extensionManaged,omitempty"`
	// DestinationParametersSchema validates MonitoringDestination.spec.parameters.
	// +optional
	DestinationParametersSchema *common.ParametersSchema `json:"destinationParametersSchema,omitempty"`
	// InstanceParametersSchema validates
	// Instance.spec.monitoring.destinations[].parameters.
	// +optional
	InstanceParametersSchema *common.ParametersSchema `json:"instanceParametersSchema,omitempty"`
	// CredentialsSchema lists the keys the MonitoringDestination credentials
	// Secret must carry.
	// +optional
	CredentialsSchema *CredentialsSchema `json:"credentialsSchema,omitempty"`
	// UI holds the hints the UI needs to present destinations of this class.
	// +optional
	UI *MonitoringClassUI `json:"ui,omitempty"`
	// UISchema holds free-form form-rendering hints consumed only by the UI,
	// grouped by the form they decorate (destination, instance, credentials).
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	UISchema *runtime.RawExtension `json:"uiSchema,omitempty"`
}

// ProviderManagedFulfilment describes how a provider renders this class
// through its engine operator's native integration.
type ProviderManagedFulfilment struct {
	// Integration is the integration name providers declare in
	// Provider.spec.monitoring.integrations (e.g. "pmm"). Immutable.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="spec.providerManaged.integration is immutable"
	Integration string `json:"integration"`
	// AgentImage is the agent image the provider renders into the engine
	// pods. It is admin-controlled: only class writers can change what runs
	// next to the data volumes.
	// +optional
	AgentImage string `json:"agentImage,omitempty"`
	// AgentVersion is the semantic version of AgentImage, matched against
	// Provider.spec.monitoring.integrations.<integration>.agentVersions.
	// +optional
	AgentVersion string `json:"agentVersion,omitempty"`
	// Features lists what the integration delivers beyond metrics
	// (e.g. "queryAnalytics"), used for user-facing disclosure.
	// +optional
	Features []string `json:"features,omitempty"`
}

// ExtensionManagedFulfilment describes what a provider must expose for the
// class controller to wire the backend from the monitoring contract.
type ExtensionManagedFulfilment struct {
	// Requires lists the source capabilities a component must satisfy to
	// be eligible for this class.
	// +optional
	Requires SourceRequirements `json:"requires,omitempty"`
	// Degrades lists the capabilities lost compared to a ProviderManaged
	// fulfilment of the same class (e.g. "slowLogQAN").
	// +optional
	Degrades []string `json:"degrades,omitempty"`
}

// SourceRequirements is matched against
// Provider.spec.monitoring.componentTypes.<type>: a component is eligible when
// every flag holds and, if Kinds is set, its kind is in Kinds.
type SourceRequirements struct {
	// Kinds restricts eligibility to components of these kinds (any-of).
	// +optional
	Kinds []string `json:"kinds,omitempty"`
	// Metrics requires the component to expose OpenMetrics
	// (metrics: Native or Exporter).
	// +optional
	Metrics bool `json:"metrics,omitempty"`
	// CredentialProfiles requires the component to offer these credential
	// profiles (all-of).
	// +optional
	CredentialProfiles []string `json:"credentialProfiles,omitempty"`
	// Features requires the component to offer these engine features
	// (all-of).
	// +optional
	Features []string `json:"features,omitempty"`
}

// CredentialsSchema lists the keys a MonitoringDestination credentials Secret
// must carry.
type CredentialsSchema struct {
	// Required is the list of required Secret keys.
	// +optional
	Required []string `json:"required,omitempty"`
}

// MonitoringClassUI holds the hints the UI needs to present destinations of
// this class.
type MonitoringClassUI struct {
	// SummaryParameter names the destination parameter the UI shows as the
	// destination's address (e.g. "url").
	// +optional
	SummaryParameter string `json:"summaryParameter,omitempty"`
	// Links are substitution-only URL templates the UI renders, e.g.
	// "{{ .destination.parameters.url }}/graph". Available values:
	// .destination.parameters.*, .instance.name, .instance.namespace and
	// .sources.identity.labels.*; substituted values are URL-escaped.
	// +listType=map
	// +listMapKey=name
	// +optional
	Links []MonitoringClassLink `json:"links,omitempty"`
}

// LinkScope says where the UI shows a MonitoringClassLink.
//
// +kubebuilder:validation:Enum=Destination;Instance
type LinkScope string

const (
	// LinkScopeDestination shows the link on the destination.
	LinkScopeDestination LinkScope = "Destination"
	// LinkScopeInstance shows the link on the page of every Instance that
	// sends to the destination.
	LinkScopeInstance LinkScope = "Instance"
)

// MonitoringClassLink is one link the UI renders for a destination.
type MonitoringClassLink struct {
	// Name is the link text.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`
	// Scope says where the link is shown.
	// +kubebuilder:default=Destination
	// +optional
	Scope LinkScope `json:"scope,omitempty"`
	// Kinds restricts an Instance-scoped link to Instances with a component
	// of one of these engine kinds.
	// +optional
	Kinds []string `json:"kinds,omitempty"`
	// URL is the link template.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	URL string `json:"url"`
}

// Disruption is what enabling a class does to the pods of a component.
//
// +kubebuilder:validation:Enum=None;RollingRestart
type Disruption string

const (
	// DisruptionNone means the pods are not touched.
	DisruptionNone Disruption = "None"
	// DisruptionRollingRestart means the pods restart one at a time.
	DisruptionRollingRestart Disruption = "RollingRestart"
)

// ProviderCompatibility is the class's compatibility with one Provider,
// computed by core from Provider.spec.monitoring.
type ProviderCompatibility struct {
	// Name is the Provider name.
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// Mode is the execution mode a binding of this class gets for the
	// Provider's Instances. Empty when the Provider is incompatible.
	// +optional
	Mode corev1alpha1.MonitoringExecutionMode `json:"mode,omitempty"`
	// Components lists, for ExtensionManaged, what each eligible component
	// type exposes.
	// +listType=map
	// +listMapKey=type
	// +optional
	Components []ComponentCompatibility `json:"components,omitempty"`
	// Reason explains an empty Mode.
	// +optional
	Reason string `json:"reason,omitempty"`
	// Message is the human-readable detail for Reason.
	// +optional
	Message string `json:"message,omitempty"`
}

// ComponentCompatibility is what one component type exposes to an
// ExtensionManaged class.
type ComponentCompatibility struct {
	// Type is the component type, a key of
	// Provider.spec.monitoring.componentTypes.
	// +kubebuilder:validation:Required
	Type string `json:"type"`
	// Metrics states how the component exposes OpenMetrics.
	// +optional
	Metrics corev1alpha1.MetricsSource `json:"metrics,omitempty"`
	// Disruption is what enabling the class does to the component's pods.
	// +optional
	Disruption Disruption `json:"disruption,omitempty"`
}

// MonitoringClassStatus defines the observed state of MonitoringClass. It
// has two writers, each applying only its own fields: the claiming
// controller owns the Accepted condition and core owns providers.
type MonitoringClassStatus struct {
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// Providers is the compatibility of this class with every installed
	// Provider.
	// +listType=map
	// +listMapKey=name
	// +optional
	Providers []ProviderCompatibility `json:"providers,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,shortName=mclass
// +kubebuilder:printcolumn:name="Controller",type="string",JSONPath=".spec.controllerName"
// +kubebuilder:printcolumn:name="Accepted",type="string",JSONPath=".status.conditions[?(@.type==\"Accepted\")].status"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:validation:XValidation:rule="size(self.metadata.name) <= 63",message="a MonitoringClass name must be at most 63 characters"

// MonitoringClass is one monitoring technology, claimed by the out-of-tree
// controller named in spec.controllerName.
type MonitoringClass struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// +required
	Spec MonitoringClassSpec `json:"spec"`
	// +optional
	// +kubebuilder:default={conditions: {{type: "Accepted", status: "Unknown", reason: "Pending", message: "Waiting for controller", lastTransitionTime: "1970-01-01T00:00:00Z"}}}
	Status MonitoringClassStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// MonitoringClassList contains a list of MonitoringClass.
type MonitoringClassList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`

	Items []MonitoringClass `json:"items"`
}

func init() { //nolint:gochecknoinits // kubebuilder scheme registration convention
	SchemeBuilder.Register(&MonitoringClass{}, &MonitoringClassList{})
}
