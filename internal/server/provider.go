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
	"net/http"

	"github.com/labstack/echo/v4"
)

// ListProviders lists all providers in the cluster.
func (e *EverestServer) ListProviders(c echo.Context, cluster string) error {
	result, err := e.handler.ListProviders(c.Request().Context(), cluster)
	if err != nil {
		e.l.Errorf("ListProviders failed: %v", err)
		return err
	}
	return c.JSON(http.StatusOK, result)
}

// GetProvider returns a specific provider.
func (e *EverestServer) GetProvider(c echo.Context, cluster string, provider string) error {
	result, err := e.handler.GetProvider(c.Request().Context(), cluster, provider)
	if err != nil {
		e.l.Errorf("GetProvider failed: %v", err)
		return err
	}
	return c.JSON(http.StatusOK, result)
}
