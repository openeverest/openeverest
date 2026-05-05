// everest
// Copyright (C) 2025 Percona LLC
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
	"fmt"
	"slices"

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
	"github.com/percona/everest/pkg/rbac"
)

// CreateLoadBalancerConfig creates a new load balancer config.
func (h *rbacHandler) CreateLoadBalancerConfig(ctx context.Context, lbc *everestv1alpha1.LoadBalancerConfig) (*everestv1alpha1.LoadBalancerConfig, error) {
	if err := h.enforce(ctx, rbac.ResourceLoadBalancerConfigs, rbac.ActionCreate, rbac.ObjectName(lbc.GetName())); err != nil {
		return nil, err
	}
	return h.next.CreateLoadBalancerConfig(ctx, lbc)
}

// UpdateLoadBalancerConfig updates an existing load balancer config.
func (h *rbacHandler) UpdateLoadBalancerConfig(ctx context.Context, lbc *everestv1alpha1.LoadBalancerConfig) (*everestv1alpha1.LoadBalancerConfig, error) {
	if err := h.enforce(ctx, rbac.ResourceLoadBalancerConfigs, rbac.ActionUpdate, rbac.ObjectName(lbc.GetName())); err != nil {
		return nil, err
	}
	return h.next.UpdateLoadBalancerConfig(ctx, lbc)
}

// ListLoadBalancerConfigs lists all load balancer configs.
func (h *rbacHandler) ListLoadBalancerConfigs(ctx context.Context) (*everestv1alpha1.LoadBalancerConfigList, error) {
	lbcList, err := h.next.ListLoadBalancerConfigs(ctx)
	if err != nil {
		return nil, fmt.Errorf("ListLoadBalancerConfigs failed: %w", err)
	}
	// filter out LoadBalancerConfigs that the user does not have access to.
	lbcList.Items = slices.DeleteFunc(lbcList.Items, func(lbc everestv1alpha1.LoadBalancerConfig) bool {
		return h.enforce(ctx, rbac.ResourceLoadBalancerConfigs, rbac.ActionRead, rbac.ObjectName(lbc.GetName())) != nil
	})
	return lbcList, nil

	return h.next.ListLoadBalancerConfigs(ctx)
}

// DeleteLoadBalancerConfig deletes a load balancer config.
func (h *rbacHandler) DeleteLoadBalancerConfig(ctx context.Context, name string) error {
	if err := h.enforce(ctx, rbac.ResourceLoadBalancerConfigs, rbac.ActionDelete, rbac.ObjectName(name)); err != nil {
		return err
	}
	return h.next.DeleteLoadBalancerConfig(ctx, name)
}

// GetLoadBalancerConfig retrieves a load balancer config by name.
func (h *rbacHandler) GetLoadBalancerConfig(ctx context.Context, name string) (*everestv1alpha1.LoadBalancerConfig, error) {
	if err := h.enforce(ctx, rbac.ResourceLoadBalancerConfigs, rbac.ActionRead, rbac.ObjectName(name)); err != nil {
		return nil, err
	}
	return h.next.GetLoadBalancerConfig(ctx, name)
}
