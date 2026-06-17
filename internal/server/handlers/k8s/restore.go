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

// GetRestore returns a specific restore by namespace and name.
func (h *k8sHandler) GetRestore(ctx context.Context, namespace, name string) (*backupv1alpha1.Restore, error) {
	return h.kubeConnector.GetRestore(ctx, types.NamespacedName{Namespace: namespace, Name: name})
}

// CreateRestore creates a new restore.
func (h *k8sHandler) CreateRestore(ctx context.Context, restore *backupv1alpha1.Restore) (*backupv1alpha1.Restore, error) {
	return h.kubeConnector.CreateRestore(ctx, restore)
}

// DeleteRestore deletes a restore by namespace and name.
func (h *k8sHandler) DeleteRestore(ctx context.Context, namespace, name string) error {
	restore := &backupv1alpha1.Restore{}
	restore.Name = name
	restore.Namespace = namespace
	return h.kubeConnector.DeleteRestore(ctx, restore)
}
