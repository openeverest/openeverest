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

	"github.com/AlekSi/pointer"
	"github.com/labstack/echo/v4"
	corev1 "k8s.io/api/core/v1"

	api "github.com/openeverest/openeverest/v2/internal/server/api"
)

// CreateSecret creates a new secret in the specified namespace and cluster.
func (e *EverestServer) CreateSecret(c echo.Context, cluster, namespace string) error {
	secret := &corev1.Secret{}
	if err := c.Bind(secret); err != nil {
		e.l.Errorf("CreateSecret: failed to bind request body: %v", err)
		return err
	}

	result, err := e.handler.CreateSecret(c.Request().Context(), cluster, namespace, secret)
	if err != nil {
		e.l.Errorf("CreateSecret failed: %v", err)
		return err
	}

	return c.JSON(http.StatusCreated, result)
}

// ListSecrets lists OpenEverest-managed secrets in the namespace, optionally filtered by
// provider and definition. Only metadata is returned; secret data is never included.
func (e *EverestServer) ListSecrets(c echo.Context, cluster, namespace string, params api.ListSecretsParams) error {
	result, err := e.handler.ListSecrets(
		c.Request().Context(),
		cluster, namespace,
		pointer.GetString(params.Provider),
		pointer.GetString(params.Definition),
	)
	if err != nil {
		e.l.Errorf("ListSecrets failed: %v", err)
		return err
	}

	return c.JSON(http.StatusOK, result)
}

// GetSecret returns the secret specified by name (metadata only, no data).
func (e *EverestServer) GetSecret(c echo.Context, cluster, namespace, name string) error {
	result, err := e.handler.GetSecret(c.Request().Context(), cluster, namespace, name)
	if err != nil {
		e.l.Errorf("GetSecret failed: %v", err)
		return err
	}

	return c.JSON(http.StatusOK, result)
}

// DeleteSecret deletes the secret specified by name in the namespace and cluster.
func (e *EverestServer) DeleteSecret(c echo.Context, cluster, namespace, name string) error {
	if err := e.handler.DeleteSecret(c.Request().Context(), cluster, namespace, name); err != nil {
		e.l.Errorf("DeleteSecret failed: %v", err)
		return err
	}

	return c.NoContent(http.StatusNoContent)
}
