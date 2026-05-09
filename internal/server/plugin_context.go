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

	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

type pluginContextResponse struct {
	User       string   `json:"user"`
	Groups     []string `json:"groups"`
	Namespaces []string `json:"namespaces"`
}

// pluginContextHandler returns the current user's identity and accessible namespaces.
// Plugins use this to scope their queries per tenant.
func (e *EverestServer) pluginContextHandler(c echo.Context) error {
	user, err := rbac.GetUser(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, "unauthorized")
	}

	nsList, err := e.kubeConnector.GetDBNamespaces(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "failed to list namespaces")
	}

	namespaces := make([]string, 0, len(nsList.Items))
	for _, ns := range nsList.Items {
		namespaces = append(namespaces, ns.Name)
	}

	return c.JSON(http.StatusOK, pluginContextResponse{
		User:       user.Subject,
		Groups:     user.Groups,
		Namespaces: namespaces,
	})
}
