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
	corev1 "k8s.io/api/core/v1"

	api "github.com/openeverest/openeverest/v2/internal/server/api"
)

// CreateConfigMap creates a new ConfigMap in the specified cluster and namespace.
func (e *EverestServer) CreateConfigMap(ctx echo.Context, cluster, namespace string) error {
	var configMap corev1.ConfigMap
	if err := ctx.Bind(&configMap); err != nil {
		e.l.Errorf("CreateConfigMap failed: failed to read request body %v", err)

		return err
	}

	result, err := e.handler.CreateConfigMap(ctx.Request().Context(), cluster, namespace, &configMap)
	if err != nil {
		e.l.Errorf("CreateConfigMap failed: %v", err)

		return err
	}

	return ctx.JSON(http.StatusCreated, result)
}

// ListConfigMaps lists OpenEverest-managed ConfigMaps in the namespace, optionally filtered by
// provider and definition.
func (e *EverestServer) ListConfigMaps(ctx echo.Context, cluster, namespace string, params api.ListConfigMapsParams) error {
	result, err := e.handler.ListConfigMaps(
		ctx.Request().Context(),
		cluster,
		namespace,
		pointer.GetString(params.Provider),
		pointer.GetString(params.Definition),
	)
	if err != nil {
		e.l.Errorf("ListConfigMaps failed: %v", err)

		return err
	}

	return ctx.JSON(http.StatusOK, result)
}

// GetConfigMap gets OpenEverest-managed ConfigMaps in the namespace.
func (e *EverestServer) GetConfigMap(ctx echo.Context, cluster, namespace, name string) error {
	result, err := e.handler.GetConfigMap(ctx.Request().Context(), cluster, namespace, name)
	if err != nil {
		e.l.Errorf("GetConfigMap failed: %v", err)

		return err
	}

	return ctx.JSON(http.StatusOK, result)
}

// DeleteConfigMap deletes an OpenEverest-managed ConfigMap in the namespace.
func (e *EverestServer) DeleteConfigMap(ctx echo.Context, cluster, namespace, name string) error {
	if err := e.handler.DeleteConfigMap(ctx.Request().Context(), cluster, namespace, name); err != nil {
		e.l.Errorf("DeleteConfigMap failed: %v", err)

		return err
	}

	return ctx.NoContent(http.StatusNoContent)
}
