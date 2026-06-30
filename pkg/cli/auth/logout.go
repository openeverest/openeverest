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
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/openeverest/openeverest/v2/client"
	"github.com/openeverest/openeverest/v2/pkg/cli"
)

// Logout revokes the current session on the server and removes its context, user,
// and (if unreferenced) server entry from the local config.
//
// The /auth/revoke endpoint is unauthenticated per RFC 7009: the refresh token in
// the request body is sufficient proof of session ownership. If the access token is
// still valid it is included opportunistically so the server can blocklist it;
// if expired it is omitted and the server skips blocklisting (acceptable given the
// 15-minute TTL). Local credentials are always cleared regardless of server response.
func (lo *Login) Logout(ctx context.Context, cfgPath string) error {
	sess, err := cli.LoadSession(cfgPath, "")
	if err != nil {
		return err
	}

	cfg := sess.Cfg
	currentCtx := sess.Ctx
	usr := sess.User
	srv := sess.Server

	c, err := client.NewClient(cli.NormalizeServerURL(srv.URL))
	if err != nil {
		return fmt.Errorf("failed to create API client: %w", err)
	}

	// Include the access token only if it is still valid so the server can
	// blocklist it. If expired, omit it — the refresh token alone is enough.
	var reqEditors []client.RequestEditorFn
	if time.Now().Before(usr.ExpiresAt) {
		reqEditors = append(reqEditors, cli.BearerToken(usr.AccessToken))
	}

	resp, err := c.RevokeAuthToken(ctx, client.RevokeAuthTokenJSONRequestBody{
		Token: &usr.RefreshToken,
	}, reqEditors...)
	if err != nil {
		lo.l.Warnf("revoke request failed: %v — clearing local credentials anyway", err)
	} else {
		defer resp.Body.Close() //nolint:errcheck
		if resp.StatusCode != http.StatusNoContent {
			body, _ := io.ReadAll(resp.Body)
			lo.l.Warnf("server returned %d during logout: %s — clearing local credentials anyway",
				resp.StatusCode, strings.TrimSpace(string(body)))
		}
	}

	// Always clear local credentials regardless of server response.
	contextName := cfg.CurrentContext
	srvName := currentCtx.Server
	userName := currentCtx.User

	cfg.RemoveContext(contextName)
	cfg.CurrentContext = ""

	if !cfg.IsUserReferenced(userName) {
		cfg.RemoveUser(userName)
	}
	if !cfg.IsServerReferenced(srvName) {
		cfg.RemoveServer(srvName)
	}

	return cfg.Save(cfgPath)
}

