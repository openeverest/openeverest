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

package k8s

import (
	"context"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
)

// ListBackupStorages returns list of backup storages in a namespace.
func (h *k8sHandler) ListBackupStorages(ctx context.Context, cluster, namespace string) (*backupv1alpha1.BackupStorageList, error) {
	return h.kubeConnector.ListBackupStorages(ctx, ctrlclient.InNamespace(namespace))
}

// GetBackupStorage returns a backup storage by name and namespace.
func (h *k8sHandler) GetBackupStorage(ctx context.Context, cluster, namespace, name string) (*backupv1alpha1.BackupStorage, error) {
	return h.kubeConnector.GetBackupStorage(ctx, types.NamespacedName{Namespace: namespace, Name: name})
}

// CreateBackupStorage creates a backup storage.
func (h *k8sHandler) CreateBackupStorage(ctx context.Context, cluster string, bs *backupv1alpha1.BackupStorage) (*backupv1alpha1.BackupStorage, error) {
	if err := h.applyS3Credentials(ctx, bs); err != nil {
		return nil, fmt.Errorf("failed to apply S3 credentials: %w", err)
	}
	return h.kubeConnector.CreateBackupStorage(ctx, bs)
}

// UpdateBackupStorage updates a backup storage.
func (h *k8sHandler) UpdateBackupStorage(ctx context.Context, cluster string, bs *backupv1alpha1.BackupStorage) (*backupv1alpha1.BackupStorage, error) {
	if err := h.applyS3Credentials(ctx, bs); err != nil {
		return nil, fmt.Errorf("failed to apply S3 credentials: %w", err)
	}
	return h.kubeConnector.UpdateBackupStorage(ctx, bs)
}

// PatchBackupStorage patches a backup storage by fetching the current state and
// merging only the non-zero fields from bs onto it before updating.
func (h *k8sHandler) PatchBackupStorage(ctx context.Context, cluster string, bs *backupv1alpha1.BackupStorage) (*backupv1alpha1.BackupStorage, error) {
	current, err := h.kubeConnector.GetBackupStorage(ctx, types.NamespacedName{
		Namespace: bs.GetNamespace(),
		Name:      bs.GetName(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get backup storage: %w", err)
	}

	if bs.Spec.S3 != nil {
		if current.Spec.S3 == nil {
			current.Spec.S3 = &backupv1alpha1.BackupStorageS3Spec{}
		}
		s3 := bs.Spec.S3
		if s3.Bucket != "" {
			current.Spec.S3.Bucket = s3.Bucket
		}
		if s3.Region != "" {
			current.Spec.S3.Region = s3.Region
		}
		if s3.EndpointURL != "" {
			current.Spec.S3.EndpointURL = s3.EndpointURL
		}
		if s3.CredentialsSecretName != "" {
			current.Spec.S3.CredentialsSecretName = s3.CredentialsSecretName
		}
		if s3.VerifyTLS != nil {
			current.Spec.S3.VerifyTLS = s3.VerifyTLS
		}
		if s3.ForcePathStyle != nil {
			current.Spec.S3.ForcePathStyle = s3.ForcePathStyle
		}
		// Credential fields: merge into current so applyS3Credentials sees them.
		current.Spec.S3.AccessKeyID = s3.AccessKeyID
		current.Spec.S3.SecretAccessKey = s3.SecretAccessKey
	}

	if err := h.applyS3Credentials(ctx, current); err != nil {
		return nil, fmt.Errorf("failed to apply S3 credentials: %w", err)
	}
	return h.kubeConnector.UpdateBackupStorage(ctx, current)
}

// DeleteBackupStorage deletes a backup storage.
func (h *k8sHandler) DeleteBackupStorage(ctx context.Context, cluster, namespace, name string) error {
	bs, err := h.kubeConnector.GetBackupStorage(ctx, types.NamespacedName{Namespace: namespace, Name: name})
	if ctrlclient.IgnoreNotFound(err) != nil {
		return fmt.Errorf("failed to get backup storage: %w", err)
	}

	if bs == nil {
		// nothing to delete
		return nil
	}

	if err := h.kubeConnector.DeleteBackupStorage(ctx, bs); ctrlclient.IgnoreNotFound(err) != nil {
		return fmt.Errorf("failed to delete backup storage: %w", err)
	}

	return nil
}

// applyS3Credentials creates or updates the credentials Secret when AccessKeyID
// and SecretAccessKey are set on the spec, then clears the write-only fields so
// they are never persisted on the BackupStorage CR.
// Returns an error if exactly one of the two credential fields is provided.
func (h *k8sHandler) applyS3Credentials(ctx context.Context, bs *backupv1alpha1.BackupStorage) error {
	if bs.Spec.S3 == nil {
		return nil
	}
	s3 := bs.Spec.S3
	switch {
	case s3.AccessKeyID != "" && s3.SecretAccessKey == "":
		return errors.New("secretAccessKey is not provided")
	case s3.AccessKeyID == "" && s3.SecretAccessKey != "":
		return errors.New("accessKeyID is not provided")
	case s3.AccessKeyID == "" && s3.SecretAccessKey == "":
		return nil // no credentials to update
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s3.CredentialsSecretName,
			Namespace: bs.GetNamespace(),
		},
		Type: corev1.SecretTypeOpaque,
		StringData: map[string]string{
			"AWS_ACCESS_KEY_ID":     s3.AccessKeyID,
			"AWS_SECRET_ACCESS_KEY": s3.SecretAccessKey,
		},
	}
	if _, err := h.kubeConnector.CreateSecret(ctx, secret); k8serrors.IsAlreadyExists(err) {
		existing, err := h.kubeConnector.GetSecret(ctx, types.NamespacedName{
			Name:      s3.CredentialsSecretName,
			Namespace: bs.GetNamespace(),
		})
		if err != nil {
			return fmt.Errorf("failed to get existing credentials secret: %w", err)
		}
		existing.StringData = secret.StringData
		if _, err := h.kubeConnector.UpdateSecret(ctx, existing); err != nil {
			return fmt.Errorf("failed to update credentials secret: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to create credentials secret: %w", err)
	}

	s3.AccessKeyID = ""
	s3.SecretAccessKey = ""
	return nil
}
