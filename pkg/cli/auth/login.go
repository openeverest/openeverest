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

// Package auth provides CLI authentication functionality.
package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/openeverest/openeverest/v2/client"
	"github.com/openeverest/openeverest/v2/pkg/cli"
	"github.com/openeverest/openeverest/v2/pkg/cli/config"
	"github.com/openeverest/openeverest/v2/pkg/output"
)

// Config holds configuration for auth commands.
type Config struct {
	Pretty bool
}

// LoginOptions holds the parameters for a login attempt.
type LoginOptions struct {
	Server      string
	Username    string
	Password    string
	ContextName string // optional; derived from Server if empty
}

// Login provides login and token-refresh functionality.
type Login struct {
	l      *zap.SugaredLogger
	config Config
}

// NewLogin creates a new Login.
func NewLogin(cfg Config, l *zap.SugaredLogger) *Login {
	lo := &Login{config: cfg, l: l.With("component", "auth")}
	if cfg.Pretty {
		lo.l = zap.NewNop().Sugar()
	}
	return lo
}

// Run exchanges username/password for tokens and saves them to cfgPath.
func (lo *Login) Run(ctx context.Context, opts LoginOptions, cfgPath string) error {
	if err := cli.ValidateServerURL(opts.Server); err != nil {
		return err
	}

	c, err := client.NewClient(cli.NormalizeServerURL(opts.Server))
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	delivery := client.AuthTokenRequestRefreshTokenDeliveryBody
	resp, err := c.CreateAuthToken(ctx, client.CreateAuthTokenJSONRequestBody{
		GrantType:            client.AuthTokenRequestGrantTypePassword,
		Username:             &opts.Username,
		Password:             &opts.Password,
		RefreshTokenDelivery: &delivery,
	})
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("authentication failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var tokenResp client.AuthTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("failed to parse token response: %w", err)
	}

	srvName := serverName(opts.Server)
	userName := opts.Username + "@" + srvName
	contextName := opts.ContextName
	if contextName == "" {
		contextName = userName
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		return err
	}
	cfg.UpsertServer(srvName, config.Server{URL: opts.Server})
	cfg.UpsertUser(userName, config.User{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
	})
	cfg.UpsertContext(contextName, config.Context{Server: srvName, User: userName})
	cfg.CurrentContext = contextName

	if err := cfg.Save(cfgPath); err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	lo.l.Infof("Logged in to %s as %s", opts.Server, opts.Username)
	if lo.config.Pretty {
		_, _ = fmt.Fprint(os.Stdout, output.Success("Logged in to %s as %s", opts.Server, opts.Username))
	}
	return nil
}

// serverName strips the URL scheme from server, returning just the host[:port].
func serverName(server string) string {
	for _, prefix := range []string{"https://", "http://"} {
		if strings.HasPrefix(server, prefix) {
			return server[len(prefix):]
		}
	}
	return server
}
