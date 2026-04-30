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
)

// RestoreSpec defines the desired state of Restore.
type RestoreSpec struct {
	// InstanceName is the name of the Instance to restore into. The Instance
	// must already exist in the same namespace and use a provider listed in
	// the BackupClass's SupportedProviders.
	// +kubebuilder:validation:Required
	InstanceName string `json:"instanceName"`
	// DataSource defines where the backup data to restore from is located.
	// +kubebuilder:validation:Required
	DataSource RestoreDataSource `json:"dataSource"`
	// Config is the restore-time configuration validated against the
	// BackupClass's .spec.restoreConfig.openAPIV3Schema.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +optional
	Config *runtime.RawExtension `json:"config,omitempty"`
}

// RestoreDataSource defines the source of the backup data for the restore
// operation. Exactly one of BackupName or External must be set.
type RestoreDataSource struct {
	// BackupName references an existing Backup CR in the same namespace to
	// restore from. The BackupClass and storage are resolved from the
	// referenced Backup.
	// +optional
	BackupName string `json:"backupName,omitempty"`
	// External describes a backup that has no corresponding Backup CR in the
	// cluster (e.g., a backup taken outside of OpenEverest).
	// +optional
	External *ExternalRestoreSource `json:"external,omitempty"`
	// PITR defines point-in-time recovery options. Requires the resolved
	// BackupClass to advertise PITR support via .spec.providerManaged.
	// +optional
	PITR *PITR `json:"pitr,omitempty"`
}

// ExternalRestoreSource defines an external backup source for restore
// operations when no Backup CR exists in the cluster.
type ExternalRestoreSource struct {
	// BackupClassName is the name of the BackupClass that defines how to
	// restore this external backup.
	// +kubebuilder:validation:Required
	BackupClassName string `json:"backupClassName"`
	// StorageName references the BackupStorage in the same namespace that
	// describes where the external backup data is located.
	// +kubebuilder:validation:Required
	StorageName string `json:"storageName"`
	// Config is forwarded to the BackupClass's restore configuration. It is
	// validated against the same schema as Restore.spec.config.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +optional
	Config *runtime.RawExtension `json:"config,omitempty"`
}

// PITRType defines the type of point-in-time recovery.
//
// +kubebuilder:validation:Enum=date;latest
type PITRType string

const (
	// PITRTypeDate indicates recovery to a specific date and time.
	PITRTypeDate PITRType = "date"
	// PITRTypeLatest indicates recovery to the latest available point in time.
	PITRTypeLatest PITRType = "latest"
)

// PITR defines point-in-time recovery configuration.
type PITR struct {
	// Type is the type of point-in-time recovery: "date" or "latest".
	// +kubebuilder:validation:Required
	Type PITRType `json:"type"`
	// Date is the target recovery point in time. Required when Type is "date".
	// +optional
	Date *metav1.Time `json:"date,omitempty"`
}

// RestoreState is a type representing the state of a restore.
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
	OperatorRestoreRef *corev1.TypedLocalObjectReference `json:"operatorRestoreRef,omitempty"`
	// JobName is the reference to the Job that is running the restore.
	// Populated only for Job classes.
	// +optional
	JobName string `json:"jobName,omitempty"`
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
// +kubebuilder:printcolumn:name="Instance",type="string",JSONPath=".spec.instanceName"
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
