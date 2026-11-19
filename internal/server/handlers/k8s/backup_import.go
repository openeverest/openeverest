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

package k8s

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
)

// ListBackupImports returns a list of backup imports in a namespace.
func (h *k8sHandler) ListBackupImports(ctx context.Context, cluster, namespace string) (*backupv1alpha1.BackupImportList, error) {
	kubeConnector, err := h.connector(ctx, cluster)
	if err != nil {
		return nil, err
	}
	return kubeConnector.ListBackupImports(ctx, ctrlclient.InNamespace(namespace))
}

// GetBackupImport returns a backup import by name and namespace.
func (h *k8sHandler) GetBackupImport(ctx context.Context, cluster, namespace, name string) (*backupv1alpha1.BackupImport, error) {
	kubeConnector, err := h.connector(ctx, cluster)
	if err != nil {
		return nil, err
	}
	return kubeConnector.GetBackupImport(ctx, types.NamespacedName{Namespace: namespace, Name: name})
}

// CreateBackupImport creates a backup import.
func (h *k8sHandler) CreateBackupImport(ctx context.Context, cluster string, backupImport *backupv1alpha1.BackupImport) (*backupv1alpha1.BackupImport, error) {
	stampActor(ctx, backupImport)
	kubeConnector, err := h.connector(ctx, cluster)
	if err != nil {
		return nil, err
	}
	return kubeConnector.CreateBackupImport(ctx, backupImport)
}

// DeleteBackupImport deletes a backup import by namespace and name.
func (h *k8sHandler) DeleteBackupImport(ctx context.Context, cluster, namespace, name string) error {
	kubeConnector, err := h.connector(ctx, cluster)
	if err != nil {
		return err
	}
	backupImport := &backupv1alpha1.BackupImport{}
	backupImport.Name = name
	backupImport.Namespace = namespace
	if err := kubeConnector.DeleteBackupImport(ctx, backupImport); ctrlclient.IgnoreNotFound(err) != nil {
		return fmt.Errorf("failed to delete backup import: %w", err)
	}

	return nil
}
