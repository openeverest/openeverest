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

package backup

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openeverest/openeverest/v2/client"
	"github.com/openeverest/openeverest/v2/pkg/cli"
	authcli "github.com/openeverest/openeverest/v2/pkg/cli/auth"
	"github.com/openeverest/openeverest/v2/pkg/cli/wait"
)

// backupFromJSON builds a client.Backup from a JSON literal.
func backupFromJSON(t *testing.T, body string) *client.Backup {
	t.Helper()
	var b client.Backup
	require.NoError(t, json.Unmarshal([]byte(body), &b))
	return &b
}

// newTestClient builds an unauthenticated client for testing a poll/query
// func directly, bypassing the config-and-auth-backed Run path. It still
// normalizes the URL the same way authcli.NewAPIClient does (appending /v1),
// so mux routes registered here match what the real client actually requests.
func newTestClient(serverURL string) (*client.ClientWithResponses, error) {
	return client.NewClientWithResponses(cli.NormalizeServerURL(serverURL))
}

func TestBackupCondition(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		body        string
		wantOutcome wait.Outcome
		wantMsgHas  string
	}{
		{
			name:        "succeeded is success",
			body:        `{"status":{"state":"Succeeded"}}`,
			wantOutcome: wait.Succeeded,
		},
		{
			name:        "failed with message",
			body:        `{"status":{"state":"Failed","message":"engine returned a non-zero exit code"}}`,
			wantOutcome: wait.Failed,
			wantMsgHas:  "engine returned a non-zero exit code",
		},
		{
			name:        "failed without message",
			body:        `{"status":{"state":"Failed"}}`,
			wantOutcome: wait.Failed,
			wantMsgHas:  "entered the Failed state",
		},
		{
			name:        "running is pending",
			body:        `{"status":{"state":"Running"}}`,
			wantOutcome: wait.Pending,
			wantMsgHas:  "Running",
		},
		{
			// Regression guard: Error is retryable, not terminal.
			name:        "error state is pending, not failed",
			body:        `{"status":{"state":"Error"}}`,
			wantOutcome: wait.Pending,
			wantMsgHas:  "Error",
		},
		{
			name:        "no status is pending",
			body:        `{}`,
			wantOutcome: wait.Pending,
			wantMsgHas:  "-",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			outcome, msg := backupCondition(backupFromJSON(t, tc.body))
			assert.Equal(t, tc.wantOutcome, outcome)
			if tc.wantMsgHas != "" {
				assert.Contains(t, msg, tc.wantMsgHas)
			}
		})
	}
}

func TestNewBackupPoll_DeletedMidWait(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/clusters/main/namespaces/everest/backups/gone", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c, err := newTestClient(srv.URL)
	require.NoError(t, err)

	poll := newBackupPoll(c, "main", "everest", "gone")
	_, err = poll(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "was deleted while waiting")
}

func TestNewBackupPoll_UnauthorizedIsTerminal(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/clusters/main/namespaces/everest/backups/my-mongo-abcde", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c, err := newTestClient(srv.URL)
	require.NoError(t, err)

	poll := newBackupPoll(c, "main", "everest", "my-mongo-abcde")
	_, err = poll(context.Background())
	require.Error(t, err)
	var re *wait.RetryableError
	require.NotErrorAs(t, err, &re, "401 must be terminal, not retryable")
	assert.Contains(t, err.Error(), "run 'everestctl auth login' again")
}

func TestNewBackupPoll_FailedTokenRefreshIsTerminal(t *testing.T) {
	t.Parallel()

	// A RoundTripper simulating the auth transport's behavior when a 401
	// triggers a token refresh that itself fails.
	rt := roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return nil, fmt.Errorf("wrap: %w", authcli.ErrTokenRefresh)
	})
	c, err := client.NewClientWithResponses("http://example.invalid", client.WithHTTPClient(&http.Client{Transport: rt}))
	require.NoError(t, err)

	poll := newBackupPoll(c, "main", "everest", "my-mongo-abcde")
	_, err = poll(context.Background())
	require.Error(t, err)
	require.ErrorIs(t, err, authcli.ErrTokenRefresh)
	var re *wait.RetryableError
	require.NotErrorAs(t, err, &re, "a failed token refresh must be terminal, not retryable")
}

func TestNewBackupPoll_UnexpectedStatusIsRetryable(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/clusters/main/namespaces/everest/backups/my-mongo-abcde", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c, err := newTestClient(srv.URL)
	require.NoError(t, err)

	poll := newBackupPoll(c, "main", "everest", "my-mongo-abcde")
	_, err = poll(context.Background())
	require.Error(t, err)
	var re *wait.RetryableError
	require.ErrorAs(t, err, &re, "a transient 500 must be retryable, not terminal")
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
