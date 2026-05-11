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

// Package validation provides the validation handler.
package validation

import (
	"context"

	"github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
)

// GetBackup proxies the request to the next handler.
func (h *validateHandler) GetBackup(ctx context.Context, cluster, namespace, name string) (*v1alpha1.Backup, error) {
	return h.next.GetBackup(ctx, cluster, namespace, name)
}

// CreateBackup proxies the request to the next handler.
func (h *validateHandler) CreateBackup(ctx context.Context, cluster string, backup *v1alpha1.Backup) (*v1alpha1.Backup, error) {
	// Add validation here if needed in the future
	return h.next.CreateBackup(ctx, cluster, backup)
}

// DeleteBackup proxies the request to the next handler.
func (h *validateHandler) DeleteBackup(ctx context.Context, cluster, namespace, name string) error {
	return h.next.DeleteBackup(ctx, cluster, namespace, name)
}
