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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
)

// ListPodSchedulingPolicies returns a list of pod scheduling policy that matches the criteria.
// This method returns a list of full objects (meta and spec).
func (k *Kubernetes) ListPodSchedulingPolicies(ctx context.Context, opts ...ctrlclient.ListOption) (*everestv1alpha1.PodSchedulingPolicyList, error) {
	result := &everestv1alpha1.PodSchedulingPolicyList{}
	if err := k.k8sClient.List(ctx, result, opts...); err != nil {
		return nil, err
	}
	return result, nil
}

// GetPodSchedulingPolicy returns pod scheduling policy(full object) that matches the criteria.
func (k *Kubernetes) GetPodSchedulingPolicy(ctx context.Context, key ctrlclient.ObjectKey) (*everestv1alpha1.PodSchedulingPolicy, error) {
	result := &everestv1alpha1.PodSchedulingPolicy{}
	if err := k.k8sClient.Get(ctx, key, result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetPodSchedulingPolicyMeta returns pod scheduling policy(metadata only) that matches the criteria.
func (k *Kubernetes) GetPodSchedulingPolicyMeta(ctx context.Context, key ctrlclient.ObjectKey) (*metav1.PartialObjectMetadata, error) {
	objMeta := &metav1.PartialObjectMetadata{}
	objMeta.SetGroupVersionKind(everestv1alpha1.GroupVersion.WithKind("PodSchedulingPolicy"))
	if err := k.k8sClient.Get(ctx, key, objMeta); err != nil {
		return nil, err
	}
	return objMeta, nil
}

// DeletePodSchedulingPolicy deletes pod scheduling policy that matches the criteria.
func (k *Kubernetes) DeletePodSchedulingPolicy(ctx context.Context, obj *everestv1alpha1.PodSchedulingPolicy) error {
	return k.k8sClient.Delete(ctx, obj)
}

// CreatePodSchedulingPolicy creates pod scheduling policy.
func (k *Kubernetes) CreatePodSchedulingPolicy(ctx context.Context, psp *everestv1alpha1.PodSchedulingPolicy) (*everestv1alpha1.PodSchedulingPolicy, error) {
	if err := k.k8sClient.Create(ctx, psp); err != nil {
		return nil, err
	}
	return psp, nil
}

// UpdatePodSchedulingPolicy updates pod scheduling policy.
func (k *Kubernetes) UpdatePodSchedulingPolicy(ctx context.Context, psp *everestv1alpha1.PodSchedulingPolicy) (*everestv1alpha1.PodSchedulingPolicy, error) {
	if err := k.k8sClient.Update(ctx, psp); err != nil {
		return nil, err
	}
	return psp, nil
}
