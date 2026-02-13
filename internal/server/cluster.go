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

package server

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type APICluster struct {
	Name   string `json:"name"`
	Server string `json:"server"`
}

type APIClusterList struct {
	Items []APICluster `json:"items"`
}

// ListClusters handles GET /clusters
func (e *EverestServer) ListClusters(ctx echo.Context) error {
	clusters, err := e.handler.ListClusters(ctx.Request().Context())
	if err != nil {
		e.l.Errorf("ListClusters failed: %w", err)
		return err
	}
	return ctx.JSON(http.StatusOK, clusters)
}

// GetCluster handles GET /clusters/{name}
func (e *EverestServer) GetCluster(ctx echo.Context, name string) error {
	cluster, err := e.handler.GetCluster(ctx.Request().Context(), name)
	if err != nil {
		e.l.Errorf("GetCluster failed: %w", err)
		return err
	}
	return ctx.JSON(http.StatusOK, cluster)
}
