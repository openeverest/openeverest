// everest
// Copyright (C) 2025 Percona LLC
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

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
)

// ListLoadBalancerConfig lists all load balancer configs.
func (e *EverestServer) ListLoadBalancerConfig(ctx echo.Context) error {
	list, err := e.handler.ListLoadBalancerConfigs(ctx.Request().Context())
	if err != nil {
		e.l.Errorf("ListLoadBalancerConfig failed: %v", err)
		return err
	}
	return ctx.JSON(http.StatusOK, list)
}

// CreateLoadBalancerConfig creates a new load balancer config.
func (e *EverestServer) CreateLoadBalancerConfig(c echo.Context) error {
	lbc := &everestv1alpha1.LoadBalancerConfig{}
	if err := e.getBodyFromContext(c, lbc); err != nil {
		return errors.Join(errFailedToReadRequestBody, err)
	}

	result, err := e.handler.CreateLoadBalancerConfig(c.Request().Context(), lbc)
	if err != nil {
		e.l.Errorf("CreateLoadBalancerConfig failed: %v", err)
		return err
	}

	return c.JSON(http.StatusOK, result)
}

// DeleteLoadBalancerConfig deletes a load balancer confi.
func (e *EverestServer) DeleteLoadBalancerConfig(c echo.Context, configName string) error {
	if err := e.handler.DeleteLoadBalancerConfig(c.Request().Context(), configName); err != nil {
		e.l.Errorf("DeleteLoadBalancerConfig failed: %v", err)
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

// GetLoadBalancerConfig retrieves a load balancer config by name.
func (e *EverestServer) GetLoadBalancerConfig(c echo.Context, configName string) error {
	result, err := e.handler.GetLoadBalancerConfig(c.Request().Context(), configName)
	if err != nil {
		e.l.Errorf("GetLoadBalancerConfig failed: %v", err)
		return err
	}
	return c.JSON(http.StatusOK, result)
}

// UpdateLoadBalancerConfig updates an existing load balancer config.
func (e *EverestServer) UpdateLoadBalancerConfig(c echo.Context, configName string) error {
	lbc := &everestv1alpha1.LoadBalancerConfig{}
	if err := e.getBodyFromContext(c, lbc); err != nil {
		return errors.Join(errFailedToReadRequestBody, err)
	}
	lbc.SetName(configName)

	result, err := e.handler.UpdateLoadBalancerConfig(c.Request().Context(), lbc)
	if err != nil {
		e.l.Errorf("UpdateLoadBalancerConfig failed: %v", err)
		return err
	}
	return c.JSON(http.StatusOK, result)
}
