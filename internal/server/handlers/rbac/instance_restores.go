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

// Package rbac provides the RBAC handler.
package rbac

import (
	"context"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
)

// ListInstanceRestores proxies the request to the next handler.
func (h *rbacHandler) ListInstanceRestores(ctx context.Context, namespace, instanceName string) (*backupv1alpha1.RestoreList, error) {
	// Add RBAC checks here if needed in the future.
	return h.next.ListInstanceRestores(ctx, namespace, instanceName)
}
