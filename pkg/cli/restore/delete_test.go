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

package restore

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/openeverest/openeverest/v2/client"
	"github.com/openeverest/openeverest/v2/pkg/cli/confirm"
	"github.com/openeverest/openeverest/v2/pkg/cli/wait"
)

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

func isTerminalFalse() bool { return false }

// newDeleteServer fakes the restore DELETE (and optional GET) endpoint.
func newDeleteServer(t *testing.T, deleteHandler, getHandler http.HandlerFunc) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	path := "/v1/clusters/main/namespaces/everest/restores/my-mongo-restore-x7k2q"
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodDelete:
			deleteHandler(w, r)
		case http.MethodGet:
			if getHandler != nil {
				getHandler(w, r)
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	return httptest.NewServer(mux)
}

// getHandlerWithState replies 200 with a restore fixture in the given state.
// An empty state omits the status block entirely (unknown/never-observed).
func getHandlerWithState(t *testing.T, state client.RestoreStatusState) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(restoreWithState(t, "my-mongo-restore-x7k2q", state))
	}
}

// yesOpts sets --yes so tests skip the interactive prompt.
func yesOpts() DeleteOptions {
	return DeleteOptions{
		Name:      "my-mongo-restore-x7k2q",
		Namespace: "everest",
		Cluster:   "main",
		Yes:       true,
	}
}

func TestDelete_HappyPath_NoWait(t *testing.T) {
	t.Parallel()

	srv := newDeleteServer(t,
		func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) },
		getHandlerWithState(t, client.RestoreStatusStateSucceeded),
	)
	defer srv.Close()

	rd := NewDeleter(Config{}, zap.NewNop().Sugar())
	err := rd.Run(context.Background(), yesOpts(), newConfigPath(t, srv.URL))
	assert.NoError(t, err)
}

func TestDelete_NotFound_WithoutIgnoreNotFound_Errors(t *testing.T) {
	t.Parallel()

	deleteCalled := false
	srv := newDeleteServer(t, func(w http.ResponseWriter, _ *http.Request) {
		deleteCalled = true
		w.WriteHeader(http.StatusNotFound)
	}, nil)
	defer srv.Close()

	rd := NewDeleter(Config{}, zap.NewNop().Sugar())
	err := rd.Run(context.Background(), yesOpts(), newConfigPath(t, srv.URL))
	require.Error(t, err)
	assert.Contains(t, err.Error(), `restore "my-mongo-restore-x7k2q" not found in namespace "everest"`)
	assert.False(t, deleteCalled, "a restore we already know is gone must not reach the DELETE call")
}

// TestDelete_NotFound_WithoutIgnoreNotFound_SkipsConfirmation proves the
// review fix: knowing the restore is already gone means we don't need to
// prompt to find that out, even without --yes or a TTY.
func TestDelete_NotFound_WithoutIgnoreNotFound_SkipsConfirmation(t *testing.T) {
	t.Parallel()

	deleteCalled := false
	srv := newDeleteServer(t, func(w http.ResponseWriter, _ *http.Request) {
		deleteCalled = true
		w.WriteHeader(http.StatusNoContent)
	}, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) })
	defer srv.Close()

	opts := DeleteOptions{
		Name:       "my-mongo-restore-x7k2q",
		Namespace:  "everest",
		Cluster:    "main",
		IsTerminal: isTerminalFalse,
	}

	rd := NewDeleter(Config{}, zap.NewNop().Sugar())
	err := rd.Run(context.Background(), opts, newConfigPath(t, srv.URL))
	require.Error(t, err)
	assert.Contains(t, err.Error(), `restore "my-mongo-restore-x7k2q" not found in namespace "everest"`)
	assert.NotContains(t, err.Error(), "confirmation required", "already knowing it's gone must skip the prompt entirely")
	assert.False(t, deleteCalled, "delete must not be issued for a restore we already know is gone")
}

func TestDelete_NotFound_WithIgnoreNotFound_Succeeds(t *testing.T) {
	t.Parallel()

	srv := newDeleteServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}, nil)
	defer srv.Close()

	opts := yesOpts()
	opts.IgnoreNotFound = true

	rd := NewDeleter(Config{}, zap.NewNop().Sugar())
	err := rd.Run(context.Background(), opts, newConfigPath(t, srv.URL))
	assert.NoError(t, err)
}

// TestDelete_IgnoreNotFound_AlreadyGone_SkipsConfirmationAndWait proves
// confirm and --wait are both skipped when the restore is already gone.
func TestDelete_IgnoreNotFound_AlreadyGone_SkipsConfirmationAndWait(t *testing.T) {
	t.Parallel()

	deleteCalled := false
	srv := newDeleteServer(t,
		func(w http.ResponseWriter, _ *http.Request) {
			deleteCalled = true
			w.WriteHeader(http.StatusNoContent)
		},
		func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) },
	)
	defer srv.Close()

	opts := DeleteOptions{
		Name:           "my-mongo-restore-x7k2q",
		Namespace:      "everest",
		Cluster:        "main",
		IgnoreNotFound: true,
		Wait:           true,
		IsTerminal:     isTerminalFalse,
	}

	rd := NewDeleter(Config{}, zap.NewNop().Sugar())
	err := rd.Run(context.Background(), opts, newConfigPath(t, srv.URL))
	require.NoError(t, err)
	assert.False(t, deleteCalled, "delete must not be issued when the restore is already gone")
}

// TestDelete_IgnoreNotFound_DeleteRaces404_ReportsNotDeleted covers PR #2765's
// gap: GET sees it present, DELETE 404s anyway — must report deleted:false.
//
//nolint:paralleltest // captureStdout mutates global os.Stdout; must run serially
func TestDelete_IgnoreNotFound_DeleteRaces404_ReportsNotDeleted(t *testing.T) {
	srv := newDeleteServer(t,
		func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) },
		getHandlerWithState(t, client.RestoreStatusStateSucceeded),
	)
	defer srv.Close()

	opts := yesOpts()
	opts.IgnoreNotFound = true

	rd := NewDeleter(Config{Pretty: false}, zap.NewNop().Sugar())
	var runErr error
	out := captureStdout(t, func() {
		runErr = rd.Run(context.Background(), opts, newConfigPath(t, srv.URL))
	})
	require.NoError(t, runErr)
	assert.JSONEq(t, `{"name":"my-mongo-restore-x7k2q","namespace":"everest","deleted":false}`, out)
}

func TestDelete_ServerError_ReturnsMessage(t *testing.T) {
	t.Parallel()

	srv := newDeleteServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"message": "boom"})
	}, getHandlerWithState(t, client.RestoreStatusStateSucceeded))
	defer srv.Close()

	rd := NewDeleter(Config{}, zap.NewNop().Sugar())
	err := rd.Run(context.Background(), yesOpts(), newConfigPath(t, srv.URL))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "boom")
}

func TestDelete_NonInteractiveWithoutYes_FailsFast(t *testing.T) {
	t.Parallel()

	called := false
	srv := newDeleteServer(t, func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}, getHandlerWithState(t, client.RestoreStatusStateSucceeded))
	defer srv.Close()

	opts := DeleteOptions{
		Name:       "my-mongo-restore-x7k2q",
		Namespace:  "everest",
		Cluster:    "main",
		IsTerminal: isTerminalFalse,
	}

	rd := NewDeleter(Config{}, zap.NewNop().Sugar())
	err := rd.Run(context.Background(), opts, newConfigPath(t, srv.URL))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "confirmation required in non-interactive mode")
	assert.False(t, called, "delete must not be issued when confirmation couldn't be obtained")
}

func TestDelete_JSONMode_NonInteractiveWithoutYes_FailsFast(t *testing.T) {
	t.Parallel()

	srv := newDeleteServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}, getHandlerWithState(t, client.RestoreStatusStateSucceeded))
	defer srv.Close()

	opts := DeleteOptions{
		Name:      "my-mongo-restore-x7k2q",
		Namespace: "everest",
		Cluster:   "main",
		JSON:      true,
		// IsTerminal is true here to prove --json alone forces non-interactive,
		// independent of whether stdin is actually a real terminal.
		IsTerminal: func() bool { return true },
	}

	rd := NewDeleter(Config{Pretty: false}, zap.NewNop().Sugar())
	err := rd.Run(context.Background(), opts, newConfigPath(t, srv.URL))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "confirmation required in non-interactive mode")
}

// TestDelete_VerboseAloneDoesNotForceNonInteractive proves --verbose alone
// does not force non-interactive mode on a real TTY, only --json should.
func TestDelete_VerboseAloneDoesNotForceNonInteractive(t *testing.T) {
	t.Parallel()

	srv := newDeleteServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}, getHandlerWithState(t, client.RestoreStatusStateSucceeded))
	defer srv.Close()

	opts := DeleteOptions{
		Name:       "my-mongo-restore-x7k2q",
		Namespace:  "everest",
		Cluster:    "main",
		JSON:       false,
		IsTerminal: func() bool { return true },
	}

	rd := NewDeleter(Config{Pretty: false}, zap.NewNop().Sugar())
	err := rd.Run(context.Background(), opts, newConfigPath(t, srv.URL))
	assert.NotErrorIs(t, err, confirm.ErrNonInteractive, "verbose alone must not be treated as non-interactive")
}

func TestDelete_WaitUntilGone_Succeeds(t *testing.T) {
	t.Parallel()

	getCalls := 0
	srv := newDeleteServer(t,
		func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) },
		func(w http.ResponseWriter, r *http.Request) {
			getCalls++
			if getCalls == 1 {
				// The pre-delete guard fetch: present and Succeeded, not in-flight.
				getHandlerWithState(t, client.RestoreStatusStateSucceeded)(w, r)
				return
			}
			// Every fetch after the delete: gone.
			w.WriteHeader(http.StatusNotFound)
		},
	)
	defer srv.Close()

	opts := yesOpts()
	opts.Wait = true
	opts.Timeout = 0 // no timeout needed: the post-delete GET 404s on the very first poll

	rd := NewDeleter(Config{}, zap.NewNop().Sugar())
	err := rd.Run(context.Background(), opts, newConfigPath(t, srv.URL))
	require.NoError(t, err)
	assert.GreaterOrEqual(t, getCalls, 2)
}

func TestDelete_WaitTimesOut(t *testing.T) {
	t.Parallel()

	srv := newDeleteServer(t,
		func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) },
		// Restore stays present (still finalizing), so --wait times out.
		getHandlerWithState(t, client.RestoreStatusStateSucceeded),
	)
	defer srv.Close()

	opts := yesOpts()
	opts.Wait = true
	opts.Timeout = 50 * time.Millisecond // the immediate first poll sees the restore still present; the timeout then elapses before the 2s default poll interval ticks again

	rd := NewDeleter(Config{}, zap.NewNop().Sugar())
	err := rd.Run(context.Background(), opts, newConfigPath(t, srv.URL))
	require.Error(t, err)
	assert.ErrorIs(t, err, wait.ErrTimeout)
}

// TestDelete_IgnoreNotFound_AlreadyGone_JSONOutput proves "deleted" is false.
//
//nolint:paralleltest // mutates global os.Stdout; must run serially
func TestDelete_IgnoreNotFound_AlreadyGone_JSONOutput(t *testing.T) {
	deleteCalled := false
	srv := newDeleteServer(t,
		func(w http.ResponseWriter, _ *http.Request) {
			deleteCalled = true
			w.WriteHeader(http.StatusNoContent)
		},
		func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) },
	)
	defer srv.Close()

	opts := DeleteOptions{
		Name:           "my-mongo-restore-x7k2q",
		Namespace:      "everest",
		Cluster:        "main",
		IgnoreNotFound: true,
		IsTerminal:     isTerminalFalse,
	}

	rd := NewDeleter(Config{Pretty: false}, zap.NewNop().Sugar())
	var runErr error
	out := captureStdout(t, func() {
		runErr = rd.Run(context.Background(), opts, newConfigPath(t, srv.URL))
	})
	require.NoError(t, runErr)
	assert.False(t, deleteCalled)
	assert.JSONEq(t, `{"name":"my-mongo-restore-x7k2q","namespace":"everest","deleted":false}`, out)
}

// TestDelete_InFlight_ForceWithoutYes_StillRequiresConfirmation pins the
// #2658 rule that --force is never a --yes synonym: overriding the guard
// must still go through the confirmation gate.
func TestDelete_InFlight_ForceWithoutYes_StillRequiresConfirmation(t *testing.T) {
	t.Parallel()

	called := false
	srv := newDeleteServer(t,
		func(w http.ResponseWriter, _ *http.Request) {
			called = true
			w.WriteHeader(http.StatusNoContent)
		},
		getHandlerWithState(t, client.RestoreStatusStateRunning),
	)
	defer srv.Close()

	opts := DeleteOptions{
		Name:       "my-mongo-restore-x7k2q",
		Namespace:  "everest",
		Cluster:    "main",
		Force:      true,
		IsTerminal: isTerminalFalse,
	}

	rd := NewDeleter(Config{}, zap.NewNop().Sugar())
	err := rd.Run(context.Background(), opts, newConfigPath(t, srv.URL))
	require.ErrorIs(t, err, confirm.ErrNonInteractive)
	assert.False(t, called, "--force must not imply --yes")
}

func TestDelete_InFlight_Pending_RefusesWithoutForce(t *testing.T) {
	t.Parallel()

	called := false
	srv := newDeleteServer(t,
		func(w http.ResponseWriter, _ *http.Request) {
			called = true
			w.WriteHeader(http.StatusNoContent)
		},
		getHandlerWithState(t, client.RestoreStatusStatePending),
	)
	defer srv.Close()

	rd := NewDeleter(Config{}, zap.NewNop().Sugar())
	err := rd.Run(context.Background(), yesOpts(), newConfigPath(t, srv.URL))
	require.Error(t, err)
	assert.Contains(t, err.Error(), `restore "my-mongo-restore-x7k2q" is still running; wait for it to finish or re-run with --force`)
	assert.False(t, called, "delete must not be issued while the restore is in flight")
}

// TestDelete_InFlight_NoStatusYet_RefusesWithoutForce covers the window
// right after create, before any controller has observed the restore.
func TestDelete_InFlight_NoStatusYet_RefusesWithoutForce(t *testing.T) {
	t.Parallel()

	called := false
	srv := newDeleteServer(t,
		func(w http.ResponseWriter, _ *http.Request) {
			called = true
			w.WriteHeader(http.StatusNoContent)
		},
		getHandlerWithState(t, ""),
	)
	defer srv.Close()

	rd := NewDeleter(Config{}, zap.NewNop().Sugar())
	err := rd.Run(context.Background(), yesOpts(), newConfigPath(t, srv.URL))
	require.Error(t, err)
	assert.Contains(t, err.Error(), `restore "my-mongo-restore-x7k2q" hasn't reported its status yet`)
	assert.False(t, called, "delete must not be issued before the restore has any status")
}

func TestDelete_InFlight_Running_RefusesWithoutForce(t *testing.T) {
	t.Parallel()

	called := false
	srv := newDeleteServer(t,
		func(w http.ResponseWriter, _ *http.Request) {
			called = true
			w.WriteHeader(http.StatusNoContent)
		},
		getHandlerWithState(t, client.RestoreStatusStateRunning),
	)
	defer srv.Close()

	rd := NewDeleter(Config{}, zap.NewNop().Sugar())
	err := rd.Run(context.Background(), yesOpts(), newConfigPath(t, srv.URL))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is still running")
	assert.False(t, called, "delete must not be issued while the restore is in flight")
}

func TestDelete_InFlight_WithForce_Succeeds(t *testing.T) {
	t.Parallel()

	srv := newDeleteServer(t,
		func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) },
		getHandlerWithState(t, client.RestoreStatusStateRunning),
	)
	defer srv.Close()

	opts := yesOpts()
	opts.Force = true

	rd := NewDeleter(Config{}, zap.NewNop().Sugar())
	err := rd.Run(context.Background(), opts, newConfigPath(t, srv.URL))
	assert.NoError(t, err)
}

func TestDelete_InFlight_NoStatusYet_WithForce_Succeeds(t *testing.T) {
	t.Parallel()

	srv := newDeleteServer(t,
		func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) },
		getHandlerWithState(t, ""),
	)
	defer srv.Close()

	opts := yesOpts()
	opts.Force = true

	rd := NewDeleter(Config{}, zap.NewNop().Sugar())
	err := rd.Run(context.Background(), opts, newConfigPath(t, srv.URL))
	assert.NoError(t, err)
}

// TestDelete_ErrorState_NotGuarded proves Error is never blocked by the
// in-flight guard — deleting a stuck restore is the normal remediation.
func TestDelete_ErrorState_NotGuarded(t *testing.T) {
	t.Parallel()

	srv := newDeleteServer(t,
		func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) },
		getHandlerWithState(t, client.RestoreStatusStateError),
	)
	defer srv.Close()

	rd := NewDeleter(Config{}, zap.NewNop().Sugar())
	err := rd.Run(context.Background(), yesOpts(), newConfigPath(t, srv.URL))
	assert.NoError(t, err)
}

// TestDelete_GuardFetchFails_DoesNotBlock proves an ambiguous pre-delete
// fetch (a 500) must not itself block the delete; DELETE is the real check.
func TestDelete_GuardFetchFails_DoesNotBlock(t *testing.T) {
	t.Parallel()

	srv := newDeleteServer(t,
		func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) },
		func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusInternalServerError) },
	)
	defer srv.Close()

	rd := NewDeleter(Config{}, zap.NewNop().Sugar())
	err := rd.Run(context.Background(), yesOpts(), newConfigPath(t, srv.URL))
	assert.NoError(t, err)
}

// TestDeleteConfirmMessage covers the normal message and the
// forcingInFlight warning text.
func TestDeleteConfirmMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		state           client.RestoreStatusState
		forcingInFlight bool
		want            string
	}{
		{
			name:            "normal delete",
			state:           client.RestoreStatusStateSucceeded,
			forcingInFlight: false,
			want:            `Delete restore "my-mongo-restore-x7k2q" in namespace "everest"?`,
		},
		{
			name:            "forcing a Running restore",
			state:           client.RestoreStatusStateRunning,
			forcingInFlight: true,
			want:            `Restore "my-mongo-restore-x7k2q" in namespace "everest" may still be in progress (state: Running). Interrupting it triggers the provider's cleanup of the engine restore and can leave the target instance in an inconsistent state. Delete anyway?`,
		},
		{
			name:            "forcing a restore with no status yet",
			state:           "",
			forcingInFlight: true,
			want:            `Restore "my-mongo-restore-x7k2q" in namespace "everest" may still be in progress (state: not yet reported). Interrupting it triggers the provider's cleanup of the engine restore and can leave the target instance in an inconsistent state. Delete anyway?`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := deleteConfirmMessage("my-mongo-restore-x7k2q", "everest", tc.state, tc.forcingInFlight)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestInFlight(t *testing.T) {
	t.Parallel()

	assert.True(t, inFlight(client.RestoreStatusStatePending, true))
	assert.True(t, inFlight(client.RestoreStatusStateRunning, true))
	assert.True(t, inFlight("", true), "read successfully but no status yet is the riskiest window, right after create")
	assert.False(t, inFlight(client.RestoreStatusStateSucceeded, true))
	assert.False(t, inFlight(client.RestoreStatusStateFailed, true))
	assert.False(t, inFlight(client.RestoreStatusStateError, true))
	assert.False(t, inFlight("", false), "a failed/ambiguous fetch must never block — best-effort guard, not an invariant")
}

func TestRestoreStateForGuard(t *testing.T) {
	t.Parallel()

	state, ok := restoreStateForGuard(nil)
	assert.Empty(t, state)
	assert.False(t, ok, "nil restore means the fetch found nothing or failed")

	state, ok = restoreStateForGuard(restoreWithState(t, "my-mongo-restore-x7k2q", ""))
	assert.Empty(t, state)
	assert.True(t, ok, "fetched successfully, just no status yet")

	state, ok = restoreStateForGuard(restoreWithState(t, "my-mongo-restore-x7k2q", client.RestoreStatusStateRunning))
	assert.Equal(t, client.RestoreStatusStateRunning, state)
	assert.True(t, ok)
}
