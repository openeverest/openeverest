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

	"github.com/AlekSi/pointer"
	"github.com/labstack/echo/v4"

	api "github.com/openeverest/openeverest/v2/internal/server/api"
)

// ListInstancePresets lists all instance presets in the cluster, optionally filtered by provider.
func (e *EverestServer) ListInstancePresets(c echo.Context, cluster string, params api.ListInstancePresetsParams) error {
	result, err := e.handler.ListInstancePresets(c.Request().Context(), cluster, pointer.GetString(params.Provider))
	if err != nil {
		e.l.Errorf("ListInstancePresets failed: %v", err)
		return err
	}
	return c.JSON(http.StatusOK, result)
}

// GetInstancePreset returns a specific instance preset.
func (e *EverestServer) GetInstancePreset(c echo.Context, cluster string, name string) error {
	result, err := e.handler.GetInstancePreset(c.Request().Context(), cluster, name)
	if err != nil {
		e.l.Errorf("GetInstancePreset failed: %v", err)
		return err
	}
	return c.JSON(http.StatusOK, result)
}

// ResolveInstancePreset returns an instance preset with namespace-specific default values populated.
func (e *EverestServer) ResolveInstancePreset(c echo.Context, cluster string, name string, params api.ResolveInstancePresetParams) error {
	result, err := e.handler.ResolveInstancePreset(c.Request().Context(), cluster, name, params.Namespace)
	if err != nil {
		e.l.Errorf("ResolveInstancePreset failed: %v", err)
		return err
	}
	return c.JSON(http.StatusOK, result)
}
