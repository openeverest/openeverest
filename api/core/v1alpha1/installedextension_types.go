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

// InstalledExtensionType discriminates between a generic plugin install
// and a spec 001 provider install.
// +kubebuilder:validation:Enum=plugin;provider
type InstalledExtensionType string

const (
	// InstalledExtensionTypePlugin selects spec.plugin.
	InstalledExtensionTypePlugin InstalledExtensionType = "plugin"
	// InstalledExtensionTypeProvider selects spec.provider.
	InstalledExtensionTypeProvider InstalledExtensionType = "provider"
)

// PluginInstallScope controls how kubePermissions translate to RBAC.
// +kubebuilder:validation:Enum=Cluster;Namespaces
type PluginInstallScope string

const (
	// PluginInstallScopeCluster grants the plugin's kubePermissions via a
	// ClusterRole/ClusterRoleBinding. Requires spec.plugin.allowClusterScope=true.
	PluginInstallScopeCluster PluginInstallScope = "Cluster"
	// PluginInstallScopeNamespaces grants the plugin's kubePermissions via
	// per-namespace Role/RoleBinding for each entry in
	// spec.plugin.namespaces[].
	PluginInstallScopeNamespaces PluginInstallScope = "Namespaces"
)

// InstalledExtensionSpec defines the desired state of an InstalledExtension.
type InstalledExtensionSpec struct {
	// Type discriminates between plugin and provider installs. Exactly one of
	// Plugin or Provider must be set, matching Type.
	// +required
	Type InstalledExtensionType `json:"type"`

	// CatalogID identifies the catalog this install was sourced from.
	// Empty for manual installs.
	// +optional
	CatalogID string `json:"catalogId,omitempty"`

	// Channel is the catalog channel selected at install time (e.g. "stable").
	// +optional
	Channel string `json:"channel,omitempty"`

	// Version is the SemVer version of the extension that was installed.
	// +optional
	Version string `json:"version,omitempty"`

	// ChartDigest is the OCI digest of the Helm chart installed for this
	// extension. Optional in early phases; required once hub install lands.
	// +optional
	ChartDigest string `json:"chartDigest,omitempty"`

	// Plugin holds plugin-specific install state. Required when type=plugin;
	// must be nil when type=provider.
	// +optional
	Plugin *PluginInstall `json:"plugin,omitempty"`

	// Provider holds provider-specific install state. Required when
	// type=provider; must be nil when type=plugin.
	// +optional
	Provider *ProviderInstall `json:"provider,omitempty"`
}

// PluginInstall captures plugin-specific install state.
type PluginInstall struct {
	// PluginCRName is the name of the cluster-scoped Plugin CR that this
	// install record points at.
	// +required
	PluginCRName string `json:"pluginCRName"`

	// FrontendDigest pins the OCI digest of the frontend bundle artifact.
	// +optional
	FrontendDigest string `json:"frontendDigest,omitempty"`

	// BackendImageDigest pins the OCI digest of the backend image.
	// +optional
	BackendImageDigest string `json:"backendImageDigest,omitempty"`

	// Scope controls how kubePermissions translate to RBAC.
	// Defaults to Cluster.
	// +optional
	// +kubebuilder:default=Cluster
	Scope PluginInstallScope `json:"scope,omitempty"`

	// AllowClusterScope must be true for the reconciler to provision a
	// ClusterRole/ClusterRoleBinding when Scope=Cluster. Without it, the
	// reconciler refuses to create cluster-wide RBAC and sets
	// RoleSynced=False, reason=ClusterScopeNotAllowed.
	// +optional
	AllowClusterScope bool `json:"allowClusterScope,omitempty"`

	// Namespaces lists the namespaces this plugin is enabled in. Required
	// when Scope=Namespaces. Each entry may attach a per-tenant config
	// secret reference.
	// +optional
	// +listType=map
	// +listMapKey=name
	Namespaces []PluginNamespaceConfig `json:"namespaces,omitempty"`
}

// PluginNamespaceConfig pairs a namespace name with an optional per-tenant
// configuration secret reference.
type PluginNamespaceConfig struct {
	// Name is the Kubernetes namespace this plugin is enabled in.
	// +required
	Name string `json:"name"`

	// ConfigSecretRef names a Secret in Name whose data is mounted as env
	// vars on the plugin backend for this tenant.
	// +optional
	ConfigSecretRef string `json:"configSecretRef,omitempty"`
}

// ProviderInstall captures provider-specific install state.
type ProviderInstall struct {
	// ProviderName is the name of the cluster-scoped Provider CR that this
	// install record points at.
	// +required
	ProviderName string `json:"providerName"`
}

// InstalledExtensionPhase rolls up the conditions into a single status word.
// +kubebuilder:validation:Enum=Pending;Installed;Upgrading;Failed;Uninstalling
type InstalledExtensionPhase string

const (
	InstalledExtensionPhasePending      InstalledExtensionPhase = "Pending"
	InstalledExtensionPhaseInstalled    InstalledExtensionPhase = "Installed"
	InstalledExtensionPhaseUpgrading    InstalledExtensionPhase = "Upgrading"
	InstalledExtensionPhaseFailed       InstalledExtensionPhase = "Failed"
	InstalledExtensionPhaseUninstalling InstalledExtensionPhase = "Uninstalling"
)

// AvailableUpgrade advertises a newer version pulled from the catalog.
type AvailableUpgrade struct {
	// Version is the SemVer version of the upgrade candidate.
	// +required
	Version string `json:"version"`
	// ChartDigest is the OCI digest of the upgrade's Helm chart.
	// +optional
	ChartDigest string `json:"chartDigest,omitempty"`
}

// InstalledExtensionStatus defines the observed state of an InstalledExtension.
type InstalledExtensionStatus struct {
	// Phase is the rollup of Conditions.
	// +optional
	Phase InstalledExtensionPhase `json:"phase,omitempty"`

	// Conditions describes the current state of the install.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// InstalledAt is the time the extension first reached Phase=Installed.
	// +optional
	InstalledAt *metav1.Time `json:"installedAt,omitempty"`

	// LastCheckedAt is the last time the reconciler polled the catalog for
	// available upgrades.
	// +optional
	LastCheckedAt *metav1.Time `json:"lastCheckedAt,omitempty"`

	// AvailableUpgrade, when set, advertises a newer version available.
	// +optional
	AvailableUpgrade *AvailableUpgrade `json:"availableUpgrade,omitempty"`
}

// Condition types surfaced by the InstalledExtension reconciler.
const (
	// ConditionReady is the rollup condition. True when the install is
	// fully reconciled.
	ConditionReady = "Ready"
	// ConditionRoleSynced reports the state of kubePermissions RBAC
	// provisioning (plugin-only).
	ConditionRoleSynced = "RoleSynced"
	// ConditionBundleServed reports that the frontend bundle is being served
	// (plugin-only, frontend mode).
	ConditionBundleServed = "BundleServed"
	// ConditionBackendReachable reports that the host can reach the plugin
	// backend (plugin-only, backend mode).
	ConditionBackendReachable = "BackendReachable"
	// ConditionTokenIssued reports daemon service-token provisioning
	// (plugin-only, daemon mode).
	ConditionTokenIssued = "TokenIssued"
	// ConditionCRDsInstalled reports custom-resource CRD installation
	// (plugin-only, stateful plugins).
	ConditionCRDsInstalled = "CRDsInstalled"
	// ConditionProviderRegistered reports the Provider CR is present and
	// healthy (provider-only).
	ConditionProviderRegistered = "ProviderRegistered"
	// ConditionUpgradeAvailable is True when a newer version is published
	// by the source catalog.
	ConditionUpgradeAvailable = "UpgradeAvailable"
)

// Reason constants for status conditions.
const (
	ReasonReady                  = "Ready"
	ReasonPluginNotFound         = "PluginNotFound"
	ReasonPluginDisabled         = "PluginDisabled"
	ReasonProviderNotFound       = "ProviderNotFound"
	ReasonClusterScopeNotAllowed = "ClusterScopeNotAllowed"
	ReasonInvalidSpec            = "InvalidSpec"
	ReasonReconciling            = "Reconciling"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,shortName=ie;extinst
// +kubebuilder:printcolumn:name="Type",type="string",JSONPath=".spec.type"
// +kubebuilder:printcolumn:name="Version",type="string",JSONPath=".spec.version"
// +kubebuilder:printcolumn:name="Channel",type="string",JSONPath=".spec.channel"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// InstalledExtension is the cluster-scoped record of an installed extension
// (a generic plugin or a spec 001 provider). It is created by
// `everestctl extension install` and owned by cluster-admin.
type InstalledExtension struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// +required
	Spec InstalledExtensionSpec `json:"spec"`
	// +optional
	Status InstalledExtensionStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// InstalledExtensionList contains a list of InstalledExtension.
type InstalledExtensionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []InstalledExtension `json:"items"`
}

func init() {
	SchemeBuilder.Register(&InstalledExtension{}, &InstalledExtensionList{})
}
