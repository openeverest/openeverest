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

	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	extensionsv1alpha1 "github.com/openeverest/openeverest/v2/api/extensions/v1alpha1"
)

// ListClusterConnections returns all registered remote clusters.
func (k *Kubernetes) ListClusterConnections(
	ctx context.Context,
	opts ...ctrlclient.ListOption,
) (*extensionsv1alpha1.ClusterConnectionList, error) {
	result := &extensionsv1alpha1.ClusterConnectionList{}
	if err := k.k8sClient.List(ctx, result, opts...); err != nil {
		return nil, err
	}
	return result, nil
}

// GetClusterConnection returns a registered remote cluster.
func (k *Kubernetes) GetClusterConnection(
	ctx context.Context,
	key ctrlclient.ObjectKey,
) (*extensionsv1alpha1.ClusterConnection, error) {
	result := &extensionsv1alpha1.ClusterConnection{}
	if err := k.k8sClient.Get(ctx, key, result); err != nil {
		return nil, err
	}
	return result, nil
}

// CreateClusterConnection registers a remote cluster.
func (k *Kubernetes) CreateClusterConnection(
	ctx context.Context,
	connection *extensionsv1alpha1.ClusterConnection,
) (*extensionsv1alpha1.ClusterConnection, error) {
	if err := k.k8sClient.Create(ctx, connection); err != nil {
		return nil, err
	}
	return connection, nil
}

// DeleteClusterConnection unregisters a remote cluster.
func (k *Kubernetes) DeleteClusterConnection(
	ctx context.Context,
	connection *extensionsv1alpha1.ClusterConnection,
) error {
	return k.k8sClient.Delete(ctx, connection)
}
