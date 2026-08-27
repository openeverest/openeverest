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

// ImportSpec defines the desired state of Import.
type ImportSpec struct {
	// ClassRef references the cluster-scoped BackupClass that determines how
	// backups in the storage are parsed. The class's executionMode controls
	// how the import is executed. The ProviderManaged classes are reconciled
	// by the provider.
	// +kubebuilder:validation:Required
	ClassRef common.ObjectRef `json:"classRef"`
	// StorageRef references a BackupStorage in the same namespace whose
	// contents are listed and parsed. The reconciler reads the storage and
	// its credentials secret read-only.
	// +kubebuilder:validation:Required
	StorageRef common.ObjectRef `json:"storageRef"`
}

// ImportPhase is the coarse lifecycle state of an import request.
//
// +kubebuilder:validation:Enum=Pending;Importing;Succeeded;Failed
type ImportPhase string

const (
	// ImportPhasePending indicates the request has been accepted but
	// import has not yet started.
	ImportPhasePending ImportPhase = "Pending"
	// ImportPhaseImporting indicates the owner is listing and
	// parsing the storage.
	ImportPhaseImporting ImportPhase = "Importing"
	// ImportPhaseSucceeded indicates import completed and the
	// corresponding Backup CRs have been created.
	ImportPhaseSucceeded ImportPhase = "Succeeded"
	// ImportPhaseFailed indicates import failed terminally; see
	// Status.Message and Status.Conditions.
	ImportPhaseFailed ImportPhase = "Failed"
)

// ImportStatus defines the observed state of Import.
type ImportStatus struct {
	// Phase is the coarse lifecycle state of the import request.
	// +optional
	Phase ImportPhase `json:"phase,omitempty"`
	// DiscoveredCount is the number of restorable backups found in the
	// storage during the last reconcile.
	// +optional
	DiscoveredCount int32 `json:"discoveredCount,omitempty"`
	// CreatedCount is the number of Backup CRs created (or already present)
	// for the discovered backups. Backups are deduped on
	// (storageRef, path), so re-running import does not
	// create duplicates.
	// +optional
	CreatedCount int32 `json:"createdCount,omitempty"`
	// Message is a human-readable message about the current state.
	// +optional
	Message string `json:"message,omitempty"`
	// LastObservedGeneration is the last reconciled generation of the spec.
	// +optional
	LastObservedGeneration int64 `json:"lastObservedGeneration,omitempty"`
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=imp
// +kubebuilder:printcolumn:name="Class",type="string",JSONPath=".spec.classRef.name"
// +kubebuilder:printcolumn:name="Storage",type="string",JSONPath=".spec.storageRef.name"
// +kubebuilder:printcolumn:name="Phase",type="string",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Created",type="integer",JSONPath=".status.createdCount"

// Import is the Schema for the imports API. It discovers
// the restorable backups already present in a BackupStorage and creates a
// Backup CR (spec.imported) for each so they can be restored.
type Import struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of Import
	// +required
	Spec ImportSpec `json:"spec"`

	// status defines the observed state of Import
	// +optional
	Status ImportStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// ImportList contains a list of Import.
type ImportList struct {
	metav1.TypeMeta `json:",inline"`

	metav1.ListMeta `json:"metadata,omitzero"`

	Items []Import `json:"items"`
}

func init() { //nolint:gochecknoinits // kubebuilder scheme registration convention
	SchemeBuilder.Register(&Import{}, &ImportList{})
}
