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
	"slices"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

// Condition types for MonitoringClass.
const (
	// MonitoringClassConditionAccepted is set by the controller serving the
	// class once it has claimed it. It defaults to Unknown so a class nobody
	// claims is visibly unusable.
	MonitoringClassConditionAccepted = "Accepted"
)

// MonitoringClassSpec defines the desired state of MonitoringClass.
//
// +kubebuilder:validation:XValidation:rule="has(self.providerManaged) || has(self.extensionManaged)",message="a MonitoringClass must declare at least one fulfilment block (providerManaged or extensionManaged)"
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.providerManaged) || has(self.providerManaged)",message="spec.providerManaged cannot be removed once set"
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.extensionManaged) || has(self.extensionManaged)",message="spec.extensionManaged cannot be removed once set"
type MonitoringClassSpec struct {
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
	// AuthModes lists the authentication methods a MonitoringDestination of
	// this class can use. A mode the class defines is Secret-backed and lists
	// its credentials keys; WorkloadIdentity and Anonymous are reserved by
	// core and take no Secret.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=16
	// +listType=map
	// +listMapKey=name
	AuthModes []common.AuthMode `json:"authModes"`
	// UISchema holds free-form form-rendering hints consumed only by the UI,
	// grouped by the form they decorate (destination, instance, and
	// credentials keyed by authentication mode).
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
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	AgentImage string `json:"agentImage"`
	// AgentVersion is the semantic version of AgentImage, matched against
	// Provider.spec.monitoring.integrations.<integration>.agentVersions.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	AgentVersion string `json:"agentVersion"`
}

// ExtensionManagedFulfilment describes what a provider must expose for the
// class controller to wire the backend from the monitoring contract.
type ExtensionManagedFulfilment struct {
	// Requires lists the source capabilities a component must satisfy to
	// be eligible for this class.
	// +optional
	Requires SourceRequirements `json:"requires,omitzero"`
}

// SourceRequirements is matched against
// Provider.spec.monitoring.componentTypes.<type>: a component is eligible when
// every flag holds.
type SourceRequirements struct {
	// Metrics requires the component to expose OpenMetrics
	// (metrics: Native or Exporter).
	// +optional
	Metrics bool `json:"metrics,omitempty"`
}

// FindAuthMode returns the class's authentication mode named name, or nil.
func (s *MonitoringClassSpec) FindAuthMode(name string) *common.AuthMode {
	i := slices.IndexFunc(s.AuthModes, func(m common.AuthMode) bool { return m.Name == name })
	if i < 0 {
		return nil
	}
	return &s.AuthModes[i]
}

// AuthModeNames returns the names of the class's authentication modes.
func (s *MonitoringClassSpec) AuthModeNames() []string {
	names := make([]string, 0, len(s.AuthModes))
	for _, m := range s.AuthModes {
		names = append(names, m.Name)
	}
	return names
}

// Disruption is what enabling a class does to the pods of an Instance.
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
	// Disruption is the worst that enabling the class does to the pods of
	// one of the Provider's Instances. Set when Mode is set.
	// +optional
	Disruption Disruption `json:"disruption,omitempty"`
	// Reason explains an empty Mode.
	// +optional
	Reason string `json:"reason,omitempty"`
	// Message is the human-readable detail for Reason.
	// +optional
	Message string `json:"message,omitempty"`
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
// +kubebuilder:printcolumn:name="Accepted",type="string",JSONPath=".status.conditions[?(@.type==\"Accepted\")].reason"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:validation:XValidation:rule="size(self.metadata.name) <= 63",message="a MonitoringClass name must be at most 63 characters"

// MonitoringClass is one monitoring technology, claimed by the out-of-tree
// controller that serves it.
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
