// everest
// Copyright (C) 2023 Percona LLC
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

// GetBackup returns a specific backup.
func (e *EverestServer) GetBackup(c echo.Context, cluster string, namespace string, backup string) error {
	result, err := e.handler.GetBackup(c.Request().Context(), cluster, namespace, backup)
	if err != nil {
		e.l.Errorf("GetBackup failed: %v", err)
		return err
	}
	return c.JSON(http.StatusOK, result)
}

// CreateBackup creates a new backup.
func (e *EverestServer) CreateBackup(c echo.Context, cluster string, namespace string) error {
	backup := &backupv1alpha1.Backup{}
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		e.l.Errorf("CreateBackup: failed to read request body: %v", err)
		return err
	}
	if err := json.Unmarshal(body, backup); err != nil {
		e.l.Errorf("CreateBackup: failed to decode request body: %v", err)
		return err
	}

	backup.Namespace = namespace
	result, err := e.handler.CreateBackup(c.Request().Context(), cluster, backup)
	if err != nil {
		e.l.Errorf("CreateBackup failed: %v", err)
		return err
	}
	return c.JSON(http.StatusCreated, result)
}

// DeleteBackup deletes a backup.
func (e *EverestServer) DeleteBackup(c echo.Context, cluster string, namespace string, backup string) error {
	if err := e.handler.DeleteBackup(c.Request().Context(), cluster, namespace, backup); err != nil {
		e.l.Errorf("DeleteBackup failed: %v", err)
		return err
	}
	return c.NoContent(http.StatusNoContent)
}
