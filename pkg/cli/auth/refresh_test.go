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
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/openeverest/openeverest/v2/client"
	"github.com/openeverest/openeverest/v2/pkg/cli/config"
)

func TestRefresh_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(client.AuthTokenResponse{
			AccessToken:  "rotated-access-jwt",
			RefreshToken: "everest_rt_new",
			ExpiresIn:    900,
			TokenType:    client.AuthTokenResponseTokenTypeBearer,
		})
	}))
	defer srv.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	cfg := &config.Config{
		APIVersion:     "v1",
		Kind:           "Config",
		CurrentContext: "local",
		Contexts: []config.NamedContext{
			{
				Name: "local",
				Context: config.Context{
					Server:       srv.URL,
					AccessToken:  "old-access",
					RefreshToken: "everest_rt_old",
					ExpiresAt:    time.Now().Add(-time.Minute),
				},
			},
		},
	}
	require.NoError(t, cfg.Save(cfgPath))

	lo := NewLogin(Config{}, zap.NewNop().Sugar())
	require.NoError(t, lo.Refresh(t.Context(), cfgPath))

	updated, err := config.Load(cfgPath)
	require.NoError(t, err)
	require.Len(t, updated.Contexts, 1)
	ctx := updated.Contexts[0].Context
	assert.Equal(t, "rotated-access-jwt", ctx.AccessToken)
	assert.Equal(t, "everest_rt_new", ctx.RefreshToken)
	assert.True(t, ctx.ExpiresAt.After(time.Now()))
}

func TestRefresh_InvalidToken(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "Invalid refresh token"})
	}))
	defer srv.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	cfg := &config.Config{
		APIVersion:     "v1",
		Kind:           "Config",
		CurrentContext: "local",
		Contexts: []config.NamedContext{
			{
				Name: "local",
				Context: config.Context{
					Server:       srv.URL,
					RefreshToken: "everest_rt_expired",
				},
			},
		},
	}
	require.NoError(t, cfg.Save(cfgPath))

	lo := NewLogin(Config{}, zap.NewNop().Sugar())
	err := lo.Refresh(t.Context(), cfgPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "401")

	// config must be unchanged
	loaded, err := config.Load(cfgPath)
	require.NoError(t, err)
	assert.Equal(t, "everest_rt_expired", loaded.Contexts[0].Context.RefreshToken)
}

func TestRefresh_NoActiveContext(t *testing.T) {
	t.Parallel()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	cfg := &config.Config{
		APIVersion:     "v1",
		Kind:           "Config",
		CurrentContext: "missing",
	}
	require.NoError(t, cfg.Save(cfgPath))

	lo := NewLogin(Config{}, zap.NewNop().Sugar())
	err := lo.Refresh(t.Context(), cfgPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing")
}
