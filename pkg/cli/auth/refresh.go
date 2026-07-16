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
	"github.com/openeverest/openeverest/v2/pkg/cli"
	"github.com/openeverest/openeverest/v2/pkg/cli/config"
)

// Refresh exchanges the stored refresh token for a new token pair, persists it to cfgPath,
// and returns the updated session. Pass "" for contextName to use the current context.
func (lo *Login) Refresh(ctx context.Context, cfgPath, contextName string) (*cli.Session, error) {
	sess, err := cli.LoadSession(cfgPath, contextName)
	if err != nil {
		return nil, err
	}

	cfg := sess.Cfg
	currentCtx := sess.Ctx
	usr := sess.User
	srv := sess.Server

	c, err := client.NewClient(cli.NormalizeServerURL(srv.URL))
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}

	resp, err := c.CreateAuthToken(ctx, client.CreateAuthTokenJSONRequestBody{
		GrantType:    client.AuthTokenRequestGrantTypeRefreshToken,
		RefreshToken: &usr.RefreshToken,
	})
	if err != nil {
		return nil, fmt.Errorf("refresh request failed: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token refresh failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var tokenResp client.AuthTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	updatedUser := config.User{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
	}

	cfg.UpsertUser(currentCtx.User, updatedUser)

	if err := cfg.Save(cfgPath); err != nil {
		return nil, err
	}

	return &cli.Session{
		Cfg:    cfg,
		Ctx:    currentCtx,
		User:   updatedUser,
		Server: srv,
	}, nil
}
