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

package server

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	api "github.com/openeverest/openeverest/v2/internal/server/api"
)

// ListBackupStorages lists backup storages.
func (e *EverestServer) ListBackupStorages(c echo.Context, namespace string) error {
	ctx := c.Request().Context()
	list, err := e.handler.ListBackupStorages(ctx, namespace)
	if err != nil {
		e.l.Errorf("ListBackupStorages failed: %v", err)
		return err
	}

	result := make([]api.BackupStorage, 0, len(list.Items))
	for _, s := range list.Items {
		out := &api.BackupStorage{}
		out.FromCR(&s)
		result = append(result, *out)
	}
	return c.JSON(http.StatusOK, result)
}

// CreateBackupStorage creates a new backup storage object.
func (e *EverestServer) CreateBackupStorage(c echo.Context, namespace string) error {
	ctx := c.Request().Context()
	req := api.CreateBackupStorageParams{}
	if err := c.Bind(&req); err != nil {
		return errors.Join(errFailedToReadRequestBody, err)
	}
	result, err := e.handler.CreateBackupStorage(ctx, namespace, &req)
	if err != nil {
		e.l.Errorf("CreateBackupStorage failed: %v", err)
		return err
	}
	out := &api.BackupStorage{}
	out.FromCR(result)
	return c.JSON(http.StatusCreated, out)
}

// DeleteBackupStorage deletes the specified backup storage.
func (e *EverestServer) DeleteBackupStorage(c echo.Context, namespace, name string) error {
	ctx := c.Request().Context()
	if err := e.handler.DeleteBackupStorage(ctx, namespace, name); err != nil {
		e.l.Errorf("DeleteBackupStorage failed: %v", err)
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

// GetBackupStorage retrieves the specified backup storage.
func (e *EverestServer) GetBackupStorage(c echo.Context, namespace, name string) error {
	ctx := c.Request().Context()
	result, err := e.handler.GetBackupStorage(ctx, namespace, name)
	if err != nil {
		e.l.Errorf("GetBackupStorage failed: %v", err)
		return err
	}

	out := &api.BackupStorage{}
	out.FromCR(result)
	return c.JSON(http.StatusOK, out)
}

// UpdateBackupStorage updates of the specified backup storage.
func (e *EverestServer) UpdateBackupStorage(c echo.Context, namespace, name string) error {
	ctx := c.Request().Context()
	req := api.UpdateBackupStorageParams{}
	if err := c.Bind(&req); err != nil {
		return errors.Join(errFailedToReadRequestBody, err)
	}
	result, err := e.handler.UpdateBackupStorage(ctx, namespace, name, &req)
	if err != nil {
		e.l.Errorf("UpdateBackupStorage failed: %v", err)
		return err
	}
	out := &api.BackupStorage{}
	out.FromCR(result)
	return c.JSON(http.StatusOK, out)
}
