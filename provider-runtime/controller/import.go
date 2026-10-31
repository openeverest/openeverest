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
	"errors"
	"fmt"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	"github.com/openeverest/openeverest/v2/api/backup/v1alpha1/storageclasses"
	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
)

// NewS3Client builds an S3 client for a BackupStorage of the s3 class. It
// reads the credentials and CA bundle the storage references, and verifies
// TLS unless the storage turns verification off.
func NewS3Client(
	ctx context.Context,
	c client.Client,
	storage *backupv1alpha1.BackupStorage,
) (*s3.Client, error) {
	if storage == nil {
		return nil, errors.New("nil BackupStorage")
	}
	if class := storage.Spec.BackupStorageClassRef.Name; class != storageclasses.ClassS3 {
		return nil, fmt.Errorf("BackupStorage %q is of BackupStorageClass %q; only %q is supported", storage.Name, class, storageclasses.ClassS3)
	}

	var data map[string][]byte
	if ref := storage.Spec.Authentication.CredentialsSecretRef; ref != nil {
		secret := &corev1.Secret{}
		if err := c.Get(ctx, client.ObjectKey{Namespace: storage.Namespace, Name: ref.Name}, secret); err != nil {
			return nil, fmt.Errorf("get credentials secret %q: %w", ref.Name, err)
		}
		data = secret.Data
	}
	params, creds, err := storageclasses.S3From(storage.Spec.Location, storage.Spec.Parameters, storage.Spec.Authentication.Mode, data)
	if err != nil {
		return nil, fmt.Errorf("BackupStorage %q: %w", storage.Name, err)
	}
	tlsConfig, err := storageTLSConfig(ctx, c, storage)
	if err != nil {
		return nil, err
	}

	httpClient := awshttp.NewBuildableClient().WithTransportOptions(func(tr *http.Transport) {
		tr.TLSClientConfig = tlsConfig
	})
	cfg := aws.Config{
		Region:      params.Region,
		Credentials: credentials.NewStaticCredentialsProvider(creds.AccessKeyID, creds.SecretAccessKey, ""),
		HTTPClient:  httpClient,
	}
	// S3-compatible gateways reject the checksums the SDK sends to Amazon S3
	// by default.
	whenRequired := params.EffectiveImplementation() != storageclasses.S3ImplementationAWS
	return s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(params.EndpointURL)
		o.UsePathStyle = params.ForcePathStyle
		if whenRequired {
			o.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
			o.ResponseChecksumValidation = aws.ResponseChecksumValidationWhenRequired
		}
	}), nil
}

// storageTLSConfig returns the TLS client configuration of a storage's
// spec.tls: its CA bundle, or no verification.
func storageTLSConfig(ctx context.Context, c client.Client, storage *backupv1alpha1.BackupStorage) (*tls.Config, error) {
	cfg := &tls.Config{MinVersion: tls.VersionTLS12}
	t := storage.Spec.TLS
	switch {
	case t == nil:
	case t.InsecureSkipVerify:
		cfg.InsecureSkipVerify = true
	case t.CA != nil && t.CA.SecretKeyRef != nil:
		ref := t.CA.SecretKeyRef
		secret := &corev1.Secret{}
		if err := c.Get(ctx, client.ObjectKey{Namespace: storage.Namespace, Name: ref.Name}, secret); err != nil {
			return nil, fmt.Errorf("get CA secret %q: %w", ref.Name, err)
		}
		pool, err := common.ParseCABundle(secret.Data[ref.Key])
		if err != nil {
			return nil, fmt.Errorf("CA secret %q key %q: %w", ref.Name, ref.Key, err)
		}
		cfg.RootCAs = pool
	}
	return cfg, nil
}
