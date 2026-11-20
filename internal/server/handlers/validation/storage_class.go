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

	storagev1 "k8s.io/api/storage/v1"
)

// ListStorageClasses proxies the request to the next handler.
func (h *validateHandler) ListStorageClasses(ctx context.Context, cluster string) (*storagev1.StorageClassList, error) {
	return h.next.ListStorageClasses(ctx, cluster)
}

// GetStorageClass proxies the request to the next handler.
func (h *validateHandler) GetStorageClass(ctx context.Context, cluster, name string) (*storagev1.StorageClass, error) {
	return h.next.GetStorageClass(ctx, cluster, name)
}
