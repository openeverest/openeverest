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
	"encoding/json"
	"io"
	"net/http"

	"github.com/AlekSi/pointer"
	"github.com/labstack/echo/v4"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	api "github.com/openeverest/openeverest/v2/internal/server/api"
)

// ListClusterInstancePresets lists all cluster instance presets in the cluster, optionally filtered by provider.
func (e *EverestServer) ListClusterInstancePresets(c echo.Context, cluster string, params api.ListClusterInstancePresetsParams) error {
	result, err := e.handler.ListClusterInstancePresets(c.Request().Context(), cluster, pointer.GetString(params.Provider))
	if err != nil {
		e.l.Errorf("ListClusterInstancePresets failed: %v", err)
		return err
	}
	return c.JSON(http.StatusOK, result)
}

// GetClusterInstancePreset returns a specific cluster instance preset.
func (e *EverestServer) GetClusterInstancePreset(c echo.Context, cluster string, name string) error {
	result, err := e.handler.GetClusterInstancePreset(c.Request().Context(), cluster, name)
	if err != nil {
		e.l.Errorf("GetClusterInstancePreset failed: %v", err)
		return err
	}
	return c.JSON(http.StatusOK, result)
}

// ResolveClusterInstancePreset returns a cluster instance preset with namespace-specific default values populated.
func (e *EverestServer) ResolveClusterInstancePreset(c echo.Context, cluster string, name string, params api.ResolveClusterInstancePresetParams) error {
	result, err := e.handler.ResolveClusterInstancePreset(c.Request().Context(), cluster, name, params.Namespace)
	if err != nil {
		e.l.Errorf("ResolveClusterInstancePreset failed: %v", err)
		return err
	}
	return c.JSON(http.StatusOK, result)
}

// ListInstancePresets lists all namespace-scoped instance presets, optionally filtered by provider.
func (e *EverestServer) ListInstancePresets(c echo.Context, cluster string, namespace string, params api.ListInstancePresetsParams) error {
	result, err := e.handler.ListInstancePresets(c.Request().Context(), cluster, namespace, pointer.GetString(params.Provider))
	if err != nil {
		e.l.Errorf("ListInstancePresets failed: %v", err)
		return err
	}
	return c.JSON(http.StatusOK, result)
}

// GetInstancePreset returns a specific namespace-scoped instance preset.
func (e *EverestServer) GetInstancePreset(c echo.Context, cluster string, namespace string, name string) error {
	result, err := e.handler.GetInstancePreset(c.Request().Context(), cluster, namespace, name)
	if err != nil {
		e.l.Errorf("GetInstancePreset failed: %v", err)
		return err
	}
	return c.JSON(http.StatusOK, result)
}

// CreateInstancePreset creates a new namespace-scoped instance preset.
func (e *EverestServer) CreateInstancePreset(c echo.Context, cluster string, namespace string) error {
	preset := &corev1alpha1.InstancePreset{}
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		e.l.Errorf("CreateInstancePreset: failed to read request body: %v", err)
		return err
	}
	if err := json.Unmarshal(body, preset); err != nil {
		e.l.Errorf("CreateInstancePreset: failed to decode request body: %v", err)
		return err
	}

	// Ensure the namespace matches
	preset.Namespace = namespace

	result, err := e.handler.CreateInstancePreset(c.Request().Context(), cluster, preset)
	if err != nil {
		e.l.Errorf("CreateInstancePreset failed: %v", err)
		return err
	}
	return c.JSON(http.StatusCreated, result)
}

// UpdateInstancePreset updates an existing namespace-scoped instance preset.
func (e *EverestServer) UpdateInstancePreset(c echo.Context, cluster string, namespace string, name string) error {
	preset := &corev1alpha1.InstancePreset{}
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		e.l.Errorf("UpdateInstancePreset: failed to read request body: %v", err)
		return err
	}
	if err := json.Unmarshal(body, preset); err != nil {
		e.l.Errorf("UpdateInstancePreset: failed to decode request body: %v", err)
		return err
	}

	// Ensure the namespace and name match
	preset.Namespace = namespace
	preset.Name = name

	result, err := e.handler.UpdateInstancePreset(c.Request().Context(), cluster, preset)
	if err != nil {
		e.l.Errorf("UpdateInstancePreset failed: %v", err)
		return err
	}
	return c.JSON(http.StatusOK, result)
}

// DeleteInstancePreset deletes a namespace-scoped instance preset.
func (e *EverestServer) DeleteInstancePreset(c echo.Context, cluster string, namespace string, name string) error {
	if err := e.handler.DeleteInstancePreset(c.Request().Context(), cluster, namespace, name); err != nil {
		e.l.Errorf("DeleteInstancePreset failed: %v", err)
		return err
	}
	return c.NoContent(http.StatusNoContent)
}
