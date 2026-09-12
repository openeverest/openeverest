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

// ListBackupImports proxies the request to the next handler.
func (h *validateHandler) ListBackupImports(ctx context.Context, cluster, namespace string) (*backupv1alpha1.BackupImportList, error) {
	return h.next.ListBackupImports(ctx, cluster, namespace)
}

// GetBackupImport proxies the request to the next handler.
func (h *validateHandler) GetBackupImport(ctx context.Context, cluster, namespace, name string) (*backupv1alpha1.BackupImport, error) {
	return h.next.GetBackupImport(ctx, cluster, namespace, name)
}

// CreateBackupImport validates the BackupImport's referenced resources before creating it.
func (h *validateHandler) CreateBackupImport(ctx context.Context, cluster string, backupImport *backupv1alpha1.BackupImport) (*backupv1alpha1.BackupImport, error) {
	if err := h.validateBackupImportRefs(ctx, backupImport); err != nil {
		if isValidationError(err) {
			return nil, errors.Join(ErrInvalidRequest, err)
		}
		return nil, err
	}
	return h.next.CreateBackupImport(ctx, cluster, backupImport)
}

// validateBackupImportRefs rejects backup imports whose classRef or storageRef
// do not point to existing resources.
func (h *validateHandler) validateBackupImportRefs(ctx context.Context, backupImport *backupv1alpha1.BackupImport) error {
	if _, err := h.kubeConnector.GetBackupClass(ctx, ctrlclient.ObjectKey{
		Name: backupImport.Spec.ClassRef.Name,
	}); err != nil {
		if k8serrors.IsNotFound(err) {
			return fmt.Errorf(
				"%w: '%s'",
				controller.ErrBackupClassNotFound,
				backupImport.Spec.ClassRef.Name,
			)
		}
		return fmt.Errorf("failed to get backup class '%s': %w", backupImport.Spec.ClassRef.Name, err)
	}

	if _, err := h.kubeConnector.GetBackupStorage(ctx, ctrlclient.ObjectKey{
		Namespace: backupImport.GetNamespace(),
		Name:      backupImport.Spec.StorageRef.Name,
	}); err != nil {
		if k8serrors.IsNotFound(err) {
			return fmt.Errorf(
				"%w: backup storage '%s' does not exist",
				controller.ErrBackupStorageNotFound,
				backupImport.Spec.StorageRef.Name,
			)
		}
		return fmt.Errorf("failed to get backup storage '%s': %w", backupImport.Spec.StorageRef.Name, err)
	}

	return nil
}

// DeleteBackupImport proxies the request to the next handler.
func (h *validateHandler) DeleteBackupImport(ctx context.Context, cluster, namespace, name string) error {
	return h.next.DeleteBackupImport(ctx, cluster, namespace, name)
}
