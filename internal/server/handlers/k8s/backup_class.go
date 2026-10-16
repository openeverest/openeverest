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
	"slices"

	"k8s.io/apimachinery/pkg/types"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
)

// ListBackupClasses returns list of backup classes.
func (h *k8sHandler) ListBackupClasses(ctx context.Context, _ string, provider, executionMode *string) (*backupv1alpha1.BackupClassList, error) {
	list, err := h.kubeConnector.ListBackupClasses(ctx)
	if err != nil || provider == nil && executionMode == nil {
		return list, err
	}

	list.Items = filterBackupClasses(list.Items, provider, executionMode)
	return list, nil
}

func filterBackupClasses(
	backupClasses []backupv1alpha1.BackupClass,
	provider, executionMode *string,
) []backupv1alpha1.BackupClass {
	filtered := make([]backupv1alpha1.BackupClass, 0, len(backupClasses))
	for _, backupClass := range backupClasses {
		if executionMode != nil && string(backupClass.Spec.ExecutionMode) != *executionMode {
			continue
		}
		if provider != nil && len(backupClass.Spec.SupportedProviders) > 0 &&
			!slices.Contains(backupClass.Spec.SupportedProviders, *provider) {
			continue
		}
		filtered = append(filtered, backupClass)
	}
	return filtered
}

// GetBackupClass returns backup class that matches the criteria.
func (h *k8sHandler) GetBackupClass(ctx context.Context, cluster, name string) (*backupv1alpha1.BackupClass, error) {
	return h.kubeConnector.GetBackupClass(ctx, types.NamespacedName{Name: name})
}
