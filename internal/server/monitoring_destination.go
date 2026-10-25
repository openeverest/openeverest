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

	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
)

// ListMonitoringDestinations lists all monitoring destinations in a namespace.
func (e *EverestServer) ListMonitoringDestinations(c echo.Context, cluster, namespace string) error {
	result, err := e.handler.ListMonitoringDestinations(c.Request().Context(), cluster, namespace)
	if err != nil {
		e.l.Errorf("ListMonitoringDestinations failed: %v", err)
		return err
	}
	return c.JSON(http.StatusOK, result)
}

// CreateMonitoringDestination creates a new monitoring destination.
func (e *EverestServer) CreateMonitoringDestination(c echo.Context, cluster, namespace string) error {
	destination := &monitoringv1alpha1.MonitoringDestination{}
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		e.l.Errorf("CreateMonitoringDestination: failed to read request body: %v", err)
		return err
	}
	if err := json.Unmarshal(body, destination); err != nil {
		e.l.Errorf("CreateMonitoringDestination: failed to decode request body: %v", err)
		return err
	}

	result, err := e.handler.CreateMonitoringDestination(c.Request().Context(), cluster, namespace, destination)
	if err != nil {
		e.l.Errorf("CreateMonitoringDestination failed: %v", err)
		return err
	}
	return c.JSON(http.StatusCreated, result)
}

// GetMonitoringDestination retrieves a specific monitoring destination.
func (e *EverestServer) GetMonitoringDestination(c echo.Context, cluster, namespace, name string) error {
	result, err := e.handler.GetMonitoringDestination(c.Request().Context(), cluster, namespace, name)
	if err != nil {
		e.l.Errorf("GetMonitoringDestination failed: %v", err)
		return err
	}
	return c.JSON(http.StatusOK, result)
}

// UpdateMonitoringDestination updates a monitoring destination.
func (e *EverestServer) UpdateMonitoringDestination(c echo.Context, cluster, namespace, name string) error {
	destination := &monitoringv1alpha1.MonitoringDestination{}
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		e.l.Errorf("UpdateMonitoringDestination: failed to read request body: %v", err)
		return err
	}
	if err := json.Unmarshal(body, destination); err != nil {
		e.l.Errorf("UpdateMonitoringDestination: failed to decode request body: %v", err)
		return err
	}

	result, err := e.handler.UpdateMonitoringDestination(c.Request().Context(), cluster, namespace, name, destination)
	if err != nil {
		e.l.Errorf("UpdateMonitoringDestination failed: %v", err)
		return err
	}
	return c.JSON(http.StatusOK, result)
}

// DeleteMonitoringDestination deletes a monitoring destination.
func (e *EverestServer) DeleteMonitoringDestination(c echo.Context, cluster, namespace, name string) error {
	if err := e.handler.DeleteMonitoringDestination(c.Request().Context(), cluster, namespace, name); err != nil {
		e.l.Errorf("DeleteMonitoringDestination failed: %v", err)
		return err
	}
	return c.NoContent(http.StatusNoContent)
}
