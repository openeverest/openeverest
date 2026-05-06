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
	"net/http"

	"github.com/labstack/echo/v4"
)

// ListInstanceRestores lists all restores for a specific instance.
func (e *EverestServer) ListInstanceRestores(c echo.Context, cluster string, namespace string, instance string) error {
	// The cluster parameter is currently ignored.
	result, err := e.handler.ListInstanceRestores(c.Request().Context(), namespace, instance)
	if err != nil {
		e.l.Errorf("ListInstanceRestores failed: %v", err)
		return err
	}
	return c.JSON(http.StatusOK, result)
}
