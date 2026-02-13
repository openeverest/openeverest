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

package k8s

import (
	"context"
	"fmt"

	"github.com/openeverest/openeverest/v2/api"
)

// ListClusters returns a list of clusters.
func (h *k8sHandler) ListClusters(ctx context.Context) (*api.ClusterList, error) {
	// For now, we only support a single cluster, which is the Kubernetes
	// cluster where Everest is running. When we add support for multiple
	// clusters, this will need to be updated.
	clustersList := &api.ClusterList{
		Items: []api.Cluster{
			{
				Name:   "main",
				Server: "https://kubernetes.default.svc",
			},
		},
	}
	return clustersList, nil
}

// GetCluster returns a cluster by name.
func (h *k8sHandler) GetCluster(ctx context.Context, name string) (*api.Cluster, error) {
	// For now, we only support a single cluster, which is the Kubernetes
	// cluster where Everest is running. When we add support for multiple
	// clusters, this will need to be updated.
	if name != "main" {
		return nil, fmt.Errorf("cluster not found: %s", name)
	}

	return &api.Cluster{
		Name:   "main",
		Server: "https://kubernetes.default.svc",
	}, nil
}
