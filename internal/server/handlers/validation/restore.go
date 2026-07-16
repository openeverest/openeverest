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
	"fmt"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

// GetRestore returns a specific restore by namespace and name.
func (h *validateHandler) GetRestore(ctx context.Context, namespace, name string) (*backupv1alpha1.Restore, error) {
	return h.next.GetRestore(ctx, namespace, name)
}

// CreateRestore creates a new restore.
func (h *validateHandler) CreateRestore(ctx context.Context, restore *backupv1alpha1.Restore) (*backupv1alpha1.Restore, error) {
	if err := h.validateRestorePITR(ctx, restore); err != nil {
		return nil, errors.Join(ErrInvalidRequest, err)
	}
	return h.next.CreateRestore(ctx, restore)
}

// validateRestorePITR rejects restores that request point-in-time recovery
// when the BackupClass resolved via the source Backup does not advertise
// PITR support. Restores without PITR options pass through untouched.
func (h *validateHandler) validateRestorePITR(ctx context.Context, restore *backupv1alpha1.Restore) error {
	ds := restore.Spec.DataSource
	if ds.Backup == nil || ds.Backup.PITR == nil {
		return nil
	}

	backup, err := h.kubeConnector.GetBackup(ctx, ctrlclient.ObjectKey{
		Namespace: restore.GetNamespace(),
		Name:      ds.Backup.BackupName,
	})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return fmt.Errorf("backup '%s' does not exist", ds.Backup.BackupName)
		}
		return fmt.Errorf("failed to get backup '%s': %w", ds.Backup.BackupName, err)
	}

	bc, err := h.kubeConnector.GetBackupClass(ctx, ctrlclient.ObjectKey{Name: backup.Spec.BackupClassName})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return fmt.Errorf("backup class '%s' does not exist", backup.Spec.BackupClassName)
		}
		return fmt.Errorf("failed to get backup class '%s': %w", backup.Spec.BackupClassName, err)
	}

	return controller.ValidateRestorePITR(restore, bc)
}

// DeleteRestore deletes a restore by namespace and name.
func (h *validateHandler) DeleteRestore(ctx context.Context, namespace, name string) error {
	return h.next.DeleteRestore(ctx, namespace, name)
}
