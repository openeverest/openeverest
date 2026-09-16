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

package auth

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/openeverest/openeverest/v2/client"
	"github.com/openeverest/openeverest/v2/pkg/cli"
	"github.com/openeverest/openeverest/v2/pkg/cli/config"
)

// newTransportConfig builds a config with a valid (non-expired) access token.
func newTransportConfig(serverURL string) *config.Config {
	cfg := newRefreshConfig(serverURL, "rt-valid")
	cfg.Users[0].User.AccessToken = "valid-access"
	cfg.Users[0].User.ExpiresAt = time.Now().Add(time.Hour)
	return cfg
}

// loadTokenSource is a test helper that builds a tokenSource from cfgPath.
func loadTokenSource(t *testing.T, cfgPath string, l *Login) *tokenSource {
	t.Helper()
	cfg, err := config.Load(cfgPath)
	require.NoError(t, err)
	ctx, ok := cfg.GetCurrentContext()
	require.True(t, ok)
	usr, _ := cfg.GetUser(ctx.User)
	srv, _ := cfg.GetServer(ctx.Server)
	return &tokenSource{
		sess:        &cli.Session{Cfg: cfg, Ctx: ctx, User: usr, Server: srv},
		login:       l,
		cfgPath:     cfgPath,
		contextName: "",
	}
}

func newTestAuthTransport(t *testing.T, source *tokenSource) *authTransport {
	t.Helper()
	base := newDefaultTransport()
	t.Cleanup(base.CloseIdleConnections)
	return &authTransport{source: source, base: base}
}

func TestTransport_InjectsBearerToken(t *testing.T) {
	t.Parallel()

	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, newTransportConfig(srv.URL).Save(cfgPath))

	ts := loadTokenSource(t, cfgPath, NewLogin(Config{}, zap.NewNop().Sugar()))
	tr := newTestAuthTransport(t, ts)

	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL, nil)
	resp, err := tr.RoundTrip(req)
	require.NoError(t, err)
	defer resp.Body.Close() //nolint:errcheck
	assert.Equal(t, "Bearer valid-access", gotAuth)
}

func TestTransport_ProactiveRefresh(t *testing.T) {
	t.Parallel()

	refreshResp := client.AuthTokenResponse{
		AccessToken:  "refreshed-access",
		RefreshToken: "refreshed-rt",
		ExpiresIn:    900,
		TokenType:    client.AuthTokenResponseTokenTypeBearer,
	}

	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(refreshResp) //nolint:gosec
			return
		}
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	cfg := newTransportConfig(srv.URL)
	cfg.Users[0].User.ExpiresAt = time.Now().Add(10 * time.Millisecond) // within 30s window
	require.NoError(t, cfg.Save(cfgPath))

	ts := loadTokenSource(t, cfgPath, NewLogin(Config{}, zap.NewNop().Sugar()))
	tr := newTestAuthTransport(t, ts)

	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL, nil)
	resp, err := tr.RoundTrip(req)
	require.NoError(t, err)
	defer resp.Body.Close() //nolint:errcheck
	assert.Equal(t, "Bearer refreshed-access", gotAuth)
}

func TestTransport_On401_RefreshAndRetry(t *testing.T) {
	t.Parallel()

	refreshResp := client.AuthTokenResponse{
		AccessToken:  "rotated-access",
		RefreshToken: "rotated-rt",
		ExpiresIn:    900,
		TokenType:    client.AuthTokenResponseTokenTypeBearer,
	}

	var getCount int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(refreshResp) //nolint:gosec
			return
		}
		getCount++
		if getCount == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, newTransportConfig(srv.URL).Save(cfgPath))

	ts := loadTokenSource(t, cfgPath, NewLogin(Config{}, zap.NewNop().Sugar()))
	tr := newTestAuthTransport(t, ts)

	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL, nil)
	resp, err := tr.RoundTrip(req)
	require.NoError(t, err)
	defer resp.Body.Close() //nolint:errcheck
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 2, getCount)
}

func TestTransport_On401_RefreshFails_ReturnsErrTokenRefresh(t *testing.T) {
	t.Parallel()

	var getCount int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			getCount++
		}
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, newTransportConfig(srv.URL).Save(cfgPath))

	ts := loadTokenSource(t, cfgPath, NewLogin(Config{}, zap.NewNop().Sugar()))
	tr := newTestAuthTransport(t, ts)

	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, srv.URL, nil)
	resp, err := tr.RoundTrip(req)
	if resp != nil {
		defer resp.Body.Close() //nolint:errcheck
	}
	require.ErrorIs(t, err, ErrTokenRefresh)
	assert.Equal(t, 1, getCount, "should not retry when refresh also fails")
}

func TestTransport_On401_RetryReplaysBody(t *testing.T) {
	t.Parallel()

	refreshResp := client.AuthTokenResponse{
		AccessToken:  "rotated-access",
		RefreshToken: "rotated-rt",
		ExpiresIn:    900,
		TokenType:    client.AuthTokenResponseTokenTypeBearer,
	}

	var bodies []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/token") {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(refreshResp) //nolint:gosec
			return
		}
		b, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(b))
		if len(bodies) == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, newTransportConfig(srv.URL).Save(cfgPath))

	ts := loadTokenSource(t, cfgPath, NewLogin(Config{}, zap.NewNop().Sugar()))
	tr := newTestAuthTransport(t, ts)

	payload := `{"metadata":{"name":"my-mongo"}}`
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodPost, srv.URL+"/instances", bytes.NewReader([]byte(payload)))
	resp, err := tr.RoundTrip(req)
	require.NoError(t, err)
	defer resp.Body.Close() //nolint:errcheck
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	require.Len(t, bodies, 2)
	assert.Equal(t, payload, bodies[1], "retried request must carry the original body")
}

func TestNewAPIClient_LoadsSession(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, newTransportConfig(srv.URL).Save(cfgPath))

	c, err := NewAPIClient(Config{}, zap.NewNop().Sugar(), cfgPath, "")
	require.NoError(t, err)
	require.NotNil(t, c)

	generatedClient, ok := c.ClientInterface.(*client.Client)
	require.True(t, ok)
	httpClient, ok := generatedClient.Client.(*http.Client)
	require.True(t, ok)
	transport, ok := httpClient.Transport.(*authTransport)
	require.True(t, ok)
	base, ok := transport.base.(*http.Transport)
	require.True(t, ok)
	t.Cleanup(base.CloseIdleConnections)
	require.NotSame(t, http.DefaultTransport, base)
}
