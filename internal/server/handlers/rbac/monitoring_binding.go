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

	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
	api "github.com/openeverest/openeverest/v2/internal/server/api"
	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

// ListMonitoringBindings returns only the monitoring bindings the caller may
// read, see enforceReadMonitoringBinding.
func (h *rbacHandler) ListMonitoringBindings(ctx context.Context, cluster, namespace string, params *api.ListMonitoringBindingsParams) (*monitoringv1alpha1.MonitoringBindingList, error) {
	list, err := h.next.ListMonitoringBindings(ctx, cluster, namespace, params)
	if err != nil {
		return nil, fmt.Errorf("ListMonitoringBindings failed: %w", err)
	}
	filtered := make([]monitoringv1alpha1.MonitoringBinding, 0, len(list.Items))
	for _, binding := range list.Items {
		if err := h.enforceReadMonitoringBinding(ctx, cluster, &binding); errors.Is(err, ErrInsufficientPermissions) {
			continue
		} else if err != nil {
			return nil, fmt.Errorf("enforce failed: %w", err)
		}
		filtered = append(filtered, binding)
	}
	list.Items = filtered
	return list, nil
}

// GetMonitoringBinding returns the monitoring binding if the caller may read
// it, see enforceReadMonitoringBinding.
//
// Not found and denied both return ErrInsufficientPermissions to prevent name
// probing.
func (h *rbacHandler) GetMonitoringBinding(ctx context.Context, cluster, namespace, name string) (*monitoringv1alpha1.MonitoringBinding, error) {
	binding, err := h.next.GetMonitoringBinding(ctx, cluster, namespace, name)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return nil, ErrInsufficientPermissions
		}
		return nil, fmt.Errorf("GetMonitoringBinding failed: %w", err)
	}
	if err := h.enforceReadMonitoringBinding(ctx, cluster, binding); err != nil {
		return nil, err
	}
	return binding, nil
}

// enforceReadMonitoringBinding returns an error unless the caller may read
// monitoring-bindings and instances on the binding's Instance. Bindings are
// keyed by their Instance, like backups, because their names are generated;
// the instances grant keeps them from revealing Instances the caller cannot see.
func (h *rbacHandler) enforceReadMonitoringBinding(ctx context.Context, cluster string, binding *monitoringv1alpha1.MonitoringBinding) error {
	object := rbac.ClusterNamespacedObjectName(cluster, binding.GetNamespace(), binding.Spec.InstanceRef.Name)
	if err := h.enforce(ctx, rbac.ResourceMonitoringBindings, rbac.ActionRead, object); err != nil {
		return err
	}
	return h.enforce(ctx, rbac.ResourceInstances, rbac.ActionRead, object)
}
