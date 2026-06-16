// everest
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
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	api "github.com/openeverest/openeverest/v2/internal/server/api"
	"github.com/openeverest/openeverest/v2/internal/tokenregistry"
	"github.com/openeverest/openeverest/v2/pkg/accounts"
	"github.com/openeverest/openeverest/v2/pkg/common"
)

const (
	jwtSubjectTml = "%s:%s" // username:capability

	// refreshTokenCookieName is the cookie that carries the refresh token when
	// the client requests cookie delivery.
	refreshTokenCookieName = "everest_refresh_token"
	// refreshTokenCookiePath restricts the cookie to the auth endpoints.
	refreshTokenCookiePath = "/v1/auth"
)

// CreateAuthToken issues API tokens (POST /v1/auth/token).
func (e *EverestServer) CreateAuthToken(ctx echo.Context) error {
	var params api.AuthTokenRequest
	if err := ctx.Bind(&params); err != nil {
		return err
	}

	switch params.GrantType {
	case api.AuthTokenRequestGrantTypePassword:
		return e.handlePasswordGrant(ctx, params)
	case api.AuthTokenRequestGrantTypeRefreshToken:
		return e.handleRefreshTokenGrant(ctx, params)
	}
	return ctx.JSON(http.StatusBadRequest, api.Error{
		Message: pointer.To(fmt.Sprintf("Unsupported grant_type %q", params.GrantType)),
	})
}

func (e *EverestServer) handlePasswordGrant(ctx echo.Context, params api.AuthTokenRequest) error {
	if params.Username == nil || params.Password == nil {
		return ctx.JSON(http.StatusBadRequest, api.Error{
			Message: pointer.To("username and password are required for the password grant"),
		})
	}

	c := ctx.Request().Context()
	if err := e.sessionMgr.Authenticate(c, *params.Username, *params.Password); err != nil {
		e.attemptsStore.IncreaseTimeout(ctx.RealIP())
		return sessionErrToHTTPRes(ctx, err)
	}

	refreshToken, _, err := e.tokenRegistry.Mint(c, *params.Username, tokenregistry.TypeRefresh, e.config.RefreshTokenTTL)
	if err != nil {
		e.l.Errorf("failed to mint refresh token: %v", err)
		return errInternalTokenIssue(ctx)
	}

	e.attemptsStore.CleanupVisitor(ctx.RealIP())

	return e.respondWithTokens(ctx, *params.Username, refreshToken, useCookieDelivery(params))
}

func (e *EverestServer) handleRefreshTokenGrant(ctx echo.Context, params api.AuthTokenRequest) error {
	c := ctx.Request().Context()

	presented, fromCookie := refreshTokenFromRequest(ctx, params.RefreshToken)
	if presented == "" {
		return ctx.JSON(http.StatusBadRequest, api.Error{
			Message: pointer.To("refresh_token is required for the refresh_token grant"),
		})
	}

	rec, err := e.tokenRegistry.Validate(c, presented)
	if err != nil {
		if errors.Is(err, tokenregistry.ErrInvalidToken) {
			e.attemptsStore.IncreaseTimeout(ctx.RealIP())
			return errInvalidRefreshToken(ctx)
		}
		e.l.Errorf("failed to validate refresh token: %v", err)
		return errInternalTokenIssue(ctx)
	}
	if rec.Type != tokenregistry.TypeRefresh {
		e.attemptsStore.IncreaseTimeout(ctx.RealIP())
		return errInvalidRefreshToken(ctx)
	}

	// The owning account must still be allowed to log in.
	account, err := e.sessionMgr.CanLogin(c, rec.OwnerSubject)
	if err != nil {
		if revokeErr := e.tokenRegistry.Revoke(c, rec.ID); revokeErr != nil {
			e.l.Errorf("failed to revoke refresh token: %v", revokeErr)
		}
		return sessionErrToHTTPRes(ctx, err)
	}

	// Refresh tokens minted before the last password change are rejected.
	if account.PasswordMtime != "" {
		passwordMtime, err := time.Parse(time.RFC3339, account.PasswordMtime)
		if err != nil {
			e.l.Errorf("failed to parse password mtime: %v", err)
			return errInternalTokenIssue(ctx)
		}
		if rec.CreatedAt.Before(passwordMtime) {
			if revokeErr := e.tokenRegistry.Revoke(c, rec.ID); revokeErr != nil {
				e.l.Errorf("failed to revoke refresh token: %v", revokeErr)
			}
			return errInvalidRefreshToken(ctx)
		}
	}

	// Rotate-on-use with a sliding TTL window.
	newRefreshToken, _, err := e.tokenRegistry.Rotate(c, rec, e.config.RefreshTokenTTL)
	if err != nil {
		e.l.Errorf("failed to rotate refresh token: %v", err)
		return errInternalTokenIssue(ctx)
	}

	e.attemptsStore.CleanupVisitor(ctx.RealIP())

	return e.respondWithTokens(ctx, rec.OwnerSubject, newRefreshToken, fromCookie || useCookieDelivery(params))
}

// respondWithTokens mints an access JWT for the user and writes the token response,
// delivering the refresh token either in the body or as an HttpOnly cookie.
func (e *EverestServer) respondWithTokens(ctx echo.Context, username, refreshToken string, cookieDelivery bool) error {
	uniqueID, err := uuid.NewRandom()
	if err != nil {
		return err
	}
	subject := fmt.Sprintf(jwtSubjectTml, username, accounts.AccountCapabilityLogin)
	accessToken, err := e.sessionMgr.Create(subject, int64(e.config.AccessTokenTTL.Seconds()), uniqueID.String())
	if err != nil {
		e.l.Errorf("failed to create access token: %v", err)
		return errInternalTokenIssue(ctx)
	}

	resp := api.AuthTokenResponse{
		AccessToken: accessToken,
		TokenType:   api.AuthTokenResponseTokenTypeBearer,
		ExpiresIn:   int(e.config.AccessTokenTTL.Seconds()),
	}
	if cookieDelivery {
		ctx.SetCookie(e.newRefreshTokenCookie(ctx, refreshToken, int(e.config.RefreshTokenTTL.Seconds())))
	} else {
		resp.RefreshToken = refreshToken
	}
	return ctx.JSON(http.StatusOK, resp)
}

// RevokeAuthToken revokes the caller's tokens (POST /v1/auth/revoke).
// The refresh token (from the body or cookie) is deleted from the registry and
// the presented access JWT is added to the blocklist until it expires.
func (e *EverestServer) RevokeAuthToken(ctx echo.Context) error {
	e.attemptsStore.IncreaseTimeout(ctx.RealIP())
	c := ctx.Request().Context()

	var params api.AuthRevokeRequest
	if err := ctx.Bind(&params); err != nil {
		return err
	}

	// Revoke the refresh token, if one was presented.
	// Per RFC 7009, an invalid token does not fail the revocation request.
	if presented, fromCookie := refreshTokenFromRequest(ctx, params.Token); presented != "" {
		if rec, err := e.tokenRegistry.Validate(c, presented); err == nil {
			if err := e.tokenRegistry.Revoke(c, rec.ID); err != nil {
				e.l.Errorf("failed to revoke refresh token: %v", err)
				return errFailedLogout(ctx)
			}
		}
		if fromCookie {
			ctx.SetCookie(e.newRefreshTokenCookie(ctx, "", -1))
		}
	}

	// Blocklist the presented access JWT.
	token, err := common.ExtractToken(c)
	if err != nil {
		return err
	}
	if err := e.sessionMgr.Block(c, token); err != nil {
		e.l.Errorf("blocklist error: %v", err)
		return errFailedLogout(ctx)
	}

	return ctx.NoContent(http.StatusNoContent)
}

// refreshTokenFromRequest returns the refresh token presented by the request,
// preferring the request body and falling back to the refresh token cookie,
// and whether it came from the cookie.
func refreshTokenFromRequest(ctx echo.Context, fromBody *string) (string, bool) {
	if fromBody != nil && *fromBody != "" {
		return *fromBody, false
	}
	if cookie, err := ctx.Cookie(refreshTokenCookieName); err == nil && cookie.Value != "" {
		return cookie.Value, true
	}
	return "", false
}

func useCookieDelivery(params api.AuthTokenRequest) bool {
	return params.RefreshTokenDelivery != nil &&
		*params.RefreshTokenDelivery == api.AuthTokenRequestRefreshTokenDeliveryCookie
}

// newRefreshTokenCookie builds the HttpOnly refresh token cookie.
// A negative maxAge expires the cookie immediately.
func (e *EverestServer) newRefreshTokenCookie(ctx echo.Context, value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     refreshTokenCookieName,
		Value:    value,
		Path:     refreshTokenCookiePath,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   ctx.Scheme() == "https",
		SameSite: http.SameSiteStrictMode,
	}
}

func errInvalidRefreshToken(ctx echo.Context) error {
	return ctx.JSON(http.StatusUnauthorized, api.Error{
		Message: pointer.To("Invalid refresh token"),
	})
}

func errInternalTokenIssue(ctx echo.Context) error {
	return ctx.JSON(http.StatusInternalServerError, api.Error{
		Message: pointer.To("Failed to issue tokens"),
	})
}

func errFailedLogout(ctx echo.Context) error {
	return ctx.JSON(http.StatusInternalServerError, api.Error{
		Message: pointer.To("Failed to logout user"),
	})
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
