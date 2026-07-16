// everest
// Copyright (C) 2025 Percona LLC
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

	olmv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// GetClusterServiceVersion retrieves a ClusterServiceVersion that matches the criteria.
func (k *Kubernetes) GetClusterServiceVersion(ctx context.Context, key ctrlclient.ObjectKey) (*olmv1alpha1.ClusterServiceVersion, error) {
	result := &olmv1alpha1.ClusterServiceVersion{}
	if err := k.k8sClient.Get(ctx, key, result); err != nil {
		return nil, err
	}
	return result, nil
}

// ListClusterServiceVersion list all CSVs that match the criteria.
// This method returns a list of full objects (meta and spec).
func (k *Kubernetes) ListClusterServiceVersion(ctx context.Context, opts ...ctrlclient.ListOption) (*olmv1alpha1.ClusterServiceVersionList, error) {
	result := &olmv1alpha1.ClusterServiceVersionList{}
	if err := k.k8sClient.List(ctx, result, opts...); err != nil {
		return nil, err
	}
	return result, nil
}

// listClusterServiceVersionMeta list all CSVs that match the criteria.
// This method returns a list of simplified objects (meta only).
func (k *Kubernetes) listClusterServiceVersionMeta(ctx context.Context, opts ...ctrlclient.ListOption) (*metav1.PartialObjectMetadataList, error) {
	result := &metav1.PartialObjectMetadataList{}
	result.SetGroupVersionKind(olmv1alpha1.SchemeGroupVersion.WithKind("ClusterServiceVersionList"))
	if err := k.k8sClient.List(ctx, result, opts...); err != nil {
		return nil, err
	}
	return result, nil
}

// DeleteClusterServiceVersion deletes a ClusterServiceVersion that matches the criteria.
func (k *Kubernetes) DeleteClusterServiceVersion(ctx context.Context, obj *olmv1alpha1.ClusterServiceVersion) error {
	return k.k8sClient.Delete(ctx, obj)
}

// DeleteClusterServiceVersions deletes all ClusterServiceVersion that match the criteria.
// This function will wait until all ClusterServiceVersion are deleted.
//
//nolint:dupl // per-resource client wrappers are intentionally similar
func (k *Kubernetes) DeleteClusterServiceVersions(ctx context.Context, opts ...ctrlclient.ListOption) error {
	// No need to fetch full objects, we only need the fact there are objects that match the criteria(opts).
	delList, err := k.listClusterServiceVersionMeta(ctx, opts...)
	if err != nil {
		k.l.Errorf("Could not list cluster service versions: %s", err)
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

	k.l.Debugf("Setting cluster service versions removal timeout to %s", pollTimeout)
	return wait.PollUntilContextTimeout(ctx, pollInterval, pollTimeout, true, func(ctx context.Context) (bool, error) {
		// Skip fetching the list of objects to delete again, we already have it (see code above).
		if delList == nil {
			var err error
			if delList, err = k.listClusterServiceVersionMeta(ctx, opts...); err != nil {
				k.l.Errorf("Could not list cluster service versions in polling: %s", err)
				return false, err
			}

			if delList == nil || len(delList.Items) == 0 {
				// Nothing to delete.
				return true, nil
			}
		}

		// Reset the list to nil to fetch it again on the next iteration.
		delList = nil

		if err := k.k8sClient.DeleteAllOf(ctx, &olmv1alpha1.ClusterServiceVersion{}, delOpts); err != nil {
			return false, err
		}
		return false, nil
	})
}
