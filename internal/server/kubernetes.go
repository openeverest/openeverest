// everest
// Copyright (C) 2023 Percona LLC
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

// Package server contains the API server implementation.
package server

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
)

// GetKubernetesClusterResourcesForCluster returns all and available resources of a Kubernetes cluster.
func (e *EverestServer) GetKubernetesClusterResourcesForCluster(ctx echo.Context, cluster string) error {
	resources, err := e.handler.GetKubernetesClusterResources(ctx.Request().Context(), cluster)
	if err != nil {
		e.l.Errorf("GetKubernetesClusterResources failed: %w", err)
		return err
	}
	return ctx.JSON(http.StatusOK, resources)
}

// GetKubernetesClusterResources serves the deprecated unscoped endpoint.
func (e *EverestServer) GetKubernetesClusterResources(ctx echo.Context) error {
	ctx.Response().Header().Set("Deprecation", "true")
	ctx.Response().Header().Set(
		"Link",
		fmt.Sprintf(`</v1/clusters/%s/resources>; rel="successor-version"`, e.config.ClusterName),
	)
	return e.GetKubernetesClusterResourcesForCluster(ctx, e.config.ClusterName)
}
