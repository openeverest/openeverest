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
// when the resolved BackupClass does not advertise PITR support. Restores that
// do not request point-in-time recovery pass through untouched.
//
// The class is the *read* class, resolved the same way as in the Restore
// controller: from the source Backup for type=Backup, and from the Instance
// owning the stream for type=PointInTime.
func (h *validateHandler) validateRestorePITR(ctx context.Context, restore *backupv1alpha1.Restore) error {
	if restore.Spec.DataSource.PointInTime == nil {
		return nil
	}

	// Point-in-time recovery resolves its class from the Instance owning the
	// stream; when no source Instance is named, that is the target Instance.
	instanceName := restore.Spec.InstanceRef.Name
	if src := restore.Spec.DataSource.PointInTime.Source; src.InstanceRef != nil {
		instanceName = src.InstanceRef.Name
	}

	instance, err := h.kubeConnector.GetInstance(ctx, ctrlclient.ObjectKey{
		Namespace: restore.GetNamespace(),
		Name:      instanceName,
	})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return fmt.Errorf("instance '%s' does not exist", instanceName)
		}
		return fmt.Errorf("failed to get instance '%s': %w", instanceName, err)
	}
	if instance.Spec.Backup == nil || instance.Spec.Backup.ClassRef.Name == "" {
		return fmt.Errorf(
			"instance '%s' has no backup class configured; point-in-time recovery is not available",
			instanceName,
		)
	}

	bc, err := h.kubeConnector.GetBackupClass(ctx,
		ctrlclient.ObjectKey{Name: instance.Spec.Backup.ClassRef.Name})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return fmt.Errorf("backup class '%s' does not exist", instance.Spec.Backup.ClassRef.Name)
		}
		return fmt.Errorf("failed to get backup class '%s': %w", instance.Spec.Backup.ClassRef.Name, err)
	}

	if err := controller.ValidateRestorePITR(restore, bc); err != nil {
		return err
	}

	// The named storage must actually carry a stream on that Instance.
	// Rejecting here turns a wrong-storage restore into an actionable error at
	// request time rather than a reconcile-time failure.
	if err := controller.ValidatePITRStorage(restore.Spec.DataSource.PointInTime, instance); err != nil {
		return err
	}
	return nil
}

// DeleteRestore deletes a restore by namespace and name.
func (h *validateHandler) DeleteRestore(ctx context.Context, namespace, name string) error {
	return h.next.DeleteRestore(ctx, namespace, name)
}
