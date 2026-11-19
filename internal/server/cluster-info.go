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

package server

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
)

// GetKubernetesClusterInfoForCluster returns cluster type and storage classes.
func (e *EverestServer) GetKubernetesClusterInfoForCluster(ctx echo.Context, cluster string) error {
	result, err := e.handler.GetKubernetesClusterInfo(ctx.Request().Context(), cluster)
	if err != nil {
		e.l.Errorf("GetKubernetesClusterInfo failed: %w", err)
		return err
	}
	return ctx.JSON(http.StatusOK, result)
}

// GetKubernetesClusterInfo serves the deprecated unscoped endpoint.
func (e *EverestServer) GetKubernetesClusterInfo(ctx echo.Context) error {
	ctx.Response().Header().Set("Deprecation", "true")
	ctx.Response().Header().Set(
		"Link",
		fmt.Sprintf(`</v1/clusters/%s/cluster-info>; rel="successor-version"`, e.config.ClusterName),
	)
	return e.GetKubernetesClusterInfoForCluster(ctx, e.config.ClusterName)
}
