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

package k8s

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
)

func (h *k8sHandler) CreateLoadBalancerConfig(ctx context.Context, psp *everestv1alpha1.LoadBalancerConfig) (*everestv1alpha1.LoadBalancerConfig, error) {
	return h.kubeConnector.CreateLoadBalancerConfig(ctx, psp)
}

func (h *k8sHandler) UpdateLoadBalancerConfig(ctx context.Context, psp *everestv1alpha1.LoadBalancerConfig) (*everestv1alpha1.LoadBalancerConfig, error) {
	return h.kubeConnector.UpdateLoadBalancerConfig(ctx, psp)
}

func (h *k8sHandler) ListLoadBalancerConfigs(ctx context.Context) (*everestv1alpha1.LoadBalancerConfigList, error) {
	return h.kubeConnector.ListLoadBalancerConfigs(ctx)
}

func (h *k8sHandler) DeleteLoadBalancerConfig(ctx context.Context, name string) error {
	delObj := &everestv1alpha1.LoadBalancerConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	return h.kubeConnector.DeleteLoadBalancerConfig(ctx, delObj)
}

func (h *k8sHandler) GetLoadBalancerConfig(ctx context.Context, name string) (*everestv1alpha1.LoadBalancerConfig, error) {
	return h.kubeConnector.GetLoadBalancerConfig(ctx, types.NamespacedName{Name: name})
}
