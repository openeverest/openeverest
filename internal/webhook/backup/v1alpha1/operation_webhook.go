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

// Package v1alpha1 holds the validating webhooks of the backup operations,
// Backup, Restore and BackupImport. They reject only what cannot run until an
// object it references changes, and admit the rest with a warning for each
// verdict or object the operation waits for; backupadmission decides.
package v1alpha1

import (
	"context"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	"github.com/openeverest/openeverest/v2/internal/backupadmission"
)

// +kubebuilder:rbac:groups=core.openeverest.io,resources=instances,verbs=get
// +kubebuilder:rbac:groups=backup.openeverest.io,resources=backups;backupclasses;backupstorages;backupstorageclasses,verbs=get

// +kubebuilder:webhook:path=/validate-backup-openeverest-io-v1alpha1-backup,mutating=false,failurePolicy=fail,sideEffects=None,groups=backup.openeverest.io,resources=backups,verbs=create,versions=v1alpha1,name=vbackup-v1alpha1.kb.io,admissionReviewVersions=v1,timeoutSeconds=5

// SetupBackupWebhookWithManager registers the webhook for Backup in the manager.
func SetupBackupWebhookWithManager(mgr ctrl.Manager) error {
	return setup(mgr, &backupv1alpha1.Backup{}, backupadmission.Backup)
}

// +kubebuilder:webhook:path=/validate-backup-openeverest-io-v1alpha1-restore,mutating=false,failurePolicy=fail,sideEffects=None,groups=backup.openeverest.io,resources=restores,verbs=create,versions=v1alpha1,name=vrestore-v1alpha1.kb.io,admissionReviewVersions=v1,timeoutSeconds=5

// SetupRestoreWebhookWithManager registers the webhook for Restore in the manager.
func SetupRestoreWebhookWithManager(mgr ctrl.Manager) error {
	return setup(mgr, &backupv1alpha1.Restore{}, backupadmission.Restore)
}

// +kubebuilder:webhook:path=/validate-backup-openeverest-io-v1alpha1-backupimport,mutating=false,failurePolicy=fail,sideEffects=None,groups=backup.openeverest.io,resources=backupimports,verbs=create,versions=v1alpha1,name=vbackupimport-v1alpha1.kb.io,admissionReviewVersions=v1,timeoutSeconds=5

// SetupBackupImportWebhookWithManager registers the webhook for BackupImport in the manager.
func SetupBackupImportWebhookWithManager(mgr ctrl.Manager) error {
	return setup(mgr, &backupv1alpha1.BackupImport{}, backupadmission.Import)
}

// Decide decides whether an operation of type T may be created.
type Decide[T client.Object] func(ctx context.Context, r backupadmission.Reader, obj T) (backupadmission.Decision, error)

func setup[T client.Object](mgr ctrl.Manager, obj T, decide Decide[T]) error {
	return ctrl.NewWebhookManagedBy(mgr, obj).
		WithValidator(&OperationValidator[T]{
			// Uncached, so a verdict published just before the operation is
			// created is the one it is judged by.
			Reader: backupadmission.ClientReader{Reader: mgr.GetAPIReader()},
			Decide: decide,
		}).
		Complete()
}

// OperationValidator validates the creation of a backup operation of type T.
type OperationValidator[T client.Object] struct {
	Reader backupadmission.Reader
	Decide Decide[T]
}

var _ admission.Validator[*backupv1alpha1.Backup] = &OperationValidator[*backupv1alpha1.Backup]{}

// ValidateCreate rejects an operation that cannot run until an object it
// references changes, and warns about what an admitted one waits for.
func (v *OperationValidator[T]) ValidateCreate(ctx context.Context, obj T) (admission.Warnings, error) {
	d, err := v.Decide(ctx, v.Reader, obj)
	if err != nil {
		return nil, apierrors.NewInternalError(err)
	}
	if d.Denied != nil {
		logf.FromContext(ctx).Info("Denied", "reason", d.Denied.Error())
	}
	return d.Warnings(), d.Denied
}

// ValidateUpdate admits every update: an operation's spec is judged once,
// when it is created.
func (v *OperationValidator[T]) ValidateUpdate(context.Context, T, T) (admission.Warnings, error) {
	return nil, nil
}

// ValidateDelete admits every deletion.
func (v *OperationValidator[T]) ValidateDelete(context.Context, T) (admission.Warnings, error) {
	return nil, nil
}
