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
)

// PluginSpec defines the desired state of a Plugin.
type PluginSpec struct {
	// DisplayName is the human-readable name shown in the UI sidebar.
	// +required
	DisplayName string `json:"displayName"`

	// BackendURL is the in-cluster base URL where the plugin's backend
	// is reachable (e.g. "http://hello-plugin.everest-system:3001").
	// The Everest server reverse-proxies /v1/plugins/<name>/* to this URL.
	// +required
	BackendURL string `json:"backendUrl"`

	// BundlePath is the path appended to BackendURL to fetch the plugin's
	// frontend ESM bundle (e.g. "/main.js"). The Everest server exposes
	// this as /v1/plugins/<name>/<bundlePath> for the UI to import().
	// +optional
	// +kubebuilder:default="/main.js"
	BundlePath string `json:"bundlePath,omitempty"`

	// Enabled controls whether the plugin is active. A disabled plugin
	// is not returned by the list endpoint and its proxy routes are inactive.
	// +optional
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`

	// CLI defines an optional CLI contribution. When set, `everestctl plugin run`
	// can exec a container from the specified image.
	// +optional
	CLI *PluginCLI `json:"cli,omitempty"`
}

// PluginCLI describes the CLI contribution of a plugin.
type PluginCLI struct {
	// Image is the OCI image reference for the CLI container.
	// +required
	Image string `json:"image"`

	// Subcommand is the name used under `everestctl plugin run <subcommand>`.
	// Defaults to the plugin name if not set.
	// +optional
	Subcommand string `json:"subcommand,omitempty"`

	// Description is a short human-readable description for the CLI help text.
	// +optional
	Description string `json:"description,omitempty"`
}

// PluginStatus defines the observed state of a Plugin.
type PluginStatus struct {
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=plg;plugin
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:printcolumn:name="Display Name",type="string",JSONPath=".spec.displayName"
// +kubebuilder:printcolumn:name="Backend URL",type="string",JSONPath=".spec.backendUrl"
// +kubebuilder:printcolumn:name="Enabled",type="boolean",JSONPath=".spec.enabled"

// Plugin is the Schema for the plugins API. It registers an external plugin
// with the Everest platform, enabling its UI bundle to be loaded dynamically
// and its backend to be reverse-proxied through the Everest server.
type Plugin struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// +required
	Spec PluginSpec `json:"spec"`
	// +optional
	Status PluginStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// PluginList contains a list of Plugin.
type PluginList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []Plugin `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Plugin{}, &PluginList{})
}
