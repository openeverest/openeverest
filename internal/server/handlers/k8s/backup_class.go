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

	"k8s.io/apimachinery/pkg/types"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
)

// ListBackupClasses returns the backup classes, only those that provider
// executes when it is set.
func (h *k8sHandler) ListBackupClasses(ctx context.Context, cluster, provider string) (*backupv1alpha1.BackupClassList, error) {
	kubeConnector, err := h.connector(ctx, cluster)
	if err != nil {
		return nil, err
	}
	list, err := kubeConnector.ListBackupClasses(ctx)
	if err != nil || provider == "" {
		return list, err
	}
	filtered := make([]backupv1alpha1.BackupClass, 0, len(list.Items))
	for _, bc := range list.Items {
		if bc.Spec.ExecutorProvider() == provider {
			filtered = append(filtered, bc)
		}
	}
	list.Items = filtered
	return list, nil
}

// GetBackupClass returns backup class that matches the criteria.
func (h *k8sHandler) GetBackupClass(ctx context.Context, cluster, name string) (*backupv1alpha1.BackupClass, error) {
	kubeConnector, err := h.connector(ctx, cluster)
	if err != nil {
		return nil, err
	}
	return kubeConnector.GetBackupClass(ctx, types.NamespacedName{Name: name})
}
