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
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/openeverest/openeverest/v2/client"
	"github.com/openeverest/openeverest/v2/pkg/cli"
)

// ErrTokenRefresh is returned when token refresh fails after a 401 or proactive expiry
// check. Watch loops should treat it as terminal rather than retrying.
var ErrTokenRefresh = errors.New("access token expired and refresh failed")

// tokenSource yields valid access tokens, proactively refreshing within 30s of expiry.
// It is safe for concurrent use.
type tokenSource struct {
	mu          sync.Mutex
	sess        *cli.Session
	login       *Login
	cfgPath     string
	contextName string
}

// token returns a valid access token, refreshing first if within 30s of expiry.
func (ts *tokenSource) token(ctx context.Context) (string, error) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if time.Now().After(ts.sess.User.ExpiresAt.Add(-30 * time.Second)) {
		newSess, err := ts.login.Refresh(ctx, ts.cfgPath, ts.contextName)
		if err != nil {
			return "", fmt.Errorf("%w: %w", ErrTokenRefresh, err)
		}
		ts.sess = newSess
	}

	return ts.sess.User.AccessToken, nil
}

// forceRefresh unconditionally fetches a new token pair, ignoring the cached expiry.
func (ts *tokenSource) forceRefresh(ctx context.Context) (string, error) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	newSess, err := ts.login.Refresh(ctx, ts.cfgPath, ts.contextName)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrTokenRefresh, err)
	}
	ts.sess = newSess

	return ts.sess.User.AccessToken, nil
}

// authTransport is an http.RoundTripper that injects Bearer tokens and handles
// 401 responses by refreshing the token and retrying the request once.
type authTransport struct {
	source *tokenSource
	base   http.RoundTripper
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	tok, err := t.source.token(req.Context())
	if err != nil {
		return nil, err
	}

	// Clone before mutating — required by the RoundTripper contract.
	req2 := req.Clone(req.Context())
	req2.Header.Set("Authorization", "Bearer "+tok)

	resp, err := t.base.RoundTrip(req2)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusUnauthorized {
		return resp, nil
	}

	// 401 — only retry when the body can be replayed.
	if req.GetBody == nil && req.Body != nil {
		return resp, nil
	}

	io.Copy(io.Discard, resp.Body) //nolint:errcheck,gosec
	resp.Body.Close()              //nolint:errcheck,gosec

	newTok, err := t.source.forceRefresh(req.Context())
	if err != nil {
		return nil, err
	}

	retryReq := req.Clone(req.Context())
	if req.GetBody != nil {
		body, err := req.GetBody()
		if err != nil {
			return nil, err
		}
		retryReq.Body = body
	}
	retryReq.Header.Set("Authorization", "Bearer "+newTok)

	return t.base.RoundTrip(retryReq)
}

// NewAPIClient returns a ClientWithResponses wired with automatic token refresh.
// Proactive rotation (30s before expiry) and 401 retry are transparent to callers.
func NewAPIClient(cfg Config, l *zap.SugaredLogger, cfgPath, contextName string) (*client.ClientWithResponses, error) {
	sess, err := cli.LoadSession(cfgPath, contextName)
	if err != nil {
		return nil, err
	}

	ts := &tokenSource{
		sess:        sess,
		login:       NewLogin(cfg, l),
		cfgPath:     cfgPath,
		contextName: contextName,
	}

	c, err := client.NewClientWithResponses(
		cli.NormalizeServerURL(sess.Server.URL),
		client.WithHTTPClient(&http.Client{
			Transport: &authTransport{source: ts, base: http.DefaultTransport},
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}
	return c, nil
}
