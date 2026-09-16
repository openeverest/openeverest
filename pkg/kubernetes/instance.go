// everest
// Copyright (C) 2023 Percona LLC
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

// Package kubernetes ...
package kubernetes

import (
	"context"

	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

// ListInstances returns list of instances that match the criteria.
func (k *Kubernetes) ListInstances(ctx context.Context, opts ...ctrlclient.ListOption) (*v1alpha1.InstanceList, error) {
	result := &v1alpha1.InstanceList{}
	if err := k.k8sClient.List(ctx, result, opts...); err != nil {
		return nil, err
	}
	return result, nil
}

// GetInstance returns instance that matches the criteria.
func (k *Kubernetes) GetInstance(ctx context.Context, key ctrlclient.ObjectKey) (*v1alpha1.Instance, error) {
	result := &v1alpha1.Instance{}
	if err := k.k8sClient.Get(ctx, key, result); err != nil {
		return nil, err
	}
	return result, nil
}

// DeleteInstance deletes instance that matches the criteria.
func (k *Kubernetes) DeleteInstance(ctx context.Context, obj *v1alpha1.Instance) error {
	return k.k8sClient.Delete(ctx, obj)
}

// CreateInstance creates instance.
func (k *Kubernetes) CreateInstance(ctx context.Context, instance *v1alpha1.Instance) (*v1alpha1.Instance, error) {
	if err := k.k8sClient.Create(ctx, instance); err != nil {
		return nil, err
	}
	return instance, nil
}

// UpdateInstance updates instance.This is for PUT operation, which replaces the entire resource with the new one.
// If you want to update only specific fields, use PatchInstance instead.
func (k *Kubernetes) UpdateInstance(ctx context.Context, instance *v1alpha1.Instance) (*v1alpha1.Instance, error) {
	if err := k.k8sClient.Update(ctx, instance); err != nil {
		return nil, err
	}
	return instance, nil
}

// PatchInstance patches instance using the provided patch.
func (k *Kubernetes) PatchInstance(ctx context.Context, instance *v1alpha1.Instance, patch ctrlclient.Patch, opts ...ctrlclient.PatchOption) (*v1alpha1.Instance, error) {
	if err := k.k8sClient.Patch(ctx, instance, patch, opts...); err != nil {
		return nil, err
	}
	return instance, nil
}
