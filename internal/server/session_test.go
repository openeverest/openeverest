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

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/time/rate"

	"github.com/percona/everest/pkg/oidc"
)

//nolint:ireturn // echo.Context is an interface by design; this is a test helper.
func newSSOTestContext(t *testing.T, body string) (echo.Context, *httptest.ResponseRecorder) {
	t.Helper()
	e := echo.New()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/session/sso", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec), rec
}

// The happy path (200) is covered by session.TestCreateSSO plus the e2e flow, because minting
// the Everest JWT needs a signing key that is only loaded from a fixed on-disk path. These cases
// pin the request-validation guards, which are the security-relevant branches of the handler.
func TestCreateSSOSession(t *testing.T) {
	t.Parallel()

	attemptsStore := NewRateLimiterMemoryStoreWithConfig(RateLimiterMemoryStoreConfig{Rate: rate.Limit(10)})

	t.Run("oidc not configured -> 400", func(t *testing.T) {
		t.Parallel()
		srv := &EverestServer{l: zap.NewNop().Sugar(), ssoAttemptsStore: attemptsStore}
		ctx, rec := newSSOTestContext(t, `{"token":"anything"}`)
		require.NoError(t, srv.CreateSSOSession(ctx))
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "OIDC is not configured")
	})

	t.Run("empty token -> 400", func(t *testing.T) {
		t.Parallel()
		srv := &EverestServer{
			l:                zap.NewNop().Sugar(),
			ssoAttemptsStore: attemptsStore,
			oidcProvider:     &oidc.ProviderConfig{UserInfoURL: "http://127.0.0.1:0/userinfo"},
		}
		ctx, rec := newSSOTestContext(t, `{"token":""}`)
		require.NoError(t, srv.CreateSSOSession(ctx))
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "'token' is required")
	})

	t.Run("userinfo rejects token -> 401", func(t *testing.T) {
		t.Parallel()
		idp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer idp.Close()

		srv := &EverestServer{
			l:                zap.NewNop().Sugar(),
			ssoAttemptsStore: attemptsStore,
			oidcProvider:     &oidc.ProviderConfig{UserInfoURL: idp.URL},
		}
		ctx, rec := newSSOTestContext(t, `{"token":"opaque-but-invalid"}`)
		require.NoError(t, srv.CreateSSOSession(ctx))
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Contains(t, rec.Body.String(), "Invalid OIDC token")
	})

	t.Run("failed exchange does not penalize password logins", func(t *testing.T) {
		t.Parallel()
		idp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer idp.Close()

		passwordStore := NewRateLimiterMemoryStoreWithConfig(RateLimiterMemoryStoreConfig{Rate: rate.Limit(10)})
		ssoStore := NewRateLimiterMemoryStoreWithConfig(RateLimiterMemoryStoreConfig{Rate: rate.Limit(10)})
		srv := &EverestServer{
			l:                zap.NewNop().Sugar(),
			attemptsStore:    passwordStore,
			ssoAttemptsStore: ssoStore,
			oidcProvider:     &oidc.ProviderConfig{UserInfoURL: idp.URL},
		}
		ctx, _ := newSSOTestContext(t, `{"token":"opaque-but-invalid"}`)
		ip := ctx.RealIP()
		// The rate-limit middleware registers the visitor in each store before the handler runs.
		_, _ = passwordStore.Allow(ip)
		_, _ = ssoStore.Allow(ip)

		require.NoError(t, srv.CreateSSOSession(ctx))

		assert.Equal(t, initialTimeout, passwordStore.visitors[ip].timeout)
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

// TestPathRateLimiterIsolation pins that SSO renewals have their own budget: exhausting the
// strict password-login limit must not throttle /v1/session/sso, and vice versa.
func TestPathRateLimiterIsolation(t *testing.T) {
	t.Parallel()

	e := echo.New()
	passwordMW, passwordStore := pathRateLimiter("/v1/session", 1)
	ssoMW, ssoStore := pathRateLimiter("/v1/session/sso", 10)
	// The store also enforces a short gap between requests from one client; step a fake clock past it.
	now := time.Now()
	clock := func() time.Time { return now }
	passwordStore.timeNow = clock
	ssoStore.timeNow = clock
	e.Use(passwordMW, ssoMW)
	ok := func(c echo.Context) error { return c.NoContent(http.StatusOK) }
	e.POST("/v1/session", ok)
	e.POST("/v1/session/sso", ok)

	post := func(path string) int {
		now = now.Add(50 * time.Millisecond)
		req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, path, nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		return rec.Code
	}

	require.Equal(t, http.StatusOK, post("/v1/session"))
	require.Equal(t, http.StatusTooManyRequests, post("/v1/session"))

	// Several tabs renewing within the same second all succeed.
	for range 5 {
		assert.Equal(t, http.StatusOK, post("/v1/session/sso"))
	}
}
