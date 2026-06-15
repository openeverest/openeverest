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

package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AlekSi/pointer"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"

	api "github.com/openeverest/openeverest/v2/internal/server/api"
)

func newTestEchoContext(t *testing.T, cookies ...*http.Cookie) echo.Context {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/token", nil)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	return echo.New().NewContext(req, httptest.NewRecorder())
}

func TestRefreshTokenFromRequest(t *testing.T) {
	t.Parallel()

	t.Run("prefers body over cookie", func(t *testing.T) {
		t.Parallel()
		ctx := newTestEchoContext(t, &http.Cookie{Name: refreshTokenCookieName, Value: "from-cookie"})
		token, fromCookie := refreshTokenFromRequest(ctx, pointer.To("from-body"))
		assert.Equal(t, "from-body", token)
		assert.False(t, fromCookie)
	})

	t.Run("falls back to cookie", func(t *testing.T) {
		t.Parallel()
		ctx := newTestEchoContext(t, &http.Cookie{Name: refreshTokenCookieName, Value: "from-cookie"})
		token, fromCookie := refreshTokenFromRequest(ctx, nil)
		assert.Equal(t, "from-cookie", token)
		assert.True(t, fromCookie)
	})

	t.Run("empty body string falls back to cookie", func(t *testing.T) {
		t.Parallel()
		ctx := newTestEchoContext(t, &http.Cookie{Name: refreshTokenCookieName, Value: "from-cookie"})
		token, fromCookie := refreshTokenFromRequest(ctx, pointer.To(""))
		assert.Equal(t, "from-cookie", token)
		assert.True(t, fromCookie)
	})

	t.Run("no token presented", func(t *testing.T) {
		t.Parallel()
		ctx := newTestEchoContext(t)
		token, fromCookie := refreshTokenFromRequest(ctx, nil)
		assert.Empty(t, token)
		assert.False(t, fromCookie)
	})
}

func TestUseCookieDelivery(t *testing.T) {
	t.Parallel()

	assert.False(t, useCookieDelivery(api.AuthTokenRequest{}))
	assert.False(t, useCookieDelivery(api.AuthTokenRequest{
		RefreshTokenDelivery: pointer.To(api.AuthTokenRequestRefreshTokenDeliveryBody),
	}))
	assert.True(t, useCookieDelivery(api.AuthTokenRequest{
		RefreshTokenDelivery: pointer.To(api.AuthTokenRequestRefreshTokenDeliveryCookie),
	}))
}

func TestNewRefreshTokenCookie(t *testing.T) {
	t.Parallel()
	e := &EverestServer{}

	t.Run("set cookie", func(t *testing.T) {
		t.Parallel()
		ctx := newTestEchoContext(t)
		cookie := e.newRefreshTokenCookie(ctx, "everest_rt_abc", 3600)
		assert.Equal(t, refreshTokenCookieName, cookie.Name)
		assert.Equal(t, "everest_rt_abc", cookie.Value)
		assert.Equal(t, refreshTokenCookiePath, cookie.Path)
		assert.Equal(t, 3600, cookie.MaxAge)
		assert.True(t, cookie.HttpOnly)
		assert.Equal(t, http.SameSiteStrictMode, cookie.SameSite)
		assert.False(t, cookie.Secure) // plain HTTP test request
	})

	t.Run("expire cookie", func(t *testing.T) {
		t.Parallel()
		ctx := newTestEchoContext(t)
		cookie := e.newRefreshTokenCookie(ctx, "", -1)
		assert.Empty(t, cookie.Value)
		assert.Negative(t, cookie.MaxAge)
	})

	t.Run("secure on https", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodPost, "/v1/auth/token", nil)
		req.Header.Set(echo.HeaderXForwardedProto, "https")
		ctx := echo.New().NewContext(req, httptest.NewRecorder())
		cookie := e.newRefreshTokenCookie(ctx, "everest_rt_abc", 3600)
		assert.True(t, cookie.Secure)
	})
}
