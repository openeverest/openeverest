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

package rbac

import (
	"context"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
)

// GetRestore returns a specific restore by namespace and name.
func (h *rbacHandler) GetRestore(ctx context.Context, namespace, name string) (*backupv1alpha1.Restore, error) {
	return h.next.GetRestore(ctx, namespace, name)
}

// CreateRestore creates a new restore.
func (h *rbacHandler) CreateRestore(ctx context.Context, restore *backupv1alpha1.Restore) (*backupv1alpha1.Restore, error) {
	return h.next.CreateRestore(ctx, restore)
}

// DeleteRestore deletes a restore by namespace and name.
func (h *rbacHandler) DeleteRestore(ctx context.Context, namespace, name string) error {
	return h.next.DeleteRestore(ctx, namespace, name)
}
