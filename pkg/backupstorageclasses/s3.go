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

package backupstorageclasses

const (
	// ClassS3 is the built-in BackupStorageClass for Amazon S3 and
	// S3-compatible object stores.
	ClassS3 = "s3"
	// AuthModeAccessKey authenticates with a static access key pair.
	AuthModeAccessKey = "AccessKey"
	// KeyAWSAccessKeyID is the access key ID in an AccessKey credentials
	// Secret.
	KeyAWSAccessKeyID = "AWS_ACCESS_KEY_ID"
	// KeyAWSSecretAccessKey is the secret access key in an AccessKey
	// credentials Secret.
	KeyAWSSecretAccessKey = "AWS_SECRET_ACCESS_KEY" //nolint:gosec // a Secret key name, not a credential
)

// S3Implementation names the software serving an S3 endpoint, for executors
// that speak to S3-compatible gateways differently from Amazon S3.
//
// +kubebuilder:validation:Enum=aws;minio;ceph;gcs;other
type S3Implementation string

// S3 implementations.
const (
	S3ImplementationAWS   S3Implementation = "aws"
	S3ImplementationMinIO S3Implementation = "minio"
	S3ImplementationCeph  S3Implementation = "ceph"
	S3ImplementationGCS   S3Implementation = "gcs"
	S3ImplementationOther S3Implementation = "other"
)

// S3 is the connection parameters of the s3 class.
type S3 struct {
	// Region of the bucket.
	// +openeverest:param=connection
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	Region string `json:"region"`
	// EndpointURL is the URL of the S3 API, for example
	// https://s3.us-east-1.amazonaws.com. An http:// URL uses plain HTTP.
	// +openeverest:param=connection
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MaxLength=2048
	// +kubebuilder:validation:Pattern=`^https?://[^/?#@]+([/?#].*)?$`
	EndpointURL string `json:"endpointURL"` //nolint:tagliatelle // the name BackupStorage has always used
	// ForcePathStyle addresses the bucket in the URL path instead of the host
	// name, as most S3-compatible gateways require.
	// +openeverest:param=connection
	// +optional
	ForcePathStyle bool `json:"forcePathStyle,omitempty"`
	// Implementation names the software serving the endpoint. Unset means
	// Amazon S3.
	// +openeverest:param=connection
	// +optional
	Implementation S3Implementation `json:"implementation,omitempty"`
}

// EffectiveImplementation returns the implementation executors act on: aws
// when unset, other for a value this version does not know.
func (s S3) EffectiveImplementation() S3Implementation {
	switch s.Implementation {
	case "":
		return S3ImplementationAWS
	case S3ImplementationAWS, S3ImplementationMinIO, S3ImplementationCeph, S3ImplementationGCS, S3ImplementationOther:
		return s.Implementation
	default:
		return S3ImplementationOther
	}
}

// Endpoint parses EndpointURL.
func (s S3) Endpoint() (Endpoint, error) {
	return ParseEndpoint(s.EndpointURL)
}
