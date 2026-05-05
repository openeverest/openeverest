// everest
// Copyright (C) 2025 Percona LLC
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

// Package server contains the API engine features implementation.
package server

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	enginefeaturesv1alpha1 "github.com/percona/everest-operator/api/enginefeatures.everest/v1alpha1"
	"github.com/percona/everest/api"
)

// CreateSplitHorizonDNSConfig handles the creation of a SplitHorizonDNSConfig resource.
func (e *EverestServer) CreateSplitHorizonDNSConfig(ctx echo.Context, namespace string) error {
	shdc := &enginefeaturesv1alpha1.SplitHorizonDNSConfig{}
	if err := e.getBodyFromContext(ctx, shdc); err != nil {
		return errors.Join(errFailedToReadRequestBody, err)
	}

	shdc.SetNamespace(namespace)

	result, err := e.handler.CreateSplitHorizonDNSConfig(ctx.Request().Context(), shdc)
	if err != nil {
		e.l.Errorf("CreateSplitHorizonDNSConfig failed: %v", err)
		return err
	}

	return ctx.JSON(http.StatusOK, result)
}

// ListSplitHorizonDNSConfigs handles listing SplitHorizonDNSConfig resources in a given namespace.
func (e *EverestServer) ListSplitHorizonDNSConfigs(ctx echo.Context, namespace string) error {
	list, err := e.handler.ListSplitHorizonDNSConfigs(ctx.Request().Context(), namespace)
	if err != nil {
		e.l.Errorf("ListSplitHorizonDNSConfigs failed: %v", err)
		return err
	}
	return ctx.JSON(http.StatusOK, list)
}

// GetSplitHorizonDNSConfig handles retrieving a specific SplitHorizonDNSConfig resource by name.
func (e *EverestServer) GetSplitHorizonDNSConfig(ctx echo.Context, namespace string, name string) error {
	result, err := e.handler.GetSplitHorizonDNSConfig(ctx.Request().Context(), namespace, name)
	if err != nil {
		e.l.Errorf("GetSplitHorizonDNSConfig failed: %v", err)
		return err
	}
	return ctx.JSON(http.StatusOK, result)
}

// UpdateSplitHorizonDNSConfig handles updating a specific SplitHorizonDNSConfig resource.
func (e *EverestServer) UpdateSplitHorizonDNSConfig(ctx echo.Context, namespace, name string) error {
	req := &api.SplitHorizonDNSConfigUpdateParams{}
	if err := ctx.Bind(req); err != nil {
		return errors.Join(errFailedToReadRequestBody, err)
	}

	result, err := e.handler.UpdateSplitHorizonDNSConfig(ctx.Request().Context(), namespace, name, req)
	if err != nil {
		e.l.Errorf("UpdateSplitHorizonDNSConfig failed: %v", err)
		return err
	}
	return ctx.JSON(http.StatusOK, result)
}

// DeleteSplitHorizonDNSConfig handles deleting a specific SplitHorizonDNSConfig resource by name.
func (e *EverestServer) DeleteSplitHorizonDNSConfig(ctx echo.Context, namespace, name string) error {
	if err := e.handler.DeleteSplitHorizonDNSConfig(ctx.Request().Context(), namespace, name); err != nil {
		e.l.Errorf("DeleteSplitHorizonDNSConfig failed: %v", err)
		return err
	}
	return ctx.NoContent(http.StatusNoContent)
}
