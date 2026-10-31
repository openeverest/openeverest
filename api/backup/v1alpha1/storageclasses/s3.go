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

package storageclasses

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
)

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
func (s S3) Endpoint() (common.Endpoint, error) {
	return common.ParseEndpoint(s.EndpointURL)
}

// S3Credentials are the credentials of an s3 storage. They are empty for
// authentication modes without a credentials Secret.
type S3Credentials struct {
	AccessKeyID     string
	SecretAccessKey string
}

// S3From decodes an s3 BackupStorage: its location, parameters and the data
// of its credentials Secret for mode. Unknown parameters are an error, never
// ignored.
func S3From(location backupv1alpha1.BackupStorageLocation, parameters *runtime.RawExtension,
	mode string, data map[string][]byte,
) (S3, S3Credentials, error) {
	if err := validateS3Location(location); err != nil {
		return S3{}, S3Credentials{}, err
	}
	s3, err := decodeS3(parameters)
	if err != nil {
		return S3{}, S3Credentials{}, err
	}
	switch mode {
	case AuthModeAccessKey:
		creds := S3Credentials{
			AccessKeyID:     string(data[KeyAWSAccessKeyID]),
			SecretAccessKey: string(data[KeyAWSSecretAccessKey]),
		}
		if creds.AccessKeyID == "" || creds.SecretAccessKey == "" {
			return S3{}, S3Credentials{}, fmt.Errorf("%w: mode %s needs %s and %s",
				ErrCredentialsMissing, AuthModeAccessKey, KeyAWSAccessKeyID, KeyAWSSecretAccessKey,
			)
		}
		return s3, creds, nil
	default:
		return S3{}, S3Credentials{}, fmt.Errorf("%w: %q", ErrAuthModeUnsupported, mode)
	}
}

// s3BucketPattern is the S3 bucket naming rule, relaxed to the underscores
// GCS accepts through its S3 interoperability endpoint.
var s3BucketPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{1,61}[a-z0-9]$`)

func validateS3Location(location backupv1alpha1.BackupStorageLocation) error {
	if !s3BucketPattern.MatchString(location.Bucket) || strings.Contains(location.Bucket, "..") {
		return fmt.Errorf("%w: bucket %q must be 3 to 63 lowercase letters, digits, '.', '-' or '_', "+
			"start and end with a letter or digit, and contain no '..'", ErrInvalidLocation, location.Bucket,
		)
	}
	return nil
}

func decodeS3(parameters *runtime.RawExtension) (S3, error) {
	var s3 S3
	if parameters != nil && len(parameters.Raw) > 0 {
		dec := json.NewDecoder(bytes.NewReader(parameters.Raw))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&s3); err != nil {
			if strings.HasPrefix(err.Error(), "json: unknown field ") {
				return S3{}, fmt.Errorf("%w: %s", ErrUnsupportedParameter, strings.TrimPrefix(err.Error(), "json: "))
			}
			return S3{}, fmt.Errorf("%w: %w", ErrInvalidParameters, err)
		}
	}
	switch {
	case s3.Region == "":
		return S3{}, fmt.Errorf("%w: region is required", ErrInvalidParameters)
	case s3.EndpointURL == "":
		return S3{}, fmt.Errorf("%w: endpointURL is required", ErrInvalidParameters)
	}
	if _, err := s3.Endpoint(); err != nil {
		return S3{}, fmt.Errorf("%w: endpointURL: %w", ErrInvalidParameters, err)
	}
	return s3, nil
}
