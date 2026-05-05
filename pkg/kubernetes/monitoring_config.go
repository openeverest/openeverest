// everest
// Copyright (C) 2023 Percona LLC
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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
)

// ListMonitoringConfigs returns list of managed monitoring configs that match the criteria.
// This method returns a list of full objects (meta and spec).
func (k *Kubernetes) ListMonitoringConfigs(ctx context.Context, opts ...ctrlclient.ListOption) (*everestv1alpha1.MonitoringConfigList, error) {
	result := &everestv1alpha1.MonitoringConfigList{}
	if err := k.k8sClient.List(ctx, result, opts...); err != nil {
		return nil, err
	}
	return result, nil
}

// listMonitoringConfigsMeta returns list of managed monitoring configs that match the criteria.
// This method returns a list of simplified objects (meta only).
func (k *Kubernetes) listMonitoringConfigsMeta(ctx context.Context, opts ...ctrlclient.ListOption) (*metav1.PartialObjectMetadataList, error) {
	result := &metav1.PartialObjectMetadataList{}
	result.SetGroupVersionKind(everestv1alpha1.GroupVersion.WithKind("MonitoringConfigList"))
	if err := k.k8sClient.List(ctx, result, opts...); err != nil {
		return nil, err
	}
	return result, nil
}

// GetMonitoringConfig returns monitoring config(full object) that matches the criteria.
func (k *Kubernetes) GetMonitoringConfig(ctx context.Context, key ctrlclient.ObjectKey) (*everestv1alpha1.MonitoringConfig, error) {
	result := &everestv1alpha1.MonitoringConfig{}
	if err := k.k8sClient.Get(ctx, key, result); err != nil {
		return nil, err
	}
	return result, nil
}

// GetMonitoringConfigMeta returns monitoring config(metadata only) that matches the criteria.
func (k *Kubernetes) GetMonitoringConfigMeta(ctx context.Context, key ctrlclient.ObjectKey) (*metav1.PartialObjectMetadata, error) {
	objMeta := &metav1.PartialObjectMetadata{}
	objMeta.SetGroupVersionKind(everestv1alpha1.GroupVersion.WithKind("MonitoringConfig"))
	if err := k.k8sClient.Get(ctx, key, objMeta); err != nil {
		return nil, err
	}
	return objMeta, nil
}

// CreateMonitoringConfig creates monitoring config.
func (k *Kubernetes) CreateMonitoringConfig(ctx context.Context, config *everestv1alpha1.MonitoringConfig) (*everestv1alpha1.MonitoringConfig, error) {
	if err := k.k8sClient.Create(ctx, config); err != nil {
		return nil, err
	}
	return config, nil
}

// UpdateMonitoringConfig updates monitoring config.
func (k *Kubernetes) UpdateMonitoringConfig(ctx context.Context, config *everestv1alpha1.MonitoringConfig) (*everestv1alpha1.MonitoringConfig, error) {
	if err := k.k8sClient.Update(ctx, config); err != nil {
		return nil, err
	}
	return config, nil
}

// DeleteMonitoringConfig deletes monitoring config that matches the criteria.
func (k *Kubernetes) DeleteMonitoringConfig(ctx context.Context, obj *everestv1alpha1.MonitoringConfig) error {
	return k.k8sClient.Delete(ctx, obj)
}

// DeleteMonitoringConfigs deletes monitoring configs that matches the criteria.
// This function will wait until all configs are deleted.
func (k *Kubernetes) DeleteMonitoringConfigs(ctx context.Context, opts ...ctrlclient.ListOption) error {
	// No need to fetch full objects, we only need the fact there are objects that match the criteria(opts).
	delList, err := k.listMonitoringConfigsMeta(ctx, opts...)
	if err != nil {
		k.l.Errorf("Could not list monitoring configs: %s", err)
		return err
	}

	if delList == nil || len(delList.Items) == 0 {
		// Nothing to delete.
		return nil
	}

	// need to convert ListOptions to DeleteAllOfOptions
	delOpts := &ctrlclient.DeleteAllOfOptions{}
	for _, opt := range opts {
		opt.ApplyToList(&delOpts.ListOptions)
	}

	k.l.Debugf("Setting monitoring configs removal timeout to %s", pollTimeout)
	return wait.PollUntilContextTimeout(ctx, pollInterval, pollTimeout, true, func(ctx context.Context) (bool, error) {
		// Skip fetching the list of objects to delete again, we already have it (see code above).
		if delList == nil {
			var err error
			if delList, err = k.listMonitoringConfigsMeta(ctx, opts...); err != nil {
				k.l.Errorf("Could not list monitoring configs in polling: %s", err)
				return false, err
			}

			if delList == nil || len(delList.Items) == 0 {
				// Nothing to delete.
				return true, nil
			}
		}

		// Reset the list to nil to fetch it again on the next iteration.
		delList = nil

		if err := k.k8sClient.DeleteAllOf(ctx, &everestv1alpha1.MonitoringConfig{}, delOpts); err != nil {
			return false, err
		}
		return false, nil
	})
}
