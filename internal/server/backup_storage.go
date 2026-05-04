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

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
)

// ListBackupStorages lists all backup storages in a namespace.
func (e *EverestServer) ListBackupStorages(c echo.Context, cluster, namespace string) error {
	// The cluster parameter is currently ignored.
	result, err := e.handler.ListBackupStorages(c.Request().Context(), namespace)
	if err != nil {
		e.l.Errorf("ListBackupStorages failed: %v", err)
		return err
	}
	return c.JSON(http.StatusOK, result)
}

// CreateBackupStorage creates a new backup storage.
func (e *EverestServer) CreateBackupStorage(c echo.Context, cluster, namespace string) error {
	// The cluster parameter is currently ignored.
	bs := &backupv1alpha1.BackupStorage{}
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		e.l.Errorf("CreateBackupStorage: failed to read request body: %v", err)
		return err
	}
	if err := json.Unmarshal(body, bs); err != nil {
		e.l.Errorf("CreateBackupStorage: failed to decode request body: %v", err)
		return err
	}

	bs.Namespace = namespace
	result, err := e.handler.CreateBackupStorage(c.Request().Context(), bs)
	if err != nil {
		e.l.Errorf("CreateBackupStorage failed: %v", err)
		return err
	}
	return c.JSON(http.StatusCreated, result)
}

// GetBackupStorage retrieves a specific backup storage.
func (e *EverestServer) GetBackupStorage(c echo.Context, cluster, namespace, name string) error {
	// The cluster parameter is currently ignored.
	result, err := e.handler.GetBackupStorage(c.Request().Context(), namespace, name)
	if err != nil {
		e.l.Errorf("GetBackupStorage failed: %v", err)
		return err
	}
	return c.JSON(http.StatusOK, result)
}

// UpdateBackupStorage updates a backup storage.
func (e *EverestServer) UpdateBackupStorage(c echo.Context, cluster, namespace, name string) error {
	// The cluster parameter is currently ignored.
	bs := &backupv1alpha1.BackupStorage{}
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		e.l.Errorf("UpdateBackupStorage: failed to read request body: %v", err)
		return err
	}
	if err := json.Unmarshal(body, bs); err != nil {
		e.l.Errorf("UpdateBackupStorage: failed to decode request body: %v", err)
		return err
	}

	bs.Namespace = namespace
	bs.Name = name
	result, err := e.handler.UpdateBackupStorage(c.Request().Context(), bs)
	if err != nil {
		e.l.Errorf("UpdateBackupStorage failed: %v", err)
		return err
	}
	return c.JSON(http.StatusOK, result)
}

// DeleteBackupStorage deletes a backup storage.
func (e *EverestServer) DeleteBackupStorage(c echo.Context, cluster, namespace, name string) error {
	// The cluster parameter is currently ignored.
	if err := e.handler.DeleteBackupStorage(c.Request().Context(), namespace, name); err != nil {
		e.l.Errorf("DeleteBackupStorage failed: %v", err)
		return err
	}
	return c.NoContent(http.StatusNoContent)
}
