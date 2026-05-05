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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
)

// ListDatabaseClusters returns list of managed database clusters that match the criteria.
// This method returns a list of full objects (meta and spec).
func (k *Kubernetes) ListDatabaseClusters(ctx context.Context, opts ...ctrlclient.ListOption) (*everestv1alpha1.DatabaseClusterList, error) {
	result := &everestv1alpha1.DatabaseClusterList{}
	if err := k.k8sClient.List(ctx, result, opts...); err != nil {
		return nil, err
	}
	return result, nil
}

// ListDatabaseClusters returns list of managed database clusters that match the criteria.
// This method returns a list of simplified objects (meta only).
func (k *Kubernetes) listDatabaseClustersMeta(ctx context.Context, opts ...ctrlclient.ListOption) (*metav1.PartialObjectMetadataList, error) {
	result := &metav1.PartialObjectMetadataList{}
	result.SetGroupVersionKind(everestv1alpha1.GroupVersion.WithKind("DatabaseClusterList"))
	if err := k.k8sClient.List(ctx, result, opts...); err != nil {
		return nil, err
	}
	return result, nil
}

// GetDatabaseCluster returns database cluster that matches the criteria.
func (k *Kubernetes) GetDatabaseCluster(ctx context.Context, key ctrlclient.ObjectKey) (*everestv1alpha1.DatabaseCluster, error) {
	result := &everestv1alpha1.DatabaseCluster{}
	if err := k.k8sClient.Get(ctx, key, result); err != nil {
		return nil, err
	}
	return result, nil
}

// DeleteDatabaseCluster deletes database cluster that matches the criteria.
func (k *Kubernetes) DeleteDatabaseCluster(ctx context.Context, obj *everestv1alpha1.DatabaseCluster) error {
	return k.k8sClient.Delete(ctx, obj)
}

// CreateDatabaseCluster creates database cluster.
func (k *Kubernetes) CreateDatabaseCluster(ctx context.Context, cluster *everestv1alpha1.DatabaseCluster) (*everestv1alpha1.DatabaseCluster, error) {
	if err := k.k8sClient.Create(ctx, cluster); err != nil {
		return nil, err
	}
	return cluster, nil
}

// UpdateDatabaseCluster updates database cluster.
func (k *Kubernetes) UpdateDatabaseCluster(ctx context.Context, cluster *everestv1alpha1.DatabaseCluster) (*everestv1alpha1.DatabaseCluster, error) {
	if err := k.k8sClient.Update(ctx, cluster); err != nil {
		return nil, err
	}
	return cluster, nil
}

// DeleteDatabaseClusters deletes all database clusters that match the criteria.
// This function will wait until all clusters are deleted.
func (k *Kubernetes) DeleteDatabaseClusters(ctx context.Context, opts ...ctrlclient.ListOption) error {
	// No need to fetch full objects, we only need the fact there are objects that match the criteria(opts).
	delList, err := k.listDatabaseClustersMeta(ctx, opts...)
	if err != nil {
		k.l.Errorf("Could not list DB clusters: %s", err)
		return err
	}

	if delList == nil || len(delList.Items) == 0 {
		// Nothing to delete.
		return nil
	}

	// We increase the timeout if there's too many DB clusters.
	const countTimeoutMultiply = 3
	timeout := max(pollTimeout, pollTimeout*time.Duration(len(delList.Items)/countTimeoutMultiply))

	// need to convert ListOptions to DeleteAllOfOptions
	delOpts := &ctrlclient.DeleteAllOfOptions{}
	for _, opt := range opts {
		opt.ApplyToList(&delOpts.ListOptions)
	}

	k.l.Debugf("Setting DB clusters removal timeout to %s", timeout)
	return wait.PollUntilContextTimeout(ctx, pollInterval, timeout, true, func(ctx context.Context) (bool, error) {
		// Skip fetching the list of objects to delete again, we already have it (see code above).
		if delList == nil {
			var err error
			if delList, err = k.listDatabaseClustersMeta(ctx, opts...); err != nil {
				k.l.Errorf("Could not list DB clusters in polling: %s", err)
				return false, err
			}

			if delList == nil || len(delList.Items) == 0 {
				// Nothing to delete.
				return true, nil
			}
		}

		// Reset the list to nil to fetch it again on the next iteration.
		delList = nil

		if err := k.k8sClient.DeleteAllOf(ctx, &everestv1alpha1.DatabaseCluster{}, delOpts); err != nil {
			return false, err
		}
		return false, nil
	})
}

// DatabasesExist checks if there are databases that match criteria exist.
func (k *Kubernetes) DatabasesExist(ctx context.Context, opts ...ctrlclient.ListOption) (bool, error) {
	list, err := k.listDatabaseClustersMeta(ctx, opts...)
	if err != nil {
		return false, err
	}

	if list != nil && len(list.Items) > 0 {
		return true, nil
	}
	return false, nil
}
