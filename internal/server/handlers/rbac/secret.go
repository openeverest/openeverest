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

	corev1 "k8s.io/api/core/v1"

	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

// CreateSecret creates a secret, gated by RBAC.
func (h *rbacHandler) CreateSecret(ctx context.Context, cluster, namespace string, secret *corev1.Secret) (*corev1.Secret, error) {
	object := rbac.ClusterNamespacedObjectName(cluster, namespace, secret.GetName())
	if err := h.enforce(ctx, rbac.ResourceSecrets, rbac.ActionCreate, object); err != nil {
		return nil, err
	}
	return h.next.CreateSecret(ctx, cluster, namespace, secret)
}

// ListSecrets returns secrets filtered by RBAC permissions.
func (h *rbacHandler) ListSecrets(ctx context.Context, cluster, namespace, provider, definition string) (*corev1.SecretList, error) {
	list, err := h.next.ListSecrets(ctx, cluster, namespace, provider, definition)
	if err != nil {
		return nil, fmt.Errorf("ListSecrets failed: %w", err)
	}
	filtered := make([]corev1.Secret, 0, len(list.Items))
	for _, s := range list.Items {
		object := rbac.ClusterNamespacedObjectName(cluster, s.GetNamespace(), s.GetName())
		if err := h.enforce(ctx, rbac.ResourceSecrets, rbac.ActionRead, object); errors.Is(err, ErrInsufficientPermissions) {
			continue
		} else if err != nil {
			return nil, fmt.Errorf("enforce failed: %w", err)
		}
		filtered = append(filtered, s)
	}
	list.Items = filtered
	return list, nil
}

// GetSecret returns a secret, gated by RBAC.
func (h *rbacHandler) GetSecret(ctx context.Context, cluster, namespace, name string) (*corev1.Secret, error) {
	object := rbac.ClusterNamespacedObjectName(cluster, namespace, name)
	if err := h.enforce(ctx, rbac.ResourceSecrets, rbac.ActionRead, object); err != nil {
		return nil, err
	}
	return h.next.GetSecret(ctx, cluster, namespace, name)
}

// DeleteSecret deletes a secret, gated by RBAC.
func (h *rbacHandler) DeleteSecret(ctx context.Context, cluster, namespace, name string) error {
	object := rbac.ClusterNamespacedObjectName(cluster, namespace, name)
	if err := h.enforce(ctx, rbac.ResourceSecrets, rbac.ActionDelete, object); err != nil {
		return err
	}
	return h.next.DeleteSecret(ctx, cluster, namespace, name)
}
