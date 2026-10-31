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

// BackupStorageType is the type of object storage backing a BackupStorage.
//
// +kubebuilder:validation:Enum=s3
type BackupStorageType string

const (
	// BackupStorageTypeS3 indicates an S3-compatible object store.
	BackupStorageTypeS3 BackupStorageType = "s3"
)

// BackupStorageSpec defines the desired state of a BackupStorage.
//
// A BackupStorage is a reusable, namespaced reference to an object store
// (today only S3-compatible) plus the credentials needed to talk to it.
// It is referenced by name from:
//
//   - Instance.spec.backup.storages[].storageRef
//   - Backup.spec.storageRef
//
// Decoupling storage from individual Backup CRs makes provider-managed
// backups (e.g. PBM, pgBackRest) practical: the provider can register a
// fixed set of storages on the engine without recomputing them from a
// dynamic list of Backup CRs.
type BackupStorageSpec struct {
	// Type is the object storage type. Today only "s3" is supported.
	// +kubebuilder:validation:Required
	Type BackupStorageType `json:"type"`

	// S3 contains S3-compatible storage configuration.
	// Required when Type is "s3".
	// +optional
	S3 *BackupStorageS3Spec `json:"s3,omitempty"`
}

// BackupStorageS3Spec defines an S3-compatible object store.
type BackupStorageS3Spec struct {
	// Bucket is the name of the S3 bucket.
	// +kubebuilder:validation:Required
	Bucket string `json:"bucket"`

	// Region is the region of the S3 bucket.
	// +kubebuilder:validation:Required
	Region string `json:"region"`

	// EndpointURL is the endpoint URL of the S3-compatible service.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="isURL(self)",message="endpointURL must be a valid URL"
	EndpointURL string `json:"endpointURL"`

	// VerifyTLS enables TLS certificate verification.
	// Defaults to true.
	//
	// +kubebuilder:default:=true
	// +optional
	VerifyTLS *bool `json:"verifyTLS,omitempty"`

	// ForcePathStyle forces path-style URLs (bucket name in the path
	// instead of the host). Defaults to false.
	//
	// +kubebuilder:default:=false
	// +optional
	ForcePathStyle *bool `json:"forcePathStyle,omitempty"`

	// CredentialsSecretRef references the Secret in the same namespace
	// that holds the AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY keys.
	// +kubebuilder:validation:Required
	CredentialsSecretRef common.SecretRef `json:"credentialsSecretRef"`

	// AccessKeyID is a write-only convenience input. When set, a webhook
	// stores it in the Secret named by CredentialsSecretRef and clears
	// this field. It is never persisted on the BackupStorage object.
	// +optional
	AccessKeyID string `json:"accessKeyId,omitempty"`

	// SecretAccessKey is a write-only convenience input. See AccessKeyID.
	// +optional
	SecretAccessKey string `json:"secretAccessKey,omitempty"`
}

// BackupStorageStatus defines the observed state of a BackupStorage.
type BackupStorageStatus struct{}

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
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=bstor
// +kubebuilder:printcolumn:name="Type",type="string",JSONPath=".spec.type"
// +kubebuilder:printcolumn:name="Bucket",type="string",JSONPath=".spec.s3.bucket"

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
