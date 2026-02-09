package v2alpha1

import (
	"encoding/json"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=wl;wld;wload
type Workload struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorkloadSpec   `json:"spec,omitempty"`
	Status WorkloadStatus `json:"status,omitempty"`
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

type WorkloadSpec struct {
	// Provider is the name of the database provider (e.g., "psmdb", "postgresql").
	Provider string `json:"provider,omitempty"`

	// Topology defines the deployment topology and its configuration.
	// +optional
	Topology *TopologySpec `json:"topology,omitempty"`

	// Global contains provider-level configuration that applies to the entire cluster.
	// The schema for this field is defined by the provider's GlobalSchema.
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Global *runtime.RawExtension `json:"global,omitempty"`

	// Components defines the component instances for this cluster.
	// The keys are component names (e.g., "engine", "proxy", "backupAgent").
	// Which components are valid depends on the selected topology.
	Components map[string]ComponentSpec `json:"components,omitempty"`
}

// GetComponentsOfType returns all components that match the given type.
func (wl *Workload) GetComponentsOfType(t string) []ComponentSpec {
	var result []ComponentSpec
	for _, c := range wl.Spec.Components {
		if c.Type == t {
			result = append(result, c)
		}
	}
	return result
}

// GetTopologyType returns the topology type, or empty string if not specified.
func (wl *Workload) GetTopologyType() string {
	if wl.Spec.Topology == nil {
		return ""
	}
	return wl.Spec.Topology.Type
}

// GetTopologyConfig returns the topology configuration as runtime.RawExtension.
// Returns nil if no topology or topology config is specified.
func (wl *Workload) GetTopologyConfig() *runtime.RawExtension {
	if wl.Spec.Topology == nil {
		return nil
	}
	return wl.Spec.Topology.Config
}

type WorkloadPhase string

const (
	WorkloadPhaseCreating WorkloadPhase = "Creating"
	WorkloadPhaseRunning  WorkloadPhase = "Running"
	WorkloadPhaseFailed   WorkloadPhase = "Failed"
	WorkloadPhaseDeleting WorkloadPhase = "Deleting"
)

type WorkloadStatus struct {
	// Phase of the database cluster.
	Phase WorkloadPhase `json:"phase,omitempty"`
	// ConnectionURL is the URL to connect to the database cluster.
	ConnectionURL string `json:"connectionURL,omitempty"`
	// CredentialSecretRef is a reference to the secret containing the credentials.
	// This Secret contains the keys `username` and `password`.
	CredentialSecretRef corev1.LocalObjectReference `json:"credentialSecretRef,omitempty"`
	// Components is the status of the components in the database cluster.
	Components []ComponentStatus `json:"components,omitempty"`
	// TODO: more fields
}

const (
	StateReady      = "Ready"
	StateInProgress = "InProgress"
	StateError      = "Error"
)

type ComponentStatus struct {
	Pods  []corev1.LocalObjectReference `json:"pods,omitempty"`
	Total *int32                        `json:"total,omitempty"`
	Ready *int32                        `json:"ready,omitempty"`
	State string                        `json:"state,omitempty"`
}

type CustomOptions map[string]json.RawMessage

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
	// TODO: Should we change to corev1.ResourceRequirements?
	Resources *Resources `json:"resources,omitempty"`
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

type Config struct {
	SecretRef    corev1.LocalObjectReference `json:"secretRef,omitempty"`
	ConfigMapRef corev1.LocalObjectReference `json:"configMapRef,omitempty"`
	Key          string                      `json:"key,omitempty"`
}

type Storage struct {
	Size         resource.Quantity `json:"size,omitempty"`
	StorageClass *string           `json:"storageClass,omitempty"`
}

type Resources struct {
	CPU    resource.Quantity `json:"cpu,omitempty"`
	Memory resource.Quantity `json:"memory,omitempty"`
}

//+kubebuilder:object:root=true

// WorkloadList contains a list of Workload.
type WorkloadList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Workload `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Workload{}, &WorkloadList{})
}
