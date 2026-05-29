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

// Package v1alpha1 contains admission webhooks for the backup v1alpha1 API group.
package v1alpha1

import (
	"context"
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
)

// SetupBackupStorageWebhookWithManager registers the mutating webhook for BackupStorage.
func SetupBackupStorageWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &backupv1alpha1.BackupStorage{}).
		WithDefaulter(&BackupStorageDefaulter{
			Client: mgr.GetClient(),
		}).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-backup-openeverest-io-v1alpha1-backupstorage,mutating=true,failurePolicy=fail,sideEffects=None,groups=backup.openeverest.io,resources=backupstorages,verbs=create;update,versions=v1alpha1,name=mbackupstorage-v1alpha1.kb.io,admissionReviewVersions=v1
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=create;update;get

// BackupStorageDefaulter handles mutating admission for BackupStorage resources.
type BackupStorageDefaulter struct {
	Client client.Client
}

// Default implements admission.CustomDefaulter.
// When AccessKeyID and SecretAccessKey are provided, it stores them in the
// Secret named by CredentialsSecretName and clears them from the object.
func (d *BackupStorageDefaulter) Default(ctx context.Context, bs *backupv1alpha1.BackupStorage) error {
	if bs.Spec.S3 == nil {
		return nil
	}
	return handleS3CredentialsSecret(ctx, d.Client, bs)
}

func handleS3CredentialsSecret(ctx context.Context, c client.Client, bs *backupv1alpha1.BackupStorage) error {
	s3 := bs.Spec.S3
	accessKeyID := s3.AccessKeyID
	secretAccessKey := s3.SecretAccessKey

	switch {
	case accessKeyID != "" && secretAccessKey == "":
		return errors.New("secretAccessKey is not provided")
	case accessKeyID == "" && secretAccessKey != "":
		return errors.New("accessKeyID is not provided")
	case accessKeyID == "" && secretAccessKey == "":
		return nil // nothing to do
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s3.CredentialsSecretName,
			Namespace: bs.GetNamespace(),
		},
	}
	if _, err := controllerutil.CreateOrUpdate(ctx, c, secret, func() error {
		secret.Type = corev1.SecretTypeOpaque
		secret.StringData = map[string]string{
			"AWS_ACCESS_KEY_ID":     accessKeyID,
			"AWS_SECRET_ACCESS_KEY": secretAccessKey,
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to create or update S3 credentials secret: %w", err)
	}

	// Clear the write-only fields so they are never persisted on the CR.
	s3.AccessKeyID = ""
	s3.SecretAccessKey = ""
	return nil
}
