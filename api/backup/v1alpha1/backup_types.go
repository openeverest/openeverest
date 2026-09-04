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

// BackupSpec defines the desired state of Backup.
//
// +kubebuilder:validation:XValidation:rule="self.origin.type != 'External' || self.deletionPolicy != 'Delete'",message="deletionPolicy Delete is not allowed when origin.type is External; use Retain"
type BackupSpec struct {
	// Origin identifies where this Backup's data comes from: produced by a
	// live Instance, or imported from data already present in a BackupStorage.
	// +kubebuilder:validation:Required
	Origin BackupOrigin `json:"origin"`
	// ClassRef references the cluster-scoped BackupClass that defines how
	// this Backup is executed. The class's executionMode controls the runtime
	// path: Job classes are reconciled by the in-cluster Backup job
	// controller; ProviderManaged classes are reconciled by the provider's
	// runtime.
	// +kubebuilder:validation:Required
	ClassRef common.ObjectRef `json:"classRef"`
	// StorageRef references a BackupStorage in the same namespace that
	// defines where the backup data is written. For ProviderManaged classes
	// the referenced storage must already be registered on the Instance via
	// .spec.backup.storages so the engine can write to it.
	// +kubebuilder:validation:Required
	StorageRef common.ObjectRef `json:"storageRef"`
	// ScheduleName, when set, identifies the InstanceBackupSchedule that
	// produced this Backup. Backups created via the API or `kubectl apply`
	// leave this field empty (on-demand). The provider's mirroring loop
	// sets it when surfacing operator-produced scheduled backups as Backup
	// CRs.
	// +optional
	ScheduleName string `json:"scheduleName,omitempty"`
	// Parameters is the backup-time structured configuration validated
	// against the BackupClass's .spec.parametersSchema.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +optional
	Parameters *runtime.RawExtension `json:"parameters,omitempty"`
	// DeletionPolicy controls what happens to the underlying backup data
	// (e.g., the object stored in S3) when this Backup CR is deleted.
	// Delete (default) instructs the provider to remove both the
	// engine-native backup resource and the data in the configured
	// BackupStorage. Retain instructs the provider to remove the
	// engine-native backup resource but to leave the underlying data in
	// place, so it can be recovered later out-of-band.
	//
	// The field is mutable on a live Backup but is frozen once deletion
	// has started: switching policies after .metadata.deletionTimestamp
	// has been set is rejected so the cleanup path cannot race with
	// itself.
	// +kubebuilder:default=Delete
	// +optional
	DeletionPolicy BackupDeletionPolicy `json:"deletionPolicy,omitempty"`
}

// BackupDeletionPolicy controls what happens to the underlying backup data
// when a Backup CR is deleted. See BackupSpec.DeletionPolicy for the full
// semantics.
//
// +kubebuilder:validation:Enum=Retain;Delete
type BackupDeletionPolicy string

const (
	// BackupDeletionPolicyDelete instructs the provider to remove both the
	// engine-native backup resource and the underlying data in the
	// BackupStorage. This is the default and matches the historical
	// behavior of the platform.
	BackupDeletionPolicyDelete BackupDeletionPolicy = "Delete"

	// BackupDeletionPolicyRetain instructs the provider to remove the
	// engine-native backup resource but to leave the underlying data in
	// the BackupStorage untouched. The data can then be recovered or
	// pruned out-of-band by an operator.
	BackupDeletionPolicyRetain BackupDeletionPolicy = "Retain"
)

// BackupOriginType selects how a Backup came to exist: produced by a live
// Instance, or imported from data already present in a BackupStorage.
//
// +kubebuilder:validation:Enum=Instance;External
type BackupOriginType string

const (
	// BackupOriginTypeInstance marks a Backup produced by a live Instance in
	// the same namespace, identified by origin.instanceRef.
	BackupOriginTypeInstance BackupOriginType = "Instance"
	// BackupOriginTypeExternal marks a Backup imported from data already
	// present in a BackupStorage, identified by origin.external.
	BackupOriginTypeExternal BackupOriginType = "External"
)

// BackupOrigin describes where a Backup's data comes from. Type selects which
// variant-specific field is populated.
//
// +kubebuilder:validation:XValidation:rule="self.type == 'Instance' ? has(self.instanceRef) : !has(self.instanceRef)",message="instanceRef must be set if and only if type is Instance"
// +kubebuilder:validation:XValidation:rule="self.type == 'External' ? has(self.external) : !has(self.external)",message="external must be set if and only if type is External"
type BackupOrigin struct {
	// Type selects the origin variant.
	// +kubebuilder:validation:Required
	Type BackupOriginType `json:"type"`
	// InstanceRef references the Instance that produced this Backup. The
	// Instance must live in the same namespace as this Backup. Required when
	// Type is Instance.
	// +optional
	InstanceRef *common.ObjectRef `json:"instanceRef,omitempty"`
	// External identifies data already present in the referenced BackupStorage
	// rather than produced by a live Instance. Required when Type is External.
	// When set, the restore is built directly from storageRef + external.path
	// with no live operator object.
	// +optional
	External *BackupOriginExternal `json:"external,omitempty"`
}

// BackupOriginExternal marks a Backup as imported and identifies where its data already
// lives within the BackupStorage referenced by Backup.spec.storageRef, so it
// can be restored without a live source Instance or operator-native backup
// object.
//
// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="spec.origin.external is immutable"
type BackupOriginExternal struct {
	// Path is the backup's path within the BackupStorage. The bucket is
	// already determined by storageRef, so it is not repeated here. The path
	// is unique within its storage and is used for restore.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	Path string `json:"path"`
	// StartedAt is the time when the backup started.
	// +kubebuilder:validation:Required
	StartedAt metav1.Time `json:"startedAt"`
	// CompletedAt is the time when the backup completed.
	// +kubebuilder:validation:Required
	CompletedAt metav1.Time `json:"completedAt"`
}

// BackupStatus defines the observed state of Backup.
type BackupStatus struct {
	// ExecutionMode is the resolved execution mode at the time the Backup
	// started. Recorded for observability.
	// +optional
	ExecutionMode BackupExecutionMode `json:"executionMode,omitempty"`
	// Size is the size of the backup data as reported by the engine.
	// Empty for external backups.
	// +optional
	Size *string `json:"size,omitempty"`
	// OperatorBackupRef points at the operator-native backup resource the
	// provider created (e.g., PerconaServerMongoDBBackup). Populated only
	// for ProviderManaged classes. Empty for external backups, which have no
	// operator-native backup object.
	// +optional
	OperatorBackupRef *common.TypedObjectRef `json:"operatorBackupRef,omitempty"`
	// JobRef references the Job that is running the backup.
	// Populated only for Job classes. Empty for external backups, which run
	// no Job.
	// +optional
	JobRef *common.ObjectRef `json:"jobRef,omitempty"`
	// StartedAt is the time when the backup started.
	// For external backups this mirrors spec.origin.external.startedAt.
	// +optional
	StartedAt *metav1.Time `json:"startedAt,omitempty"`
	// CompletedAt is the time when the backup completed successfully.
	// For external backups this mirrors spec.origin.external.completedAt.
	// +optional
	CompletedAt *metav1.Time `json:"completedAt,omitempty"`
	// LastObservedGeneration is the last observed generation of the Backup CR.
	// +optional
	LastObservedGeneration int64 `json:"lastObservedGeneration,omitempty"`
	// State is the current state of the backup.
	// For external backups, the state is Succeeded if the backup has valid
	// StartedAt and CompletedAt set.
	// +optional
	State BackupState `json:"state,omitempty"`
	// Message is a human-readable message about the current state.
	// +optional
	Message string `json:"message,omitempty"`
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// BackupState is a type representing the state of a backup.
//
// +kubebuilder:validation:Enum=Pending;Running;Succeeded;Failed;Error;Deleting
type BackupState string

const (
	// BackupStatePending indicates that the backup has been accepted but
	// has not yet started.
	BackupStatePending BackupState = "Pending"
	// BackupStateRunning indicates that the backup is currently running.
	BackupStateRunning BackupState = "Running"
	// BackupStateSucceeded indicates that the backup completed successfully.
	BackupStateSucceeded BackupState = "Succeeded"
	// BackupStateFailed indicates that the backup has failed terminally.
	BackupStateFailed BackupState = "Failed"
	// BackupStateError indicates a transient error; the controller may retry.
	BackupStateError BackupState = "Error"
	// BackupStateDeleting indicates that the backup is in the process of being deleted.
	BackupStateDeleting BackupState = "Deleting"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=bk;bak
// +kubebuilder:printcolumn:name="Instance",type="string",JSONPath=".spec.origin.instanceRef.name"
// +kubebuilder:printcolumn:name="Storage",type="string",JSONPath=".spec.storageRef.name"
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.state"

// Backup is the Schema for the backups API.
type Backup struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// +required
	Spec BackupSpec `json:"spec"`
	// +optional
	Status BackupStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// BackupList contains a list of Backup.
type BackupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []Backup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Backup{}, &BackupList{})
}
