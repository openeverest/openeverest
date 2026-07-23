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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openeverest/openeverest/v2/client"
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
// func directly, bypassing the config-and-auth-backed Run path.
func newTestClient(serverURL string) (*client.ClientWithResponses, error) {
	return client.NewClientWithResponses(serverURL)
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
