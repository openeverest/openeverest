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
	"k8s.io/apimachinery/pkg/runtime"

	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
)

// RestoreSpec defines the desired state of Restore.
type RestoreSpec struct {
	// InstanceRef references the Instance to restore into. The Instance
	// must already exist in the same namespace and use a provider listed in
	// the BackupClass's SupportedProviders.
	// +kubebuilder:validation:Required
	InstanceRef common.ObjectRef `json:"instanceRef"`
	// DataSource identifies the data to restore from. The same type is used
	// by Instance.spec.dataSource when seeding a new Instance, so both paths
	// identify a source identically.
	// +kubebuilder:validation:Required
	DataSource DataSource `json:"dataSource"`
	// Parameters is the restore-time structured configuration validated
	// against the resolved BackupClass's .spec.restoreParametersSchema. It
	// carries restore *operation* modifiers -- how the data is applied --
	// and applies to both data source types.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +optional
	Parameters *runtime.RawExtension `json:"parameters,omitempty"`
}

// DataSourceType selects the restore intent: recover the state captured by a
// specific Backup, or roll a backup stream forward to a point in time.
//
// Note this discriminates the kind of *source*, not the kind of restore
// operation: the operation is identical in both cases, and no engine models
// point-in-time recovery as a separate operation.
//
// +kubebuilder:validation:Enum=Backup;PointInTime
type DataSourceType string

const (
	// DataSourceTypeBackup restores the state captured by an existing Backup
	// CR in the same namespace, whatever its engine-level type (full,
	// differential or incremental -- engines resolve the chain themselves).
	DataSourceTypeBackup DataSourceType = "Backup"
	// DataSourceTypePointInTime rolls a backup stream forward to a target
	// point. The provider selects whatever base backup the engine requires.
	DataSourceTypePointInTime DataSourceType = "PointInTime"
)

// DataSourceBackup identifies the backup to restore.
type DataSourceBackup struct {
	// BackupRef references the Backup CR in the same namespace.
	// +kubebuilder:validation:Required
	BackupRef common.ObjectRef `json:"backupRef"`
}

// StreamSource identifies the backup stream to recover from: InstanceRef says
// whose data it is, StorageRef says where the stream lives.
//
// StorageRef is always required. One BackupStorage commonly holds the streams
// of many Instances, and an Instance may archive to several storages with
// different retention -- so naming it is the only way a manifest can be read
// and validated on its own, without consulting the Instance it points at.
type StreamSource struct {
	// InstanceRef names the Instance whose stream to recover. Defaults to the
	// Restore's target Instance when omitted; required when seeding a new
	// Instance via Instance.spec.dataSource, which has no stream of its own.
	// +optional
	InstanceRef *common.ObjectRef `json:"instanceRef,omitempty"`
	// StorageRef selects which of the source Instance's registered
	// BackupStorages to read the stream from. It must name a storage with
	// .pitr.enabled=true on that Instance.
	// +kubebuilder:validation:Required
	StorageRef common.ObjectRef `json:"storageRef"`
}

// DataSourcePointInTime rolls a backup stream forward to a target point.
//
// +kubebuilder:validation:XValidation:rule="self.recoveryTarget == 'date' ? has(self.date) : !has(self.date)",message="date must be set if and only if recoveryTarget is date"
type DataSourcePointInTime struct {
	// Source identifies the backup stream to recover from.
	// +kubebuilder:validation:Required
	Source StreamSource `json:"source"`
	// RecoveryTarget selects date-based or latest recovery. This enum is
	// deliberately closed: date and latest are the only recovery targets that
	// are meaningful without knowing which engine is running. Engine-specific
	// targets (GTID, LSN, XID, named restore points) are out of scope.
	// +kubebuilder:validation:Required
	RecoveryTarget RecoveryTarget `json:"recoveryTarget"`
	// Date is the recovery point, RFC 3339 with an explicit UTC offset.
	// Required when RecoveryTarget is "date", forbidden otherwise. Providers
	// convert it to the engine's expected representation; several engines
	// interpret timezone-less timestamps as node-local, so the offset is not
	// optional.
	// +optional
	Date *metav1.Time `json:"date,omitempty"`
}

// DataSource describes the data a restore or an initial Instance seeding
// operation should read from. The Type field selects which intent-specific
// block is populated.
//
// +kubebuilder:validation:XValidation:rule="self.type == 'Backup' ? has(self.backup) : !has(self.backup)",message="backup must be set if and only if type is Backup"
// +kubebuilder:validation:XValidation:rule="self.type == 'PointInTime' ? has(self.pointInTime) : !has(self.pointInTime)",message="pointInTime must be set if and only if type is PointInTime"
type DataSource struct {
	// Type selects the restore intent.
	// +kubebuilder:validation:Required
	Type DataSourceType `json:"type"`
	// Backup identifies the backup to restore. Required when type=Backup.
	// +optional
	Backup *DataSourceBackup `json:"backup,omitempty"`
	// PointInTime identifies the stream and the point to recover to.
	// Required when type=PointInTime.
	// +optional
	PointInTime *DataSourcePointInTime `json:"pointInTime,omitempty"`
}

// RecoveryTarget selects which point in a backup stream to recover to.
//
// The name follows the established "recovery target" terminology (PostgreSQL's
// recovery_target_*, CloudNativePG's recoveryTarget). The qualifier matters: an
// unqualified "target" would read as the Instance being restored into, which is
// .spec.instanceRef.
//
// +kubebuilder:validation:Enum=date;latest
type RecoveryTarget string

const (
	// RecoveryTargetDate recovers to a specific date and time.
	RecoveryTargetDate RecoveryTarget = "date"
	// RecoveryTargetLatest recovers as far forward as the stream allows.
	RecoveryTargetLatest RecoveryTarget = "latest"
)

// RestoreState is a type representing the state of a restore.
//
// +kubebuilder:validation:Enum=Pending;Running;Succeeded;Failed;Error
type RestoreState string

const (
	// RestoreStatePending indicates that the restore has been accepted but
	// has not yet started.
	RestoreStatePending RestoreState = "Pending"
	// RestoreStateRunning indicates that the restore is currently running.
	RestoreStateRunning RestoreState = "Running"
	// RestoreStateSucceeded indicates that the restore completed successfully.
	RestoreStateSucceeded RestoreState = "Succeeded"
	// RestoreStateFailed indicates that the restore has failed terminally.
	RestoreStateFailed RestoreState = "Failed"
	// RestoreStateError indicates a transient error; the controller may retry.
	RestoreStateError RestoreState = "Error"
)

// RestoreStatus defines the observed state of Restore.
type RestoreStatus struct {
	// ExecutionMode is the resolved execution mode at the time the Restore
	// started. Recorded for observability.
	// +optional
	ExecutionMode BackupExecutionMode `json:"executionMode,omitempty"`
	// OperatorRestoreRef points at the operator-native restore resource the
	// provider created (e.g., PerconaServerMongoDBRestore). Populated only
	// for ProviderManaged classes.
	// +optional
	OperatorRestoreRef *common.TypedObjectRef `json:"operatorRestoreRef,omitempty"`
	// JobRef references the Job that is running the restore.
	// Populated only for Job classes.
	// +optional
	JobRef *common.ObjectRef `json:"jobRef,omitempty"`
	// StartedAt is the time when the restore started.
	// +optional
	StartedAt *metav1.Time `json:"startedAt,omitempty"`
	// CompletedAt is the time when the restore completed successfully.
	// +optional
	CompletedAt *metav1.Time `json:"completedAt,omitempty"`
	// LastObservedGeneration is the last observed generation of the Restore CR.
	// +optional
	LastObservedGeneration int64 `json:"lastObservedGeneration,omitempty"`
	// State is the current state of the restore.
	// +optional
	State RestoreState `json:"state,omitempty"`
	// Message is a human-readable message about the current state.
	// +optional
	Message string `json:"message,omitempty"`
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=rs;rst
// +kubebuilder:printcolumn:name="Instance",type="string",JSONPath=".spec.instanceRef.name"
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.state"

// Restore is the Schema for the restores API.
type Restore struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// +required
	Spec RestoreSpec `json:"spec"`
	// +optional
	Status RestoreStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// RestoreList contains a list of Restore.
type RestoreList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []Restore `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Restore{}, &RestoreList{})
}
