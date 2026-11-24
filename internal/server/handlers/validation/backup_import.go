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
	"github.com/openeverest/openeverest/v2/internal/backupadmission"
)

// ListBackupImports proxies the request to the next handler.
func (h *validateHandler) ListBackupImports(ctx context.Context, cluster, namespace string) (*backupv1alpha1.BackupImportList, error) {
	return h.next.ListBackupImports(ctx, cluster, namespace)
}

// GetBackupImport proxies the request to the next handler.
func (h *validateHandler) GetBackupImport(ctx context.Context, cluster, namespace, name string) (*backupv1alpha1.BackupImport, error) {
	return h.next.GetBackupImport(ctx, cluster, namespace, name)
}

// CreateBackupImport rejects a BackupImport that cannot run, or that references an object that does not exist, before
// creating it.
func (h *validateHandler) CreateBackupImport(ctx context.Context, cluster string, backupImport *backupv1alpha1.BackupImport) (*backupv1alpha1.BackupImport, error) {
	d, err := backupadmission.Import(ctx, h.kubeConnector, backupImport)
	if err := admitOperation(ctx, d, err); err != nil {
		return nil, err
	}
	return h.next.CreateBackupImport(ctx, cluster, backupImport)
}

// DeleteBackupImport proxies the request to the next handler.
func (h *validateHandler) DeleteBackupImport(ctx context.Context, cluster, namespace, name string) error {
	return h.next.DeleteBackupImport(ctx, cluster, namespace, name)
}
