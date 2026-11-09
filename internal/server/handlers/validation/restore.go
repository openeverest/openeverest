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

package validation

import (
	"context"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	"github.com/openeverest/openeverest/v2/internal/backupadmission"
)

// GetRestore returns a specific restore by namespace and name.
func (h *validateHandler) GetRestore(ctx context.Context, cluster, namespace, name string) (*backupv1alpha1.Restore, error) {
	return h.next.GetRestore(ctx, cluster, namespace, name)
}

// CreateRestore rejects a Restore that cannot run, or that references an object that does not exist, before creating it.
func (h *validateHandler) CreateRestore(ctx context.Context, cluster string, restore *backupv1alpha1.Restore) (*backupv1alpha1.Restore, error) {
	d, err := backupadmission.Restore(ctx, h.kubeConnector, restore)
	if err := admitOperation(ctx, d, err); err != nil {
		return nil, err
	}
	return h.next.CreateRestore(ctx, cluster, restore)
}

// DeleteRestore deletes a restore by namespace and name.
func (h *validateHandler) DeleteRestore(ctx context.Context, cluster, namespace, name string) error {
	return h.next.DeleteRestore(ctx, cluster, namespace, name)
}
