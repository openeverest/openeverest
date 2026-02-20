
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ProviderSpec defines the desired state of Provider
type ProviderSpec struct {
	ComponentTypes map[string]ComponentType `json:"componentTypes,omitempty"`
	Components     map[string]Component     `json:"components,omitempty"`
	Topologies     map[string]Topology      `json:"topologies,omitempty"`
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
}

type Topology struct {
	Components map[string]TopologyComponent `json:"components,omitempty"`
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
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitzero"`

	Spec   ProviderSpec   `json:"spec"`
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
