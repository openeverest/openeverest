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

package validation

import (
	"context"

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
)

// CreateLoadBalancerConfig creates a new load balancer config.
func (h *validateHandler) CreateLoadBalancerConfig(ctx context.Context, lbc *everestv1alpha1.LoadBalancerConfig) (*everestv1alpha1.LoadBalancerConfig, error) {
	return h.next.CreateLoadBalancerConfig(ctx, lbc)
}

// UpdateLoadBalancerConfig updates an existing load balancer config.
func (h *validateHandler) UpdateLoadBalancerConfig(ctx context.Context, lbc *everestv1alpha1.LoadBalancerConfig) (*everestv1alpha1.LoadBalancerConfig, error) {
	return h.next.UpdateLoadBalancerConfig(ctx, lbc)
}

// ListLoadBalancerConfigs lists all load balancer configs.
func (h *validateHandler) ListLoadBalancerConfigs(ctx context.Context) (*everestv1alpha1.LoadBalancerConfigList, error) {
	return h.next.ListLoadBalancerConfigs(ctx)
}

// DeleteLoadBalancerConfig deletes a load balancer config.
func (h *validateHandler) DeleteLoadBalancerConfig(ctx context.Context, name string) error {
	return h.next.DeleteLoadBalancerConfig(ctx, name)
}

// GetLoadBalancerConfig retrieves a load balancer config by name.
func (h *validateHandler) GetLoadBalancerConfig(ctx context.Context, name string) (*everestv1alpha1.LoadBalancerConfig, error) {
	return h.next.GetLoadBalancerConfig(ctx, name)
}
