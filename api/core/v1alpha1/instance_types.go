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

	// Backup configures the backup feature for this Instance. When enabled,
	// the provider's reconciler is given the resolved BackupClass and storage
	// list so it can configure the engine accordingly (sidecars, agent
	// configuration, etc.). Required for ProviderManaged BackupClasses; Job
	// classes do not need an entry here because they read directly from
	// individual Backup CRs.
	// +optional
	Backup *InstanceBackupSpec `json:"backup,omitempty"`
}

// InstanceBackupSpec configures the backup feature on an Instance.
//
// Schedules and PITR are configured per storage (under
// .spec.backup.storages[].schedules and .spec.backup.storages[].pitr) so
// that each storage carries its own backup policy. Schedule names must be
// unique across all storages on the Instance because engines use them as
// global identifiers and they appear on mirrored Backup CRs as
// .spec.scheduleName.
//
// +kubebuilder:validation:XValidation:rule="!has(self.storages) || self.storages.all(s1, !has(s1.schedules) || s1.schedules.all(sch1, self.storages.filter(s2, has(s2.schedules) && s2.schedules.exists(sch2, sch2.name == sch1.name)).size() <= 1 && s1.schedules.filter(sch2, sch2.name == sch1.name).size() == 1))",message="schedule names must be unique across all storages"
type InstanceBackupSpec struct {
	// Enabled toggles the backup feature for this Instance. When false the
	// runtime skips ConfigureBackup() and the rest of this struct is ignored.
	Enabled bool `json:"enabled"`
	// ClassRef references the BackupClass that the provider should use to
	// configure the engine. The class must have ExecutionMode=ProviderManaged
	// and list the Instance's provider in its SupportedProviders.
	// +kubebuilder:validation:Required
	ClassRef BackupClassReference `json:"classRef"`
	// Storages registers BackupStorages on the engine. Each entry maps a
	// logical name (visible to the engine and reused by Backup CRs via
	// .spec.storageName) to a BackupStorage resource. Schedules and PITR are
	// configured per storage via the nested .schedules and .pitr fields.
	// +optional
	// +kubebuilder:validation:MaxItems=10
	Storages []InstanceBackupStorage `json:"storages,omitempty"`
}

// BackupClassReference references a BackupClass by name.
type BackupClassReference struct {
	// Name is the BackupClass name. BackupClasses are cluster-scoped.
	// +kubebuilder:validation:Required
	Name string `json:"name"`
}

// InstanceBackupStorage registers a BackupStorage on the Instance and
// carries the backup policy (schedules, PITR) that targets it.
type InstanceBackupStorage struct {
	// Name is the logical name the engine uses for this storage. It is also
	// the value that Backup CRs target via .spec.storageName.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MaxLength=63
	Name string `json:"name"`
	// StorageRef references a BackupStorage in the same namespace.
	// +kubebuilder:validation:Required
	StorageRef corev1.LocalObjectReference `json:"storageRef"`
	// Main marks this storage as the engine's default. At most one storage
	// per Instance may be marked main.
	// +optional
	Main bool `json:"main,omitempty"`
	// Schedules registers recurring backup tasks that write to this storage.
	// Schedules produce Backup CRs (via the provider's mirroring loop) using
	// the operator-native scheduler — the runtime never spawns CronJobs for
	// ProviderManaged BackupClasses. Schedule names must be unique across
	// all storages on the Instance.
	// +optional
	// +kubebuilder:validation:MaxItems=10
	Schedules []InstanceBackupSchedule `json:"schedules,omitempty"`
	// PITR enables and configures point-in-time recovery writing to this
	// storage. Requires the BackupClass to advertise PITR support via
	// .spec.providerManaged. Engines that support only a single PITR stream
	// (e.g. PSMDB, PXC) require at most one storage on the Instance to set
	// .pitr.enabled=true; this is enforced by the provider, not by the
	// core schema (PG legitimately archives WAL to every configured repo).
	// +optional
	PITR *InstanceBackupStoragePITR `json:"pitr,omitempty"`
}

// InstanceBackupSchedule configures a recurring backup task on the engine
// for the parent storage. The provider translates each schedule into the
// engine's native scheduler (e.g. PSMDB BackupTaskSpec, PXC
// PXCScheduledBackupSchedule, pgBackRest schedule). Operator-produced
// backups are mirrored back into Backup CRs by the provider, sharing the
// operator backup's name.
type InstanceBackupSchedule struct {
	// Name uniquely identifies the schedule. The provider uses it as the
	// schedule key on the engine and as the value of Backup.spec.scheduleName
	// on mirrored Backup CRs. Names must be unique across all storages on
	// the Instance.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	Name string `json:"name"`
	// Enabled toggles the schedule. A disabled schedule is removed from
	// the engine without losing its definition on the Instance.
	Enabled bool `json:"enabled"`
	// Cron is a standard 5-field cron expression. The provider may reject
	// expressions the engine does not support.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Cron string `json:"cron"`
	// RetentionCopies is the number of recent backups to keep for this
	// schedule. Zero (or unset) means "keep all". Negative values are
	// rejected.
	// +kubebuilder:validation:Minimum=0
	// +optional
	RetentionCopies int32 `json:"retentionCopies,omitempty"`
}

// InstanceBackupStoragePITR configures point-in-time recovery writing to
// the parent storage.
type InstanceBackupStoragePITR struct {
	// Enabled toggles PITR for this storage.
	Enabled bool `json:"enabled"`
	// Config holds provider-specific PITR options. The schema is defined by
	// the BackupClass via .spec.providerManaged.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +optional
	Config *runtime.RawExtension `json:"config,omitempty"`
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

	// ConditionBackupConfigured indicates whether the provider has successfully
	// configured the backup feature on the instance engine. This condition is
	// only set when Backup.Enabled=true; it remains absent otherwise. When False,
	// the reason and message explain the configuration failure (e.g., storage
	// resolution or PITR wiring error).
	ConditionBackupConfigured = "BackupConfigured"
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
