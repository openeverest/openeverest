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

package s3

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/openeverest/openeverest/v2/backupstorageclasses"
)

// Location is the location of an s3 BackupStorage, its
// spec.location.objectStore.
type Location struct {
	Bucket string
	// Prefix is the path within the bucket, without leading or trailing
	// slash.
	Prefix string
}

// Credentials are the credentials of an s3 storage. They are empty for
// authentication modes without a credentials Secret.
type Credentials struct {
	AccessKeyID     string
	SecretAccessKey string
}

// Decode checks the location of an s3 BackupStorage and decodes its raw
// spec.parameters. Unknown parameters are an error, never ignored.
func Decode(location Location, parameters []byte) (Parameters, error) {
	if err := validateLocation(location); err != nil {
		return Parameters{}, err
	}
	var p Parameters
	if err := backupstorageclasses.DecodeParameters(parameters, &p); err != nil {
		return Parameters{}, err
	}
	switch {
	case p.Region == "":
		return Parameters{}, fmt.Errorf("%w: region is required", backupstorageclasses.ErrInvalidParameters)
	case p.EndpointURL == "":
		return Parameters{}, fmt.Errorf("%w: endpointURL is required", backupstorageclasses.ErrInvalidParameters)
	}
	if _, err := p.Endpoint(); err != nil {
		return Parameters{}, fmt.Errorf("%w: endpointURL: %w", backupstorageclasses.ErrInvalidParameters, err)
	}
	return p, nil
}

// DecodeCredentials decodes the data of the credentials Secret of an s3
// storage authenticating with mode.
func DecodeCredentials(mode string, data map[string][]byte) (Credentials, error) {
	switch mode {
	case AuthModeAccessKey:
		creds := Credentials{
			AccessKeyID:     string(data[KeyAWSAccessKeyID]),
			SecretAccessKey: string(data[KeyAWSSecretAccessKey]),
		}
		if creds.AccessKeyID == "" || creds.SecretAccessKey == "" {
			return Credentials{}, fmt.Errorf("%w: mode %s needs %s and %s",
				backupstorageclasses.ErrCredentialsMissing, AuthModeAccessKey, KeyAWSAccessKeyID, KeyAWSSecretAccessKey,
			)
		}
		return creds, nil
	default:
		return Credentials{}, fmt.Errorf("%w: %q", backupstorageclasses.ErrAuthModeUnsupported, mode)
	}
}

// bucketPattern is the S3 bucket naming rule, relaxed to the underscores GCS
// accepts through its S3 interoperability endpoint.
var bucketPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{1,61}[a-z0-9]$`)

func validateLocation(location Location) error {
	if !bucketPattern.MatchString(location.Bucket) || strings.Contains(location.Bucket, "..") {
		return fmt.Errorf("%w: bucket %q must be 3 to 63 lowercase letters, digits, '.', '-' or '_', "+
			"start and end with a letter or digit, and contain no '..'", backupstorageclasses.ErrInvalidLocation, location.Bucket,
		)
	}
	return nil
}
