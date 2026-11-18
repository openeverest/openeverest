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

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	"github.com/openeverest/openeverest/v2/internal/backupadmission"
	api "github.com/openeverest/openeverest/v2/internal/server/api"
)

// ListBackups proxies the request to the next handler.
func (h *validateHandler) ListBackups(ctx context.Context, cluster, namespace string, params *api.ListBackupsParams) (*backupv1alpha1.BackupList, error) {
	return h.next.ListBackups(ctx, cluster, namespace, params)
}

// GetBackup proxies the request to the next handler.
func (h *validateHandler) GetBackup(ctx context.Context, cluster, namespace, name string) (*backupv1alpha1.Backup, error) {
	return h.next.GetBackup(ctx, cluster, namespace, name)
}

// CreateBackup rejects a Backup that cannot run, or that references an object that does not exist, before creating it.
func (h *validateHandler) CreateBackup(ctx context.Context, cluster string, backup *backupv1alpha1.Backup) (*backupv1alpha1.Backup, error) {
	d, err := backupadmission.Backup(ctx, h.kubeConnector, backup)
	if err := admitOperation(ctx, d, err); err != nil {
		return nil, err
	}
	return h.next.CreateBackup(ctx, cluster, backup)
}

// DeleteBackup proxies the request to the next handler.
func (h *validateHandler) DeleteBackup(ctx context.Context, cluster, namespace, name string, params *api.DeleteBackupParams) error {
	return h.next.DeleteBackup(ctx, cluster, namespace, name, params)
}
