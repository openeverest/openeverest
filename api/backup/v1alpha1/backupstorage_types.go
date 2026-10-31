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
	"slices"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
)

// BackupStorageSpec defines the desired state of a BackupStorage: an object
// store, of the technology a BackupStorageClass describes, that backups are
// written to. It is referenced by name from Instance.spec.backup.storages,
// Backup.spec.storageRef, Restore point-in-time sources and BackupImports.
type BackupStorageSpec struct {
	// BackupStorageClassRef names the BackupStorageClass, the object storage
	// technology.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="backupStorageClassRef is immutable"
	BackupStorageClassRef common.ObjectRef `json:"backupStorageClassRef"`
	// Location is where backups are kept. Engines treat it as their catalog,
	// so it cannot change: create a new BackupStorage instead.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="location is immutable"
	Location BackupStorageLocation `json:"location"`
	// Parameters are the connection parameters of the technology, validated
	// against the class's parametersSchema.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +optional
	Parameters *runtime.RawExtension `json:"parameters,omitempty"`
	// Authentication names one of the class's authModes and, for every mode
	// but WorkloadIdentity, the Secret in the same namespace holding that
	// mode's credentials keys.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="self.mode != 'Anonymous'",message="a BackupStorage cannot use the Anonymous authentication mode"
	Authentication common.Authentication `json:"authentication"`
	// TLS is how executors verify the server certificate of the store.
	// Absent means the system trust store.
	// +optional
	TLS *common.TLSConfig `json:"tls,omitempty"`
}

// SecretNames returns the names of the Secrets the storage references.
func (s *BackupStorageSpec) SecretNames() []string {
	var names []string
	if ref := s.Authentication.CredentialsSecretRef; ref != nil {
		names = append(names, ref.Name)
	}
	if s.TLS != nil && s.TLS.CA != nil && s.TLS.CA.SecretKeyRef != nil && !slices.Contains(names, s.TLS.CA.SecretKeyRef.Name) {
		names = append(names, s.TLS.CA.SecretKeyRef.Name)
	}
	return names
}

// BackupStorageStatus defines the observed state of a BackupStorage.
type BackupStorageStatus struct {
	// Conditions: Accepted, written by core, and Deleting while a deleted
	// storage waits for its referrers.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// BackupClasses lists every BackupClass and whether it can write to the
	// storage.
	// +listType=map
	// +listMapKey=name
	// +optional
	BackupClasses []BackupClassCompatibility `json:"backupClasses,omitempty"`
}

// BackupClassCompatibility is whether a BackupClass can write to a
// BackupStorage.
type BackupClassCompatibility struct {
	// Name of the BackupClass.
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// Compatible is true when the BackupClass supports the storage's class and
	// authentication mode.
	Compatible bool `json:"compatible"`
	// Reason is why the BackupClass cannot write to the storage.
	// +optional
	Reason string `json:"reason,omitempty"`
}

// BackupStorageLocation is where a BackupStorage keeps its backups.
type BackupStorageLocation struct {
	// Bucket is the bucket, or the container for technologies that call it
	// so.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=255
	// +kubebuilder:validation:Pattern=`^[^/]+$`
	Bucket string `json:"bucket"`
	// Prefix is the path within the bucket under which backups are written,
	// without leading or trailing slash.
	// +kubebuilder:validation:MaxLength=512
	// +kubebuilder:validation:Pattern=`^([^/]+(/[^/]+)*)?$`
	// +kubebuilder:validation:XValidation:rule="!self.split('/').exists(s, s == '.' || s == '..')",message="prefix must not contain '.' or '..' segments"
	// +optional
	Prefix string `json:"prefix,omitempty"`
}

// BackupStorage conditions and their reasons.
const (
	// BackupStorageConditionAccepted reports whether the storage is valid for
	// its BackupStorageClass. It does not test that the store is reachable.
	BackupStorageConditionAccepted = "Accepted"

	// ReasonAccepted means the storage is valid for its class.
	ReasonAccepted = "Accepted"
	// ReasonClassNotFound means the referenced BackupStorageClass does not
	// exist.
	ReasonClassNotFound = "ClassNotFound"
	// ReasonInvalidClass means the BackupStorageClass itself breaks a rule
	// every class must follow.
	ReasonInvalidClass = "InvalidClass"
	// ReasonInvalidLocation means spec.location breaks a naming rule of the
	// technology.
	ReasonInvalidLocation = "InvalidLocation"
	// ReasonInvalidParameters means spec.parameters is rejected by the class.
	ReasonInvalidParameters = "InvalidParameters"
	// ReasonAuthModeUnsupported means the class has no authentication mode
	// named by spec.authentication.mode.
	ReasonAuthModeUnsupported = "AuthModeUnsupported"
	// ReasonCredentialsNotFound means the credentials Secret, or a key the
	// authentication mode requires, is missing.
	ReasonCredentialsNotFound = "CredentialsNotFound"
	// ReasonInvalidCABundle means the CA bundle named by spec.tls.ca is
	// missing or holds no PEM certificate.
	ReasonInvalidCABundle = "InvalidCABundle"

	// BackupStorageConditionDeleting is True while a deleted storage waits
	// for its referrers to go.
	BackupStorageConditionDeleting = "Deleting"
	// ReasonWaitingForReferrers means Instances, Backups, Restores or
	// BackupImports still reference the storage.
	ReasonWaitingForReferrers = "WaitingForReferrers"

	// ReasonBackupStorageClassUnsupported means a BackupClass does not list
	// the storage's BackupStorageClass.
	ReasonBackupStorageClassUnsupported = "BackupStorageClassUnsupported"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=bstor
// +kubebuilder:printcolumn:name="Type",type="string",JSONPath=".spec.backupStorageClassRef.name"
// +kubebuilder:printcolumn:name="Bucket",type="string",JSONPath=".spec.location.bucket"
// +kubebuilder:printcolumn:name="Auth",type="string",JSONPath=".spec.authentication.mode"
// +kubebuilder:printcolumn:name="Accepted",type="string",JSONPath=".status.conditions[?(@.type==\"Accepted\")].reason"
// +kubebuilder:printcolumn:name="Insecure",type="boolean",JSONPath=".spec.tls.insecureSkipVerify",priority=1
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// BackupStorage is the Schema for the backupstorages API.
type BackupStorage struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// +required
	Spec BackupStorageSpec `json:"spec"`
	// +optional
	Status BackupStorageStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// BackupStorageList contains a list of BackupStorage.
type BackupStorageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []BackupStorage `json:"items"`
}

func init() {
	SchemeBuilder.Register(&BackupStorage{}, &BackupStorageList{})
}
