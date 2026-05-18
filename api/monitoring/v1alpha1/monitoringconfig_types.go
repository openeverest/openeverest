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

// Package v1alpha1 contains API Schema definitions for the monitoring v1alpha1 API group.
// +kubebuilder:object:generate=true
// +groupName=monitoring.openeverest.io
package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

const (
	// PMMMonitoringType represents monitoring via PMM.
	PMMMonitoringType MonitoringType = "pmm"
)

// MonitoringType is a type of monitoring.
type MonitoringType string

// PMMServerVersion is the version of the PMM server.
type PMMServerVersion string

// MonitoringConfigSpec defines the desired state of MonitoringConfig.
//
// +kubebuilder:validation:XValidation:rule="self.type != 'pmm' || has(self.pmm)",message="pmm config is required when type is pmm"
// +kubebuilder:validation:XValidation:rule="!has(self.pmm) || self.type == 'pmm'",message="pmm config is only allowed when type is pmm"
type MonitoringConfigSpec struct {
	// Type is the name of monitoring tool (e.g., "pmm").
	// +kubebuilder:validation:Enum=pmm
	Type MonitoringType `json:"type"`
	// PMM contains PMM-specific monitoring configuration.
	// Required when type is "pmm".
	// +optional
	PMM *PMMMonitoringSpec `json:"pmm,omitempty"`
}

// PMMMonitoringSpec contains configuration specific to PMM monitoring.
type PMMMonitoringSpec struct {
	// CredentialsSecretName is the reference to the secret containing the API key.
	// It contains `apiKey` key with the API key value.
	CredentialsSecretName string `json:"credentialsSecretName"`
	// URL is the URL of the PMM server.
	URL string `json:"url"`
	// VerifyTLS is set to ensure TLS/SSL verification.
	// If unspecified, the default value is true.
	//
	// +kubebuilder:default:=true
	VerifyTLS *bool `json:"verifyTLS,omitempty"`
}

// MonitoringConfigStatus defines the observed state of MonitoringConfig.
//
// FIXME: Add []metav1.Condition to represent the current state of the MonitoringConfig resource. Currently adding Conditions fails due to
// producing duplicated Go constants in generated code.
type MonitoringConfigStatus struct {
	// InUse is a flag that indicates if any Instance uses the monitoring config.
	// +kubebuilder:default=false
	InUse bool `json:"inUse,omitempty"`
	// LastObservedGeneration is the most recent generation observed for this MonitoringConfig.
	LastObservedGeneration int64 `json:"lastObservedGeneration,omitempty"`
	// PMM contains PMM-specific status information.
	// +optional
	PMM *PMMMonitoringStatus `json:"pmm,omitempty"`
}

// PMMMonitoringStatus contains PMM-specific observed state.
type PMMMonitoringStatus struct {
	// ServerVersion shows the PMM server version.
	ServerVersion PMMServerVersion `json:"serverVersion,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Type",type="string",JSONPath=".spec.type",description="Monitoring tool type (e.g., pmm)"
// +kubebuilder:printcolumn:name="InUse",type="boolean",JSONPath=".status.inUse",description="Indicates if any Instance uses the monitoring config"

// MonitoringConfig is the Schema for the monitoringconfigs API.
type MonitoringConfig struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of MonitoringConfig
	// +required
	Spec MonitoringConfigSpec `json:"spec"`

	// status defines the observed state of MonitoringConfig
	// +optional
	// +kubebuilder:default={"inUse": false}
	Status MonitoringConfigStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// MonitoringConfigList contains a list of MonitoringConfig.
type MonitoringConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []MonitoringConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MonitoringConfig{}, &MonitoringConfigList{})
}
