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

package kubernetes

import (
	"context"

	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
)

// ListLoadBalancerConfigs returns a list of load balancer config that matches the criteria.
// This method returns a list of full objects (meta and spec).
func (k *Kubernetes) ListLoadBalancerConfigs(ctx context.Context, opts ...ctrlclient.ListOption) (*everestv1alpha1.LoadBalancerConfigList, error) {
	result := &everestv1alpha1.LoadBalancerConfigList{}
	if err := k.k8sClient.List(ctx, result, opts...); err != nil {
		return nil, err
	}
	return result, nil
}

// GetLoadBalancerConfig returns load balancer config(full object) that matches the criteria.
func (k *Kubernetes) GetLoadBalancerConfig(ctx context.Context, key ctrlclient.ObjectKey) (*everestv1alpha1.LoadBalancerConfig, error) {
	result := &everestv1alpha1.LoadBalancerConfig{}
	if err := k.k8sClient.Get(ctx, key, result); err != nil {
		return nil, err
	}
	return result, nil
}

// DeleteLoadBalancerConfig deletes load balancer config that matches the criteria.
func (k *Kubernetes) DeleteLoadBalancerConfig(ctx context.Context, obj *everestv1alpha1.LoadBalancerConfig) error {
	return k.k8sClient.Delete(ctx, obj)
}

// CreateLoadBalancerConfig creates load balancer config.
func (k *Kubernetes) CreateLoadBalancerConfig(ctx context.Context, lbc *everestv1alpha1.LoadBalancerConfig) (*everestv1alpha1.LoadBalancerConfig, error) {
	if err := k.k8sClient.Create(ctx, lbc); err != nil {
		return nil, err
	}
	return lbc, nil
}

// UpdateLoadBalancerConfig updates load balancer config.
func (k *Kubernetes) UpdateLoadBalancerConfig(ctx context.Context, lbc *everestv1alpha1.LoadBalancerConfig) (*everestv1alpha1.LoadBalancerConfig, error) {
	if err := k.k8sClient.Update(ctx, lbc); err != nil {
		return nil, err
	}
	return lbc, nil
}
