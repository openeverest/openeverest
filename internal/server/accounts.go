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
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/AlekSi/pointer"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/percona/everest/api"
	"github.com/percona/everest/pkg/accounts"
	"github.com/percona/everest/pkg/common"
	"github.com/percona/everest/pkg/session"
)

func (e *EverestServer) GetUserAccount(c echo.Context, username string) error {
	token, err := common.ExtractToken(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusForbidden, api.Error{
			Message: pointer.ToString(err.Error()),
		})
	}

	claims, err := common.ExtractClaims(token)
	if err != nil {
		return c.JSON(http.StatusForbidden, api.Error{
			Message: pointer.ToString(err.Error()),
		})
	}

	tokeSubject, err := claims.GetSubject()
	if err != nil {
		return c.JSON(http.StatusForbidden, api.Error{
			Message: pointer.ToString(err.Error()),
		})
	}

	subject, _, ok := strings.Cut(tokeSubject, ":")
	if !ok {
		return errors.New("cannot find subject in token")
	}

	if subject != username {
		return c.JSON(http.StatusForbidden, api.Error{
			Message: pointer.ToString("invalid token"),
		})
	}

	isSecure, err := e.sessionMgr.IsSecure(c.Request().Context(), username)
	if err != nil {
		return err
	}

	issuer, err := claims.GetIssuer()
	if err != nil {
		return err
	}

	authSource := api.Local
	if issuer == session.SessionManagerClaimsIssuer {
		authSource = api.Local
	} else {
		authSource = api.Oidc
	}

	return c.JSON(http.StatusOK, api.Account{
		AuthSource:       authSource,
		PasswordInsecure: !isSecure,
		Username:         username,
	})
}

func (e *EverestServer) SetUserPassword(ctx echo.Context, username string) error {
	var params api.SetUserPasswordJSONBody
	if err := ctx.Bind(&params); err != nil {
		return err
	}

	c := ctx.Request().Context()
	err := e.sessionMgr.Authenticate(c, username, params.OldPassword)
	if err != nil {
		e.attemptsStore.IncreaseTimeout(ctx.RealIP())
		return sessionErrToHTTPRes(ctx, err)
	}

	isSecure, err := e.sessionMgr.IsSecure(c, username)
	if err != nil {
		return err
	}

	if err := accounts.ValidatePassword(params.NewPassword); err != nil {
		return err
	}

	e.l.Infof("Setting a new password for user '%s'", username)
	if err := e.kubeConnector.Accounts().SetPassword(c, username, params.NewPassword, isSecure); err != nil {
		return err
	}
	e.l.Infof("Password for user '%s' has been set succesfully", username)

	uniqueID, err := uuid.NewRandom()
	if err != nil {
		return err
	}
	subject := fmt.Sprintf(jwtSubjectTml, username, accounts.AccountCapabilityLogin)
	secondsBeforeExpiry := int64(jwtDefaultExpiry.Seconds())

	jwtToken, err := e.sessionMgr.Create(subject, secondsBeforeExpiry, uniqueID.String(), isSecure)
	if err != nil {
		return err
	}

	e.attemptsStore.CleanupVisitor(ctx.RealIP())

	return ctx.JSON(http.StatusOK, map[string]string{"token": jwtToken})
}
