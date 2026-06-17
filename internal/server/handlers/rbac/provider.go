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
	"errors"
	"fmt"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

// ListProviders returns providers filtered by RBAC permissions.
func (h *rbacHandler) ListProviders(ctx context.Context, cluster string) (*corev1alpha1.ProviderList, error) {
	list, err := h.next.ListProviders(ctx, cluster)
	if err != nil {
		return nil, fmt.Errorf("ListProviders failed: %w", err)
	}
	filtered := make([]corev1alpha1.Provider, 0, len(list.Items))
	for _, p := range list.Items {
		object := rbac.ClusterObjectName(cluster, p.GetName())
		if err := h.enforce(ctx, rbac.ResourceProviders, rbac.ActionRead, object); errors.Is(err, ErrInsufficientPermissions) {
			continue
		} else if err != nil {
			return nil, fmt.Errorf("enforce failed: %w", err)
		}
		filtered = append(filtered, p)
	}
	list.Items = filtered
	return list, nil
}

// GetProvider returns a provider, gated by RBAC.
func (h *rbacHandler) GetProvider(ctx context.Context, cluster, name string) (*corev1alpha1.Provider, error) {
	object := rbac.ClusterObjectName(cluster, name)
	if err := h.enforce(ctx, rbac.ResourceProviders, rbac.ActionRead, object); err != nil {
		return nil, err
	}
	return h.next.GetProvider(ctx, cluster, name)
}
