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

// ListDatabaseClusterRestores List of the created database cluster restores on the specified kubernetes cluster.
func (e *EverestServer) ListDatabaseClusterRestores(ctx echo.Context, namespace, name string) error {
	result, err := e.handler.ListDatabaseClusterRestores(ctx.Request().Context(), namespace, name)
	if err != nil {
		e.l.Errorf("ListDatabaseClusterRestores failed: %w", err)
		return err
	}
	return ctx.JSON(http.StatusOK, result)
}

// CreateDatabaseClusterRestore Create a database cluster restore on the specified kubernetes cluster.
func (e *EverestServer) CreateDatabaseClusterRestore(ctx echo.Context, namespace string) error {
	restore := &everestv1alpha1.DatabaseClusterRestore{}
	if err := e.getBodyFromContext(ctx, restore); err != nil {
		return errors.Join(errFailedToReadRequestBody, err)
	}
	restore.SetNamespace(namespace)

	result, err := e.handler.CreateDatabaseClusterRestore(ctx.Request().Context(), restore)
	if err != nil {
		e.l.Errorf("CreateDatabaseClusterRestore failed: %w", err)
		return err
	}
	return ctx.JSON(http.StatusCreated, result)
}

// DeleteDatabaseClusterRestore Delete the specified cluster restore on the specified kubernetes cluster.
func (e *EverestServer) DeleteDatabaseClusterRestore(ctx echo.Context, namespace, name string) error {
	if err := e.handler.DeleteDatabaseClusterRestore(ctx.Request().Context(), namespace, name); err != nil {
		e.l.Errorf("DeleteDatabaseClusterRestore failed: %w", err)
		return err
	}
	return ctx.NoContent(http.StatusNoContent)
}

// GetDatabaseClusterRestore Returns the specified cluster restore on the specified kubernetes cluster.
func (e *EverestServer) GetDatabaseClusterRestore(ctx echo.Context, namespace, name string) error {
	rs, err := e.handler.GetDatabaseClusterRestore(ctx.Request().Context(), namespace, name)
	if err != nil {
		e.l.Errorf("GetDatabaseClusterRestore failed: %w", err)
		return err
	}
	return ctx.JSON(http.StatusOK, rs)
}

// UpdateDatabaseClusterRestore Replace the specified cluster restore on the specified kubernetes cluster.
func (e *EverestServer) UpdateDatabaseClusterRestore(ctx echo.Context, namespace, name string) error {
	restore := &everestv1alpha1.DatabaseClusterRestore{}
	if err := e.getBodyFromContext(ctx, restore); err != nil {
		return errors.Join(errFailedToReadRequestBody, err)
	}
	restore.SetNamespace(namespace)
	restore.SetName(name)

	result, err := e.handler.UpdateDatabaseClusterRestore(ctx.Request().Context(), restore)
	if err != nil {
		e.l.Errorf("UpdateDatabaseClusterRestore failed: %w", err)
		return err
	}
	return ctx.JSON(http.StatusOK, result)
}
