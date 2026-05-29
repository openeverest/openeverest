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
	"fmt"

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
	return h.kubeConnector.CreateBackupStorage(ctx, bs)
}

// UpdateBackupStorage updates a backup storage.
func (h *k8sHandler) UpdateBackupStorage(ctx context.Context, cluster string, bs *backupv1alpha1.BackupStorage) (*backupv1alpha1.BackupStorage, error) {
	return h.kubeConnector.UpdateBackupStorage(ctx, bs)
}

// PatchBackupStorage patches a backup storage by fetching the current state and
// merging only the non-zero fields from bs onto it before updating.
// Write-only credential fields (AccessKeyID, SecretAccessKey) are forwarded as-is
// so the mutating webhook can extract them into the credentials Secret.
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
		// Write-only fields: pass through to be consumed by the mutating webhook.
		current.Spec.S3.AccessKeyID = s3.AccessKeyID
		current.Spec.S3.SecretAccessKey = s3.SecretAccessKey
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
