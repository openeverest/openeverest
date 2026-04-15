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

	// Version selects a provider-defined version bundle, resolving compatible
	// versions for all components automatically. Per-component versions set
	// in Components take precedence over the bundle.
	// If omitted and the provider defines a default bundle, that bundle is used.
	// +optional
	Version string `json:"version,omitempty"`

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

	// Version is the effective version bundle that is currently applied to this
	// Instance. On the first reconciliation the provider-runtime writes the
	// resolved default bundle name here and uses this value on every subsequent
	// reconciliation when spec.version is empty. This ensures that a Provider
	// upgrade (which may change the default bundle) never silently triggers an
	// unintended database upgrade on existing Instances.
	//
	// GitOps tools (ArgoCD, Flux) exclude status from diff calculations by
	// default, so this field does not cause spurious out-of-sync alerts.
	//
	// +optional
	Version string `json:"version,omitempty"`
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

// InstancePhase represents the high-level, mutually exclusive lifecycle state
// of an Instance. These phases are designed for human readability, providing an
// immediate understanding of the instance's current lifecycle stage.
//
// +kubebuilder:validation:Enum=Pending;Provisioning;Initializing;Ready;Updating;Terminating;Failed;Restoring;Suspending;Suspended;Resuming
type InstancePhase string

const (
	// --- Core Lifecycle Phases ---

	// InstancePhasePending indicates the Instance CR has been accepted by the
	// API server, but the provider has not yet begun provisioning (e.g.,
	// waiting on resource quotas or prerequisite checks).
	InstancePhasePending InstancePhase = "Pending"

	// InstancePhaseProvisioning indicates the provider is actively creating the
	// underlying Kubernetes infrastructure (StatefulSets, PVCs, Services,
	// Secrets, ConfigMaps).
	InstancePhaseProvisioning InstancePhase = "Provisioning"

	// InstancePhaseInitializing indicates the infrastructure exists and a fresh
	// instance engine is booting. This covers operations such as bootstrap
	// scripts, default user setup, or initial quorum establishment.
	InstancePhaseInitializing InstancePhase = "Initializing"

	// InstancePhaseReady indicates the instance is fully operational, healthy,
	// and actively accepting client connections. This is the target steady
	// state.
	InstancePhaseReady InstancePhase = "Ready"

	// InstancePhaseUpdating indicates the provider is actively rolling out a
	// mutation (e.g., scaling resources, modifying configuration flags, or
	// performing a version upgrade).
	InstancePhaseUpdating InstancePhase = "Updating"

	// InstancePhaseTerminating indicates the user has requested deletion. The
	// instance is actively spinning down and resources are being reclaimed.
	InstancePhaseTerminating InstancePhase = "Terminating"

	// InstancePhaseFailed indicates a terminal or semi-terminal error requiring
	// human intervention (e.g., persistent CrashLoopBackOff or unrecoverable
	// disk corruption).
	InstancePhaseFailed InstancePhase = "Failed"

	// --- Data Recovery Phase ---

	// InstancePhaseRestoring indicates the instance is actively downloading and
	// unpacking data from an external backup source (e.g., S3 bucket or volume
	// snapshot). This phase is distinct from Initializing because it can take
	// hours, has different failure domains (network/storage vs. compute), and
	// is triggered by a spec.init.fromBackup directive or a Restore CR.
	InstancePhaseRestoring InstancePhase = "Restoring"

	// --- Cost-Saving (Compute-to-Zero) Phases ---

	// InstancePhaseSuspending indicates the provider is gracefully shutting
	// down the instance engine, flushing memory buffers to disk, and preparing
	// to scale compute replicas to zero.
	InstancePhaseSuspending InstancePhase = "Suspending"

	// InstancePhaseSuspended indicates the instance compute is scaled to zero.
	// The instance is completely offline and not incurring compute charges, but
	// PersistentVolumes remain intact.
	InstancePhaseSuspended InstancePhase = "Suspended"

	// InstancePhaseResuming indicates the user has requested the instance to
	// wake up. The provider is scaling compute back up, reattaching existing
	// storage, and warming the instance engine. Once complete, the instance
	// transitions to Ready.
	InstancePhaseResuming InstancePhase = "Resuming"
)

// Condition types for Instance.
const (
	// ConditionConnectionDetailsReady indicates whether the connection
	// details Secret has been populated by the provider.
	ConditionConnectionDetailsReady = "ConnectionDetailsReady"

	// ConditionStorageResizing is a state-indicator condition that is True
	// while a PVC volume expansion is in flight, and False when storage is in
	// a steady state. Monitoring tools can use this to suppress disk I/O alerts
	// during the storage controller's block metadata rewrite.
	ConditionStorageResizing = "StorageResizing"

	// ConditionUpgrading is a state-indicator condition that is True while a
	// version upgrade is in flight, and False when the instance is running its
	// target version. External CI/CD pipelines can use this to block subsequent
	// infrastructure changes until the upgrade completes.
	ConditionUpgrading = "Upgrading"
)

// Reasons for the StorageResizing condition.
const (
	// ReasonStorageExpansionTriggered indicates the operator has updated the
	// PVC; waiting for the cloud provider to provision the additional capacity.
	ReasonStorageExpansionTriggered = "ExpansionTriggered"

	// ReasonStorageFileSystemResizePending indicates the cloud disk is already
	// larger, but the Kubelet has not yet expanded the filesystem inside the pod.
	ReasonStorageFileSystemResizePending = "FileSystemResizePending"

	// ReasonStorageResizeCompleted indicates the resize finished successfully
	// and the new capacity is available to the instance.
	ReasonStorageResizeCompleted = "ResizeCompleted"

	// ReasonStorageQuotaExceeded indicates the cloud provider rejected the
	// expansion request due to a storage quota limit.
	ReasonStorageQuotaExceeded = "QuotaExceeded"

	// ReasonStorageResizeFailed indicates the expansion failed (e.g., the
	// storage class does not support online expansion).
	ReasonStorageResizeFailed = "ResizeFailed"
)

// Reasons for the Upgrading condition.
const (
	// ReasonUpgradeMinorVersionRolling indicates a non-disruptive, pod-by-pod
	// restart is in progress (e.g., 15.1 → 15.2). Traffic continues to be
	// served throughout the rollout.
	ReasonUpgradeMinorVersionRolling = "MinorVersionRolling"

	// ReasonUpgradeMajorDataConversion indicates a disruptive logical upgrade
	// is in progress (e.g., Postgres 14 → 15) that may require downtime.
	ReasonUpgradeMajorDataConversion = "MajorDataConversion"

	// ReasonUpgradeAwaitingReplicaSync indicates the primary has been upgraded
	// but the operator is waiting for read-replicas to catch up before
	// completing the rollout.
	ReasonUpgradeAwaitingReplicaSync = "AwaitingReplicaSync"

	// ReasonUpgradeCompleted indicates the instance is successfully running the
	// version specified in spec.
	ReasonUpgradeCompleted = "UpgradeCompleted"

	// ReasonUpgradeFailed indicates the upgrade encountered a fatal error
	// (e.g., a deprecated configuration parameter) and is stuck or rolling back.
	ReasonUpgradeFailed = "UpgradeFailed"
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
