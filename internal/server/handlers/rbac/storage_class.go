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
	"errors"
	"fmt"

	storagev1 "k8s.io/api/storage/v1"

	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

// ListStorageClasses returns the storage classes filtered by RBAC permissions.
func (h *rbacHandler) ListStorageClasses(ctx context.Context, cluster string) (*storagev1.StorageClassList, error) {
	list, err := h.next.ListStorageClasses(ctx, cluster)
	if err != nil {
		return nil, fmt.Errorf("ListStorageClasses failed: %w", err)
	}
	filtered := make([]storagev1.StorageClass, 0, len(list.Items))
	for _, class := range list.Items {
		object := rbac.ClusterObjectName(cluster, class.GetName())
		if err := h.enforce(ctx, rbac.ResourceStorageClasses, rbac.ActionRead, object); errors.Is(err, ErrInsufficientPermissions) {
			continue
		} else if err != nil {
			return nil, fmt.Errorf("enforce failed: %w", err)
		}
		filtered = append(filtered, class)
	}
	list.Items = filtered
	return list, nil
}

// GetStorageClass returns a storage class, gated by RBAC.
func (h *rbacHandler) GetStorageClass(ctx context.Context, cluster, name string) (*storagev1.StorageClass, error) {
	if err := h.enforce(ctx, rbac.ResourceStorageClasses, rbac.ActionRead, rbac.ClusterObjectName(cluster, name)); err != nil {
		return nil, err
	}
	return h.next.GetStorageClass(ctx, cluster, name)
}
