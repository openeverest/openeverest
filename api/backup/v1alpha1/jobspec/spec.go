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

// Package jobspec defines the JSON payload contract between the backup/restore
// controller and the job containers that perform backup and restore operations.
//
// The controller creates a Kubernetes Secret containing the JSON-serialized Spec,
// mounts it into the job container, and passes the mount path as the first
// command-line argument. For example:
//
//	./my-backup-tool backup /payload/request.json
//	./my-backup-tool restore /payload/request.json
package jobspec

// Spec defines the structure of the JSON payload passed to backup/restore job containers.
type Spec struct {
	// Instance identifies the target database instance.
	Instance InstanceRef `json:"instance"`
	// Connection contains the resolved connection details for the target instance.
	// This is populated from the Instance's ConnectionSecretRef.
	// May be nil if the Instance is not yet running (e.g., during restore to a new instance).
	Connection *ConnectionDetails `json:"connection,omitempty"`
	// Storage contains the S3 storage details for the backup destination or restore source.
	Storage *StorageDetails `json:"storage,omitempty"`
	// PITR contains point-in-time recovery options, only set for restore operations.
	PITR *PITRDetails `json:"pitr,omitempty"`
	// Config contains BackupClass-specific configuration, passed through from the
	// Backup or Restore CR's .spec.config field.
	Config map[string]interface{} `json:"config,omitempty"`
}

// InstanceRef identifies the target database instance.
type InstanceRef struct {
	// Name is the Instance CR name.
	Name string `json:"name"`
	// Namespace is the Instance CR namespace.
	Namespace string `json:"namespace"`
}

// ConnectionDetails contains the resolved connection information for the target instance.
// The fields match the well-known keys from the Instance's ConnectionSecretRef.
type ConnectionDetails struct {
	// Type is the database type (e.g., "mongodb", "postgresql").
	Type string `json:"type,omitempty"`
	// Provider is the provider name (e.g., "percona-server-mongodb").
	Provider string `json:"provider,omitempty"`
	// Host is the hostname or IP address of the database.
	Host string `json:"host,omitempty"`
	// Port is the port number.
	Port string `json:"port,omitempty"`
	// Username is the database username.
	Username string `json:"username,omitempty"`
	// Password is the database password.
	Password string `json:"password,omitempty"`
	// URI is the full connection URI including credentials.
	URI string `json:"uri,omitempty"`
}

// StorageDetails contains the storage location for backup data.
type StorageDetails struct {
	// S3 contains S3-compatible storage details.
	S3 *S3Details `json:"s3,omitempty"`
	// Path is the key prefix or path within the storage bucket.
	Path string `json:"path,omitempty"`
}

// S3Details contains configuration for S3-compatible storage.
type S3Details struct {
	// Bucket is the name of the S3 bucket.
	Bucket string `json:"bucket,omitempty"`
	// Region is the AWS region of the S3 bucket.
	Region string `json:"region,omitempty"`
	// EndpointURL is the endpoint URL for the S3-compatible service.
	EndpointURL string `json:"endpointURL,omitempty"`
	// VerifyTLS specifies whether to verify TLS certificates.
	VerifyTLS bool `json:"verifyTLS,omitempty"`
	// ForcePathStyle specifies whether to use path-style URLs.
	ForcePathStyle bool `json:"forcePathStyle,omitempty"`
	// AccessKeyID is the AWS access key ID.
	AccessKeyID string `json:"accessKeyID,omitempty"`
	// SecretAccessKey is the AWS secret access key.
	SecretAccessKey string `json:"secretAccessKey,omitempty"`
}

// PITRDetails contains point-in-time recovery configuration.
type PITRDetails struct {
	// Type is the PITR type: "date" or "latest".
	Type string `json:"type"`
	// Date is the target recovery point in RFC3339 format.
	// Required when Type is "date".
	Date string `json:"date,omitempty"`
}
