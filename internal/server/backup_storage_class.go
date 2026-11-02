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
	"net/http"

	"github.com/labstack/echo/v4"
)

// ListBackupStorageClasses lists the backup storage classes in the cluster.
func (e *EverestServer) ListBackupStorageClasses(c echo.Context, cluster string) error {
	result, err := e.handler.ListBackupStorageClasses(c.Request().Context(), cluster)
	if err != nil {
		e.l.Errorf("ListBackupStorageClasses failed: %v", err)
		return err
	}
	return c.JSON(http.StatusOK, result)
}

// GetBackupStorageClass returns a specific backup storage class.
func (e *EverestServer) GetBackupStorageClass(c echo.Context, cluster, name string) error {
	result, err := e.handler.GetBackupStorageClass(c.Request().Context(), cluster, name)
	if err != nil {
		e.l.Errorf("GetBackupStorageClass failed: %v", err)
		return err
	}
	return c.JSON(http.StatusOK, result)
}
