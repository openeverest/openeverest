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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"

	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
)

// MonitoringExecutionMode says who renders a monitoring binding.
//
// +kubebuilder:validation:Enum=ProviderManaged;ExtensionManaged
type MonitoringExecutionMode string

const (
	// MonitoringExecutionModeProviderManaged means the provider renders the
	// engine operator's native integration inside Sync.
	MonitoringExecutionModeProviderManaged MonitoringExecutionMode = "ProviderManaged"
	// MonitoringExecutionModeExtensionManaged means the class controller wires
	// the backend from Instance.status.monitoring.sources.
	MonitoringExecutionModeExtensionManaged MonitoringExecutionMode = "ExtensionManaged"
)

// Keys of every monitoring credential Secret listed in
// MonitoringSources.credentials.
const (
	CredentialSecretKeyUsername = "username"
	CredentialSecretKeyPassword = "password"
)

// MonitoringKindPattern validates every monitoring engine kind. Kubebuilder
// markers cannot reference constants, so the Pattern markers on
// MonitoringComponentType.Kind and MetricsEndpoint.Kind must stay identical
// to it.
const MonitoringKindPattern = `^([a-z0-9.-]+/)?[a-z0-9]([a-z0-9._-]*[a-z0-9])?$`

// InstanceMonitoringSpec configures where the Instance sends monitoring data.
// It is an object so Instance-wide fields can be added without a breaking
// change.
type InstanceMonitoringSpec struct {
	// Destinations lists the MonitoringDestinations this Instance sends to.
	// Core creates one MonitoringBinding per entry.
	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:MaxItems=8
	// +kubebuilder:validation:XValidation:rule="self.all(a, self.exists_one(b, b.destinationRef.name == a.destinationRef.name))",message="each MonitoringDestination may be listed at most once"
	// +optional
	Destinations []InstanceMonitoringDestination `json:"destinations,omitempty"`
}

// InstanceMonitoringDestination sends the Instance's monitoring data to one
// MonitoringDestination.
type InstanceMonitoringDestination struct {
	// Name identifies the entry within the Instance; it is the key of
	// status.monitoring.destinations[] and part of the MonitoringBinding name.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`
	Name string `json:"name"`
	// DestinationRef references the MonitoringDestination in the Instance's
	// namespace.
	// +kubebuilder:validation:Required
	DestinationRef common.MonitoringDestinationRef `json:"destinationRef"`
	// ExecutionMode pins who renders the binding. When unset the mode is
	// resolved once when the binding is created and never changes on its own;
	// changing the pin recreates the binding.
	// +optional
	ExecutionMode MonitoringExecutionMode `json:"executionMode,omitempty"`
	// Parameters are validated against the class's instanceParametersSchema.
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Parameters *runtime.RawExtension `json:"parameters,omitempty"`
}

// InstanceMonitoringStatus is written only by provider-runtime: sources is
// what the Instance exposes, destinations mirrors spec.monitoring.
type InstanceMonitoringStatus struct {
	// Sources is the per-Instance publication of the provider's monitoring
	// contract, consumed by ExtensionManaged class controllers.
	// +optional
	Sources *MonitoringSources `json:"sources,omitempty"`
	// Destinations has one entry per spec.monitoring.destinations[] entry.
	// +listType=map
	// +listMapKey=name
	// +optional
	Destinations []InstanceMonitoringDestinationStatus `json:"destinations,omitempty"`
}

// InstanceMonitoringDestinationStatus is the read-only summary of one entry,
// mirrored from its MonitoringBinding.
type InstanceMonitoringDestinationStatus struct {
	// Name matches spec.monitoring.destinations[].name.
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// Mode is the resolved execution mode.
	// +optional
	Mode MonitoringExecutionMode `json:"mode,omitempty"`
	// Configured mirrors the binding's Configured condition; False when the
	// binding is not Accepted, Unknown while it is missing or pending.
	// +optional
	Configured metav1.ConditionStatus `json:"configured,omitempty"`
	// Reason is the machine-readable reason for Configured.
	// +optional
	Reason string `json:"reason,omitempty"`
	// Message is the human-readable detail for Configured.
	// +optional
	Message string `json:"message,omitempty"`
}

// MetricsSource states how a component type exposes OpenMetrics.
//
// +kubebuilder:validation:Enum=Native;Exporter;None
type MetricsSource string

const (
	// MetricsSourceNative means the component's own process serves
	// OpenMetrics.
	MetricsSourceNative MetricsSource = "Native"
	// MetricsSourceExporter means the provider can run an exporter for it.
	MetricsSourceExporter MetricsSource = "Exporter"
	// MetricsSourceNone means no metrics are available.
	MetricsSourceNone MetricsSource = "None"
)

// ProviderMonitoring is the provider's static monitoring contract.
type ProviderMonitoring struct {
	// Integrations lists the operator-native monitoring integrations the
	// provider renders itself, keyed by integration name (e.g. "pmm").
	// +optional
	Integrations map[string]MonitoringIntegration `json:"integrations,omitempty"`
	// ComponentTypes declares what each component type exposes, keyed by
	// component type (the software), not by component name.
	// +optional
	ComponentTypes map[string]MonitoringComponentType `json:"componentTypes,omitempty"`
}

// MonitoringIntegration describes one operator-native integration.
type MonitoringIntegration struct {
	// AgentVersions is the semver range of agent versions the provider can
	// render (e.g. ">=3.0.0 <4.0.0").
	// +optional
	AgentVersions string `json:"agentVersions,omitempty"`
}

// MonitoringComponentType declares what one component type exposes.
type MonitoringComponentType struct {
	// Kind is the engine kind (e.g. "mysql", "haproxy"). Well-known values
	// follow OpenTelemetry db.system.name; unknown kinds are allowed.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^([a-z0-9.-]+/)?[a-z0-9]([a-z0-9._-]*[a-z0-9])?$`
	Kind string `json:"kind"`
	// Metrics states how the component exposes OpenMetrics.
	// +kubebuilder:default=None
	// +optional
	Metrics MetricsSource `json:"metrics,omitempty"`
}

// MonitoringSources is the per-Instance publication of the monitoring
// contract, computed by the provider and validated and written by
// provider-runtime. Signals are keys: metrics today, logs and traces later.
type MonitoringSources struct {
	// Identity carries the labels every monitoring class must attach so
	// destinations can tell Instances and clusters apart. Stamped by the
	// runtime.
	// +optional
	Identity MonitoringIdentity `json:"identity,omitzero"`
	// Credentials lists the Instance-owned monitoring credential Secrets by
	// profile.
	// +listType=map
	// +listMapKey=profile
	// +optional
	Credentials []MonitoringCredential `json:"credentials,omitempty"`
	// Metrics lists the OpenMetrics endpoints that are currently serving.
	// +listType=atomic
	// +kubebuilder:validation:MaxItems=32
	// +optional
	Metrics []MetricsEndpoint `json:"metrics,omitempty"`
}

// MonitoringIdentity identifies the Instance towards monitoring destinations.
type MonitoringIdentity struct {
	// ClusterID is the UID of the kube-system namespace.
	// +optional
	ClusterID string `json:"clusterId,omitempty"`
	// Labels are the identity labels (openeverest_instance,
	// openeverest_namespace, openeverest_provider, k8s_cluster_id).
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
}

// MonitoringCredential references one Instance-owned monitoring credential
// Secret. The Secret's keys are fixed: "username" (CredentialSecretKeyUsername)
// and "password" (CredentialSecretKeyPassword).
type MonitoringCredential struct {
	// Profile names the credential profile (e.g. "metrics").
	// +kubebuilder:validation:Required
	Profile string `json:"profile"`
	// SecretRef references the Secret in the Instance's namespace holding
	// the "username" and "password" keys.
	// +kubebuilder:validation:Required
	SecretRef common.SecretRef `json:"secretRef"`
}

// MetricsEndpoint describes one OpenMetrics scrape target. It matches the
// Prometheus Operator PodMetricsEndpoint on every field they share; scrape
// policy (interval, timeout, relabelings) belongs to the destination.
type MetricsEndpoint struct {
	// Component is the Instance component name (e.g. "engine", "proxy").
	// +kubebuilder:validation:Required
	Component string `json:"component"`
	// Kind is the component's engine kind.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=`^([a-z0-9.-]+/)?[a-z0-9]([a-z0-9._-]*[a-z0-9])?$`
	Kind string `json:"kind"`
	// PodSelector selects the pods serving the endpoint.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinProperties=1
	PodSelector map[string]string `json:"podSelector"`
	// Port is the container port, by name or number.
	// +kubebuilder:validation:Required
	Port intstr.IntOrString `json:"port"`
	// Path is the HTTP path.
	// +kubebuilder:default="/metrics"
	// +optional
	Path string `json:"path,omitempty"`
	// Scheme is "http" or "https".
	// +kubebuilder:validation:Enum=http;https
	// +kubebuilder:default=http
	// +optional
	Scheme string `json:"scheme,omitempty"`
	// TLS describes how to verify an https endpoint.
	// +optional
	TLS *MetricsEndpointTLS `json:"tls,omitempty"`
	// Auth names the credential profile the scraper must authenticate with.
	// +optional
	Auth *MetricsEndpointAuth `json:"auth,omitempty"`
	// Params are extra scrape query parameters (e.g. target for /probe).
	// +optional
	Params map[string][]string `json:"params,omitempty"`
}

// MetricsEndpointTLS describes how to verify an https endpoint.
type MetricsEndpointTLS struct {
	// CA selects the Secret key in the Instance's namespace holding the CA
	// bundle.
	// +optional
	CA *corev1.SecretKeySelector `json:"ca,omitempty"`
	// ServerName overrides the expected server name.
	// +optional
	ServerName string `json:"serverName,omitempty"`
	// InsecureSkipVerify disables verification.
	// +optional
	InsecureSkipVerify bool `json:"insecureSkipVerify,omitempty"`
}

// MetricsEndpointAuth names the credential profile a scraper authenticates with.
type MetricsEndpointAuth struct {
	// CredentialProfile matches MonitoringSources.credentials[].profile.
	// +kubebuilder:validation:Required
	CredentialProfile string `json:"credentialProfile"`
}
