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
	"net/http"

	"github.com/labstack/echo/v4"

	api "github.com/openeverest/openeverest/v2/internal/server/api"
)

// ListMonitoringBindings lists monitoring bindings in the namespace, optionally
// filtered by destination.
func (e *EverestServer) ListMonitoringBindings(c echo.Context, cluster, namespace string, params api.ListMonitoringBindingsParams) error {
	result, err := e.handler.ListMonitoringBindings(c.Request().Context(), cluster, namespace, &params)
	if err != nil {
		e.l.Errorf("ListMonitoringBindings failed: %v", err)
		return err
	}
	return c.JSON(http.StatusOK, result)
}

// GetMonitoringBinding returns a specific monitoring binding.
func (e *EverestServer) GetMonitoringBinding(c echo.Context, cluster, namespace, name string) error {
	result, err := e.handler.GetMonitoringBinding(c.Request().Context(), cluster, namespace, name)
	if err != nil {
		e.l.Errorf("GetMonitoringBinding failed: %v", err)
		return err
	}
	return c.JSON(http.StatusOK, result)
}
