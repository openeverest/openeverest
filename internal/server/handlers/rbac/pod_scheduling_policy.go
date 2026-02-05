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
	"github.com/percona/everest/api"
	"github.com/percona/everest/pkg/rbac"
)

// CreatePodSchedulingPolicy creates a new pod scheduling policy.
func (h *rbacHandler) CreatePodSchedulingPolicy(ctx context.Context, psp *everestv1alpha1.PodSchedulingPolicy) (*everestv1alpha1.PodSchedulingPolicy, error) {
	if err := h.enforce(ctx, rbac.ResourcePodSchedulingPolicies, rbac.ActionCreate, rbac.ObjectName(psp.GetName())); err != nil {
		return nil, err
	}
	return h.next.CreatePodSchedulingPolicy(ctx, psp)
}

// UpdatePodSchedulingPolicy updates an existing pod scheduling policy.
func (h *rbacHandler) UpdatePodSchedulingPolicy(ctx context.Context, psp *everestv1alpha1.PodSchedulingPolicy) (*everestv1alpha1.PodSchedulingPolicy, error) {
	if err := h.enforce(ctx, rbac.ResourcePodSchedulingPolicies, rbac.ActionUpdate, rbac.ObjectName(psp.GetName())); err != nil {
		return nil, err
	}
	return h.next.UpdatePodSchedulingPolicy(ctx, psp)
}

// ListPodSchedulingPolicies lists all pod scheduling policies.
func (h *rbacHandler) ListPodSchedulingPolicies(ctx context.Context, params *api.ListPodSchedulingPolicyParams) (*everestv1alpha1.PodSchedulingPolicyList, error) {
	pspList, err := h.next.ListPodSchedulingPolicies(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("ListPodSchedulingPolicies failed: %w", err)
	}
	// filter out PodSchedulingPolicies that the user does not have access to.
	pspList.Items = slices.DeleteFunc(pspList.Items, func(psp everestv1alpha1.PodSchedulingPolicy) bool {
		return h.enforce(ctx, rbac.ResourcePodSchedulingPolicies, rbac.ActionRead, rbac.ObjectName(psp.GetName())) != nil
	})
	return pspList, nil
}

// DeletePodSchedulingPolicy deletes a pod scheduling policy.
func (h *rbacHandler) DeletePodSchedulingPolicy(ctx context.Context, name string) error {
	if err := h.enforce(ctx, rbac.ResourcePodSchedulingPolicies, rbac.ActionDelete, rbac.ObjectName(name)); err != nil {
		return err
	}
	return h.next.DeletePodSchedulingPolicy(ctx, name)
}

// GetPodSchedulingPolicy retrieves a pod scheduling policy by name.
func (h *rbacHandler) GetPodSchedulingPolicy(ctx context.Context, name string) (*everestv1alpha1.PodSchedulingPolicy, error) {
	if err := h.enforce(ctx, rbac.ResourcePodSchedulingPolicies, rbac.ActionRead, rbac.ObjectName(name)); err != nil {
		return nil, err
	}
	return h.next.GetPodSchedulingPolicy(ctx, name)
}
