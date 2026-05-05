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
	"slices"

	"github.com/AlekSi/pointer"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
	"github.com/percona/everest/api"
)

func (h *k8sHandler) CreatePodSchedulingPolicy(ctx context.Context, psp *everestv1alpha1.PodSchedulingPolicy) (*everestv1alpha1.PodSchedulingPolicy, error) {
	return h.kubeConnector.CreatePodSchedulingPolicy(ctx, psp)
}

func (h *k8sHandler) UpdatePodSchedulingPolicy(ctx context.Context, psp *everestv1alpha1.PodSchedulingPolicy) (*everestv1alpha1.PodSchedulingPolicy, error) {
	return h.kubeConnector.UpdatePodSchedulingPolicy(ctx, psp)
}

func (h *k8sHandler) ListPodSchedulingPolicies(ctx context.Context, params *api.ListPodSchedulingPolicyParams) (*everestv1alpha1.PodSchedulingPolicyList, error) {
	pspList, err := h.kubeConnector.ListPodSchedulingPolicies(ctx)
	if err != nil {
		return pspList, err
	}

	if params != nil {
		if engineType := everestv1alpha1.EngineType(pointer.Get(params.EngineType)); engineType != "" {
			// filter out PodSchedulingPolicies that do not match a requested engine type
			pspList.Items = slices.DeleteFunc(pspList.Items, func(psp everestv1alpha1.PodSchedulingPolicy) bool {
				return psp.Spec.EngineType != engineType
			})
		}

		if pointer.Get(params.HasRules) {
			// filter out PodSchedulingPolicies that do not have rules
			pspList.Items = slices.DeleteFunc(pspList.Items, func(psp everestv1alpha1.PodSchedulingPolicy) bool {
				return !psp.HasRules()
			})
		}
	}
	return pspList, err
}

func (h *k8sHandler) DeletePodSchedulingPolicy(ctx context.Context, name string) error {
	delObj := &everestv1alpha1.PodSchedulingPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	return h.kubeConnector.DeletePodSchedulingPolicy(ctx, delObj)
}

func (h *k8sHandler) GetPodSchedulingPolicy(ctx context.Context, name string) (*everestv1alpha1.PodSchedulingPolicy, error) {
	return h.kubeConnector.GetPodSchedulingPolicy(ctx, types.NamespacedName{Name: name})
}
