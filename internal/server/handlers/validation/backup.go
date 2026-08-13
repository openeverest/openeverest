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

// Package validation provides the validation handler.
package validation

import (
	"context"
	"errors"
	"fmt"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	api "github.com/openeverest/openeverest/v2/internal/server/api"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

// GetBackup proxies the request to the next handler.
func (h *validateHandler) GetBackup(ctx context.Context, cluster, namespace, name string) (*v1alpha1.Backup, error) {
	return h.next.GetBackup(ctx, cluster, namespace, name)
}

// CreateBackup validates the Backup's referenced resources before creating it.
func (h *validateHandler) CreateBackup(ctx context.Context, cluster string, backup *v1alpha1.Backup) (*v1alpha1.Backup, error) {
	if err := h.validateBackupRefs(ctx, backup); err != nil {
		if isValidationError(err) {
			return nil, errors.Join(ErrInvalidRequest, err)
		}
		return nil, err
	}
	return h.next.CreateBackup(ctx, cluster, backup)
}

// validateBackupRefs rejects backups whose instanceRef, storageRef, or
// classRef do not point to existing resources, or whose BackupClass does
// not support the target Instance's provider.
func (h *validateHandler) validateBackupRefs(ctx context.Context, backup *v1alpha1.Backup) error {
	instance, err := h.kubeConnector.GetInstance(ctx, ctrlclient.ObjectKey{
		Namespace: backup.GetNamespace(),
		Name:      backup.Spec.InstanceRef.Name,
	})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return fmt.Errorf("%w: instance '%s' does not exist", controller.ErrInstanceNotFound, backup.Spec.InstanceRef.Name)
		}
		return fmt.Errorf("failed to get instance '%s': %w", backup.Spec.InstanceRef.Name, err)
	}

	if _, err := h.kubeConnector.GetBackupStorage(ctx, ctrlclient.ObjectKey{
		Namespace: backup.GetNamespace(),
		Name:      backup.Spec.StorageRef.Name,
	}); err != nil {
		if k8serrors.IsNotFound(err) {
			return fmt.Errorf("%w: backup storage '%s' does not exist", controller.ErrBackupStorageNotFound, backup.Spec.StorageRef.Name)
		}
		return fmt.Errorf("failed to get backup storage '%s': %w", backup.Spec.StorageRef.Name, err)
	}

	bc, err := h.kubeConnector.GetBackupClass(ctx, ctrlclient.ObjectKey{Name: backup.Spec.ClassRef.Name})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return fmt.Errorf("%w: '%s'", controller.ErrBackupClassNotFound, backup.Spec.ClassRef.Name)
		}
		return fmt.Errorf("failed to get backup class '%s': %w", backup.Spec.ClassRef.Name, err)
	}

	return controller.ValidateClassSupportsProvider(bc, instance.Spec.ProviderRef.Name)
}

// DeleteBackup proxies the request to the next handler.
func (h *validateHandler) DeleteBackup(ctx context.Context, cluster, namespace, name string, params *api.DeleteBackupParams) error {
	return h.next.DeleteBackup(ctx, cluster, namespace, name, params)
}
