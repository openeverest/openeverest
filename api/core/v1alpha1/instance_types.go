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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// InstanceSpec defines the desired state of Instance
type InstanceSpec struct {
	// Provider is the name of the database provider (e.g., "psmdb", "postgresql").
	Provider string `json:"provider,omitempty"`

	// Topology defines the deployment topology and its configuration.
	// +optional
	Topology *TopologySpec `json:"topology,omitempty"`

	// Global contains provider-level configuration that applies to the entire cluster.
	// The schema for this field is defined by the provider's GlobalConfigSchema.
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Global *runtime.RawExtension `json:"global,omitempty"`

	// Components defines the component instances for this cluster.
	// The keys are component names (e.g., "engine", "proxy", "backupAgent").
	// Which components are valid depends on the selected topology.
	Components map[string]ComponentSpec `json:"components,omitempty"`
}

// TopologySpec defines the deployment topology and its configuration.
type TopologySpec struct {
	// Type is the topology name (e.g., "sharded", "replicaset").
	// The available topologies are defined by the provider.
	// If omitted, the provider's default topology is used.
	// +optional
	Type string `json:"type,omitempty"`

	// Config contains topology-specific configuration.
	// The schema for this field is defined by the provider's TopologyDefinition.
	// Examples: shard count for sharded topology, replication factor, etc.
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Config *runtime.RawExtension `json:"config,omitempty"`
}

type ComponentSpec struct {
	// Name of the component.
	Name string `json:"name,omitempty"`
	// Type of the component from the Provider.
	Type string `json:"type,omitempty"`
	// Version of the component from ComponentVersions.
	Version string `json:"version,omitempty"`
	// Image specifies an override for the image to use.
	// When unspecified, it is autmatically set from the ComponentVersions
	// based on the Version specified.
	// +optional
	Image string `json:"image,omitempty"`
	// Storage requirements for this component.
	// For stateless components, this is an optional field.
	// +optional
	// TODO: Should we change to corev1.PersistentVolumeClaimSpec?
	Storage *Storage `json:"storage,omitempty"`
	// Resources requirements for this component.
	// +optional
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`
	// Config specifies the component specific configuration.
	// +optional
	Config *Config `json:"config,omitempty"`
	// Replicas specifies the number of replicas for this component.
	// +optional
	Replicas *int32 `json:"replicas,omitempty"`
	// +kubebuilder:pruning:PreserveUnknownFields
	// CustomSpec provides an API for customising this component.
	// The API schema is defined by the provider's ComponentSchemas.
	CustomSpec *runtime.RawExtension `json:"customSpec,omitempty"`
}

type Storage struct {
	Size         resource.Quantity `json:"size,omitempty"`
	StorageClass *string           `json:"storageClass,omitempty"`
}

type Config struct {
	SecretRef    corev1.LocalObjectReference `json:"secretRef,omitempty"`
	ConfigMapRef corev1.LocalObjectReference `json:"configMapRef,omitempty"`
	Key          string                      `json:"key,omitempty"`
}

// GetComponentsOfType returns all components that match the given type.
func (in *Instance) GetComponentsOfType(t string) []ComponentSpec {
	var result []ComponentSpec
	for _, c := range in.Spec.Components {
		if c.Type == t {
			result = append(result, c)
		}
	}
	return result
}

// GetTopologyType returns the topology type, or empty string if not specified.
func (in *Instance) GetTopologyType() string {
	if in.Spec.Topology == nil {
		return ""
	}
	return in.Spec.Topology.Type
}

// GetTopologyConfig returns the topology configuration as runtime.RawExtension.
// Returns nil if no topology or topology config is specified.
func (in *Instance) GetTopologyConfig() *runtime.RawExtension {
	if in.Spec.Topology == nil {
		return nil
	}
	return in.Spec.Topology.Config
}

// InstanceStatus defines the observed state of Instance.
type InstanceStatus struct {
	// Phase of the database cluster.
	Phase InstancePhase `json:"phase,omitempty"`
	// ConnectionSecretRef is a reference to the Secret containing connection details.
	// The Secret is auto-generated by the provider-runtime reconciler with the name
	// "{instance-name}-conn" and owned by the Instance (auto-deleted on cleanup).
	//
	// The Secret uses well-known keys inspired by the Service Binding specification:
	//   - "type"     - Database type (e.g., "mongodb", "postgresql")
	//   - "provider" - Provider name (e.g., "percona-server-mongodb")
	//   - "host"     - Hostname or IP address
	//   - "port"     - Port number
	//   - "username" - Database username
	//   - "password" - Database password
	//   - "uri"      - Full connection URI including credentials
	//
	// +optional
	ConnectionSecretRef corev1.LocalObjectReference `json:"connectionSecretRef,omitempty"`
	// Components is the status of the components in the database cluster.
	Components []ComponentStatus `json:"components,omitempty"`
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

type InstancePhase string

const (
	InstancePhaseCreating InstancePhase = "Creating"
	InstancePhaseRunning  InstancePhase = "Running"
	InstancePhaseFailed   InstancePhase = "Failed"
	InstancePhaseDeleting InstancePhase = "Deleting"
)

const (
	StateReady      = "Ready"
	StateInProgress = "InProgress"
	StateError      = "Error"
)

// Condition types for Instance.
const (
	// ConditionConnectionDetailsReady indicates whether the connection
	// details Secret has been populated by the provider.
	ConditionConnectionDetailsReady = "ConnectionDetailsReady"
)

type ComponentStatus struct {
	Pods  []corev1.LocalObjectReference `json:"pods,omitempty"`
	Total *int32                        `json:"total,omitempty"`
	Ready *int32                        `json:"ready,omitempty"`
	State string                        `json:"state,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=in;inst

// Instance is the Schema for the instances API
type Instance struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// +required
	Spec InstanceSpec `json:"spec"`
	// +optional
	Status InstanceStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// InstanceList contains a list of Instance
type InstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []Instance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Instance{}, &InstanceList{})
}
