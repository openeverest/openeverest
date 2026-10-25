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
//
//nolint:dupl // per-resource RBAC filters are intentionally similar
package rbac

import (
	"context"
	"errors"
	"fmt"

	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

// ListMonitoringClasses returns monitoring classes filtered by RBAC permissions.
func (h *rbacHandler) ListMonitoringClasses(ctx context.Context, cluster string) (*monitoringv1alpha1.MonitoringClassList, error) {
	list, err := h.next.ListMonitoringClasses(ctx, cluster)
	if err != nil {
		return nil, fmt.Errorf("ListMonitoringClasses failed: %w", err)
	}
	filtered := make([]monitoringv1alpha1.MonitoringClass, 0, len(list.Items))
	for _, class := range list.Items {
		object := rbac.ClusterObjectName(cluster, class.GetName())
		if err := h.enforce(ctx, rbac.ResourceMonitoringClasses, rbac.ActionRead, object); errors.Is(err, ErrInsufficientPermissions) {
			continue
		} else if err != nil {
			return nil, fmt.Errorf("enforce failed: %w", err)
		}
		filtered = append(filtered, class)
	}
	list.Items = filtered
	return list, nil
}

// GetMonitoringClass returns a monitoring class, gated by RBAC.
func (h *rbacHandler) GetMonitoringClass(ctx context.Context, cluster, name string) (*monitoringv1alpha1.MonitoringClass, error) {
	object := rbac.ClusterObjectName(cluster, name)
	if err := h.enforce(ctx, rbac.ResourceMonitoringClasses, rbac.ActionRead, object); err != nil {
		return nil, err
	}
	return h.next.GetMonitoringClass(ctx, cluster, name)
}
