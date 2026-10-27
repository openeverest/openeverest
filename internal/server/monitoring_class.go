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

	"github.com/AlekSi/pointer"
	"github.com/labstack/echo/v4"

	"github.com/openeverest/openeverest/v2/internal/server/api"
)

// ListMonitoringClasses lists the monitoring classes in the cluster, optionally those compatible with a provider.
func (e *EverestServer) ListMonitoringClasses(c echo.Context, cluster string, params api.ListMonitoringClassesParams) error {
	result, err := e.handler.ListMonitoringClasses(c.Request().Context(), cluster, pointer.GetString(params.Provider))
	if err != nil {
		e.l.Errorf("ListMonitoringClasses failed: %v", err)
		return err
	}
	return c.JSON(http.StatusOK, result)
}

// GetMonitoringClass returns a specific monitoring class.
func (e *EverestServer) GetMonitoringClass(c echo.Context, cluster, name string) error {
	result, err := e.handler.GetMonitoringClass(c.Request().Context(), cluster, name)
	if err != nil {
		e.l.Errorf("GetMonitoringClass failed: %v", err)
		return err
	}
	return c.JSON(http.StatusOK, result)
}
