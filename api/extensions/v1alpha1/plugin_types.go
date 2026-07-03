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

// PluginSpec defines the desired state of Plugin
type PluginSpec struct {
	// DisplayName is the human-readable name shown in the UI sidebar.
	// +required
	DisplayName string `json:"displayName"`

	// Description is a short human-readable description of what the plugin does.
	// +optional
	Description string `json:"description,omitempty"`

	// Version is the SemVer version of the plugin.
	// +optional
	Version string `json:"version,omitempty"`

	// Vendor is the name of the plugin author or organisation.
	// +optional
	Vendor string `json:"vendor,omitempty"`

	// Icon is a URL to the plugin's icon image.
	// +optional
	Icon string `json:"icon,omitempty"`

	// CompatibleHostVersions is a SemVer range expression specifying which
	// OpenEverest host versions this plugin supports (e.g. ">=2.0.0 <3.0.0").
	// +optional
	CompatibleHostVersions string `json:"compatibleHostVersions,omitempty"`

	// Frontend defines the optional frontend contribution of the plugin.
	// +optional
	Frontend *PluginFrontend `json:"frontend,omitempty"`

	// Backend defines the optional backend contribution of the plugin.
	// +optional
	Backend *PluginBackend `json:"backend,omitempty"`

	// Enabled controls whether the plugin is active. A disabled plugin
	// is not returned by the list endpoint and its proxy routes are inactive.
	// +optional
	// +kubebuilder:default=true
	Enabled bool `json:"enabled,omitempty"`

	// Permissions declares what OpenEverest API resources this plugin needs access to.
	// +optional
	Permissions []PluginPermission `json:"permissions,omitempty"`

	// CLI defines an optional CLI contribution. When set, `everestctl extension run`
	// can exec a container from the specified image.
	// +optional
	CLI *PluginCLI `json:"cli,omitempty"`
}

// PluginFrontend defines the frontend contribution of a plugin.
type PluginFrontend struct {
	// BundlePath is the path on the backend that serves the plugin's
	// frontend ESM bundle (e.g. "/main.js"). The Everest server exposes
	// this as /v1/plugins/<name>/<bundlePath> for the UI to import().
	// +optional
	// +kubebuilder:default="/main.js"
	BundlePath string `json:"bundlePath,omitempty"`

	// BundleIntegrity is an optional SRI hash for verifying the bundle.
	// +optional
	BundleIntegrity string `json:"bundleIntegrity,omitempty"`

	// ExtensionPoints declares the UI extension points this plugin fills.
	// This enables the host to show/hide contributions before loading the bundle
	// and enables RBAC-based filtering.
	// +optional
	ExtensionPoints []PluginExtensionPoint `json:"extensionPoints,omitempty"`
}

// PluginExtensionPoint declares a single UI extension point contribution.
type PluginExtensionPoint struct {
	// Type is the kind of extension point (e.g. "route", "sidebarItem",
	// "clusterDetailTab", "clusterAction", "clusterCard",
	// "globalDashboardWidget", "settingsPanel", "instanceCreateFormSection",
	// "instanceEditFormSection", "themeOverride").
	// +required
	Type string `json:"type"`

	// Label is the human-readable label displayed for this contribution.
	// +optional
	Label string `json:"label,omitempty"`

	// Path is an optional sub-path (used by "route" and tab-type extension points).
	// +optional
	Path string `json:"path,omitempty"`

	// Icon is an optional icon identifier.
	// +optional
	Icon string `json:"icon,omitempty"`

	// Providers is an optional list of database engine types this extension point
	// applies to. Values match spec.engine.type on the DatabaseCluster CR:
	// "postgresql", "psmdb", "pxc".
	// When omitted or empty, the extension point is shown for all engine types.
	// +optional
	Providers []string `json:"providers,omitempty"`
}

// PluginBackend defines the backend contribution of a plugin.
// Exactly one of ServiceRef or ExternalURL must be set.
type PluginBackend struct {
	// ServiceRef references an in-cluster Kubernetes Service.
	// The Everest server resolves it to http://<name>.<namespace>.svc:<port>
	// and reverse-proxies /v1/plugins/<pluginName>/* to that address.
	// +optional
	ServiceRef *PluginBackendServiceRef `json:"serviceRef,omitempty"`

	// ExternalURL is the HTTPS base URL of an externally hosted backend
	// (e.g. "https://sql-explorer.example.com"). Mutually exclusive with ServiceRef.
	// +optional
	ExternalURL string `json:"externalUrl,omitempty"`

	// CredentialsSecretRef is the name of a Secret in the same namespace as
	// the InstalledExtension entry whose "token" key is forwarded as the
	// Authorization header to the external backend. Only meaningful when
	// ExternalURL is set.
	// +optional
	CredentialsSecretRef string `json:"credentialsSecretRef,omitempty"`
}

// PluginBackendServiceRef points to an in-cluster Kubernetes Service.
type PluginBackendServiceRef struct {
	// Namespace of the Service.
	// +required
	Namespace string `json:"namespace"`
	// Name of the Service.
	// +required
	Name string `json:"name"`
	// Port is the service port number.
	// +required
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	Port int32 `json:"port"`
}

// PluginPermission declares a single permission the plugin requires.
type PluginPermission struct {
	// Verb is the action (e.g. "read", "create", "update", "delete").
	// +required
	Verb string `json:"verb"`

	// Resource is the OpenEverest API resource (e.g. "database-clusters").
	// +required
	Resource string `json:"resource"`
}

// PluginCLI describes the CLI contribution of a plugin.
type PluginCLI struct {
	// Image is the OCI image reference for the CLI container.
	// +required
	Image string `json:"image"`

	// Subcommand is the name used under `everestctl extension run <subcommand>`.
	// Defaults to the plugin name if not set.
	// +optional
	Subcommand string `json:"subcommand,omitempty"`

	// Description is a short human-readable description for the CLI help text.
	// +optional
	Description string `json:"description,omitempty"`
}

// PluginStatus defines the observed state of Plugin.
type PluginStatus struct {
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,shortName=plg;plugin
// +kubebuilder:printcolumn:name="Display Name",type="string",JSONPath=".spec.displayName"
// +kubebuilder:printcolumn:name="Version",type="string",JSONPath=".spec.version"
// +kubebuilder:printcolumn:name="Backend URL",type="string",JSONPath=".spec.backend.url"
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

// PluginList contains a list of Plugin
type PluginList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []Plugin `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Plugin{}, &PluginList{})
}
