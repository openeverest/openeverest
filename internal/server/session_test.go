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
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
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
			ssoTokenVerifier: newTestSSOVerifier(t),
		}
		ctx, rec := newSSOTestContext(t, `{"token":""}`)
		require.NoError(t, srv.CreateSSOSession(ctx))
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "'token' is required")
	})

	// Access tokens, opaque or JWT, don't prove which application they were issued to.
	t.Run("opaque token -> 401", func(t *testing.T) {
		t.Parallel()
		srv := &EverestServer{
			l:                zap.NewNop().Sugar(),
			ssoAttemptsStore: attemptsStore,
			ssoTokenVerifier: newTestSSOVerifier(t),
		}
		ctx, rec := newSSOTestContext(t, `{"token":"opaque-but-invalid"}`)
		require.NoError(t, srv.CreateSSOSession(ctx))
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Contains(t, rec.Body.String(), "Invalid OIDC token")
	})

	t.Run("failed exchange does not penalize password logins", func(t *testing.T) {
		t.Parallel()
		passwordStore := NewRateLimiterMemoryStoreWithConfig(RateLimiterMemoryStoreConfig{Rate: rate.Limit(10)})
		ssoStore := NewRateLimiterMemoryStoreWithConfig(RateLimiterMemoryStoreConfig{Rate: rate.Limit(10)})
		srv := &EverestServer{
			l:                zap.NewNop().Sugar(),
			attemptsStore:    passwordStore,
			ssoAttemptsStore: ssoStore,
			ssoTokenVerifier: newTestSSOVerifier(t),
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

func TestNewSSOTokenVerifier(t *testing.T) {
	t.Parallel()

	provider := func(clientID string) *oidc.ProviderConfig {
		return &oidc.ProviderConfig{
			Issuer:   "https://idp.example.com",
			JWKSURL:  "https://idp.example.com/jwks",
			ClientID: clientID,
		}
	}

	t.Run("OIDC not configured disables SSO", func(t *testing.T) {
		t.Parallel()
		assert.Nil(t, newSSOTokenVerifier(t.Context(), zap.NewNop().Sugar(), nil))
	})

	t.Run("invalid OIDC settings disable SSO instead of failing the server", func(t *testing.T) {
		t.Parallel()
		core, logs := observer.New(zap.ErrorLevel)
		assert.Nil(t, newSSOTokenVerifier(t.Context(), zap.New(core).Sugar(), provider("")))
		assert.Equal(t, 1, logs.FilterMessageSnippet("SSO is disabled").Len())
	})

	t.Run("valid OIDC settings enable SSO", func(t *testing.T) {
		t.Parallel()
		assert.NotNil(t, newSSOTokenVerifier(t.Context(), zap.NewNop().Sugar(), provider("everest")))
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
