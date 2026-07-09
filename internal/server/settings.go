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

	api "github.com/openeverest/openeverest/v2/internal/server/api"
)

// GetSettings returns the Everest global settings.
func (e *EverestServer) GetSettings(ctx echo.Context) error {
	result, err := e.handler.GetSettings(ctx.Request().Context())
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, api.Error{
			Message: new("Failed to get Everest settings"),
		})
	}
	return ctx.JSON(http.StatusOK, result)
}
