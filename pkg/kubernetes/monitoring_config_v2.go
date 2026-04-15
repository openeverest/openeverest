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

package kubernetes

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	monitoringv1alpha2 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha2"
)

// ListMonitoringConfigsV2 returns list of managed monitoring configs that match the criteria.
// This method returns a list of full objects (meta and spec).
//
// TODO Rename it to ListMonitoringConfigs once v1 code is removed.
func (k *Kubernetes) ListMonitoringConfigsV2(ctx context.Context, opts ...ctrlclient.ListOption) (*monitoringv1alpha2.MonitoringConfigList, error) {
	result := &monitoringv1alpha2.MonitoringConfigList{}
	if err := k.k8sClient.List(ctx, result, opts...); err != nil {
		return nil, err
	}
	return result, nil
}

// GetMonitoringConfigV2 returns monitoring config(full object) that matches the criteria.
//
// TODO Rename it to GetMonitoringConfig once v1 code is removed.
func (k *Kubernetes) GetMonitoringConfigV2(ctx context.Context, key ctrlclient.ObjectKey) (*monitoringv1alpha2.MonitoringConfig, error) {
	result := &monitoringv1alpha2.MonitoringConfig{}
	if err := k.k8sClient.Get(ctx, key, result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetMonitoringConfigMetaV2 returns monitoring config(metadata only) that matches the criteria.
//
// TODO Rename it to GetMonitoringConfigMeta once v1 code is removed.
func (k *Kubernetes) GetMonitoringConfigMetaV2(ctx context.Context, key ctrlclient.ObjectKey) (*metav1.PartialObjectMetadata, error) {
	objMeta := &metav1.PartialObjectMetadata{}
	objMeta.SetGroupVersionKind(monitoringv1alpha2.GroupVersion.WithKind("MonitoringConfig"))
	if err := k.k8sClient.Get(ctx, key, objMeta); err != nil {
		return nil, err
	}
	return objMeta, nil
}

// CreateMonitoringConfigV2 creates monitoring config.
//
// TODO Rename it to CreateMonitoringConfig once v1 code is removed.
func (k *Kubernetes) CreateMonitoringConfigV2(ctx context.Context, config *monitoringv1alpha2.MonitoringConfig) (*monitoringv1alpha2.MonitoringConfig, error) {
	if err := k.k8sClient.Create(ctx, config); err != nil {
		return nil, err
	}
	return config, nil
}

// UpdateMonitoringConfigV2 updates monitoring config.
//
// TODO Rename it to UpdateMonitoringConfig once v1 code is removed.
func (k *Kubernetes) UpdateMonitoringConfigV2(ctx context.Context, config *monitoringv1alpha2.MonitoringConfig) (*monitoringv1alpha2.MonitoringConfig, error) {
	if err := k.k8sClient.Update(ctx, config); err != nil {
		return nil, err
	}
	return config, nil
}

// DeleteMonitoringConfigV2 deletes monitoring config that matches the criteria.
//
// TODO Rename it to DeleteMonitoringConfig once v1 code is removed.
func (k *Kubernetes) DeleteMonitoringConfigV2(ctx context.Context, obj *monitoringv1alpha2.MonitoringConfig) error {
	return k.k8sClient.Delete(ctx, obj)
}
