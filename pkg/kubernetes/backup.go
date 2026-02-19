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

// Package kubernetes ...
package kubernetes

import (
	"context"

	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
)

// GetBackup returns backup that matches the criteria.
func (k *Kubernetes) GetBackup(ctx context.Context, key ctrlclient.ObjectKey) (*backupv1alpha1.Backup, error) {
	result := &backupv1alpha1.Backup{}
	if err := k.k8sClient.Get(ctx, key, result); err != nil {
		return nil, err
	}
	return result, nil
}

// DeleteBackup deletes backup that matches the criteria.
func (k *Kubernetes) DeleteBackup(ctx context.Context, obj *backupv1alpha1.Backup) error {
	return k.k8sClient.Delete(ctx, obj)
}

// CreateBackup creates backup.
func (k *Kubernetes) CreateBackup(ctx context.Context, backup *backupv1alpha1.Backup) (*backupv1alpha1.Backup, error) {
	if err := k.k8sClient.Create(ctx, backup); err != nil {
		return nil, err
	}
	return backup, nil
}
