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

// ListBackupClasses returns list of backup classes that match the criteria.
func (k *Kubernetes) ListBackupClasses(ctx context.Context, opts ...ctrlclient.ListOption) (*backupv1alpha1.BackupClassList, error) {
	result := &backupv1alpha1.BackupClassList{}
	if err := k.k8sClient.List(ctx, result, opts...); err != nil {
		return nil, err
	}
	return result, nil
}

// GetBackupClass returns backup class that matches the criteria.
func (k *Kubernetes) GetBackupClass(ctx context.Context, key ctrlclient.ObjectKey) (*backupv1alpha1.BackupClass, error) {
	result := &backupv1alpha1.BackupClass{}
	if err := k.k8sClient.Get(ctx, key, result); err != nil {
		return nil, err
	}
	return result, nil
}
