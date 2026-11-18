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
	"strings"
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

// conditionJSON is one status condition, observed for generation 1.
func conditionJSON(conditionType, status, reason, message string) string {
	return fmt.Sprintf(
		`{"type":%q,"status":%q,"reason":%q,"message":%q,"observedGeneration":1,"lastTransitionTime":"2026-01-01T00:00:00Z"}`,
		conditionType, status, reason, message,
	)
}

// statusJSON is the status of an Instance whose generation 1 was observed.
func statusJSON(phase string, conditions ...string) string {
	return fmt.Sprintf(`{"observedGeneration":1,"phase":%q,"conditions":[%s]}`, phase, strings.Join(conditions, ","))
}

// creatingStatus is the status of an Instance that is being brought up.
func creatingStatus() string {
	return statusJSON(
		"Pending",
		conditionJSON("Available", "False", "NotCreated", "The database has not been available yet"),
		conditionJSON("Ready", "False", "Creating", "Waiting for the replica set to initialise"),
		conditionJSON("Reconciling", "True", "Creating", "Waiting for the replica set to initialise"),
		conditionJSON("Stalled", "False", "NoProblems", ""),
		conditionJSON("NeedsAttention", "False", "NoProblems", ""),
	)
}

// readyStatus is the status of an Instance that has converged.
func readyStatus() string {
	return statusJSON(
		"Provisioned",
		conditionJSON("Available", "True", "Available", ""),
		conditionJSON("Ready", "True", "Ready", "The instance is ready"),
		conditionJSON("Reconciling", "False", "NoProblems", ""),
		conditionJSON("Stalled", "False", "NoProblems", ""),
		conditionJSON("NeedsAttention", "False", "NoProblems", ""),
	)
}

// stalledStatus is the status of an Instance whose spec the provider rejected.
func stalledStatus() string {
	return statusJSON(
		"Pending",
		conditionJSON("Available", "False", "NotCreated", "The database has not been available yet"),
		conditionJSON("Ready", "False", "ValidationFailed", "replicas must be odd"),
		conditionJSON("Reconciling", "False", "ValidationFailed", ""),
		conditionJSON("Stalled", "True", "ValidationFailed", "replicas must be odd"),
		conditionJSON("NeedsAttention", "True", "ValidationFailed", "replicas must be odd"),
	)
}

// missingSecretStatus is the status of an Instance waiting for a Secret that
// may still arrive.
func missingSecretStatus() string {
	return statusJSON(
		"Pending",
		conditionJSON("Ready", "False", "UserSecretNotFound", "secret s not found"),
		conditionJSON("Reconciling", "True", "UserSecretNotFound", "secret s not found"),
		conditionJSON("Stalled", "False", "NoProblems", ""),
		conditionJSON("NeedsAttention", "True", "UserSecretNotFound", "secret s not found"),
	)
}

// stoppedStatus is the status of an Instance stopped because its spec asks
// for it.
func stoppedStatus() string {
	return statusJSON(
		"Provisioned",
		conditionJSON("Available", "False", "Suspended", "Compute is scaled to zero"),
		conditionJSON("Ready", "True", "Suspended", "The instance is stopped"),
		conditionJSON("Suspended", "True", "Suspended", "Compute is scaled to zero"),
	)
}

// instanceJSON is an Instance at the given generation with the given status.
func instanceJSON(generation int, status string) string {
	return fmt.Sprintf(`{"metadata":{"name":"my-db","generation":%d},"status":%s}`, generation, status)
}

func TestInstanceCondition(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		body        string
		wantOutcome wait.Outcome
		wantMsg     string
	}{
		{
			name:        "ready is success",
			body:        instanceJSON(1, readyStatus()),
			wantOutcome: wait.Succeeded,
			wantMsg:     "Up · Ready",
		},
		{
			name:        "being created keeps waiting",
			body:        instanceJSON(1, creatingStatus()),
			wantOutcome: wait.Pending,
			wantMsg:     "Creating",
		},
		{
			name:        "not reconciled yet keeps waiting",
			body:        `{"metadata":{"name":"my-db","generation":1},"status":{"observedGeneration":-1}}`,
			wantOutcome: wait.Pending,
			wantMsg:     "Creating",
		},
		{
			name:        "missing status keeps waiting",
			body:        `{"metadata":{"name":"my-db"}}`,
			wantOutcome: wait.Pending,
			wantMsg:     "Creating",
		},
		{
			name:        "stalled is a failure that says why",
			body:        instanceJSON(1, stalledStatus()),
			wantOutcome: wait.Failed,
			wantMsg:     "instance will not become ready without intervention: Invalid configuration: replicas must be odd",
		},
		{
			name:        "a stall on an older spec keeps waiting: a fix may have been pushed",
			body:        instanceJSON(2, stalledStatus()),
			wantOutcome: wait.Pending,
			wantMsg:     "Creating",
		},
		{
			name:        "ready for an older spec keeps waiting",
			body:        instanceJSON(2, readyStatus()),
			wantOutcome: wait.Pending,
			wantMsg:     "Up · Applying changes",
		},
		{
			name:        "a problem that may still clear keeps waiting",
			body:        instanceJSON(1, missingSecretStatus()),
			wantOutcome: wait.Pending,
			wantMsg:     "Secret not found",
		},
		{
			name:        "stopped on purpose has converged",
			body:        instanceJSON(1, stoppedStatus()),
			wantOutcome: wait.Succeeded,
			wantMsg:     "Stopped",
		},
		{
			name: "being deleted is a failure",
			body: `{"metadata":{"name":"my-db","generation":2,"deletionTimestamp":"2026-01-01T00:00:00Z"},` +
				`"status":` + readyStatus() + `}`,
			wantOutcome: wait.Failed,
			wantMsg:     "instance is being deleted",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			outcome, msg := instanceCondition(instFromJSON(t, tc.body))
			assert.Equal(t, tc.wantOutcome, outcome)
			assert.Equal(t, tc.wantMsg, msg)
		})
	}
}

func TestProgressMessage_WithComponents(t *testing.T) {
	t.Parallel()

	inst := instFromJSON(t, `{"metadata":{"generation":1},"status":{"observedGeneration":1,"phase":"Pending","components":[
		{"name":"engine","readyReplicas":2,"replicas":3},
		{"name":"proxy","readyReplicas":1,"replicas":1}
	]}}`)
	_, msg := instanceCondition(inst)
	assert.Equal(t, "Creating (engine 2/3 ready, proxy 1/1 ready)", msg)
}

// ---- Run --wait integration -------------------------------------------------

// waitServer serves the provider fetch, POST create, and GET polls. The
// Instance statuses in getStatuses are returned in order on successive GETs
// (last repeats); getStatus != 200 makes GETs return that status instead
// (e.g. 404).
func waitServer(t *testing.T, getStatus int, getStatuses ...string) *httptest.Server {
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
			_, _ = w.Write([]byte(createdInstance))
			return
		}
		if getStatus != http.StatusOK {
			w.WriteHeader(getStatus)
			return
		}
		i := int(gets.Add(1)) - 1
		if i >= len(getStatuses) {
			i = len(getStatuses) - 1
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(instanceJSON(1, getStatuses[i])))
	})
	return httptest.NewServer(mux)
}

// createdInstance is what the API returns for an Instance it has just
// created: no provider has reconciled it, so its status is the API default.
const createdInstance = `{"metadata":{"name":"my-db","generation":1},"status":{"observedGeneration":-1}}`

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
		name        string
		getStatus   int      // status returned by GET polls; non-200 short-circuits statuses
		getStatuses []string // Instance statuses returned in order (last repeats)
		timeout     time.Duration
		wantErr     func(t *testing.T, err error)
	}{
		{
			name:        "succeeds when ready",
			getStatus:   http.StatusOK,
			getStatuses: []string{readyStatus()},
			timeout:     time.Minute,
			wantErr:     func(t *testing.T, err error) { t.Helper(); assert.NoError(t, err) },
		},
		{
			name:        "succeeds once it becomes ready",
			getStatus:   http.StatusOK,
			getStatuses: []string{`{"observedGeneration":-1}`, creatingStatus(), readyStatus()},
			timeout:     time.Minute,
			wantErr:     func(t *testing.T, err error) { t.Helper(); assert.NoError(t, err) },
		},
		{
			name:        "stalled returns FailedError",
			getStatus:   http.StatusOK,
			getStatuses: []string{creatingStatus(), stalledStatus()},
			timeout:     time.Minute,
			wantErr: func(t *testing.T, err error) {
				t.Helper()
				var fe *wait.FailedError
				require.ErrorAs(t, err, &fe)
				assert.Contains(t, fe.Error(), "Invalid configuration: replicas must be odd")
			},
		},
		{
			name:      "deleted mid-wait is terminal",
			getStatus: http.StatusNotFound,
			timeout:   time.Minute,
			wantErr: func(t *testing.T, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Contains(t, err.Error(), "no longer exists")
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
			name:        "times out when never terminal",
			getStatus:   http.StatusOK,
			getStatuses: []string{creatingStatus()},
			timeout:     40 * time.Millisecond,
			wantErr:     func(t *testing.T, err error) { t.Helper(); require.ErrorIs(t, err, wait.ErrTimeout) },
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			srv := waitServer(t, tc.getStatus, tc.getStatuses...)
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
			_, _ = w.Write([]byte(createdInstance))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if gets.Add(1) == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(instanceJSON(1, readyStatus())))
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
	srv := waitServer(t, http.StatusOK, creatingStatus())
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
	inst := instFromJSON(t, `{"metadata":{"generation":1,"deletionTimestamp":"2026-01-01T00:00:00Z"},`+
		`"status":{"observedGeneration":1,"phase":"Terminating","conditions":[`+
		conditionJSON("Ready", "False", "WaitingForBackups", "Waiting for 2 backups to be deleted")+`],`+
		`"components":[{"name":"engine","readyReplicas":0,"replicas":3}]}}`)
	outcome, msg := deleteCondition(inst)
	assert.Equal(t, wait.Pending, outcome)
	assert.Equal(t, "Deleting · Waiting for backups (engine 0/3 ready)", msg)
}
