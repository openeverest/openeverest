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

package validation

import (
	"context"

	"github.com/openeverest/openeverest/v2/api"
)

// ListClusters returns a list of clusters.
func (h *validateHandler) ListClusters(ctx context.Context) (*api.ClusterList, error) {
	// TODO: Add validation logic, then call next handler
	return h.next.ListClusters(ctx)
}

// GetCluster returns a cluster by name.
func (h *validateHandler) GetCluster(ctx context.Context, name string) (*api.Cluster, error) {
	// TODO: Add validation logic, then call next handler
	return h.next.GetCluster(ctx, name)
}
