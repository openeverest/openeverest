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

// GetRestore returns a specific restore.
func (e *EverestServer) GetRestore(c echo.Context, cluster string, namespace string, restore string) error {
	// The cluster parameter is currently ignored.
	result, err := e.handler.GetRestore(c.Request().Context(), namespace, restore)
	if err != nil {
		e.l.Errorf("GetRestore failed: %v", err)
		return err
	}
	return c.JSON(http.StatusOK, result)
}

// CreateRestore creates a new restore.
func (e *EverestServer) CreateRestore(c echo.Context, cluster string, namespace string) error {
	// The cluster parameter is currently ignored.
	restore := &backupv1alpha1.Restore{}
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		e.l.Errorf("CreateRestore: failed to read request body: %v", err)
		return err
	}
	if err := json.Unmarshal(body, restore); err != nil {
		e.l.Errorf("CreateRestore: failed to decode request body: %v", err)
		return err
	}

	restore.Namespace = namespace
	result, err := e.handler.CreateRestore(c.Request().Context(), restore)
	if err != nil {
		e.l.Errorf("CreateRestore failed: %v", err)
		return err
	}
	return c.JSON(http.StatusCreated, result)
}

// DeleteRestore deletes a restore.
func (e *EverestServer) DeleteRestore(c echo.Context, cluster string, namespace string, restore string) error {
	// The cluster parameter is currently ignored.
	if err := e.handler.DeleteRestore(c.Request().Context(), namespace, restore); err != nil {
		e.l.Errorf("DeleteRestore failed: %v", err)
		return err
	}
	return c.NoContent(http.StatusNoContent)
}
