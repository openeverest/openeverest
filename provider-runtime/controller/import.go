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

package controller

import (
	"crypto/tls"
	"fmt"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
)

// BackupImportNameLabel is the label key stamped on Backup CRs created by an
// import, carrying the name of the originating BackupImport.
const BackupImportNameLabel = "backupImportName"

// NewS3Client builds an S3 client from an S3 BackupStorage spec.
func NewS3Client(spec *backupv1alpha1.BackupStorageS3Spec, accessKeyID, secretAccessKey string) (*s3.Client, error) {
	if spec == nil {
		return nil, fmt.Errorf("nil S3 storage spec")
	}

	verifyTLS := true
	if spec.VerifyTLS != nil {
		verifyTLS = *spec.VerifyTLS
	}
	httpClient := awshttp.NewBuildableClient().WithTransportOptions(func(tr *http.Transport) {
		if !verifyTLS {
			if tr.TLSClientConfig == nil {
				tr.TLSClientConfig = &tls.Config{} //nolint:gosec // VerifyTLS explicitly disabled by the user
			}
			tr.TLSClientConfig.InsecureSkipVerify = true
		}
	})

	cfg := aws.Config{
		Region:      spec.Region,
		Credentials: credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, ""),
		HTTPClient:  httpClient,
	}

	forcePathStyle := spec.ForcePathStyle != nil && *spec.ForcePathStyle
	return s3.NewFromConfig(cfg, func(o *s3.Options) {
		if spec.EndpointURL != "" {
			o.BaseEndpoint = aws.String(spec.EndpointURL)
		}
		o.UsePathStyle = forcePathStyle
	}), nil
}
