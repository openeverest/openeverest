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

package cli

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/openeverest/openeverest/v2/client"
	"github.com/openeverest/openeverest/v2/pkg/cli/config"
)

// Session holds resolved credentials for a single Everest context.
type Session struct {
	Cfg    *config.Config
	Ctx    config.Context
	User   config.User
	Server config.Server
}

// LoadSession loads the config, resolves the named context (or current context
// when name is empty), and validates the server URL.
func LoadSession(cfgPath, contextName string) (*Session, error) {
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return nil, err
	}

	var (
		ctx config.Context
		ok  bool
	)
	if contextName != "" {
		ctx, ok = cfg.GetContext(contextName)
		if !ok {
			return nil, fmt.Errorf("context %q not found in config", contextName)
		}
	} else {
		ctx, ok = cfg.GetCurrentContext()
		if !ok {
			return nil, fmt.Errorf("no active context found in config — run 'everestctl auth login' first")
		}
	}

	usr, ok := cfg.GetUser(ctx.User)
	if !ok {
		return nil, fmt.Errorf("user %q not found in config", ctx.User)
	}

	srv, ok := cfg.GetServer(ctx.Server)
	if !ok {
		return nil, fmt.Errorf("server %q not found in config", ctx.Server)
	}

	if err := ValidateServerURL(srv.URL); err != nil {
		return nil, fmt.Errorf("invalid server URL in config: %w", err)
	}

	return &Session{Cfg: cfg, Ctx: ctx, User: usr, Server: srv}, nil
}

// ValidateServerURL returns an error if the URL scheme is not http or https.
func ValidateServerURL(server string) error {
	if !strings.HasPrefix(server, "http://") && !strings.HasPrefix(server, "https://") {
		return fmt.Errorf("server URL must start with http:// or https://")
	}
	return nil
}

// NormalizeServerURL ensures the URL ends with /v1.
func NormalizeServerURL(server string) string {
	server = strings.TrimRight(server, "/")
	if !strings.HasSuffix(server, "/v1") {
		server += "/v1"
	}
	return server
}

// BearerToken returns a request editor that sets the Authorization header.
func BearerToken(token string) client.RequestEditorFn {
	return func(_ context.Context, req *http.Request) error {
		req.Header.Set("Authorization", "Bearer "+token)
		return nil
	}
}
