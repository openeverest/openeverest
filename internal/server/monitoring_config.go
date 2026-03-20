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
	"encoding/json"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/openeverest/openeverest/v2/internal/server/api"
)

// CreateMonitoringConfig creates a new monitoring config.
func (e *EverestServer) CreateMonitoringConfig(c echo.Context, cluster, namespace string) error {
	// The cluster parameter is currently ignored as we operate on the configured cluster
	req := &api.MonitoringConfigCreateParams{}
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		e.l.Errorf("CreateMonitoringConfig: failed to read request body: %v", err)
		return err
	}

	if err := json.Unmarshal(body, req); err != nil {
		e.l.Errorf("CreateMonitoringConfig: failed to decode request body: %v", err)
		return err
	}

	result, err := e.handler.CreateMonitoringConfig(c.Request().Context(), namespace, req)
	if err != nil {
		e.l.Errorf("CreateMonitoringConfig failed: %v", err)
		return err
	}

	return c.JSON(http.StatusOK, result)
}

// ListMonitoringConfigs lists all monitoring configs.
func (e *EverestServer) ListMonitoringConfigs(c echo.Context, cluster, namespace string) error {
	// The cluster parameter is currently ignored as we operate on the configured cluster
	result, err := e.handler.ListMonitoringConfigs(c.Request().Context(), namespace)
	if err != nil {
		e.l.Errorf("ListMonitoringConfigs failed: %v", err)
		return err
	}

	return c.JSON(http.StatusOK, result)
}

// GetMonitoringConfig retrieves a monitoring config.
func (e *EverestServer) GetMonitoringConfig(c echo.Context, cluster, namespace, name string) error {
	// The cluster parameter is currently ignored as we operate on the configured cluster
	result, err := e.handler.GetMonitoringConfig(c.Request().Context(), namespace, name)
	if err != nil {
		e.l.Errorf("GetMonitoringConfig failed: %v", err)
		return err
	}

	return c.JSON(http.StatusOK, result)
}

// UpdateMonitoringConfig updates a monitoring config based on the provided fields.
func (e *EverestServer) UpdateMonitoringConfig(c echo.Context, cluster, namespace, name string) error {
	// The cluster parameter is currently ignored as we operate on the configured cluster
	req := &api.MonitoringConfigUpdateParams{}
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		e.l.Errorf("UpdateMonitoringConfig: failed to read request body: %v", err)
		return err
	}

	if err := json.Unmarshal(body, req); err != nil {
		e.l.Errorf("UpdateMonitoringConfig: failed to decode request body: %v", err)
		return err
	}

	result, err := e.handler.UpdateMonitoringConfig(c.Request().Context(), namespace, name, req)
	if err != nil {
		e.l.Errorf("UpdateMonitoringConfig failed: %v", err)
		return err
	}

	return c.JSON(http.StatusOK, result)
}

// DeleteMonitoringConfig deletes a monitoring config.
func (e *EverestServer) DeleteMonitoringConfig(c echo.Context, cluster, namespace, name string) error {
	// The cluster parameter is currently ignored as we operate on the configured cluster
	if err := e.handler.DeleteMonitoringConfig(c.Request().Context(), namespace, name); err != nil {
		e.l.Errorf("DeleteMonitoringConfig failed: %v", err)
		return err
	}

	return c.NoContent(http.StatusNoContent)
}
