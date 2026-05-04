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

package rbac

import (
	"context"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
)

// ListBackupStorages proxies the request to the next handler.
func (h *rbacHandler) ListBackupStorages(ctx context.Context, namespace string) (*backupv1alpha1.BackupStorageList, error) {
	// Add RBAC checks here if needed in the future
	return h.next.ListBackupStorages(ctx, namespace)
}

// GetBackupStorage proxies the request to the next handler.
func (h *rbacHandler) GetBackupStorage(ctx context.Context, namespace, name string) (*backupv1alpha1.BackupStorage, error) {
	// Add RBAC checks here if needed in the future
	return h.next.GetBackupStorage(ctx, namespace, name)
}

// CreateBackupStorage proxies the request to the next handler.
func (h *rbacHandler) CreateBackupStorage(ctx context.Context, bs *backupv1alpha1.BackupStorage) (*backupv1alpha1.BackupStorage, error) {
	// Add RBAC checks here if needed in the future
	return h.next.CreateBackupStorage(ctx, bs)
}

// UpdateBackupStorage proxies the request to the next handler.
func (h *rbacHandler) UpdateBackupStorage(ctx context.Context, bs *backupv1alpha1.BackupStorage) (*backupv1alpha1.BackupStorage, error) {
	// Add RBAC checks here if needed in the future
	return h.next.UpdateBackupStorage(ctx, bs)
}

// DeleteBackupStorage proxies the request to the next handler.
func (h *rbacHandler) DeleteBackupStorage(ctx context.Context, namespace, name string) error {
	// Add RBAC checks here if needed in the future
	return h.next.DeleteBackupStorage(ctx, namespace, name)
}
