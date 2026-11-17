// everest
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

package kubernetes

import (
	"context"

	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
)

// GetBackupImport returns backup import that matches the criteria.
func (k *Kubernetes) GetBackupImport(ctx context.Context, key ctrlclient.ObjectKey) (*backupv1alpha1.BackupImport, error) {
	result := &backupv1alpha1.BackupImport{}
	if err := k.k8sClient.Get(ctx, key, result); err != nil {
		return nil, err
	}
	return result, nil
}

// ListBackupImports returns a list of backup imports in a given namespace.
func (k *Kubernetes) ListBackupImports(ctx context.Context, opts ...ctrlclient.ListOption) (*backupv1alpha1.BackupImportList, error) {
	result := &backupv1alpha1.BackupImportList{}
	if err := k.k8sClient.List(ctx, result, opts...); err != nil {
		return nil, err
	}
	return result, nil
}

// CreateBackupImport creates backup import.
func (k *Kubernetes) CreateBackupImport(ctx context.Context, backupImport *backupv1alpha1.BackupImport) (*backupv1alpha1.BackupImport, error) {
	if err := k.k8sClient.Create(ctx, backupImport); err != nil {
		return nil, err
	}
	return backupImport, nil
}

// DeleteBackupImport deletes backup import that matches the criteria.
func (k *Kubernetes) DeleteBackupImport(ctx context.Context, obj *backupv1alpha1.BackupImport) error {
	return k.k8sClient.Delete(ctx, obj)
}
