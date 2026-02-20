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
)

// BackupSpec defines the desired state of Backup
type BackupSpec struct {
	// InstanceName is the name of the Instance to back up.
	// +kubebuilder:validation:Required
	InstanceName    string `json:"instanceName,omitempty"`
	*BackupTemplate `json:",inline"`
}

// BackupTemplate defines a shared template for the backup job.
type BackupTemplate struct {
	// BackupClassName is the backup tool to use for the backup.
	// +kubebuilder:validation:Required
	BackupClassName string `json:"backupClassName,omitempty"`
	// TODO: Why is this a pointer?
	// Destination is the destination for the backup data.
	// +kubebuilder:validation:Required
	Destination *BackupDestination `json:"destination,omitempty"`
	// Config defines the configuration for the backup job.
	// These options are specific to the BackupClass being used and must conform to
	// the schema defined in the BackupClass's .spec.config.openAPIV3Schema.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +optional
	Config *runtime.RawExtension `json:"config,omitempty"`
}

// BackupDestination defines the destination for the backup data.
type BackupDestination struct {
	// BackupStorageName is the name of the BackupStorage to use for the backup.
	// +optional
	BackupStorageName *string `json:"backupStorageName,omitempty"`
	// S3 contains the S3 information for the backup destination.
	// +optional
	S3 *BackupS3Destination `json:"s3,omitempty"`
}

// BackupS3Destination defines the S3 destination for the backup job.
type BackupS3Destination struct {
	// Bucket is the name of the S3 bucket.
	// +kubebuilder:validation:Required
	Bucket string `json:"bucket,omitempty"`
	// Region is the region of the S3 bucket.
	// +kubebuilder:validation:Required
	Region string `json:"region,omitempty"`
	// EndpointURL is an endpoint URL of backup storage.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="isURL(self)",message="endpointURL must be a valid URL"
	EndpointURL string `json:"endpointURL,omitempty"`
	// VerifyTLS is set to ensure TLS/SSL verification.
	// If unspecified, the default value is true.
	//
	// +kubebuilder:default:=true
	VerifyTLS *bool `json:"verifyTLS,omitempty"`
	// ForcePathStyle is set to use path-style URLs.
	// If unspecified, the default value is false.
	//
	// +kubebuilder:default:=false
	ForcePathStyle *bool `json:"forcePathStyle,omitempty"`
	// CredentialsSecreName is the reference to the secret containing the S3 credentials.
	// The Secret must contain the keys `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY`.
	// +kubebuilder:validation:Required
	CredentialsSecretName string `json:"credentialsSecretName,omitempty"`

	// AccessKeyID allows specifying the S3 access key ID inline.
	// It is provided as a write-only input field for convenience.
	// When this field is set, a webhook writes this value in the Secret specified by `credentialsSecretName`
	// and empties this field.
	// This field is not stored in the API.
	// +optional
	AccessKeyID string `json:"accessKeyId,omitempty"`
	// SecretAccessKey allows specifying the S3 secret access key inline.
	// It is provided as a write-only input field for convenience.
	// When this field is set, a webhook writes this value in the Secret specified by `credentialsSecretName`
	// and empties this field.
	// This field is not stored in the API.
	// +optional
	SecretAccessKey string `json:"secretAccessKey,omitempty"`
}

// BackupStatus defines the observed state of Backup.
type BackupStatus struct {
	// StartedAt is the time when the backup job started.
	StartedAt *metav1.Time `json:"startedAt,omitempty"`
	// CompletedAt is the time when the backup job completed successfully.
	CompletedAt *metav1.Time `json:"completedAt,omitempty"`
	// LastObservedGeneration is the last observed generation of the backup job.
	LastObservedGeneration int64 `json:"lastObservedGeneration,omitempty"`
	// State is the current state of the backup job.
	State BackupState `json:"state,omitempty"`
	// Message is the message of the backup job.
	Message string `json:"message,omitempty"`
	// JobName is the reference to the job that is running the backup.
	// +optional
	JobName string `json:"jobName,omitempty"`
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// BackupState is a type representing the state of a backup job.
type BackupState string

const (
	// BackupStatePending indicates that the backup job is pending.
	BackupStatePending BackupState = "Pending"
	// BackupStateRunning indicates that the backup job is currently running.
	BackupStateRunning BackupState = "Running"
	// BackupStateSucceeded indicates that the backup job has completed successfully.
	BackupStateSucceeded BackupState = "Succeeded"
	// BackupStateFailed indicates that the backup job has failed.
	// Once the job is in this phase, it cannot be retried.
	BackupStateFailed BackupState = "Failed"
	// BackupStateError indicates that the backup job has encountered an error.
	// This phase is used for transient errors that may allow the job to be retried.
	BackupStateError BackupState = "Error"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=bk;bak
// +kubebuilder:printcolumn:name="TargetCluster",type="string",JSONPath=".spec.targetClusterName"
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.state"

// Backup is the Schema for the backups API
type Backup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitzero"`

	Spec   BackupSpec   `json:"spec"`
	Status BackupStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// BackupList contains a list of Backup
type BackupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []Backup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Backup{}, &BackupList{})
}
