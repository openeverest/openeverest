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
	"github.com/percona/everest/api"
)

// ListPodSchedulingPolicy lists all pod scheduling policies.
func (e *EverestServer) ListPodSchedulingPolicy(ctx echo.Context, params api.ListPodSchedulingPolicyParams) error {
	list, err := e.handler.ListPodSchedulingPolicies(ctx.Request().Context(), &params)
	if err != nil {
		e.l.Errorf("ListPodSchedulingPolicies failed: %v", err)
		return err
	}
	return ctx.JSON(http.StatusOK, list)
}

// CreatePodSchedulingPolicy creates a new pod scheduling policy.
func (e *EverestServer) CreatePodSchedulingPolicy(c echo.Context) error {
	psp := &everestv1alpha1.PodSchedulingPolicy{}
	if err := e.getBodyFromContext(c, psp); err != nil {
		return errors.Join(errFailedToReadRequestBody, err)
	}

	result, err := e.handler.CreatePodSchedulingPolicy(c.Request().Context(), psp)
	if err != nil {
		e.l.Errorf("CreatePodSchedulingPolicy failed: %v", err)
		return err
	}

	return c.JSON(http.StatusOK, result)
}

// DeletePodSchedulingPolicy deletes a pod scheduling policy.
func (e *EverestServer) DeletePodSchedulingPolicy(c echo.Context, policyName string) error {
	if err := e.handler.DeletePodSchedulingPolicy(c.Request().Context(), policyName); err != nil {
		e.l.Errorf("DeletePodSchedulingPolicy failed: %v", err)
		return err
	}
	return c.NoContent(http.StatusNoContent)
}

// GetPodSchedulingPolicy retrieves a pod scheduling policy by name.
func (e *EverestServer) GetPodSchedulingPolicy(c echo.Context, policyName string) error {
	result, err := e.handler.GetPodSchedulingPolicy(c.Request().Context(), policyName)
	if err != nil {
		e.l.Errorf("GetPodSchedulingPolicy failed: %v", err)
		return err
	}
	return c.JSON(http.StatusOK, result)
}

// UpdatePodSchedulingPolicy updates an existing pod scheduling policy.
func (e *EverestServer) UpdatePodSchedulingPolicy(c echo.Context, policyName string) error {
	psp := &everestv1alpha1.PodSchedulingPolicy{}
	if err := e.getBodyFromContext(c, psp); err != nil {
		return errors.Join(errFailedToReadRequestBody, err)
	}
	psp.SetName(policyName)

	result, err := e.handler.UpdatePodSchedulingPolicy(c.Request().Context(), psp)
	if err != nil {
		e.l.Errorf("UpdatePodSchedulingPolicy failed: %v", err)
		return err
	}
	return c.JSON(http.StatusOK, result)
}
