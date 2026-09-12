// everest
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

	"github.com/labstack/echo/v4"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
)

// ListBackupImports lists backup imports.
func (e *EverestServer) ListBackupImports(c echo.Context, cluster string, namespace string) error {
	result, err := e.handler.ListBackupImports(c.Request().Context(), cluster, namespace)
	if err != nil {
		e.l.Errorf("ListBackupImports failed: %v", err)
		return err
	}
	return c.JSON(http.StatusOK, result)
}

// GetBackupImport returns a specific backup import.
func (e *EverestServer) GetBackupImport(c echo.Context, cluster string, namespace string, name string) error {
	result, err := e.handler.GetBackupImport(c.Request().Context(), cluster, namespace, name)
	if err != nil {
		e.l.Errorf("GetBackupImport failed: %v", err)
		return err
	}
	return c.JSON(http.StatusOK, result)
}

// CreateBackupImport creates a new backup import.
func (e *EverestServer) CreateBackupImport(c echo.Context, cluster string, namespace string) error {
	backupImport := &backupv1alpha1.BackupImport{}
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		e.l.Errorf("CreateBackupImport: failed to read request body: %v", err)
		return err
	}
	if err := json.Unmarshal(body, backupImport); err != nil {
		e.l.Errorf("CreateBackupImport: failed to decode request body: %v", err)
		return err
	}

	// Ensure the namespace and name match
	backupImport.Namespace = namespace
	result, err := e.handler.CreateBackupImport(c.Request().Context(), cluster, backupImport)
	if err != nil {
		e.l.Errorf("CreateBackupImport failed: %v", err)
		return err
	}
	return c.JSON(http.StatusCreated, result)
}

// DeleteBackupImport deletes a backup import.
func (e *EverestServer) DeleteBackupImport(c echo.Context, cluster string, namespace string, name string) error {
	if err := e.handler.DeleteBackupImport(c.Request().Context(), cluster, namespace, name); err != nil {
		e.l.Errorf("DeleteBackupImport failed: %v", err)
		return err
	}
	return c.NoContent(http.StatusNoContent)
}
