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

// GetRestore returns a specific restore by namespaced name.
func (k *Kubernetes) GetRestore(ctx context.Context, key ctrlclient.ObjectKey) (*backupv1alpha1.Restore, error) {
	result := &backupv1alpha1.Restore{}
	if err := k.k8sClient.Get(ctx, key, result); err != nil {
		return nil, err
	}
	return result, nil
}

// CreateRestore creates a new restore.
func (k *Kubernetes) CreateRestore(ctx context.Context, restore *backupv1alpha1.Restore) (*backupv1alpha1.Restore, error) {
	if err := k.k8sClient.Create(ctx, restore); err != nil {
		return nil, err
	}
	return restore, nil
}

// DeleteRestore deletes a restore.
func (k *Kubernetes) DeleteRestore(ctx context.Context, obj *backupv1alpha1.Restore) error {
	return k.k8sClient.Delete(ctx, obj)
}
