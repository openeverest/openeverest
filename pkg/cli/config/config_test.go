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
	assert.True(t, strings.HasSuffix(path, filepath.Join(".config", "everest", "config.yaml")))
}

func TestLoad_FileAbsent(t *testing.T) {
	t.Parallel()
	cfg, err := Load(filepath.Join(t.TempDir(), "config.yaml"))
	require.NoError(t, err)
	assert.Equal(t, "v1", cfg.APIVersion)
	assert.Equal(t, "Config", cfg.Kind)
	assert.Empty(t, cfg.Contexts)
}

func TestLoad_RoundTrip(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "config.yaml")
	expiry := time.Now().UTC().Truncate(time.Second)

	original := &Config{
		APIVersion:     "v1",
		Kind:           "Config",
		CurrentContext: "test",
		Contexts: []NamedContext{
			{
				Name: "test",
				Context: Context{
					Server:       "http://localhost:8080",
					AccessToken:  "access-jwt",
					RefreshToken: "everest_rt_abc",
					ExpiresAt:    expiry,
				},
			},
		},
	}
	require.NoError(t, original.Save(path))

	loaded, err := Load(path)
	require.NoError(t, err)
	assert.Equal(t, original.CurrentContext, loaded.CurrentContext)
	require.Len(t, loaded.Contexts, 1)
	c := loaded.Contexts[0].Context
	assert.Equal(t, "http://localhost:8080", c.Server)
	assert.Equal(t, "access-jwt", c.AccessToken)
	assert.Equal(t, "everest_rt_abc", c.RefreshToken)
	assert.Equal(t, expiry, c.ExpiresAt.UTC().Truncate(time.Second))
}

func TestConfig_Save_Permissions(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "sub", "config.yaml")
	cfg := &Config{APIVersion: "v1", Kind: "Config"}
	require.NoError(t, cfg.Save(path))

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}

func TestConfig_UpsertContext_Insert(t *testing.T) {
	t.Parallel()
	cfg := &Config{APIVersion: "v1", Kind: "Config"}
	cfg.UpsertContext("local", Context{Server: "http://localhost:8080", AccessToken: "tok"})
	require.Len(t, cfg.Contexts, 1)
	assert.Equal(t, "local", cfg.Contexts[0].Name)
	assert.Equal(t, "tok", cfg.Contexts[0].Context.AccessToken)
}

func TestConfig_UpsertContext_Update(t *testing.T) {
	t.Parallel()
	cfg := &Config{
		APIVersion: "v1",
		Kind:       "Config",
		Contexts:   []NamedContext{{Name: "local", Context: Context{AccessToken: "old"}}},
	}
	cfg.UpsertContext("local", Context{AccessToken: "new"})
	require.Len(t, cfg.Contexts, 1)
	assert.Equal(t, "new", cfg.Contexts[0].Context.AccessToken)
}
