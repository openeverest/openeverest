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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/openeverest/openeverest/v2/client"
	"github.com/openeverest/openeverest/v2/pkg/cli/config"
)

func newTokenServer(t *testing.T, statusCode int, body any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_ = json.NewEncoder(w).Encode(body)
	}))
}

func TestLogin_Run_Success(t *testing.T) {
	t.Parallel()

	srv := newTokenServer(t, http.StatusOK, client.AuthTokenResponse{
		AccessToken:  "new-access-jwt",
		RefreshToken: "everest_rt_abc",
		ExpiresIn:    900,
		TokenType:    client.AuthTokenResponseTokenTypeBearer,
	})
	defer srv.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	lo := NewLogin(Config{}, zap.NewNop().Sugar())

	err := lo.Run(t.Context(), LoginOptions{
		Server:   srv.URL,
		Username: "admin",
		Password: "secret",
	}, cfgPath)
	require.NoError(t, err)

	cfg, err := config.Load(cfgPath)
	require.NoError(t, err)
	assert.Equal(t, defaultContextName("admin", srv.URL), cfg.CurrentContext)
	require.Len(t, cfg.Contexts, 1)
	ctx := cfg.Contexts[0].Context
	assert.Equal(t, "new-access-jwt", ctx.AccessToken)
	assert.Equal(t, "everest_rt_abc", ctx.RefreshToken)
	assert.False(t, ctx.ExpiresAt.IsZero())

	info, err := os.Stat(cfgPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestLogin_Run_CustomContextName(t *testing.T) {
	t.Parallel()

	srv := newTokenServer(t, http.StatusOK, client.AuthTokenResponse{
		AccessToken:  "access-jwt",
		RefreshToken: "everest_rt_abc",
		ExpiresIn:    900,
		TokenType:    client.AuthTokenResponseTokenTypeBearer,
	})
	defer srv.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	lo := NewLogin(Config{}, zap.NewNop().Sugar())

	err := lo.Run(t.Context(), LoginOptions{
		Server:      srv.URL,
		Username:    "admin",
		Password:    "secret",
		ContextName: "my-cluster",
	}, cfgPath)
	require.NoError(t, err)

	cfg, err := config.Load(cfgPath)
	require.NoError(t, err)
	assert.Equal(t, "my-cluster", cfg.CurrentContext)
	require.Len(t, cfg.Contexts, 1)
	assert.Equal(t, "my-cluster", cfg.Contexts[0].Name)
}

func TestLogin_Run_AuthFailure(t *testing.T) {
	t.Parallel()

	srv := newTokenServer(t, http.StatusUnauthorized, map[string]string{"message": "invalid credentials"})
	defer srv.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	lo := NewLogin(Config{}, zap.NewNop().Sugar())

	err := lo.Run(t.Context(), LoginOptions{
		Server:   srv.URL,
		Username: "admin",
		Password: "wrong",
	}, cfgPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "401")

	_, statErr := os.Stat(cfgPath)
	assert.True(t, os.IsNotExist(statErr), "config file should not be written on auth failure")
}

func TestValidateServerURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input   string
		wantErr bool
	}{
		{"http://localhost:8080", false},
		{"https://everest.example", false},
		{"file:///etc/passwd", true},
		{"ftp://example.com", true},
		{"javascript:alert(1)", true},
		{"//example.com", true},
		{"", true},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			err := validateServerURL(tc.input)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNormalizeServerURL(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  string
	}{
		{"http://localhost:8080", "http://localhost:8080/v1"},
		{"http://localhost:8080/", "http://localhost:8080/v1"},
		{"http://localhost:8080/v1", "http://localhost:8080/v1"},
		{"http://localhost:8080/v1/", "http://localhost:8080/v1"},
		{"https://everest.example", "https://everest.example/v1"},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, normalizeServerURL(tc.input))
		})
	}
}

func TestDefaultContextName(t *testing.T) {
	t.Parallel()
	tests := []struct {
		username string
		server   string
		want     string
	}{
		{"admin", "http://localhost:8080", "admin@localhost:8080"},
		{"readonly", "https://everest.prod.example", "readonly@everest.prod.example"},
		{"alice", "localhost:8080", "alice@localhost:8080"},
	}
	for _, tc := range tests {
		t.Run(tc.username+"@"+tc.server, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, defaultContextName(tc.username, tc.server))
		})
	}
}
