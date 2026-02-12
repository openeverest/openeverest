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
	"encoding/json"
	"io"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/openeverest/openeverest/v2/pkg/apis/v1alpha1"
)

// ListInstances lists all instances in a namespace.
func (e *EverestServer) ListInstances(c echo.Context, cluster string, namespace string) error {
	// The cluster parameter is currently ignored as we operate on the configured cluster
	result, err := e.handler.ListInstances(c.Request().Context(), namespace)
	if err != nil {
		e.l.Errorf("ListInstances failed: %v", err)
		return err
	}
	return c.JSON(http.StatusOK, result)
}

// GetInstance returns a specific instance.
func (e *EverestServer) GetInstance(c echo.Context, cluster string, namespace string, instance string) error {
	// The cluster parameter is currently ignored as we operate on the configured cluster
	result, err := e.handler.GetInstance(c.Request().Context(), namespace, instance)
	if err != nil {
		e.l.Errorf("GetInstance failed: %v", err)
		return err
	}
	return c.JSON(http.StatusOK, result)
}

// CreateInstance creates a new instance.
func (e *EverestServer) CreateInstance(c echo.Context, cluster string, namespace string) error {
	// The cluster parameter is currently ignored as we operate on the configured cluster
	instance := &v1alpha1.Instance{}
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		e.l.Errorf("CreateInstance: failed to read request body: %v", err)
		return err
	}
	if err := json.Unmarshal(body, instance); err != nil {
		e.l.Errorf("CreateInstance: failed to decode request body: %v", err)
		return err
	}

	// Ensure the namespace matches
	instance.Namespace = namespace

	result, err := e.handler.CreateInstance(c.Request().Context(), instance)
	if err != nil {
		e.l.Errorf("CreateInstance failed: %v", err)
		return err
	}
	return c.JSON(http.StatusCreated, result)
}

// UpdateInstance updates an existing instance.
func (e *EverestServer) UpdateInstance(c echo.Context, cluster string, namespace string, instance string) error {
	// The cluster parameter is currently ignored as we operate on the configured cluster
	inst := &v1alpha1.Instance{}
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		e.l.Errorf("UpdateInstance: failed to read request body: %v", err)
		return err
	}
	if err := json.Unmarshal(body, inst); err != nil {
		e.l.Errorf("UpdateInstance: failed to decode request body: %v", err)
		return err
	}

	// Ensure the namespace and name match
	inst.Namespace = namespace
	inst.Name = instance

	result, err := e.handler.UpdateInstance(c.Request().Context(), inst)
	if err != nil {
		e.l.Errorf("UpdateInstance failed: %v", err)
		return err
	}
	return c.JSON(http.StatusOK, result)
}

// DeleteInstance deletes an instance.
func (e *EverestServer) DeleteInstance(c echo.Context, cluster string, namespace string, instance string) error {
	// The cluster parameter is currently ignored as we operate on the configured cluster
	if err := e.handler.DeleteInstance(c.Request().Context(), namespace, instance); err != nil {
		e.l.Errorf("DeleteInstance failed: %v", err)
		return err
	}
	return c.NoContent(http.StatusNoContent)
}
