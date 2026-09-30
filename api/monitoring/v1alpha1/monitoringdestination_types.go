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

// Condition types for MonitoringDestination. Each type has exactly one writer.
const (
	// MonitoringDestinationConditionAccepted is owned by core: class found,
	// parameters valid against the class schema, credential keys present.
	MonitoringDestinationConditionAccepted = "Accepted"
	// MonitoringDestinationConditionReady is owned by the class controller:
	// the backend is reachable and the credentials are valid.
	MonitoringDestinationConditionReady = "Ready"
)

// MonitoringDestinationSpec defines the desired state of MonitoringDestination.
//
// +kubebuilder:validation:XValidation:rule="self.classRef == oldSelf.classRef",message="spec.classRef is immutable"
type MonitoringDestinationSpec struct {
	// ClassRef references the cluster-scoped MonitoringClass this destination
	// belongs to. Immutable.
	// +kubebuilder:validation:Required
	ClassRef common.ObjectRef `json:"classRef"`
	// CredentialsSecretRef references the Secret in the same namespace that
	// holds the destination credentials; its keys are described by the
	// class's credentialsSchema.
	// +optional
	CredentialsSecretRef *common.SecretRef `json:"credentialsSecretRef,omitempty"`
	// Parameters are validated against
	// MonitoringClass.spec.destinationParametersSchema.
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Parameters *runtime.RawExtension `json:"parameters,omitempty"`
}

// MonitoringDestinationStatus defines the observed state of
// MonitoringDestination.
type MonitoringDestinationStatus struct {
	// ServerVersion is the backend server version probed by the class
	// controller.
	// +optional
	ServerVersion string `json:"serverVersion,omitempty"`
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=mdest
// +kubebuilder:printcolumn:name="Class",type="string",JSONPath=".spec.classRef.name"
// +kubebuilder:printcolumn:name="Accepted",type="string",JSONPath=".status.conditions[?(@.type==\"Accepted\")].status"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type==\"Ready\")].status"
// +kubebuilder:printcolumn:name="Version",type="string",JSONPath=".status.serverVersion"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// MonitoringDestination is one place a namespace sends monitoring data to,
// typed by a MonitoringClass. Instances send to it through
// spec.monitoring.destinations[].
type MonitoringDestination struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// +required
	Spec MonitoringDestinationSpec `json:"spec"`
	// +optional
	Status MonitoringDestinationStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// MonitoringDestinationList contains a list of MonitoringDestination.
type MonitoringDestinationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []MonitoringDestination `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MonitoringDestination{}, &MonitoringDestinationList{})
}
