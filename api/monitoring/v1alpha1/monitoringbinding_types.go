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

// Labels set by core on every MonitoringBinding so executors can watch by
// class and find bindings without parsing names.
const (
	// BindingInstanceLabel is the Instance name.
	BindingInstanceLabel = "openeverest.io/instance"
	// BindingEntryLabel is the Instance.spec.monitoring[] entry name.
	BindingEntryLabel = "monitoring.openeverest.io/entry"
	// BindingClassLabel is the resolved MonitoringClass name.
	BindingClassLabel = "monitoring.openeverest.io/class"
)

// Condition types for MonitoringBinding. Each type has exactly one writer.
const (
	// MonitoringBindingConditionAccepted is owned by the core materialiser:
	// destination, class and provider resolved, class claimed, parameters
	// valid.
	MonitoringBindingConditionAccepted = "Accepted"
	// MonitoringBindingConditionConfigured is owned by the executor named by
	// spec.executionMode (provider-runtime or the class controller). It means
	// configuration was handed to the agent path; it never claims data flows.
	MonitoringBindingConditionConfigured = "Configured"
)

// Reasons for the Accepted condition.
const (
	ReasonAccepted               = "Accepted"
	ReasonDestinationNotFound    = "DestinationNotFound"
	ReasonClassNotFound          = "ClassNotFound"
	ReasonClassNotAccepted       = "ClassNotAccepted"
	ReasonProviderNotFound       = "ProviderNotFound"
	ReasonIntegrationUnsupported = "IntegrationUnsupported"
	ReasonIntegrationInUse       = "IntegrationInUse"
	ReasonIncompatible           = "Incompatible"
	ReasonInvalidParameters      = "InvalidParameters"
)

// ConfiguredReason is a reason of the Configured condition. The condition
// status is derived from the reason, see Status.
type ConfiguredReason string

// Reasons for the Configured condition.
const (
	// True.
	ReasonConfigured         ConfiguredReason = "Configured"
	ReasonMaintenancePending ConfiguredReason = "MaintenancePending"
	// Unknown.
	ReasonPending  ConfiguredReason = "Pending"
	ReasonRetained ConfiguredReason = "Retained"
	// False.
	ReasonNotAccepted        ConfiguredReason = "NotAccepted"
	ReasonRenderFailed       ConfiguredReason = "RenderFailed"
	ReasonApplyFailed        ConfiguredReason = "ApplyFailed"
	ReasonAgentNotReady      ConfiguredReason = "AgentNotReady"
	ReasonEndpointNotServing ConfiguredReason = "EndpointNotServing"
	ReasonServerTooOld       ConfiguredReason = "ServerTooOld"
	ReasonNotSelected        ConfiguredReason = "NotSelected"
)

// Status returns the condition status the reason implies.
func (r ConfiguredReason) Status() metav1.ConditionStatus {
	switch r {
	case ReasonConfigured, ReasonMaintenancePending:
		return metav1.ConditionTrue
	case ReasonPending, ReasonRetained:
		return metav1.ConditionUnknown
	default:
		return metav1.ConditionFalse
	}
}

// MonitoringBindingSpec is written only by the core materialiser. It is
// immutable except for parameters; classRef and executionMode are set once,
// when the destination and class first resolve.
//
// +kubebuilder:validation:XValidation:rule="self.instanceRef == oldSelf.instanceRef",message="spec.instanceRef is immutable"
// +kubebuilder:validation:XValidation:rule="self.destinationRef == oldSelf.destinationRef",message="spec.destinationRef is immutable"
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.classRef) || (has(self.classRef) && self.classRef == oldSelf.classRef)",message="spec.classRef is immutable once set"
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.executionMode) || (has(self.executionMode) && self.executionMode == oldSelf.executionMode)",message="spec.executionMode is immutable once set"
type MonitoringBindingSpec struct {
	// InstanceRef references the Instance in the same namespace.
	// +kubebuilder:validation:Required
	InstanceRef common.ObjectRef `json:"instanceRef"`
	// DestinationRef is copied from the Instance.spec.monitoring.destinations[] entry.
	// +kubebuilder:validation:Required
	DestinationRef corev1alpha1.MonitoringDestinationRef `json:"destinationRef"`
	// ClassRef is copied from the MonitoringDestination once it resolves.
	// +optional
	ClassRef *common.ObjectRef `json:"classRef,omitempty"`
	// ExecutionMode selects the executor that owns the Configured condition.
	// Set once, at first successful resolution.
	// +optional
	ExecutionMode corev1alpha1.MonitoringExecutionMode `json:"executionMode,omitempty"`
	// Parameters are copied from the Instance.spec.monitoring.destinations[] entry.
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Parameters *runtime.RawExtension `json:"parameters,omitempty"`
}

// MonitoringBindingStatus is written per condition type by its owner.
type MonitoringBindingStatus struct {
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// AppliedRevision is the hash of the rendered content last applied by a
	// ProviderManaged executor. Reporting only.
	// +optional
	AppliedRevision string `json:"appliedRevision,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=mbind
// +kubebuilder:printcolumn:name="Instance",type="string",JSONPath=".spec.instanceRef.name"
// +kubebuilder:printcolumn:name="Destination",type="string",JSONPath=".spec.destinationRef.name"
// +kubebuilder:printcolumn:name="Class",type="string",JSONPath=".spec.classRef.name"
// +kubebuilder:printcolumn:name="Accepted",type="string",JSONPath=".status.conditions[?(@.type==\"Accepted\")].reason"
// +kubebuilder:printcolumn:name="Configured",type="string",JSONPath=".status.conditions[?(@.type==\"Configured\")].reason"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Mode",type="string",JSONPath=".spec.executionMode",priority=1
// +kubebuilder:printcolumn:name="Revision",type="string",JSONPath=".status.appliedRevision",priority=1

// MonitoringBinding records that OpenEverest has configured one Instance to
// send monitoring data to one MonitoringDestination. Created from
// Instance.spec.monitoring.destinations[]; edit the Instance, not this object.
type MonitoringBinding struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// +required
	Spec MonitoringBindingSpec `json:"spec"`
	// +optional
	Status MonitoringBindingStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// MonitoringBindingList contains a list of MonitoringBinding.
type MonitoringBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []MonitoringBinding `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MonitoringBinding{}, &MonitoringBindingList{})
}
