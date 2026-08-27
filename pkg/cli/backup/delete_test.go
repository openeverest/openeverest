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

// newDeleteServer fakes the backup DELETE (and optional GET) endpoint.
func newDeleteServer(t *testing.T, deleteHandler, getHandler http.HandlerFunc) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	path := "/v1/clusters/main/namespaces/everest/backups/pre-upgrade"
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

// backupFixture builds a Backup fixture via JSON instead of hand-spelling
// the generated anonymous Status struct. Empty state/policy omits the field.
func backupFixture(t *testing.T, name string, state client.BackupStatusState, policy string) *client.Backup {
	t.Helper()
	statusField := ""
	if state != "" {
		statusField = fmt.Sprintf(`,"status":{"state":%q}`, string(state))
	}
	policyField := ""
	if policy != "" {
		policyField = fmt.Sprintf(`,"deletionPolicy":%q`, policy)
	}
	body := fmt.Sprintf(
		`{"metadata":{"name":%q,"namespace":"everest"},"spec":{"instanceRef":{"name":"my-mongo"},"classRef":{"name":"psmdb-backup"},"storageRef":{"name":"my-s3"}%s}%s}`,
		name, policyField, statusField,
	)
	var b client.Backup
	require.NoError(t, json.Unmarshal([]byte(body), &b))
	return &b
}

func getHandlerWithStateAndPolicy(t *testing.T, state client.BackupStatusState, policy string) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(backupFixture(t, "pre-upgrade", state, policy))
	}
}

// yesOpts sets --yes so tests skip the interactive prompt.
func yesOpts() DeleteOptions {
	return DeleteOptions{
		Name:      "pre-upgrade",
		Namespace: "everest",
		Cluster:   "main",
		Yes:       true,
	}
}

func TestDelete_HappyPath_NoWait(t *testing.T) {
	t.Parallel()

	srv := newDeleteServer(t,
		func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) },
		getHandlerWithStateAndPolicy(t, client.BackupStatusStateSucceeded, backupDeletionPolicyDelete),
	)
	defer srv.Close()

	bd := NewDeleter(Config{}, zap.NewNop().Sugar())
	err := bd.Run(context.Background(), yesOpts(), newConfigPath(t, srv.URL))
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

	bd := NewDeleter(Config{}, zap.NewNop().Sugar())
	err := bd.Run(context.Background(), yesOpts(), newConfigPath(t, srv.URL))
	require.Error(t, err)
	assert.Contains(t, err.Error(), `backup "pre-upgrade" not found in namespace "everest"`)
	assert.False(t, deleteCalled, "a backup we already know is gone must not reach the DELETE call")
}

// TestDelete_NotFound_WithoutIgnoreNotFound_SkipsConfirmation proves knowing
// the backup is already gone skips the prompt entirely, even without --yes
// or a TTY — no point confirming something we already know the answer to.
func TestDelete_NotFound_WithoutIgnoreNotFound_SkipsConfirmation(t *testing.T) {
	t.Parallel()

	deleteCalled := false
	srv := newDeleteServer(t, func(w http.ResponseWriter, _ *http.Request) {
		deleteCalled = true
		w.WriteHeader(http.StatusNoContent)
	}, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) })
	defer srv.Close()

	opts := DeleteOptions{
		Name:       "pre-upgrade",
		Namespace:  "everest",
		Cluster:    "main",
		IsTerminal: isTerminalFalse,
	}

	bd := NewDeleter(Config{}, zap.NewNop().Sugar())
	err := bd.Run(context.Background(), opts, newConfigPath(t, srv.URL))
	require.Error(t, err)
	assert.Contains(t, err.Error(), `backup "pre-upgrade" not found in namespace "everest"`)
	assert.NotContains(t, err.Error(), "confirmation required", "already knowing it's gone must skip the prompt entirely")
	assert.False(t, deleteCalled, "delete must not be issued for a backup we already know is gone")
}

func TestDelete_NotFound_WithIgnoreNotFound_Succeeds(t *testing.T) {
	t.Parallel()

	srv := newDeleteServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}, nil)
	defer srv.Close()

	opts := yesOpts()
	opts.IgnoreNotFound = true

	bd := NewDeleter(Config{}, zap.NewNop().Sugar())
	err := bd.Run(context.Background(), opts, newConfigPath(t, srv.URL))
	assert.NoError(t, err)
}

// TestDelete_IgnoreNotFound_AlreadyGone_SkipsConfirmationAndWait proves
// confirm and --wait are both skipped when the backup is already gone.
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
		Name:           "pre-upgrade",
		Namespace:      "everest",
		Cluster:        "main",
		IgnoreNotFound: true,
		Wait:           true,
		IsTerminal:     isTerminalFalse,
	}

	bd := NewDeleter(Config{}, zap.NewNop().Sugar())
	err := bd.Run(context.Background(), opts, newConfigPath(t, srv.URL))
	require.NoError(t, err)
	assert.False(t, deleteCalled, "delete must not be issued when the backup is already gone")
}

// TestDelete_IgnoreNotFound_DeleteRaces404_ReportsNotDeleted covers PR #2765's
// gap: GET sees it present, DELETE 404s anyway — must report deleted:false.
//
//nolint:paralleltest // captureStdout mutates global os.Stdout; must run serially
func TestDelete_IgnoreNotFound_DeleteRaces404_ReportsNotDeleted(t *testing.T) {
	srv := newDeleteServer(t,
		func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) },
		getHandlerWithStateAndPolicy(t, client.BackupStatusStateSucceeded, backupDeletionPolicyDelete),
	)
	defer srv.Close()

	opts := yesOpts()
	opts.IgnoreNotFound = true

	bd := NewDeleter(Config{Pretty: false}, zap.NewNop().Sugar())
	var runErr error
	out := captureStdout(t, func() {
		runErr = bd.Run(context.Background(), opts, newConfigPath(t, srv.URL))
	})
	require.NoError(t, runErr)
	assert.JSONEq(t, `{"name":"pre-upgrade","namespace":"everest","deleted":false}`, out)
}

func TestDelete_ServerError_ReturnsMessage(t *testing.T) {
	t.Parallel()

	srv := newDeleteServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"message": "boom"})
	}, getHandlerWithStateAndPolicy(t, client.BackupStatusStateSucceeded, backupDeletionPolicyDelete))
	defer srv.Close()

	bd := NewDeleter(Config{}, zap.NewNop().Sugar())
	err := bd.Run(context.Background(), yesOpts(), newConfigPath(t, srv.URL))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "boom")
}

func TestDelete_NonInteractiveWithoutYes_FailsFast(t *testing.T) {
	t.Parallel()

	called := false
	srv := newDeleteServer(t, func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}, getHandlerWithStateAndPolicy(t, client.BackupStatusStateSucceeded, backupDeletionPolicyDelete))
	defer srv.Close()

	opts := DeleteOptions{
		Name:       "pre-upgrade",
		Namespace:  "everest",
		Cluster:    "main",
		IsTerminal: isTerminalFalse,
	}

	bd := NewDeleter(Config{}, zap.NewNop().Sugar())
	err := bd.Run(context.Background(), opts, newConfigPath(t, srv.URL))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "confirmation required in non-interactive mode")
	assert.False(t, called, "delete must not be issued when confirmation couldn't be obtained")
}

func TestDelete_NonInteractiveWithoutYes_RetainPolicy_FailsFast(t *testing.T) {
	t.Parallel()

	called := false
	srv := newDeleteServer(t, func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}, getHandlerWithStateAndPolicy(t, client.BackupStatusStateSucceeded, backupDeletionPolicyRetain))
	defer srv.Close()

	opts := DeleteOptions{
		Name:       "pre-upgrade",
		Namespace:  "everest",
		Cluster:    "main",
		IsTerminal: isTerminalFalse,
	}

	bd := NewDeleter(Config{}, zap.NewNop().Sugar())
	err := bd.Run(context.Background(), opts, newConfigPath(t, srv.URL))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "confirmation required in non-interactive mode")
	assert.False(t, called, "delete must not be issued when confirmation couldn't be obtained")
}

func TestDelete_JSONMode_NonInteractiveWithoutYes_FailsFast(t *testing.T) {
	t.Parallel()

	srv := newDeleteServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}, getHandlerWithStateAndPolicy(t, client.BackupStatusStateSucceeded, backupDeletionPolicyDelete))
	defer srv.Close()

	opts := DeleteOptions{
		Name:      "pre-upgrade",
		Namespace: "everest",
		Cluster:   "main",
		JSON:      true,
		// IsTerminal is true here to prove --json alone forces non-interactive,
		// independent of whether stdin is actually a real terminal.
		IsTerminal: func() bool { return true },
	}

	bd := NewDeleter(Config{Pretty: false}, zap.NewNop().Sugar())
	err := bd.Run(context.Background(), opts, newConfigPath(t, srv.URL))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "confirmation required in non-interactive mode")
}

// TestDelete_VerboseAloneDoesNotForceNonInteractive proves --verbose alone
// does not force non-interactive mode on a real TTY, only --json should.
func TestDelete_VerboseAloneDoesNotForceNonInteractive(t *testing.T) {
	t.Parallel()

	srv := newDeleteServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}, getHandlerWithStateAndPolicy(t, client.BackupStatusStateSucceeded, backupDeletionPolicyDelete))
	defer srv.Close()

	opts := DeleteOptions{
		Name:       "pre-upgrade",
		Namespace:  "everest",
		Cluster:    "main",
		JSON:       false,
		IsTerminal: func() bool { return true },
	}

	bd := NewDeleter(Config{Pretty: false}, zap.NewNop().Sugar())
	err := bd.Run(context.Background(), opts, newConfigPath(t, srv.URL))
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
				getHandlerWithStateAndPolicy(t, client.BackupStatusStateSucceeded, backupDeletionPolicyDelete)(w, r)
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

	bd := NewDeleter(Config{}, zap.NewNop().Sugar())
	err := bd.Run(context.Background(), opts, newConfigPath(t, srv.URL))
	require.NoError(t, err)
	assert.GreaterOrEqual(t, getCalls, 2)
}

func TestDelete_WaitTimesOut(t *testing.T) {
	t.Parallel()

	srv := newDeleteServer(t,
		func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) },
		// Backup stays present (artifact cleanup still running), so --wait times out.
		getHandlerWithStateAndPolicy(t, client.BackupStatusStateSucceeded, backupDeletionPolicyDelete),
	)
	defer srv.Close()

	opts := yesOpts()
	opts.Wait = true
	opts.Timeout = 50 * time.Millisecond // the immediate first poll sees the backup still present; the timeout then elapses before the 2s default poll interval ticks again

	bd := NewDeleter(Config{}, zap.NewNop().Sugar())
	err := bd.Run(context.Background(), opts, newConfigPath(t, srv.URL))
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
		Name:           "pre-upgrade",
		Namespace:      "everest",
		Cluster:        "main",
		IgnoreNotFound: true,
		IsTerminal:     isTerminalFalse,
	}

	bd := NewDeleter(Config{Pretty: false}, zap.NewNop().Sugar())
	var runErr error
	out := captureStdout(t, func() {
		runErr = bd.Run(context.Background(), opts, newConfigPath(t, srv.URL))
	})
	require.NoError(t, runErr)
	assert.False(t, deleteCalled)
	assert.JSONEq(t, `{"name":"pre-upgrade","namespace":"everest","deleted":false}`, out)
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
		getHandlerWithStateAndPolicy(t, client.BackupStatusStateRunning, backupDeletionPolicyDelete),
	)
	defer srv.Close()

	opts := DeleteOptions{
		Name:       "pre-upgrade",
		Namespace:  "everest",
		Cluster:    "main",
		Force:      true,
		IsTerminal: isTerminalFalse,
	}

	bd := NewDeleter(Config{}, zap.NewNop().Sugar())
	err := bd.Run(context.Background(), opts, newConfigPath(t, srv.URL))
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
		getHandlerWithStateAndPolicy(t, client.BackupStatusStatePending, backupDeletionPolicyDelete),
	)
	defer srv.Close()

	bd := NewDeleter(Config{}, zap.NewNop().Sugar())
	err := bd.Run(context.Background(), yesOpts(), newConfigPath(t, srv.URL))
	require.Error(t, err)
	assert.Contains(t, err.Error(), `backup "pre-upgrade" is still running; wait for it to finish or re-run with --force`)
	assert.False(t, called, "delete must not be issued while the backup is in flight")
}

func TestDelete_InFlight_Running_RefusesWithoutForce(t *testing.T) {
	t.Parallel()

	called := false
	srv := newDeleteServer(t,
		func(w http.ResponseWriter, _ *http.Request) {
			called = true
			w.WriteHeader(http.StatusNoContent)
		},
		getHandlerWithStateAndPolicy(t, client.BackupStatusStateRunning, backupDeletionPolicyDelete),
	)
	defer srv.Close()

	bd := NewDeleter(Config{}, zap.NewNop().Sugar())
	err := bd.Run(context.Background(), yesOpts(), newConfigPath(t, srv.URL))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is still running")
	assert.False(t, called, "delete must not be issued while the backup is in flight")
}

func TestDelete_InFlight_WithForce_Succeeds(t *testing.T) {
	t.Parallel()

	srv := newDeleteServer(t,
		func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) },
		getHandlerWithStateAndPolicy(t, client.BackupStatusStateRunning, backupDeletionPolicyDelete),
	)
	defer srv.Close()

	opts := yesOpts()
	opts.Force = true

	bd := NewDeleter(Config{}, zap.NewNop().Sugar())
	err := bd.Run(context.Background(), opts, newConfigPath(t, srv.URL))
	assert.NoError(t, err)
}

// TestDelete_InFlight_NoStatusYet_RefusesWithoutForce covers the window
// right after create, before any controller has observed the backup.
func TestDelete_InFlight_NoStatusYet_RefusesWithoutForce(t *testing.T) {
	t.Parallel()

	called := false
	srv := newDeleteServer(t,
		func(w http.ResponseWriter, _ *http.Request) {
			called = true
			w.WriteHeader(http.StatusNoContent)
		},
		getHandlerWithStateAndPolicy(t, "", backupDeletionPolicyDelete),
	)
	defer srv.Close()

	bd := NewDeleter(Config{}, zap.NewNop().Sugar())
	err := bd.Run(context.Background(), yesOpts(), newConfigPath(t, srv.URL))
	require.Error(t, err)
	assert.Contains(t, err.Error(), `backup "pre-upgrade" hasn't reported its status yet`)
	assert.False(t, called, "delete must not be issued before the backup has any status")
}

func TestDelete_InFlight_NoStatusYet_WithForce_Succeeds(t *testing.T) {
	t.Parallel()

	srv := newDeleteServer(t,
		func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) },
		getHandlerWithStateAndPolicy(t, "", backupDeletionPolicyDelete),
	)
	defer srv.Close()

	opts := yesOpts()
	opts.Force = true

	bd := NewDeleter(Config{}, zap.NewNop().Sugar())
	err := bd.Run(context.Background(), opts, newConfigPath(t, srv.URL))
	assert.NoError(t, err)
}

// TestDelete_ErrorState_RefusesWithoutForce proves Error is guarded, same as
// Pending/Running: BackupStateError's own doc comment says the controller
// may still retry it, so it is not safe to treat as a stable, done state.
func TestDelete_ErrorState_RefusesWithoutForce(t *testing.T) {
	t.Parallel()

	called := false
	srv := newDeleteServer(t,
		func(w http.ResponseWriter, _ *http.Request) {
			called = true
			w.WriteHeader(http.StatusNoContent)
		},
		getHandlerWithStateAndPolicy(t, client.BackupStatusStateError, backupDeletionPolicyDelete),
	)
	defer srv.Close()

	bd := NewDeleter(Config{}, zap.NewNop().Sugar())
	err := bd.Run(context.Background(), yesOpts(), newConfigPath(t, srv.URL))
	require.Error(t, err)
	assert.Contains(t, err.Error(), `backup "pre-upgrade" is in the Error state and may still be retried`)
	assert.False(t, called, "delete must not be issued while the backup may still be retried")
}

func TestDelete_ErrorState_WithForce_Succeeds(t *testing.T) {
	t.Parallel()

	srv := newDeleteServer(t,
		func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) },
		getHandlerWithStateAndPolicy(t, client.BackupStatusStateError, backupDeletionPolicyDelete),
	)
	defer srv.Close()

	opts := yesOpts()
	opts.Force = true

	bd := NewDeleter(Config{}, zap.NewNop().Sugar())
	err := bd.Run(context.Background(), opts, newConfigPath(t, srv.URL))
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

	bd := NewDeleter(Config{}, zap.NewNop().Sugar())
	err := bd.Run(context.Background(), yesOpts(), newConfigPath(t, srv.URL))
	assert.NoError(t, err)
}

func TestIsRetainTier(t *testing.T) {
	t.Parallel()

	assert.False(t, isRetainTier(nil), "unverified (fetch failed/gone) must use the stricter tier")
	assert.False(t, isRetainTier(backupFixture(t, "b", "", "")), "unset policy defaults to Delete")
	assert.False(t, isRetainTier(backupFixture(t, "b", "", backupDeletionPolicyDelete)))
	assert.True(t, isRetainTier(backupFixture(t, "b", "", backupDeletionPolicyRetain)))
}

func TestPolicyFromBackup(t *testing.T) {
	t.Parallel()

	assert.Empty(t, policyFromBackup(nil))
	assert.Equal(t, backupDeletionPolicyDelete, policyFromBackup(backupFixture(t, "b", "", "")))
	assert.Equal(t, backupDeletionPolicyDelete, policyFromBackup(backupFixture(t, "b", "", backupDeletionPolicyDelete)))
	assert.Equal(t, backupDeletionPolicyRetain, policyFromBackup(backupFixture(t, "b", "", backupDeletionPolicyRetain)))
}

func TestInFlight(t *testing.T) {
	t.Parallel()

	assert.True(t, inFlight(client.BackupStatusStatePending, true))
	assert.True(t, inFlight(client.BackupStatusStateRunning, true))
	assert.True(t, inFlight(client.BackupStatusStateError, true), "BackupStateError's own doc comment says the controller may still retry it")
	assert.True(t, inFlight("", true), "read successfully but no status yet is the riskiest window, right after create")
	assert.False(t, inFlight(client.BackupStatusStateSucceeded, true))
	assert.False(t, inFlight(client.BackupStatusStateFailed, true), "Failed is genuinely terminal, unlike Error")
	assert.False(t, inFlight(client.BackupStatusStateDeleting, true))
	assert.False(t, inFlight("", false), "a failed/ambiguous fetch must never block — best-effort guard, not an invariant")
}

func TestBackupStateForGuard(t *testing.T) {
	t.Parallel()

	state, ok := backupStateForGuard(nil)
	assert.Empty(t, state)
	assert.False(t, ok, "nil backup means the fetch found nothing or failed")

	state, ok = backupStateForGuard(backupFixture(t, "pre-upgrade", "", ""))
	assert.Empty(t, state)
	assert.True(t, ok, "fetched successfully, just no status yet")

	state, ok = backupStateForGuard(backupFixture(t, "pre-upgrade", client.BackupStatusStateRunning, ""))
	assert.Equal(t, client.BackupStatusStateRunning, state)
	assert.True(t, ok)
}

// TestRetainConfirmMessage and TestDeleteConfirmMessage cover the exact
// text a user reads right before confirming.
func TestRetainConfirmMessage(t *testing.T) {
	t.Parallel()

	got := retainConfirmMessage("nightly-20260701", "everest")
	assert.Equal(t, `Delete backup "nightly-20260701" in namespace "everest"? Its data in storage stays in place (policy Retain).`, got)
}

func TestDeleteConfirmMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		storage  string
		verified bool
		want     string
	}{
		{
			name:     "verified policy Delete names the storage",
			storage:  "my-s3",
			verified: true,
			want:     `This permanently deletes backup "pre-upgrade" in namespace "everest" and, per policy Delete, its data in storage "my-s3". Type the backup name to confirm.`,
		},
		{
			name:     "unverified fallback names neither policy nor storage",
			storage:  "my-s3",
			verified: false,
			want:     `This permanently deletes backup "pre-upgrade" in namespace "everest" and, per the backup's own deletion policy, may delete its data in storage. Type the backup name to confirm.`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := deleteConfirmMessage("pre-upgrade", "everest", tc.storage, tc.verified)
			assert.Equal(t, tc.want, got)
		})
	}
}
