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

// BackupSpec defines the desired state of Backup.
type BackupSpec struct {
	// InstanceName is the name of the Instance to back up. The Instance must
	// live in the same namespace as this Backup.
	// +kubebuilder:validation:Required
	InstanceName string `json:"instanceName"`
	// BackupClassName is the BackupClass that defines how this Backup is
	// executed. The class's executionMode controls the runtime path: Job
	// classes are reconciled by the in-cluster Backup job controller;
	// ProviderManaged classes are reconciled by the provider's runtime.
	// +kubebuilder:validation:Required
	BackupClassName string `json:"backupClassName"`
	// StorageName references a BackupStorage in the same namespace that
	// defines where the backup data is written. For ProviderManaged classes
	// the referenced storage must already be registered on the Instance via
	// .spec.backup.storages so the engine can write to it.
	// +kubebuilder:validation:Required
	StorageName string `json:"storageName"`
	// ScheduleName, when set, identifies the InstanceBackupSchedule that
	// produced this Backup. Backups created via the API or `kubectl apply`
	// leave this field empty (on-demand). The provider's mirroring loop
	// sets it when surfacing operator-produced scheduled backups as Backup
	// CRs.
	// +optional
	ScheduleName string `json:"scheduleName,omitempty"`
	// Config is the backup-time configuration validated against the
	// BackupClass's .spec.config.openAPIV3Schema.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +optional
	Config *runtime.RawExtension `json:"config,omitempty"`
}

// BackupStatus defines the observed state of Backup.
type BackupStatus struct {
	// ExecutionMode is the resolved execution mode at the time the Backup
	// started. Recorded for observability.
	// +optional
	ExecutionMode BackupExecutionMode `json:"executionMode,omitempty"`
	// OperatorBackupRef points at the operator-native backup resource the
	// provider created (e.g., PerconaServerMongoDBBackup). Populated only
	// for ProviderManaged classes.
	// +optional
	OperatorBackupRef *corev1.TypedLocalObjectReference `json:"operatorBackupRef,omitempty"`
	// JobName is the reference to the Job that is running the backup.
	// Populated only for Job classes.
	// +optional
	JobName string `json:"jobName,omitempty"`
	// StartedAt is the time when the backup started.
	// +optional
	StartedAt *metav1.Time `json:"startedAt,omitempty"`
	// CompletedAt is the time when the backup completed successfully.
	// +optional
	CompletedAt *metav1.Time `json:"completedAt,omitempty"`
	// LastObservedGeneration is the last observed generation of the Backup CR.
	// +optional
	LastObservedGeneration int64 `json:"lastObservedGeneration,omitempty"`
	// State is the current state of the backup.
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
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=bk;bak
// +kubebuilder:printcolumn:name="Instance",type="string",JSONPath=".spec.instanceName"
// +kubebuilder:printcolumn:name="Storage",type="string",JSONPath=".spec.storageName"
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
