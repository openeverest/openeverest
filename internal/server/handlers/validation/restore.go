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
	if err := h.validateRestoreRefs(ctx, restore); err != nil {
		if isValidationError(err) {
			return nil, errors.Join(ErrInvalidRequest, err)
		}
		return nil, err
	}
	return h.next.CreateRestore(ctx, restore)
}

// validateRestoreRefs rejects restores whose source Backup does not exist or
// has not reached the Succeeded state, whose target Instance does not exist,
// or whose Backup's BackupClass does not support the target Instance's
// provider. When the restore also requests point-in-time recovery, it
// additionally rejects the request if the BackupClass does not advertise
// PITR support.
func (h *validateHandler) validateRestoreRefs(ctx context.Context, restore *backupv1alpha1.Restore) error {
	ds := restore.Spec.DataSource
	if ds.Backup == nil {
		return nil
	}

	backup, err := h.kubeConnector.GetBackup(ctx, ctrlclient.ObjectKey{
		Namespace: restore.GetNamespace(),
		Name:      ds.Backup.BackupRef.Name,
	})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return fmt.Errorf("%w: '%s' in namespace '%s'", controller.ErrBackupNotFound, ds.Backup.BackupRef.Name, restore.GetNamespace())
		}
		return fmt.Errorf("failed to get backup '%s': %w", ds.Backup.BackupRef.Name, err)
	}
	if err := controller.ValidateBackupSucceeded(backup); err != nil {
		return err
	}

	instance, err := h.kubeConnector.GetInstance(ctx, ctrlclient.ObjectKey{
		Namespace: restore.GetNamespace(),
		Name:      restore.Spec.InstanceRef.Name,
	})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return fmt.Errorf("%w: instance '%s' does not exist", controller.ErrInstanceNotFound, restore.Spec.InstanceRef.Name)
		}
		return fmt.Errorf("failed to get instance '%s': %w", restore.Spec.InstanceRef.Name, err)
	}

	bc, err := h.kubeConnector.GetBackupClass(ctx, ctrlclient.ObjectKey{Name: backup.Spec.ClassRef.Name})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return fmt.Errorf("%w: '%s'", controller.ErrBackupClassNotFound, backup.Spec.ClassRef.Name)
		}
		return fmt.Errorf("failed to get backup class '%s': %w", backup.Spec.ClassRef.Name, err)
	}
	if err := controller.ValidateClassSupportsProvider(bc, instance.Spec.ProviderRef.Name); err != nil {
		return err
	}

	return controller.ValidateRestorePITR(restore, bc)
}

// DeleteRestore deletes a restore by namespace and name.
func (h *validateHandler) DeleteRestore(ctx context.Context, namespace, name string) error {
	return h.next.DeleteRestore(ctx, namespace, name)
}
