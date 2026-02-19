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

// Package server ...
package server

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	api "github.com/openeverest/openeverest/v2/internal/server/api"
	"github.com/openeverest/openeverest/v2/pkg/accounts"
	"github.com/openeverest/openeverest/v2/pkg/common"
)

const (
	jwtSubjectTml    = "%s:%s" // username:capability
	jwtDefaultExpiry = time.Hour * 24
)

// CreateSession creates a new session.
func (e *EverestServer) CreateSession(ctx echo.Context) error {
	var params api.UserCredentials
	if err := ctx.Bind(&params); err != nil {
		return err
	}

	c := ctx.Request().Context()
	err := e.sessionMgr.Authenticate(c, *params.Username, *params.Password)
	if err != nil {
		e.attemptsStore.IncreaseTimeout(ctx.RealIP())
		return sessionErrToHTTPRes(ctx, err)
	}

	uniqueID, err := uuid.NewRandom()
	if err != nil {
		return err
	}
	subject := fmt.Sprintf(jwtSubjectTml, *params.Username, accounts.AccountCapabilityLogin)
	secondsBeforeExpiry := int64(jwtDefaultExpiry.Seconds())

	jwtToken, err := e.sessionMgr.Create(subject, secondsBeforeExpiry, uniqueID.String())
	if err != nil {
		return err
	}

	e.attemptsStore.CleanupVisitor(ctx.RealIP())

	return ctx.JSON(http.StatusOK, map[string]string{"token": jwtToken})
}

// DeleteSession invalidates the user token by adding it to the blocklist
func (e *EverestServer) DeleteSession(ctx echo.Context) error {
	e.attemptsStore.IncreaseTimeout(ctx.RealIP())
	c := ctx.Request().Context()
	token, err := common.ExtractToken(c)
	if err != nil {
		return err
	}
	err = e.sessionMgr.Block(c, token)
	if err != nil {
		e.l.Errorf("blocklist error: %v", err)
		return ctx.JSON(http.StatusInternalServerError, api.Error{
			Message: pointer.To("Failed to logout user"),
		})
	}

	return ctx.NoContent(http.StatusNoContent)
}

func sessionErrToHTTPRes(ctx echo.Context, err error) error {
	if errors.Is(err, accounts.ErrAccountNotFound) ||
		errors.Is(err, accounts.ErrIncorrectPassword) {
		return ctx.JSON(http.StatusUnauthorized, api.Error{
			Message: pointer.To("Incorrect username or password provided"),
		})
	}

	if errors.Is(err, accounts.ErrAccountDisabled) {
		return ctx.JSON(http.StatusForbidden, api.Error{
			Message: pointer.To("User account is disabled"),
		})
	}

	if errors.Is(err, accounts.ErrInsufficientCapabilities) {
		return ctx.JSON(http.StatusForbidden, api.Error{
			Message: pointer.To("User account lacks required capabilities"),
		})
	}
	return err
}
