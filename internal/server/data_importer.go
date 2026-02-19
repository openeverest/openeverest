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
	"github.com/AlekSi/pointer"
	"github.com/labstack/echo/v4"

	api "github.com/openeverest/openeverest/v2/internal/server/api"
)

// ListDataImporters lists all data importers available in the Everest server.
func (e *EverestServer) ListDataImporters(c echo.Context, params api.ListDataImportersParams) error {
	list, err := e.handler.ListDataImporters(c.Request().Context(), pointer.Get(params.SupportedEngines)...)
	if err != nil {
		return err
	}
	return c.JSON(200, list)
}
