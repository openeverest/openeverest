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

package server

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	api "github.com/openeverest/openeverest/v2/internal/server/api"
)

// ListBackupStorages lists backup storages.
func (e *EverestServer) ListBackupStoragesV1(c echo.Context, namespace string) error {
	ctx := c.Request().Context()
	list, err := e.handler.ListBackupStoragesV1(ctx, namespace)
	if err != nil {
		e.l.Errorf("ListBackupStorages failed: %v", err)
		return err
	}

	result := make([]api.BackupStorageV1, 0, len(list.Items))
	for _, s := range list.Items {
		out := &api.BackupStorageV1{}
		out.FromCR(&s)
		result = append(result, *out)
	}
	return c.JSON(http.StatusOK, result)
}

// CreateBackupStorage creates a new backup storage object.
func (e *EverestServer) CreateBackupStorageV1(c echo.Context, namespace string) error {
	ctx := c.Request().Context()
	req := api.CreateBackupStorageParams{}
	if err := c.Bind(&req); err != nil {
		return errors.Join(errFailedToReadRequestBody, err)
	}
	result, err := e.handler.CreateBackupStorageV1(ctx, namespace, &req)
	if err != nil {
		e.l.Errorf("CreateBackupStorage failed: %v", err)
		return err
	}
	out := &api.BackupStorageV1{}
	out.FromCR(result)
	return c.JSON(http.StatusCreated, out)
}

// DeleteBackupStorage deletes the specified backup storage.
func (e *EverestServer) DeleteBackupStorageV1(c echo.Context, namespace, name string) error {
	ctx := c.Request().Context()
	if err := e.handler.DeleteBackupStorageV1(ctx, namespace, name); err != nil {
		e.l.Errorf("DeleteBackupStorage failed: %v", err)
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

// GetBackupStorage retrieves the specified backup storage.
func (e *EverestServer) GetBackupStorageV1(c echo.Context, namespace, name string) error {
	ctx := c.Request().Context()
	result, err := e.handler.GetBackupStorageV1(ctx, namespace, name)
	if err != nil {
		e.l.Errorf("GetBackupStorage failed: %v", err)
		return err
	}

	out := &api.BackupStorageV1{}
	out.FromCR(result)
	return c.JSON(http.StatusOK, out)
}

// UpdateBackupStorage updates of the specified backup storage.
func (e *EverestServer) UpdateBackupStorageV1(c echo.Context, namespace, name string) error {
	ctx := c.Request().Context()
	req := api.UpdateBackupStorageParams{}
	if err := c.Bind(&req); err != nil {
		return errors.Join(errFailedToReadRequestBody, err)
	}
	result, err := e.handler.UpdateBackupStorageV1(ctx, namespace, name, &req)
	if err != nil {
		e.l.Errorf("UpdateBackupStorage failed: %v", err)
		return err
	}
	out := &api.BackupStorageV1{}
	out.FromCR(result)
	return c.JSON(http.StatusOK, out)
}
