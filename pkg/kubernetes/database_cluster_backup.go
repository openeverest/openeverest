// everest
// Copyright (C) 2023 Percona LLC
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

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
)

// GetDatabaseClusterBackup returns database cluster backup that matches the criteria.
// This method returns a list of full objects (meta and spec).
func (k *Kubernetes) GetDatabaseClusterBackup(ctx context.Context, key ctrlclient.ObjectKey) (*everestv1alpha1.DatabaseClusterBackup, error) {
	result := &everestv1alpha1.DatabaseClusterBackup{}
	if err := k.k8sClient.Get(ctx, key, result); err != nil {
		return nil, err
	}
	return result, nil
}

// ListDatabaseClusterBackups returns database cluster backups that match the criteria.
// This method returns a list of full objects (meta and spec).
func (k *Kubernetes) ListDatabaseClusterBackups(ctx context.Context, opts ...ctrlclient.ListOption) (*everestv1alpha1.DatabaseClusterBackupList, error) {
	result := &everestv1alpha1.DatabaseClusterBackupList{}
	if err := k.k8sClient.List(ctx, result, opts...); err != nil {
		return nil, err
	}
	return result, nil
}

// UpdateDatabaseClusterBackup updates database cluster backup.
func (k *Kubernetes) UpdateDatabaseClusterBackup(ctx context.Context, backup *everestv1alpha1.DatabaseClusterBackup) (*everestv1alpha1.DatabaseClusterBackup, error) {
	if err := k.k8sClient.Update(ctx, backup); err != nil {
		return nil, err
	}
	return backup, nil
}

// DeleteDatabaseClusterBackup deletes database cluster backup that matches the criteria.
func (k *Kubernetes) DeleteDatabaseClusterBackup(ctx context.Context, obj *everestv1alpha1.DatabaseClusterBackup) error {
	return k.k8sClient.Delete(ctx, obj)
}

// CreateDatabaseClusterBackup creates database cluster backup.
func (k *Kubernetes) CreateDatabaseClusterBackup(ctx context.Context, backup *everestv1alpha1.DatabaseClusterBackup) (*everestv1alpha1.DatabaseClusterBackup, error) {
	if err := k.k8sClient.Create(ctx, backup); err != nil {
		return nil, err
	}
	return backup, nil
}
