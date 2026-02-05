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

// GetDatabaseClusterRestore returns database cluster restore that matches the criteria.
func (k *Kubernetes) GetDatabaseClusterRestore(ctx context.Context, key ctrlclient.ObjectKey) (*everestv1alpha1.DatabaseClusterRestore, error) {
	result := &everestv1alpha1.DatabaseClusterRestore{}
	if err := k.k8sClient.Get(ctx, key, result); err != nil {
		return nil, err
	}
	return result, nil
}

// ListDatabaseClusterRestores returns database cluster restores that match the criteria.
// This method returns a list of full objects (meta and spec).
func (k *Kubernetes) ListDatabaseClusterRestores(ctx context.Context, opts ...ctrlclient.ListOption) (*everestv1alpha1.DatabaseClusterRestoreList, error) {
	result := &everestv1alpha1.DatabaseClusterRestoreList{}
	if err := k.k8sClient.List(ctx, result, opts...); err != nil {
		return nil, err
	}
	return result, nil
}

// UpdateDatabaseClusterRestore updates database cluster restore.
func (k *Kubernetes) UpdateDatabaseClusterRestore(ctx context.Context, restore *everestv1alpha1.DatabaseClusterRestore) (*everestv1alpha1.DatabaseClusterRestore, error) {
	if err := k.k8sClient.Update(ctx, restore); err != nil {
		return nil, err
	}
	return restore, nil
}

// DeleteDatabaseClusterRestore deletes database cluster restore that matches the criteria.
func (k *Kubernetes) DeleteDatabaseClusterRestore(ctx context.Context, obj *everestv1alpha1.DatabaseClusterRestore) error {
	return k.k8sClient.Delete(ctx, obj)
}

// CreateDatabaseClusterRestore creates database cluster restore.
func (k *Kubernetes) CreateDatabaseClusterRestore(ctx context.Context, restore *everestv1alpha1.DatabaseClusterRestore) (*everestv1alpha1.DatabaseClusterRestore, error) {
	if err := k.k8sClient.Create(ctx, restore); err != nil {
		return nil, err
	}
	return restore, nil
}
