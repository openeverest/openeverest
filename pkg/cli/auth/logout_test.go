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
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/openeverest/openeverest/v2/pkg/cli/config"
)

// newLogoutConfig builds a minimal config with a fresh (non-expired) access token.
func newLogoutConfig(serverURL string) *config.Config {
	srvName := serverName(serverURL)
	userName := "admin@" + srvName
	return &config.Config{
		APIVersion:     "config.openeverest.io/v1alpha1",
		Kind:           "ClientConfig",
		CurrentContext: userName,
		Contexts: []config.NamedContext{
			{Name: userName, Context: config.Context{Server: srvName, User: userName}},
		},
		Servers: []config.NamedServer{
			{Name: srvName, Server: config.Server{URL: serverURL}},
		},
		Users: []config.NamedUser{
			{Name: userName, User: config.User{
				AccessToken:  "access-jwt",
				RefreshToken: "everest_rt_old",
				ExpiresAt:    time.Now().Add(time.Minute),
			}},
		},
	}
}

func TestLogout_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, newLogoutConfig(srv.URL).Save(cfgPath))

	lo := NewLogin(Config{}, zap.NewNop().Sugar())
	require.NoError(t, lo.Logout(t.Context(), cfgPath))

	updated, err := config.Load(cfgPath)
	require.NoError(t, err)
	assert.Empty(t, updated.CurrentContext)
	assert.Empty(t, updated.Contexts)
	assert.Empty(t, updated.Users)
	assert.Empty(t, updated.Servers)
}

func TestLogout_ServerError_ClearsLocalConfig(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, newLogoutConfig(srv.URL).Save(cfgPath))

	lo := NewLogin(Config{}, zap.NewNop().Sugar())
	// server error must not block logout — local credentials are always cleared
	require.NoError(t, lo.Logout(t.Context(), cfgPath))

	updated, err := config.Load(cfgPath)
	require.NoError(t, err)
	assert.Empty(t, updated.CurrentContext)
	assert.Empty(t, updated.Contexts)
	assert.Empty(t, updated.Users)
	assert.Empty(t, updated.Servers)
}

func TestLogout_NoActiveContext(t *testing.T) {
	t.Parallel()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	cfg := &config.Config{
		APIVersion:     "config.openeverest.io/v1alpha1",
		Kind:           "ClientConfig",
		CurrentContext: "missing",
	}
	require.NoError(t, cfg.Save(cfgPath))

	lo := NewLogin(Config{}, zap.NewNop().Sugar())
	err := lo.Logout(t.Context(), cfgPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing")
}

func TestLogout_ExpiredToken_NoBearerHeader(t *testing.T) {
	t.Parallel()

	var authHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	cfg := newLogoutConfig(srv.URL)
	cfg.Users[0].User.ExpiresAt = time.Now().Add(-time.Minute) // expired
	require.NoError(t, cfg.Save(cfgPath))

	lo := NewLogin(Config{}, zap.NewNop().Sugar())
	require.NoError(t, lo.Logout(t.Context(), cfgPath))

	assert.Empty(t, authHeader, "expired access token must not be sent as Bearer header")

	updated, err := config.Load(cfgPath)
	require.NoError(t, err)
	assert.Empty(t, updated.Contexts)
}

func TestLogout_MultiContext_PreservesSharedServer(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	srvName := serverName(srv.URL)
	userA := "user-a@" + srvName
	userB := "user-b@" + srvName
	cfg := &config.Config{
		APIVersion:     "config.openeverest.io/v1alpha1",
		Kind:           "ClientConfig",
		CurrentContext: "ctx-a",
		Contexts: []config.NamedContext{
			{Name: "ctx-a", Context: config.Context{Server: srvName, User: userA}},
			{Name: "ctx-b", Context: config.Context{Server: srvName, User: userB}},
		},
		Servers: []config.NamedServer{
			{Name: srvName, Server: config.Server{URL: srv.URL}},
		},
		Users: []config.NamedUser{
			{Name: userA, User: config.User{AccessToken: "access-a", RefreshToken: "rt-a", ExpiresAt: time.Now().Add(time.Minute)}},
			{Name: userB, User: config.User{AccessToken: "access-b", RefreshToken: "rt-b", ExpiresAt: time.Now().Add(time.Minute)}},
		},
	}

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, cfg.Save(cfgPath))

	lo := NewLogin(Config{}, zap.NewNop().Sugar())
	require.NoError(t, lo.Logout(t.Context(), cfgPath))

	updated, err := config.Load(cfgPath)
	require.NoError(t, err)
	assert.Empty(t, updated.CurrentContext)
	assert.Len(t, updated.Contexts, 1, "ctx-b should remain")
	assert.Equal(t, "ctx-b", updated.Contexts[0].Name)
	assert.Len(t, updated.Servers, 1, "shared server must not be removed")
	assert.Len(t, updated.Users, 1, "user-b should remain")
	assert.Equal(t, userB, updated.Users[0].Name)
}
