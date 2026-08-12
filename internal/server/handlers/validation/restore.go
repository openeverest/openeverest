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
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
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

// validateRestoreRefs rejects restores whose source cannot be resolved, or is
// incompatible with the target Instance:
//
//   - the target Instance must exist;
//   - type=Backup: the source Backup must exist and have Succeeded;
//   - type=PointInTime: the Instance owning the stream must have a backup class
//     configured, and must be archiving to the named storage;
//   - the resolved read class must support the target Instance's provider, and
//     must advertise PITR support when point-in-time recovery is requested.
//
// The read class describes how the data was written and therefore how it must
// be read. For type=Backup it comes from the source Backup; for
// type=PointInTime it comes from the Instance owning the stream, which is not
// necessarily the Instance being restored into.
func (h *validateHandler) validateRestoreRefs(ctx context.Context, restore *backupv1alpha1.Restore) error {
	// The data source is validated first so a bad source is reported ahead of
	// anything about the target.
	var (
		bc  *backupv1alpha1.BackupClass
		err error
	)
	switch restore.Spec.DataSource.Type {
	case backupv1alpha1.DataSourceTypeBackup:
		bc, err = h.resolveClassFromBackup(ctx, restore)
	case backupv1alpha1.DataSourceTypePointInTime:
		bc, err = h.resolveClassFromStream(ctx, restore)
	default:
		return fmt.Errorf("%w: unsupported dataSource type '%s'",
			controller.ErrInvalidReference, restore.Spec.DataSource.Type)
	}
	if err != nil {
		return err
	}

	target, err := h.getInstanceRef(ctx, restore.GetNamespace(), restore.Spec.InstanceRef.Name)
	if err != nil {
		return err
	}

	if err := controller.ValidateClassSupportsProvider(bc, target.Spec.ProviderRef.Name); err != nil {
		return err
	}
	return controller.ValidateRestorePITR(restore, bc)
}

// resolveClassFromBackup validates the source Backup of a type=Backup restore
// and returns the class it was written with.
func (h *validateHandler) resolveClassFromBackup(
	ctx context.Context, restore *backupv1alpha1.Restore,
) (*backupv1alpha1.BackupClass, error) {
	ref := restore.Spec.DataSource.Backup
	if ref == nil {
		return nil, fmt.Errorf("%w: spec.dataSource.backup is required when type is 'Backup'",
			controller.ErrInvalidReference)
	}

	backup, err := h.kubeConnector.GetBackup(ctx, ctrlclient.ObjectKey{
		Namespace: restore.GetNamespace(),
		Name:      ref.BackupRef.Name,
	})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, fmt.Errorf("%w: '%s' in namespace '%s'",
				controller.ErrBackupNotFound, ref.BackupRef.Name, restore.GetNamespace())
		}
		return nil, fmt.Errorf("failed to get backup '%s': %w", ref.BackupRef.Name, err)
	}
	if err := controller.ValidateBackupSucceeded(backup); err != nil {
		return nil, err
	}

	return h.getBackupClassRef(ctx, backup.Spec.ClassRef.Name)
}

// resolveClassFromStream validates the stream a type=PointInTime restore reads
// from and returns the class that Instance writes with.
func (h *validateHandler) resolveClassFromStream(
	ctx context.Context, restore *backupv1alpha1.Restore,
) (*backupv1alpha1.BackupClass, error) {
	pitr := restore.Spec.DataSource.PointInTime
	if pitr == nil {
		return nil, fmt.Errorf("%w: spec.dataSource.pointInTime is required when type is 'PointInTime'",
			controller.ErrInvalidReference)
	}
	instanceName := controller.RestoreStreamInstanceName(restore)

	instance, err := h.getInstanceRef(ctx, restore.GetNamespace(), instanceName)
	if err != nil {
		return nil, err
	}

	if instance.Spec.Backup == nil || instance.Spec.Backup.ClassRef.Name == "" {
		return nil, fmt.Errorf(
			"%w: instance '%s' has no backup class configured, so it has no stream to recover from",
			controller.ErrInvalidReference, instanceName,
		)
	}

	// The named storage must actually carry a stream on that Instance.
	// Rejecting here turns a wrong-storage restore into an actionable error at
	// request time rather than a reconcile-time failure.
	if err := controller.ValidatePITRStorage(pitr, instance); err != nil {
		return nil, err
	}

	return h.getBackupClassRef(ctx, instance.Spec.Backup.ClassRef.Name)
}

// getInstanceRef fetches an Instance, mapping not-found onto the reference
// sentinel so the API layer classifies it as a client error.
func (h *validateHandler) getInstanceRef(
	ctx context.Context, namespace, name string,
) (*corev1alpha1.Instance, error) {
	instance, err := h.kubeConnector.GetInstance(ctx, ctrlclient.ObjectKey{
		Namespace: namespace,
		Name:      name,
	})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, fmt.Errorf("%w: instance '%s' does not exist", controller.ErrInstanceNotFound, name)
		}
		return nil, fmt.Errorf("failed to get instance '%s': %w", name, err)
	}
	return instance, nil
}

// getBackupClassRef fetches a BackupClass, mapping not-found onto the reference
// sentinel.
func (h *validateHandler) getBackupClassRef(
	ctx context.Context, name string,
) (*backupv1alpha1.BackupClass, error) {
	bc, err := h.kubeConnector.GetBackupClass(ctx, ctrlclient.ObjectKey{Name: name})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, fmt.Errorf("%w: '%s'", controller.ErrBackupClassNotFound, name)
		}
		return nil, fmt.Errorf("failed to get backup class '%s': %w", name, err)
	}
	return bc, nil
}

// DeleteRestore deletes a restore by namespace and name.
func (h *validateHandler) DeleteRestore(ctx context.Context, namespace, name string) error {
	return h.next.DeleteRestore(ctx, namespace, name)
}
