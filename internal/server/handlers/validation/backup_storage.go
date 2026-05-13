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

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
)

// ListBackupStorages proxies the request to the next handler.
func (h *validateHandler) ListBackupStorages(ctx context.Context, cluster, namespace string) (*backupv1alpha1.BackupStorageList, error) {
	return h.next.ListBackupStorages(ctx, cluster, namespace)
}

// GetBackupStorage proxies the request to the next handler.
func (h *validateHandler) GetBackupStorage(ctx context.Context, cluster, namespace, name string) (*backupv1alpha1.BackupStorage, error) {
	return h.next.GetBackupStorage(ctx, cluster, namespace, name)
}

// CreateBackupStorage proxies the request to the next handler.
func (h *validateHandler) CreateBackupStorage(ctx context.Context, cluster string, bs *backupv1alpha1.BackupStorage) (*backupv1alpha1.BackupStorage, error) {
	// Add validation here if needed in the future
	return h.next.CreateBackupStorage(ctx, cluster, bs)
}

// UpdateBackupStorage proxies the request to the next handler.
func (h *validateHandler) UpdateBackupStorage(ctx context.Context, cluster string, bs *backupv1alpha1.BackupStorage) (*backupv1alpha1.BackupStorage, error) {
	// Add validation here if needed in the future
	return h.next.UpdateBackupStorage(ctx, cluster, bs)
}

// DeleteBackupStorage proxies the request to the next handler.
func (h *validateHandler) DeleteBackupStorage(ctx context.Context, cluster, namespace, name string) error {
	return h.next.DeleteBackupStorage(ctx, cluster, namespace, name)
}
