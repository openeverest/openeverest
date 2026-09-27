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
)

// ListPlugins lists the enabled plugins the caller can see in the cluster.
func (e *EverestServer) ListPlugins(c echo.Context, cluster string) error {
	result, err := e.handler.ListPlugins(c.Request().Context(), cluster)
	if err != nil {
		e.l.Errorf("ListPlugins failed: %v", err)
		return err
	}
	return c.JSON(http.StatusOK, result)
}

// GetPluginContext returns the caller's identity and accessible namespaces.
func (e *EverestServer) GetPluginContext(c echo.Context, cluster string) error {
	result, err := e.handler.GetPluginContext(c.Request().Context(), cluster)
	if err != nil {
		e.l.Errorf("GetPluginContext failed: %v", err)
		return err
	}
	return c.JSON(http.StatusOK, result)
}
