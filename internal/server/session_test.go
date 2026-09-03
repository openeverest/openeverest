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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/time/rate"

	"github.com/percona/everest/pkg/oidc"
)

func newSSOTestContext(t *testing.T, body string) (echo.Context, *httptest.ResponseRecorder) {
	t.Helper()
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/v1/session/sso", strings.NewReader(body))
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
		srv := &EverestServer{l: zap.NewNop().Sugar(), attemptsStore: attemptsStore}
		ctx, rec := newSSOTestContext(t, `{"token":"anything"}`)
		require.NoError(t, srv.CreateSSOSession(ctx))
		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Contains(t, rec.Body.String(), "OIDC is not configured")
	})

	t.Run("empty token -> 400", func(t *testing.T) {
		t.Parallel()
		srv := &EverestServer{
			l:             zap.NewNop().Sugar(),
			attemptsStore: attemptsStore,
			oidcProvider:  &oidc.ProviderConfig{UserInfoURL: "http://127.0.0.1:0/userinfo"},
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
			l:             zap.NewNop().Sugar(),
			attemptsStore: attemptsStore,
			oidcProvider:  &oidc.ProviderConfig{UserInfoURL: idp.URL},
		}
		ctx, rec := newSSOTestContext(t, `{"token":"opaque-but-invalid"}`)
		require.NoError(t, srv.CreateSSOSession(ctx))
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Contains(t, rec.Body.String(), "Invalid OIDC token")
	})
}
