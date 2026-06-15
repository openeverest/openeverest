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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/openeverest/openeverest/v2/client"
	"github.com/openeverest/openeverest/v2/pkg/cli/config"
)

// Refresh exchanges the stored refresh token for a new token pair and persists it to cfgPath.
// The caller decides when to invoke this (e.g. on 401 or when ExpiresAt is near).
func (lo *Login) Refresh(ctx context.Context, cfgPath string) error {
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return err
	}

	var current *config.Context
	for i, nc := range cfg.Contexts {
		if nc.Name == cfg.CurrentContext {
			current = &cfg.Contexts[i].Context
			break
		}
	}
	if current == nil {
		return fmt.Errorf("no active context %q found in config", cfg.CurrentContext)
	}

	if err := validateServerURL(current.Server); err != nil {
		return fmt.Errorf("invalid server URL in config: %w", err)
	}

	c, err := client.NewClient(normalizeServerURL(current.Server))
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	resp, err := c.CreateAuthToken(ctx, client.CreateAuthTokenJSONRequestBody{
		GrantType:    client.AuthTokenRequestGrantTypeRefreshToken,
		RefreshToken: &current.RefreshToken,
	})
	if err != nil {
		return fmt.Errorf("refresh request failed: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("token refresh failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var tokenResp client.AuthTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("failed to parse token response: %w", err)
	}

	current.AccessToken = tokenResp.AccessToken
	current.RefreshToken = tokenResp.RefreshToken
	current.ExpiresAt = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	return cfg.Save(cfgPath)
}
