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
//
//nolint:dupl // per-resource client wrappers are intentionally similar
package kubernetes

import (
	"context"

	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
)

// ListMonitoringDestinations returns list of monitoring destinations that match the criteria.
func (k *Kubernetes) ListMonitoringDestinations(ctx context.Context, opts ...ctrlclient.ListOption) (*monitoringv1alpha1.MonitoringDestinationList, error) {
	result := &monitoringv1alpha1.MonitoringDestinationList{}
	if err := k.k8sClient.List(ctx, result, opts...); err != nil {
		return nil, err
	}
	return result, nil
}

// GetMonitoringDestination returns a monitoring destination by name and namespace.
func (k *Kubernetes) GetMonitoringDestination(ctx context.Context, key ctrlclient.ObjectKey) (*monitoringv1alpha1.MonitoringDestination, error) {
	result := &monitoringv1alpha1.MonitoringDestination{}
	if err := k.k8sClient.Get(ctx, key, result); err != nil {
		return nil, err
	}
	return result, nil
}

// CreateMonitoringDestination creates a monitoring destination.
func (k *Kubernetes) CreateMonitoringDestination(ctx context.Context, destination *monitoringv1alpha1.MonitoringDestination) (*monitoringv1alpha1.MonitoringDestination, error) {
	if err := k.k8sClient.Create(ctx, destination); err != nil {
		return nil, err
	}
	return destination, nil
}

// UpdateMonitoringDestination updates a monitoring destination.
func (k *Kubernetes) UpdateMonitoringDestination(ctx context.Context, destination *monitoringv1alpha1.MonitoringDestination) (*monitoringv1alpha1.MonitoringDestination, error) {
	if err := k.k8sClient.Update(ctx, destination); err != nil {
		return nil, err
	}
	return destination, nil
}

// DeleteMonitoringDestination deletes a monitoring destination.
func (k *Kubernetes) DeleteMonitoringDestination(ctx context.Context, obj *monitoringv1alpha1.MonitoringDestination) error {
	return k.k8sClient.Delete(ctx, obj)
}
