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
	"net/http"

	"github.com/labstack/echo/v4"
)

// ListBackupClasses lists all backup classes in the cluster.
func (e *EverestServer) ListBackupClasses(c echo.Context, cluster string) error {
	// The cluster parameter is currently ignored as we operate on the configured cluster
	result, err := e.handler.ListBackupClasses(c.Request().Context())
	if err != nil {
		e.l.Errorf("ListBackupClasses failed: %v", err)
		return err
	}
	return c.JSON(http.StatusOK, result)
}

// GetBackupClass returns a specific backup class.
func (e *EverestServer) GetBackupClass(c echo.Context, cluster string, backupClass string) error {
	// The cluster parameter is currently ignored as we operate on the configured cluster
	result, err := e.handler.GetBackupClass(c.Request().Context(), backupClass)
	if err != nil {
		e.l.Errorf("GetBackupClass failed: %v", err)
		return err
	}
	return c.JSON(http.StatusOK, result)
}
