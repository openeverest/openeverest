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

package backup

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	"github.com/openeverest/openeverest/v2/api/backup/v1alpha1/jobspec"
	"github.com/openeverest/openeverest/v2/api/backup/v1alpha1/storageclasses"
)

// jobStorageDetails resolves the BackupStorage named name into the job
// payload. Job mode writes only to the s3 class.
func jobStorageDetails(ctx context.Context, c client.Client, namespace, name string) (*jobspec.StorageDetails, error) {
	storage := &backupv1alpha1.BackupStorage{}
	if err := c.Get(ctx, client.ObjectKey{Namespace: namespace, Name: name}, storage); err != nil {
		return nil, fmt.Errorf("failed to get BackupStorage %q: %w", name, err)
	}
	if class := storage.Spec.BackupStorageClassRef.Name; class != storageclasses.ClassS3 {
		return nil, fmt.Errorf("BackupStorage %q is of BackupStorageClass %q; Job mode supports only %q", name, class, storageclasses.ClassS3)
	}
	var data map[string][]byte
	if ref := storage.Spec.Authentication.CredentialsSecretRef; ref != nil {
		secret := &corev1.Secret{}
		if err := c.Get(ctx, client.ObjectKey{Namespace: namespace, Name: ref.Name}, secret); err != nil {
			return nil, fmt.Errorf("failed to get credentials secret %q: %w", ref.Name, err)
		}
		data = secret.Data
	}
	params, creds, err := storageclasses.S3From(storage.Spec.Location, storage.Spec.Parameters, storage.Spec.Authentication.Mode, data)
	if err != nil {
		return nil, fmt.Errorf("BackupStorage %q: %w", name, err)
	}
	s3 := &jobspec.S3Details{
		Bucket:          storage.Spec.Location.Bucket,
		Region:          params.Region,
		EndpointURL:     params.EndpointURL,
		VerifyTLS:       storage.Spec.TLS == nil || !storage.Spec.TLS.InsecureSkipVerify,
		ForcePathStyle:  params.ForcePathStyle,
		AccessKeyID:     creds.AccessKeyID,
		SecretAccessKey: creds.SecretAccessKey,
		Implementation:  string(params.EffectiveImplementation()),
	}
	if t := storage.Spec.TLS; t != nil && t.CA != nil && t.CA.SecretKeyRef != nil {
		ref := t.CA.SecretKeyRef
		secret := &corev1.Secret{}
		if err := c.Get(ctx, client.ObjectKey{Namespace: namespace, Name: ref.Name}, secret); err != nil {
			return nil, fmt.Errorf("failed to get CA secret %q: %w", ref.Name, err)
		}
		s3.CABundle = string(secret.Data[ref.Key])
	}
	return &jobspec.StorageDetails{S3: s3, Path: storage.Spec.Location.Prefix}, nil
}
