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
	"context"
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	api "github.com/openeverest/openeverest/v2/internal/server/api"
	"github.com/openeverest/openeverest/v2/pkg/oidc"
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

//nolint:ireturn // echo.Context is an interface by design; this is a test helper.
func newTokenRequestContext(t *testing.T, body string) (echo.Context, *httptest.ResponseRecorder) {
	t.Helper()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/auth/token", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	return echo.New().NewContext(req, rec), rec
}

func newGrantTestStore(now time.Time) *RateLimiterMemoryStore {
	store := newGrantRateLimiter(10)
	store.timeNow = func() time.Time { return now }
	return store
}

func newTestSSOVerifier(t *testing.T) *oidc.IDTokenVerifier {
	t.Helper()
	idpKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	verifier, err := oidc.NewIDTokenVerifier(
		func(*jwt.Token) (any, error) { return idpKey.Public(), nil },
		"everest", "https://idp.example.com",
	)
	require.NoError(t, err)
	return verifier
}

// The happy path (200) is covered by session.TestCreateSSO, oidc.TestIDTokenVerifier and the e2e
// flow, because minting the Everest JWT needs a signing key that is only loaded from a fixed
// on-disk path. These cases pin the request-validation guards, which are the security-relevant
// branches of the handler.
func TestTokenExchangeGrant(t *testing.T) {
	t.Parallel()

	const exchangeBody = `{"grant_type":"urn:ietf:params:oauth:grant-type:token-exchange",` +
		`"subject_token_type":"urn:ietf:params:oauth:token-type:id_token","subject_token":"not-an-id-token"}`

	t.Run("oidc not configured -> 400", func(t *testing.T) {
		t.Parallel()
		srv := &EverestServer{l: zap.NewNop().Sugar(), ssoAttemptsStore: newGrantTestStore(time.Now())}
		ctx, rec := newTokenRequestContext(t, exchangeBody)
		require.NoError(t, srv.CreateAuthToken(ctx))
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "OIDC is not configured")
	})

	t.Run("missing subject_token -> 400", func(t *testing.T) {
		t.Parallel()
		srv := &EverestServer{
			l:                zap.NewNop().Sugar(),
			ssoAttemptsStore: newGrantTestStore(time.Now()),
			ssoTokenVerifier: newTestSSOVerifier(t),
		}
		ctx, rec := newTokenRequestContext(t, `{"grant_type":"urn:ietf:params:oauth:grant-type:token-exchange",`+
			`"subject_token_type":"urn:ietf:params:oauth:token-type:id_token","subject_token":""}`)
		require.NoError(t, srv.CreateAuthToken(ctx))
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Subject_token is required")
	})

	// Access tokens don't prove which application they were issued to.
	for name, body := range map[string]string{ //nolint:gosec // request fixtures, not credentials
		"access_token subject_token_type -> 400": `{"grant_type":"urn:ietf:params:oauth:grant-type:token-exchange",` +
			`"subject_token_type":"urn:ietf:params:oauth:token-type:access_token","subject_token":"x"}`,
		"missing subject_token_type -> 400": `{"grant_type":"urn:ietf:params:oauth:grant-type:token-exchange","subject_token":"x"}`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			srv := &EverestServer{
				l:                zap.NewNop().Sugar(),
				ssoAttemptsStore: newGrantTestStore(time.Now()),
				ssoTokenVerifier: newTestSSOVerifier(t),
			}
			ctx, rec := newTokenRequestContext(t, body)
			require.NoError(t, srv.CreateAuthToken(ctx))
			assert.Equal(t, http.StatusBadRequest, rec.Code)
		})
	}

	t.Run("invalid ID token -> 401, then throttled -> 429", func(t *testing.T) {
		t.Parallel()
		srv := &EverestServer{
			l:                zap.NewNop().Sugar(),
			ssoAttemptsStore: newGrantTestStore(time.Now()),
			ssoTokenVerifier: newTestSSOVerifier(t),
		}
		ctx, rec := newTokenRequestContext(t, exchangeBody)
		require.NoError(t, srv.CreateAuthToken(ctx))
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Contains(t, rec.Body.String(), "Invalid OIDC token")

		// The frozen clock keeps the retry inside the post-failure backoff window.
		ctx, rec = newTokenRequestContext(t, exchangeBody)
		require.NoError(t, srv.CreateAuthToken(ctx))
		assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	})

	t.Run("failed exchange does not penalize password logins", func(t *testing.T) {
		t.Parallel()
		passwordStore := newGrantTestStore(time.Now())
		ssoStore := newGrantTestStore(time.Now())
		srv := &EverestServer{
			l:                zap.NewNop().Sugar(),
			attemptsStore:    passwordStore,
			ssoAttemptsStore: ssoStore,
			ssoTokenVerifier: newTestSSOVerifier(t),
		}
		ctx, _ := newTokenRequestContext(t, exchangeBody)
		ip := ctx.RealIP()

		require.NoError(t, srv.CreateAuthToken(ctx))

		assert.NotContains(t, passwordStore.visitors, ip)
		assert.Equal(t, 2*initialTimeout, ssoStore.visitors[ip].timeout)
	})
}

func TestEverestJWTKeyFuncRejectsForeignIssuers(t *testing.T) {
	t.Parallel()

	// No session manager: a foreign token must be rejected before any key lookup.
	srv := &EverestServer{}
	for _, iss := range []string{"https://idp.example.com", ""} {
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{"iss": iss, "sub": "alice"})
		_, err := srv.everestJWTKeyFunc(token)
		require.Error(t, err, "issuer %q", iss)
	}
}
