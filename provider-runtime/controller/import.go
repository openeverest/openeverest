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
	"context"
	"crypto/tls"
	"fmt"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
)

// BackupImportNameLabel is the label key stamped on Backup CRs created by an
// import, carrying the name of the originating BackupImport.
// The label must be 63 characters or less.
const BackupImportNameLabel = "backup.openeverest.io/backup-import"

// NewS3Client builds an S3 client for the given BackupStorage. It returns an
// error if the storage is not backed by S3. It reads its credentials from the
// referenced Secret.
func NewS3Client(
	ctx context.Context,
	c client.Client,
	storage *backupv1alpha1.BackupStorage,
) (*s3.Client, error) {
	if storage == nil {
		return nil, fmt.Errorf("nil BackupStorage")
	}

	spec := storage.Spec.S3
	if spec == nil {
		return nil, fmt.Errorf("BackupStorage %q is not an S3 storage", storage.Name)
	}

	accessKeyID, secretAccessKey, err := s3Credentials(ctx, c, storage)
	if err != nil {
		return nil, err
	}

	verifyTLS := true
	if spec.VerifyTLS != nil {
		verifyTLS = *spec.VerifyTLS
	}

	httpClient := awshttp.NewBuildableClient().WithTransportOptions(func(tr *http.Transport) {
		if !verifyTLS {
			if tr.TLSClientConfig == nil {
				tr.TLSClientConfig = &tls.Config{}
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

// s3Credentials reads the S3 access key and secret key from the Secret
// referenced by the BackupStorage. It returns empty strings when no
// credentials secret is referenced.
func s3Credentials(
	ctx context.Context,
	c client.Client,
	storage *backupv1alpha1.BackupStorage,
) (string, string, error) {
	ref := storage.Spec.S3.CredentialsSecretRef.Name
	secret := &corev1.Secret{}
	if err := c.Get(ctx, client.ObjectKey{
		Namespace: storage.Namespace,
		Name:      ref,
	}, secret,
	); err != nil {
		return "", "", fmt.Errorf("get credentials secret %q: %w", ref, err)
	}
	return string(secret.Data["AWS_ACCESS_KEY_ID"]), string(secret.Data["AWS_SECRET_ACCESS_KEY"]), nil
}
