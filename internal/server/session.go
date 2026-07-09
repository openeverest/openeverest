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

// Package server ...
package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	api "github.com/openeverest/openeverest/v2/internal/server/api"
	"github.com/openeverest/openeverest/v2/pkg/accounts"
	"github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/pkg/events"
)

const (
	jwtDefaultExpiry = time.Hour * 24
)

// CreateSession creates a new session.
func (e *EverestServer) CreateSession(ctx echo.Context) error {
	var params api.UserCredentials
	if err := ctx.Bind(&params); err != nil {
		return err
	}

	c := ctx.Request().Context()
	username := ""
	if params.Username != nil {
		username = *params.Username
	}
	password := ""
	if params.Password != nil {
		password = *params.Password
	}
	err := e.sessionMgr.Authenticate(c, username, password)
	if err != nil {
		e.attemptsStore.IncreaseTimeout(ctx.RealIP())
		e.publishAuthEvent(events.UserLoginFailed, username, ctx.RealIP(), err.Error())
		return sessionErrToHTTPRes(ctx, err)
	}

	uniqueID, err := uuid.NewRandom()
	if err != nil {
		return err
	}
	subject := fmt.Sprintf(jwtSubjectTml, username, accounts.AccountCapabilityLogin)
	secondsBeforeExpiry := int64(jwtDefaultExpiry.Seconds())

	jwtToken, err := e.sessionMgr.Create(subject, secondsBeforeExpiry, uniqueID.String())
	if err != nil {
		return err
	}

	e.attemptsStore.CleanupVisitor(ctx.RealIP())
	e.publishAuthEvent(events.UserLogin, username, ctx.RealIP(), "")

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

	e.publishAuthEvent(events.UserLogout, subjectFromJWT(token), ctx.RealIP(), "")
	return ctx.NoContent(http.StatusNoContent)
}

// publishAuthEvent emits an authentication-related event onto the hub.
// `reason` is included only for the login-failed type.
func (e *EverestServer) publishAuthEvent(t events.Type, subject, remoteIP, reason string) {
	if e.eventHub == nil {
		return
	}
	evt := events.Event{
		Type:       t,
		OccurredAt: time.Now().UTC(),
		Resource: events.ResourceRef{
			Kind: "Session",
			Name: subject,
		},
		Actor: events.Actor{Type: "user", ID: subject},
	}
	if reason != "" {
		evt.NewState = events.StateSnapshot{Phase: reason}
	}
	if remoteIP != "" {
		// Use prevState slot to carry the IP without inventing new schema
		// fields. Audit plugins read both prev/new phases for free-form context.
		evt.PrevState = events.StateSnapshot{Phase: "ip=" + remoteIP}
	}
	e.eventHub.Publish(evt)
}

// subjectFromJWT returns the username portion of a "username:capability"
// JWT subject claim, or "" if it cannot be parsed.
func subjectFromJWT(token *jwt.Token) string {
	if token == nil {
		return ""
	}
	sub, err := token.Claims.GetSubject()
	if err != nil || sub == "" {
		return ""
	}
	for i := 0; i < len(sub); i++ {
		if sub[i] == ':' {
			return sub[:i]
		}
	}
	return sub
}
