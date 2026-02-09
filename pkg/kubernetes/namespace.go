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
	"errors"
	"slices"

	corev1 "k8s.io/api/core/v1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openeverest/openeverest/v2/pkg/common"
)

// CreateNamespace creates the given namespace.
func (k *Kubernetes) CreateNamespace(ctx context.Context, namespace *corev1.Namespace) (*corev1.Namespace, error) {
	if err := k.k8sClient.Create(ctx, namespace); err != nil {
		return nil, err
	}
	return namespace, nil
}

// GetNamespace returns a namespace that matches the criteria.
func (k *Kubernetes) GetNamespace(ctx context.Context, key ctrlclient.ObjectKey) (*corev1.Namespace, error) {
	result := &corev1.Namespace{}
	if err := k.k8sClient.Get(ctx, key, result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetDBNamespaces returns a list of DB namespaces that managed by the Everest and match the criteria.
func (k *Kubernetes) GetDBNamespaces(ctx context.Context, opts ...ctrlclient.ListOption) (*corev1.NamespaceList, error) {
	opts = append(opts, ctrlclient.MatchingLabels{common.KubernetesManagedByLabel: common.Everest})
	result, err := k.ListNamespaces(ctx, opts...)
	if err != nil {
		return nil, errors.Join(err, errors.New("failed to get managed namespaces"))
	}

	internalNs := []string{common.SystemNamespace, common.MonitoringNamespace}
	// filter out Everest system and monitoring namespaces.
	result.Items = slices.DeleteFunc(result.Items, func(ns corev1.Namespace) bool {
		return slices.Contains(internalNs, ns.Name)
	})
	return result, nil
}

// DeleteNamespace deletes a namespace that matches the criteria.
func (k *Kubernetes) DeleteNamespace(ctx context.Context, obj *corev1.Namespace) error {
	return k.k8sClient.Delete(ctx, obj)
}

// ListNamespaces lists all namespaces that match the criteria.
// This method returns a list of full objects (meta and spec).
func (k *Kubernetes) ListNamespaces(ctx context.Context, opts ...ctrlclient.ListOption) (*corev1.NamespaceList, error) {
	result := &corev1.NamespaceList{}
	if err := k.k8sClient.List(ctx, result, opts...); err != nil {
		return nil, err
	}
	return result, nil
}

// UpdateNamespace updates the given namespace.
func (k *Kubernetes) UpdateNamespace(ctx context.Context, namespace *corev1.Namespace) (*corev1.Namespace, error) {
	if err := k.k8sClient.Update(ctx, namespace); err != nil {
		return nil, err
	}
	return namespace, nil
}
