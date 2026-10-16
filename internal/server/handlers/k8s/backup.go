// everest
// Copyright (C) 2023 Percona LLC
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

	"github.com/AlekSi/pointer"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	api "github.com/openeverest/openeverest/v2/internal/server/api"
)

// ListBackups returns backups in the namespace that match all provided filters.
// Filters are evaluated server-side via the CRD's selectable fields.
func (h *k8sHandler) ListBackups(ctx context.Context, _, namespace string, params *api.ListBackupsParams) (*backupv1alpha1.BackupList, error) {
	fields := ctrlclient.MatchingFields{}

	if class := pointer.Get(params.BackupClass); class != "" {
		fields["spec.classRef.name"] = class
	}

	if storage := pointer.Get(params.BackupStorage); storage != "" {
		fields["spec.storageRef.name"] = storage
	}

	if origin := pointer.Get(params.BackupOrigin); origin != "" {
		fields["spec.origin.type"] = string(origin)
	}

	return h.kubeConnector.ListBackups(ctx, ctrlclient.InNamespace(namespace), fields)
}

// GetBackup returns backup that matches the criteria.
func (h *k8sHandler) GetBackup(ctx context.Context, cluster, namespace, name string) (*backupv1alpha1.Backup, error) {
	return h.kubeConnector.GetBackup(ctx, types.NamespacedName{Namespace: namespace, Name: name})
}

// CreateBackup creates a backup.
func (h *k8sHandler) CreateBackup(ctx context.Context, cluster string, backup *backupv1alpha1.Backup) (*backupv1alpha1.Backup, error) {
	stampActor(ctx, backup)
	return h.kubeConnector.CreateBackup(ctx, backup)
}

// DeleteBackup deletes a backup. If the deletionPolicy query parameter is
// provided, the backup's spec.deletionPolicy is patched before deletion so the
// controller sees the caller's intent (Delete vs Retain S3 data).
func (h *k8sHandler) DeleteBackup(ctx context.Context, cluster, namespace, name string, params *api.DeleteBackupParams) error {
	backup, err := h.kubeConnector.GetBackup(ctx, types.NamespacedName{Namespace: namespace, Name: name})
	if err != nil {
		return err
	}

	actorChanged := stampActor(ctx, backup)
	policyChanged := false
	if params != nil && params.DeletionPolicy != nil {
		policy := backupv1alpha1.BackupDeletionPolicy(*params.DeletionPolicy)
		if backup.Spec.DeletionPolicy != policy {
			backup.Spec.DeletionPolicy = policy
			policyChanged = true
		}
	}
	if actorChanged || policyChanged {
		if _, err := h.kubeConnector.UpdateBackup(ctx, backup); err != nil {
			return err
		}
	}

	return h.kubeConnector.DeleteBackup(ctx, backup)
}
