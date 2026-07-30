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

package validation

import (
	"context"
	"errors"

	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	backupvalidation "github.com/openeverest/openeverest/v2/provider-runtime/controller/backup"
)

// GetRestore returns a specific restore by namespace and name.
func (h *validateHandler) GetRestore(ctx context.Context, namespace, name string) (*backupv1alpha1.Restore, error) {
	return h.next.GetRestore(ctx, namespace, name)
}

// CreateRestore creates a new restore.
func (h *validateHandler) CreateRestore(ctx context.Context, restore *backupv1alpha1.Restore) (*backupv1alpha1.Restore, error) {
	if err := h.validateRestoreBackupRef(ctx, restore); err != nil {
		return nil, errors.Join(ErrInvalidRequest, err)
	}
	return h.next.CreateRestore(ctx, restore)
}

// validateRestoreBackupRef rejects restores whose source Backup does not
// exist or has not reached the Succeeded state. When the restore also
// requests point-in-time recovery, it additionally rejects the request if
// the Backup's BackupClass does not advertise PITR support.
func (h *validateHandler) validateRestoreBackupRef(ctx context.Context, restore *backupv1alpha1.Restore) error {
	ds := restore.Spec.DataSource
	if ds.Backup == nil {
		return nil
	}

	backup, err := backupvalidation.ResolveSucceededBackup(ctx, ds.Backup.BackupRef.Name,
		func(ctx context.Context, name string) (*backupv1alpha1.Backup, error) {
			return h.kubeConnector.GetBackup(ctx, ctrlclient.ObjectKey{Namespace: restore.GetNamespace(), Name: name})
		})
	if err != nil {
		return err
	}

	if ds.Backup.PITR == nil {
		return nil
	}

	bc, err := backupvalidation.ResolveBackupClass(ctx, backup.Spec.ClassRef.Name,
		func(ctx context.Context, name string) (*backupv1alpha1.BackupClass, error) {
			return h.kubeConnector.GetBackupClass(ctx, ctrlclient.ObjectKey{Name: name})
		})
	if err != nil {
		return err
	}

	return backupvalidation.ValidateRestorePITR(restore, bc)
}

// DeleteRestore deletes a restore by namespace and name.
func (h *validateHandler) DeleteRestore(ctx context.Context, namespace, name string) error {
	return h.next.DeleteRestore(ctx, namespace, name)
}
