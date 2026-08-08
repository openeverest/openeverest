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

package instance

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/openeverest/openeverest/v2/client"
	"github.com/openeverest/openeverest/v2/pkg/cli/wait"
)

// instFromJSON builds a client.Instance from a JSON literal — far more readable
// than the generated anonymous Status struct.
func instFromJSON(t *testing.T, body string) *client.Instance {
	t.Helper()
	var inst client.Instance
	require.NoError(t, json.Unmarshal([]byte(body), &inst))
	return &inst
}

// ---- instanceCondition ------------------------------------------------------

func TestInstanceCondition(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		body        string
		wantOutcome wait.Outcome
		wantMsgHas  string
	}{
		{
			name:        "ready is success",
			body:        `{"status":{"phase":"Ready"}}`,
			wantOutcome: wait.Succeeded,
			wantMsgHas:  "Ready",
		},
		{
			name:        "failed is terminal failure with message",
			body:        `{"status":{"phase":"Failed","message":"engine crashloop"}}`,
			wantOutcome: wait.Failed,
			wantMsgHas:  "engine crashloop",
		},
		{
			name:        "failed without message still fails",
			body:        `{"status":{"phase":"Failed"}}`,
			wantOutcome: wait.Failed,
			wantMsgHas:  "Failed phase",
		},
		{
			name:        "terminating is failure",
			body:        `{"status":{"phase":"Terminating"}}`,
			wantOutcome: wait.Failed,
			wantMsgHas:  "being deleted",
		},
		{
			name:        "provisioning keeps waiting",
			body:        `{"status":{"phase":"Provisioning"}}`,
			wantOutcome: wait.Pending,
			wantMsgHas:  "Provisioning",
		},
		{
			name:        "suspended is treated as pending, not failure",
			body:        `{"status":{"phase":"Suspended"}}`,
			wantOutcome: wait.Pending,
			wantMsgHas:  "Suspended",
		},
		{
			name:        "missing status is pending",
			body:        `{"metadata":{"name":"x"}}`,
			wantOutcome: wait.Pending,
			wantMsgHas:  "-",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			outcome, msg := instanceCondition(instFromJSON(t, tc.body))
			assert.Equal(t, tc.wantOutcome, outcome)
			assert.Contains(t, msg, tc.wantMsgHas)
		})
	}
}

func TestProgressMessage_WithComponents(t *testing.T) {
	t.Parallel()

	inst := instFromJSON(t, `{"status":{"phase":"Provisioning","components":[
		{"state":"initializing","ready":1,"total":3}
	]}}`)
	_, msg := instanceCondition(inst)
	assert.Equal(t, "Provisioning (initializing 1/3 ready)", msg)
}

// ---- Run --wait integration -------------------------------------------------

// waitServer serves the provider fetch, POST create, and GET polls. getPhases
// are returned in order on successive GETs (last repeats); getStatus != 200
// makes GETs return that status instead (e.g. 404).
func waitServer(t *testing.T, getStatus int, getPhases ...string) *httptest.Server {
	t.Helper()
	var gets atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/clusters/main/providers/psmdb", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(psmdbProvider())
	})
	mux.HandleFunc("/v1/clusters/main/namespaces/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"metadata":{"name":"my-db"},"status":{"phase":"Pending"}}`))
			return
		}
		if getStatus != http.StatusOK {
			w.WriteHeader(getStatus)
			return
		}
		i := int(gets.Add(1)) - 1
		if i >= len(getPhases) {
			i = len(getPhases) - 1
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(fmt.Appendf(nil, `{"metadata":{"name":"my-db"},"status":{"phase":%q}}`, getPhases[i]))
	})
	return httptest.NewServer(mux)
}

func runWaitCreate(t *testing.T, srv *httptest.Server, cfg Config, timeout time.Duration) error {
	t.Helper()
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, newTestConfig(srv.URL).Save(cfgPath))

	ic := NewInstanceCreator(cfg, zap.NewNop().Sugar())
	return ic.Run(context.Background(), CreateOptions{
		Name:      "my-db",
		Namespace: "everest",
		Provider:  "psmdb",
		Cluster:   "main",
		Wait:      true,
		Timeout:   timeout,
	}, cfgPath)
}

func TestRunWait(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		getStatus int      // status returned by GET polls; non-200 short-circuits phases
		getPhases []string // phases returned in order (last repeats)
		timeout   time.Duration
		wantErr   func(t *testing.T, err error)
	}{
		{
			name:      "succeeds when ready",
			getStatus: http.StatusOK,
			getPhases: []string{"Ready"},
			timeout:   time.Minute,
			wantErr:   func(t *testing.T, err error) { t.Helper(); assert.NoError(t, err) },
		},
		{
			name:      "failed phase returns FailedError",
			getStatus: http.StatusOK,
			getPhases: []string{"Failed"},
			timeout:   time.Minute,
			wantErr: func(t *testing.T, err error) {
				t.Helper()
				var fe *wait.FailedError
				require.ErrorAs(t, err, &fe)
				assert.Contains(t, fe.Error(), "Failed phase")
			},
		},
		{
			name:      "deleted mid-wait is terminal",
			getStatus: http.StatusNotFound,
			timeout:   time.Minute,
			wantErr: func(t *testing.T, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Contains(t, err.Error(), "was deleted")
			},
		},
		{
			// A 401 makes the auth transport attempt a token refresh; when that
			// fails it surfaces as ErrTokenRefresh, which must be terminal (not
			// retried until timeout).
			name:      "failed token refresh is terminal",
			getStatus: http.StatusUnauthorized,
			timeout:   time.Minute,
			wantErr: func(t *testing.T, err error) {
				t.Helper()
				require.Error(t, err)
				require.NotErrorIs(t, err, wait.ErrTimeout)
				assert.Contains(t, err.Error(), "refresh failed")
			},
		},
		{
			name:      "times out when never terminal",
			getStatus: http.StatusOK,
			getPhases: []string{"Provisioning"},
			timeout:   40 * time.Millisecond,
			wantErr:   func(t *testing.T, err error) { t.Helper(); require.ErrorIs(t, err, wait.ErrTimeout) },
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			srv := waitServer(t, tc.getStatus, tc.getPhases...)
			defer srv.Close()
			tc.wantErr(t, runWaitCreate(t, srv, Config{Pretty: true}, tc.timeout))
		})
	}
}

func TestRunWait_RetriesTransientFetchError(t *testing.T) {
	t.Parallel()

	// First poll returns 500 (transient), second returns Ready — the wait must
	// retry rather than fail on the transient status.
	var gets atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/clusters/main/providers/psmdb", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(psmdbProvider())
	})
	mux.HandleFunc("/v1/clusters/main/namespaces/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"metadata":{"name":"my-db"},"status":{"phase":"Pending"}}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if gets.Add(1) == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"metadata":{"name":"my-db"},"status":{"phase":"Ready"}}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	assert.NoError(t, runWaitCreate(t, srv, Config{Pretty: true}, time.Minute))
}

func TestRun_JSONErrorsOnEmptyCreateBody(t *testing.T) {
	t.Parallel()

	// 201 with no parseable body must error rather than emit empty stdout.
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/clusters/main/providers/psmdb", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(psmdbProvider())
	})
	mux.HandleFunc("/v1/clusters/main/namespaces/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, newTestConfig(srv.URL).Save(cfgPath))

	ic := NewInstanceCreator(Config{Pretty: false}, zap.NewNop().Sugar())
	err := ic.Run(context.Background(), CreateOptions{
		Name:      "my-db",
		Namespace: "everest",
		Provider:  "psmdb",
		Cluster:   "main",
	}, cfgPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unreadable response body")
}

// ---- create --json output contract ------------------------------------------

// captureStdout returns what fn writes to os.Stdout. Not parallel-safe
// (os.Stdout is global), so callers must not use t.Parallel.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	fn()
	require.NoError(t, w.Close())
	out, err := io.ReadAll(r)
	require.NoError(t, err)
	return string(out)
}

// Not parallel: captures global os.Stdout.
//
//nolint:paralleltest // mutates global os.Stdout; must run serially
func TestRun_JSONEmitsCreatedInstanceWithoutWait(t *testing.T) {
	srv := waitServer(t, http.StatusOK, "Pending")
	defer srv.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, newTestConfig(srv.URL).Save(cfgPath))

	out := captureStdout(t, func() {
		ic := NewInstanceCreator(Config{Pretty: false}, zap.NewNop().Sugar())
		err := ic.Run(context.Background(), CreateOptions{
			Name:      "my-db",
			Namespace: "everest",
			Provider:  "psmdb",
			Cluster:   "main",
		}, cfgPath)
		require.NoError(t, err)
	})

	// The created instance object must be emitted (not empty), so the contract
	// is stable whether or not --wait is passed.
	var got client.Instance
	require.NoError(t, json.Unmarshal([]byte(out), &got))
	require.NotNil(t, got.Metadata)
	assert.Equal(t, "my-db", got.Metadata.Name)
}

// ---- deleteCondition --------------------------------------------------------

func TestDeleteCondition_NilInstance_Succeeds(t *testing.T) {
	t.Parallel()
	outcome, msg := deleteCondition(nil)
	assert.Equal(t, wait.Succeeded, outcome)
	assert.Equal(t, "instance deleted", msg)
}

func TestDeleteCondition_StillPresent_IsPending(t *testing.T) {
	t.Parallel()
	inst := instFromJSON(t, `{"status":{"phase":"Terminating"}}`)
	outcome, msg := deleteCondition(inst)
	assert.Equal(t, wait.Pending, outcome)
	assert.Contains(t, msg, "Terminating")
}
