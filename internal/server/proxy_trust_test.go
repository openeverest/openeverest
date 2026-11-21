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
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const (
	privateProxyAddr = "10.0.0.1:1234"
	publicProxyAddr  = "198.51.100.7:1234"
	publicProxyCIDR  = "198.51.100.0/24"
	clientA          = "203.0.113.10"
	clientB          = "203.0.113.20"
)

func newProxiedRequest(t *testing.T, remoteAddr, forwardedFor string) *http.Request {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	req.RemoteAddr = remoteAddr
	if forwardedFor != "" {
		req.Header.Set(echo.HeaderXForwardedFor, forwardedFor)
	}
	req.Header.Set(echo.HeaderXForwardedProto, "https")
	return req
}

func newProxyTrustEcho(t *testing.T, trustedProxies ...string) *echo.Echo {
	t.Helper()
	e := echo.New()
	require.NoError(t, configureProxyTrust(e, trustedProxies))
	return e
}

func TestConfigureProxyTrustRateLimitsPerClient(t *testing.T) {
	t.Parallel()

	e := newProxyTrustEcho(t)
	e.Use(echomiddleware.RateLimiter(echomiddleware.NewRateLimiterMemoryStoreWithConfig(
		echomiddleware.RateLimiterMemoryStoreConfig{Rate: 1, Burst: 1},
	)))
	e.GET("/", func(c echo.Context) error { return c.NoContent(http.StatusOK) })

	status := func(forwardedFor string) int {
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, newProxiedRequest(t, privateProxyAddr, forwardedFor))
		return rec.Code
	}

	assert.Equal(t, http.StatusOK, status(clientA))
	assert.Equal(t, http.StatusTooManyRequests, status(clientA))
	// A second client behind the same ingress must not inherit the first client's bucket.
	assert.Equal(t, http.StatusOK, status(clientB))
}

func TestSSOGrantThrottlesPerClientBehindProxy(t *testing.T) {
	t.Parallel()

	const exchangeBody = `{"grant_type":"urn:ietf:params:oauth:grant-type:token-exchange",` +
		`"subject_token_type":"urn:ietf:params:oauth:token-type:id_token","subject_token":"not-an-id-token"}`

	e := newProxyTrustEcho(t)
	srv := &EverestServer{
		l:                zap.NewNop().Sugar(),
		ssoAttemptsStore: newGrantTestStore(time.Now()),
		ssoTokenVerifier: newTestSSOVerifier(t),
	}

	status := func(forwardedFor string) int {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/v1/auth/token", strings.NewReader(exchangeBody))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
		req.Header.Set(echo.HeaderXForwardedFor, forwardedFor)
		req.RemoteAddr = privateProxyAddr
		rec := httptest.NewRecorder()
		require.NoError(t, srv.CreateAuthToken(e.NewContext(req, rec)))
		return rec.Code
	}

	assert.Equal(t, http.StatusUnauthorized, status(clientA))
	// The frozen clock keeps the retry inside the post-failure backoff window.
	assert.Equal(t, http.StatusTooManyRequests, status(clientA))
	// A second client behind the same ingress must not inherit the first client's backoff.
	assert.Equal(t, http.StatusUnauthorized, status(clientB))
}

func TestConfigureProxyTrust(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		trustedProxies []string
		remoteAddr     string
		forwardedFor   string
		wantIP         string
		wantScheme     string
	}{
		{
			name:         "private proxy is trusted by default",
			remoteAddr:   privateProxyAddr,
			forwardedFor: clientA,
			wantIP:       clientA,
			wantScheme:   "https",
		},
		{
			name:         "client cannot spoof the headers without a proxy",
			remoteAddr:   clientB + ":1234",
			forwardedFor: clientA,
			wantIP:       clientB,
			wantScheme:   "http",
		},
		{
			name:         "public proxy is not trusted by default",
			remoteAddr:   publicProxyAddr,
			forwardedFor: clientA,
			wantIP:       "198.51.100.7",
			wantScheme:   "http",
		},
		{
			name:           "listed public proxy is trusted",
			trustedProxies: []string{publicProxyCIDR},
			remoteAddr:     publicProxyAddr,
			forwardedFor:   clientA,
			wantIP:         clientA,
			wantScheme:     "https",
		},
		{
			name:           "forged entry before a trusted proxy is ignored",
			trustedProxies: []string{publicProxyCIDR},
			remoteAddr:     publicProxyAddr,
			forwardedFor:   "192.0.2.99, " + clientA,
			wantIP:         clientA,
			wantScheme:     "https",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := newProxyTrustEcho(t, tt.trustedProxies...)
			ctx := e.NewContext(newProxiedRequest(t, tt.remoteAddr, tt.forwardedFor), httptest.NewRecorder())
			assert.Equal(t, tt.wantIP, ctx.RealIP())
			assert.Equal(t, tt.wantScheme, ctx.Scheme())
		})
	}
}

func TestConfigureProxyTrustRejectsInvalidCIDR(t *testing.T) {
	t.Parallel()

	err := configureProxyTrust(echo.New(), []string{"not-a-cidr"})
	require.ErrorContains(t, err, "TRUSTED_PROXIES")
}

func TestRefreshTokenCookieSecureBehindListedPublicProxy(t *testing.T) {
	t.Parallel()

	server := &EverestServer{}
	cookieSecure := func(trustedProxies ...string) bool {
		e := newProxyTrustEcho(t, trustedProxies...)
		ctx := e.NewContext(newProxiedRequest(t, publicProxyAddr, clientA), httptest.NewRecorder())
		return server.newRefreshTokenCookie(ctx, "everest_rt_abc", 3600).Secure
	}

	assert.False(t, cookieSecure())
	assert.True(t, cookieSecure(publicProxyCIDR))
}
