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

package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultPath(t *testing.T) {
	t.Parallel()
	path, err := DefaultPath()
	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(path, filepath.Join("everest", "config.yaml")))
}

func TestLoad_FileAbsent(t *testing.T) {
	t.Parallel()
	cfg, err := Load(filepath.Join(t.TempDir(), "config.yaml"))
	require.NoError(t, err)
	assert.Equal(t, "config.openeverest.io/v1alpha1", cfg.APIVersion)
	assert.Equal(t, "ClientConfig", cfg.Kind)
	assert.Empty(t, cfg.Contexts)
	assert.Empty(t, cfg.Servers)
	assert.Empty(t, cfg.Users)
}

func TestLoad_RoundTrip(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "config.yaml")
	expiry := time.Now().UTC().Truncate(time.Second)

	original := &Config{
		APIVersion:     "config.openeverest.io/v1alpha1",
		Kind:           "ClientConfig",
		CurrentContext: "admin@localhost:8080",
		Contexts: []NamedContext{
			{Name: "admin@localhost:8080", Context: Context{Server: "localhost:8080", User: "admin@localhost:8080"}},
		},
		Servers: []NamedServer{
			{Name: "localhost:8080", Server: Server{URL: "http://localhost:8080"}},
		},
		Users: []NamedUser{
			{Name: "admin@localhost:8080", User: User{
				AccessToken:  "access-jwt",
				RefreshToken: "everest_rt_abc",
				ExpiresAt:    expiry,
			}},
		},
	}
	require.NoError(t, original.Save(path))

	loaded, err := Load(path)
	require.NoError(t, err)
	assert.Equal(t, original.CurrentContext, loaded.CurrentContext)
	require.Len(t, loaded.Contexts, 1)
	require.Len(t, loaded.Servers, 1)
	require.Len(t, loaded.Users, 1)
	assert.Equal(t, "localhost:8080", loaded.Contexts[0].Context.Server)
	assert.Equal(t, "admin@localhost:8080", loaded.Contexts[0].Context.User)
	assert.Equal(t, "http://localhost:8080", loaded.Servers[0].Server.URL)
	u := loaded.Users[0].User
	assert.Equal(t, "access-jwt", u.AccessToken)
	assert.Equal(t, "everest_rt_abc", u.RefreshToken)
	assert.Equal(t, expiry, u.ExpiresAt.UTC().Truncate(time.Second))
}

func TestConfig_Save_Permissions(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "sub", "config.yaml")
	cfg := &Config{APIVersion: "config.openeverest.io/v1alpha1", Kind: "ClientConfig"}
	require.NoError(t, cfg.Save(path))

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestConfig_UpsertContext_Insert(t *testing.T) {
	t.Parallel()
	cfg := &Config{}
	cfg.UpsertContext("admin@local", Context{Server: "local", User: "admin@local"})
	require.Len(t, cfg.Contexts, 1)
	assert.Equal(t, "admin@local", cfg.Contexts[0].Name)
	assert.Equal(t, "local", cfg.Contexts[0].Context.Server)
	assert.Equal(t, "admin@local", cfg.Contexts[0].Context.User)
}

func TestConfig_UpsertContext_Update(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		Contexts: []NamedContext{{Name: "admin@local", Context: Context{Server: "old", User: "old"}}},
	}
	cfg.UpsertContext("admin@local", Context{Server: "new", User: "new"})
	require.Len(t, cfg.Contexts, 1)
	assert.Equal(t, "new", cfg.Contexts[0].Context.Server)
}

func TestConfig_UpsertServer_Insert(t *testing.T) {
	t.Parallel()
	cfg := &Config{}
	cfg.UpsertServer("local", Server{URL: "http://localhost:8080"})
	require.Len(t, cfg.Servers, 1)
	assert.Equal(t, "local", cfg.Servers[0].Name)
	assert.Equal(t, "http://localhost:8080", cfg.Servers[0].Server.URL)
}

func TestConfig_UpsertServer_Update(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		Servers: []NamedServer{{Name: "local", Server: Server{URL: "http://old"}}},
	}
	cfg.UpsertServer("local", Server{URL: "http://new"})
	require.Len(t, cfg.Servers, 1)
	assert.Equal(t, "http://new", cfg.Servers[0].Server.URL)
}

func TestConfig_UpsertUser_Insert(t *testing.T) {
	t.Parallel()
	cfg := &Config{}
	cfg.UpsertUser("admin@local", User{AccessToken: "tok"})
	require.Len(t, cfg.Users, 1)
	assert.Equal(t, "admin@local", cfg.Users[0].Name)
	assert.Equal(t, "tok", cfg.Users[0].User.AccessToken)
}

func TestConfig_UpsertUser_Update(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		Users: []NamedUser{{Name: "admin@local", User: User{AccessToken: "old"}}},
	}
	cfg.UpsertUser("admin@local", User{AccessToken: "new"})
	require.Len(t, cfg.Users, 1)
	assert.Equal(t, "new", cfg.Users[0].User.AccessToken)
}

func TestConfig_GetCurrentContext(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		CurrentContext: "admin@local",
		Contexts: []NamedContext{
			{Name: "admin@local", Context: Context{Server: "local", User: "admin@local"}},
		},
	}
	ctx, ok := cfg.GetCurrentContext()
	require.True(t, ok)
	assert.Equal(t, "local", ctx.Server)

	cfg.CurrentContext = "missing"
	_, ok = cfg.GetCurrentContext()
	assert.False(t, ok)
}

func TestConfig_GetServer(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		Servers: []NamedServer{{Name: "local", Server: Server{URL: "http://localhost:8080"}}},
	}
	srv, ok := cfg.GetServer("local")
	require.True(t, ok)
	assert.Equal(t, "http://localhost:8080", srv.URL)

	_, ok = cfg.GetServer("missing")
	assert.False(t, ok)
}

func TestConfig_GetUser(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		Users: []NamedUser{{Name: "admin@local", User: User{AccessToken: "tok"}}},
	}
	usr, ok := cfg.GetUser("admin@local")
	require.True(t, ok)
	assert.Equal(t, "tok", usr.AccessToken)

	_, ok = cfg.GetUser("missing")
	assert.False(t, ok)
}
