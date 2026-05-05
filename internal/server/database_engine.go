// everest
// Copyright (C) 2023 Percona LLC
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
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
)

// ListDatabaseEngines List of the available database engines on the specified namespace.
func (e *EverestServer) ListDatabaseEngines(ctx echo.Context, namespace string) error {
	result, err := e.handler.ListDatabaseEngines(ctx.Request().Context(), namespace)
	if err != nil {
		e.l.Errorf("ListDatabaseEngines failed: %w", err)
		return err
	}
	return ctx.JSON(http.StatusOK, result)
}

// GetDatabaseEngine Get the specified database engine on the specified namespace.
func (e *EverestServer) GetDatabaseEngine(ctx echo.Context, namespace, name string) error {
	result, err := e.handler.GetDatabaseEngine(ctx.Request().Context(), namespace, name)
	if err != nil {
		e.l.Errorf("GetDatabaseEngine failed: %w", err)
		return err
	}
	return ctx.JSON(http.StatusOK, result)
}

// UpdateDatabaseEngine Update the specified database engine on the specified namespace.
//
//nolint:dupl
func (e *EverestServer) UpdateDatabaseEngine(ctx echo.Context, namespace, name string) error {
	dbe := &everestv1alpha1.DatabaseEngine{}
	if err := e.getBodyFromContext(ctx, dbe); err != nil {
		return errors.Join(errFailedToReadRequestBody, err)
	}
	dbe.SetNamespace(namespace)
	dbe.SetName(name)

	result, err := e.handler.UpdateDatabaseEngine(ctx.Request().Context(), dbe)
	if err != nil {
		e.l.Errorf("UpdateDatabaseEngine failed: %w", err)
		return err
	}
	return ctx.JSON(http.StatusOK, result)
}

// GetUpgradePlan gets the upgrade plan for the given namespace.
func (e *EverestServer) GetUpgradePlan(
	ctx echo.Context,
	namespace string,
) error {
	result, err := e.handler.GetUpgradePlan(ctx.Request().Context(), namespace)
	if err != nil {
		e.l.Errorf("GetUpgradePlan failed: %w", err)
		return err
	}
	return ctx.JSON(http.StatusOK, result)
}

// ApproveUpgradePlan starts the upgrade of operators in the provided namespace.
func (e *EverestServer) ApproveUpgradePlan(ctx echo.Context, namespace string) error {
	if err := e.handler.ApproveUpgradePlan(ctx.Request().Context(), namespace); err != nil {
		e.l.Errorf("ApproveUpgradePlan failed: %w", err)
		return err
	}
	return nil
}
