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

package cli

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/openeverest/openeverest/v2/client"
)

// ResolveCluster returns the requested cluster or discovers it when exactly one
// cluster is registered. Callers with multiple clusters must select one explicitly.
func ResolveCluster(
	ctx context.Context,
	c *client.ClientWithResponses,
	requested string,
) (string, error) {
	if requested != "" {
		return requested, nil
	}

	resp, err := c.ListClustersWithResponse(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to list clusters: %w", err)
	}
	if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		return "", fmt.Errorf("unexpected response listing clusters: %s", resp.Status())
	}
	if len(resp.JSON200.Items) == 0 {
		return "", errors.New("no clusters are registered")
	}

	clusters := resp.JSON200.Items
	sort.Slice(clusters, func(i, j int) bool {
		return clusters[i].Name < clusters[j].Name
	})
	if len(clusters) > 1 {
		names := make([]string, 0, len(clusters))
		for _, cluster := range clusters {
			names = append(names, cluster.Name)
		}
		return "", fmt.Errorf(
			"multiple clusters are available (%s); specify --cluster",
			strings.Join(names, ", "),
		)
	}
	return clusters[0].Name, nil
}
