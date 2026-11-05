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
	"sync/atomic"
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

func newRejectingIDP(t *testing.T) *httptest.Server {
	t.Helper()
	idp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	t.Cleanup(idp.Close)
	return idp
}

// The happy path (200) is covered by session.TestCreateSSO plus the e2e flow, because minting
// the Everest JWT needs a signing key that is only loaded from a fixed on-disk path. These cases
// pin the request-validation guards, which are the security-relevant branches of the handler.
func TestTokenExchangeGrant(t *testing.T) {
	t.Parallel()

	const exchangeBody = `{"grant_type":"urn:ietf:params:oauth:grant-type:token-exchange",` +
		`"subject_token_type":"urn:ietf:params:oauth:token-type:access_token","subject_token":"opaque-but-invalid"}`

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
			oidcProvider:     &oidc.ProviderConfig{UserInfoURL: "http://127.0.0.1:0/userinfo"},
		}
		ctx, rec := newTokenRequestContext(t, `{"grant_type":"urn:ietf:params:oauth:grant-type:token-exchange","subject_token":""}`)
		require.NoError(t, srv.CreateAuthToken(ctx))
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Subject_token is required")
	})

	t.Run("unsupported subject_token_type -> 400", func(t *testing.T) {
		t.Parallel()
		srv := &EverestServer{
			l:                zap.NewNop().Sugar(),
			ssoAttemptsStore: newGrantTestStore(time.Now()),
			oidcProvider:     &oidc.ProviderConfig{UserInfoURL: "http://127.0.0.1:0/userinfo"},
		}
		ctx, rec := newTokenRequestContext(t, `{"grant_type":"urn:ietf:params:oauth:grant-type:token-exchange",`+
			`"subject_token_type":"urn:ietf:params:oauth:token-type:id_token","subject_token":"x"}`)
		require.NoError(t, srv.CreateAuthToken(ctx))
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "Unsupported subject_token_type")
	})

	t.Run("userinfo rejects token -> 401, then throttled -> 429", func(t *testing.T) {
		t.Parallel()
		srv := &EverestServer{
			l:                zap.NewNop().Sugar(),
			ssoAttemptsStore: newGrantTestStore(time.Now()),
			oidcProvider:     &oidc.ProviderConfig{UserInfoURL: newRejectingIDP(t).URL},
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
			oidcProvider:     &oidc.ProviderConfig{UserInfoURL: newRejectingIDP(t).URL},
		}
		ctx, _ := newTokenRequestContext(t, exchangeBody)
		ip := ctx.RealIP()

		require.NoError(t, srv.CreateAuthToken(ctx))

		assert.NotContains(t, passwordStore.visitors, ip)
		assert.Equal(t, 2*initialTimeout, ssoStore.visitors[ip].timeout)
	})
}

func TestResolveSSOIdentity(t *testing.T) {
	t.Parallel()

	idpKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	var keyFunc jwt.Keyfunc = func(*jwt.Token) (any, error) { return idpKey.Public(), nil }

	newIDP := func(t *testing.T, status int) (*httptest.Server, *atomic.Int32) {
		t.Helper()
		var calls atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			calls.Add(1)
			w.WriteHeader(status)
			_, _ = w.Write([]byte(`{"sub":"from-userinfo","groups":["g-userinfo"]}`))
		}))
		t.Cleanup(srv.Close)
		return srv, &calls
	}
	signedJWT := func(t *testing.T, key *rsa.PrivateKey) string {
		t.Helper()
		s, err := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
			"sub":    "from-jwt",
			"exp":    time.Now().Add(time.Hour).Unix(),
			"groups": []string{"g-jwt"},
		}).SignedString(key)
		require.NoError(t, err)
		return s
	}

	// Entra ID: the UserInfo endpoint rejects tokens minted for a custom API audience.
	t.Run("verifiable JWT is accepted without calling UserInfo", func(t *testing.T) {
		t.Parallel()
		idp, calls := newIDP(t, http.StatusUnauthorized)
		srv := &EverestServer{
			l:            zap.NewNop().Sugar(),
			oidcProvider: &oidc.ProviderConfig{UserInfoURL: idp.URL},
			oidcKeyFunc:  keyFunc,
		}
		info, err := srv.resolveSSOIdentity(context.Background(), signedJWT(t, idpKey))
		require.NoError(t, err)
		assert.Equal(t, "from-jwt", info.Subject)
		assert.Equal(t, oidc.StringList{"g-jwt"}, info.Groups)
		assert.Zero(t, calls.Load())
	})

	t.Run("opaque token is validated by UserInfo", func(t *testing.T) {
		t.Parallel()
		idp, calls := newIDP(t, http.StatusOK)
		srv := &EverestServer{
			l:            zap.NewNop().Sugar(),
			oidcProvider: &oidc.ProviderConfig{UserInfoURL: idp.URL},
			oidcKeyFunc:  keyFunc,
		}
		info, err := srv.resolveSSOIdentity(context.Background(), "opaque-random-string")
		require.NoError(t, err)
		assert.Equal(t, "from-userinfo", info.Subject)
		assert.Equal(t, oidc.StringList{"g-userinfo"}, info.Groups)
		assert.Equal(t, int32(1), calls.Load())
	})

	t.Run("unverifiable JWT is left to UserInfo to reject", func(t *testing.T) {
		t.Parallel()
		otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)
		idp, calls := newIDP(t, http.StatusUnauthorized)
		srv := &EverestServer{
			l:            zap.NewNop().Sugar(),
			oidcProvider: &oidc.ProviderConfig{UserInfoURL: idp.URL},
			oidcKeyFunc:  keyFunc,
		}
		_, err = srv.resolveSSOIdentity(context.Background(), signedJWT(t, otherKey))
		require.Error(t, err)
		assert.Equal(t, int32(1), calls.Load())
	})
}
