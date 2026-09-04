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

	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
)

// BackupImportSpec defines the desired state of BackupImport.
//
// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="spec is immutable"
type BackupImportSpec struct {
	// ClassRef references the cluster-scoped BackupClass that determines how
	// backups in the storage are parsed. The class's executionMode controls
	// how the import is executed. The ProviderManaged classes are reconciled
	// by the provider.
	// +kubebuilder:validation:Required
	ClassRef common.ObjectRef `json:"classRef"`
	// StorageRef references a BackupStorage in the same namespace whose
	// contents are listed and parsed. The reconciler reads the storage and
	// its credentials secret.
	// +kubebuilder:validation:Required
	StorageRef common.ObjectRef `json:"storageRef"`
}

// BackupImportStatus defines the observed state of BackupImport.
type BackupImportStatus struct {
	// DiscoveredCount is the number of restorable backups found in the
	// storage.
	// +optional
	DiscoveredCount int32 `json:"discoveredCount"`
	// CreatedCount is the number of Backup CRs created. Backups are deduped on
	// (storageRef, path), so the import does not create duplicates.
	// +optional
	CreatedCount int32 `json:"createdCount"`
	// LastObservedGeneration is the last observed generation of the BackupImport CR.
	// +optional
	LastObservedGeneration int64 `json:"lastObservedGeneration,omitempty"`
	// State is the current state of the backup import.
	// +optional
	State BackupImportState `json:"state,omitempty"`
	// Message is a human-readable message about the current state.
	// +optional
	Message string `json:"message,omitempty"`
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// BackupImportState is a type representing the state of a backup import request.
//
// +kubebuilder:validation:Enum=Succeeded;Failed;Error
type BackupImportState string

const (
	// BackupImportStateSucceeded indicates the import completed and the
	// corresponding Backup CRs have been created.
	BackupImportStateSucceeded BackupImportState = "Succeeded"
	// BackupImportStateFailed indicates the import failed terminally; see
	// Status.Message and Status.Conditions.
	BackupImportStateFailed BackupImportState = "Failed"
	// BackupImportStateError indicates a transient error; the controller may retry.
	BackupImportStateError BackupImportState = "Error"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=bimp
// +kubebuilder:printcolumn:name="Class",type="string",JSONPath=".spec.classRef.name"
// +kubebuilder:printcolumn:name="Storage",type="string",JSONPath=".spec.storageRef.name"
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.state"
// +kubebuilder:printcolumn:name="Created",type="integer",JSONPath=".status.createdCount"

// BackupImport is the Schema for the backupimports API.
type BackupImport struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of BackupImport
	// +required
	Spec BackupImportSpec `json:"spec"`

	// status defines the observed state of BackupImport
	// +optional
	Status BackupImportStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// BackupImportList contains a list of BackupImport.
type BackupImportList struct {
	metav1.TypeMeta `json:",inline"`

	metav1.ListMeta `json:"metadata,omitzero"`

	Items []BackupImport `json:"items"`
}

func init() { //nolint:gochecknoinits // kubebuilder scheme registration convention
	SchemeBuilder.Register(&BackupImport{}, &BackupImportList{})
}
