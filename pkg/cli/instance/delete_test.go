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
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/openeverest/openeverest/v2/pkg/cli/confirm"
	"github.com/openeverest/openeverest/v2/pkg/cli/wait"
)

func isTerminalFalse() bool { return false }

// newDeleteServer fakes the instance DELETE (and optional GET) endpoint.
func newDeleteServer(t *testing.T, deleteHandler, getHandler http.HandlerFunc) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	path := "/v1/clusters/main/namespaces/everest/instances/my-db"
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

// yesOpts sets --yes so tests skip the interactive prompt.
func yesOpts() DeleteOptions {
	return DeleteOptions{
		Name:      "my-db",
		Namespace: "everest",
		Cluster:   "main",
		Yes:       true,
	}
}

func TestDelete_HappyPath_NoWait(t *testing.T) {
	t.Parallel()

	srv := newDeleteServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}, nil)
	defer srv.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, newTestConfig(srv.URL).Save(cfgPath))

	id := NewDeleter(Config{}, zap.NewNop().Sugar())
	err := id.Run(context.Background(), yesOpts(), cfgPath)
	assert.NoError(t, err)
}

func TestDelete_NotFound_WithoutIgnoreNotFound_Errors(t *testing.T) {
	t.Parallel()

	srv := newDeleteServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}, nil)
	defer srv.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, newTestConfig(srv.URL).Save(cfgPath))

	id := NewDeleter(Config{}, zap.NewNop().Sugar())
	err := id.Run(context.Background(), yesOpts(), cfgPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `instance "my-db" not found in namespace "everest"`)
}

func TestDelete_NotFound_WithIgnoreNotFound_Succeeds(t *testing.T) {
	t.Parallel()

	srv := newDeleteServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}, nil)
	defer srv.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, newTestConfig(srv.URL).Save(cfgPath))

	opts := yesOpts()
	opts.IgnoreNotFound = true

	id := NewDeleter(Config{}, zap.NewNop().Sugar())
	err := id.Run(context.Background(), opts, cfgPath)
	assert.NoError(t, err)
}

// TestDelete_IgnoreNotFound_AlreadyGone_SkipsConfirmationAndWait proves that
// when the instance is already gone, confirmation and --wait are both
// skipped, and DELETE is never called.
func TestDelete_IgnoreNotFound_AlreadyGone_SkipsConfirmationAndWait(t *testing.T) {
	t.Parallel()

	deleteCalled := false
	srv := newDeleteServer(t,
		func(w http.ResponseWriter, _ *http.Request) {
			deleteCalled = true
			w.WriteHeader(http.StatusNoContent)
		},
		func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		},
	)
	defer srv.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, newTestConfig(srv.URL).Save(cfgPath))

	opts := DeleteOptions{
		Name:           "my-db",
		Namespace:      "everest",
		Cluster:        "main",
		IgnoreNotFound: true,
		Wait:           true,
		IsTerminal:     isTerminalFalse,
	}

	id := NewDeleter(Config{}, zap.NewNop().Sugar())
	err := id.Run(context.Background(), opts, cfgPath)
	require.NoError(t, err)
	assert.False(t, deleteCalled, "delete must not be issued when the instance is already gone")
}

func TestDelete_ServerError_ReturnsMessage(t *testing.T) {
	t.Parallel()

	srv := newDeleteServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]any{"message": "boom"})
	}, nil)
	defer srv.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, newTestConfig(srv.URL).Save(cfgPath))

	id := NewDeleter(Config{}, zap.NewNop().Sugar())
	err := id.Run(context.Background(), yesOpts(), cfgPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "boom")
}

func TestDelete_InvalidDeletionPolicy_Errors(t *testing.T) {
	t.Parallel()

	opts := yesOpts()
	opts.DeletionPolicy = "Bogus"

	id := NewDeleter(Config{}, zap.NewNop().Sugar())
	err := id.Run(context.Background(), opts, filepath.Join(t.TempDir(), "config.yaml"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid --deletion-policy")
}

func TestDelete_DeletionPolicyForwarded(t *testing.T) {
	t.Parallel()

	var gotQuery string
	srv := newDeleteServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query().Get("deletionPolicy")
		w.WriteHeader(http.StatusNoContent)
	}, nil)
	defer srv.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, newTestConfig(srv.URL).Save(cfgPath))

	opts := yesOpts()
	opts.DeletionPolicy = "Orphan"

	id := NewDeleter(Config{}, zap.NewNop().Sugar())
	err := id.Run(context.Background(), opts, cfgPath)
	require.NoError(t, err)
	assert.Equal(t, "Orphan", gotQuery)
}

func TestDelete_NonInteractiveWithoutYes_FailsFast(t *testing.T) {
	t.Parallel()

	// Delete must not be called if confirmation fails.
	called := false
	srv := newDeleteServer(t, func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}, nil)
	defer srv.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, newTestConfig(srv.URL).Save(cfgPath))

	opts := DeleteOptions{
		Name:       "my-db",
		Namespace:  "everest",
		Cluster:    "main",
		IsTerminal: isTerminalFalse,
	}

	id := NewDeleter(Config{}, zap.NewNop().Sugar())
	err := id.Run(context.Background(), opts, cfgPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "confirmation required in non-interactive mode")
	assert.False(t, called, "delete must not be issued when confirmation couldn't be obtained")
}

func TestDelete_JSONMode_NonInteractiveWithoutYes_FailsFast(t *testing.T) {
	t.Parallel()

	srv := newDeleteServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}, nil)
	defer srv.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, newTestConfig(srv.URL).Save(cfgPath))

	opts := DeleteOptions{
		Name:      "my-db",
		Namespace: "everest",
		Cluster:   "main",
		JSON:      true,
		// IsTerminal is true here to prove --json alone forces non-interactive,
		// independent of whether stdin is actually a real terminal.
		IsTerminal: func() bool { return true },
	}

	id := NewDeleter(Config{Pretty: false}, zap.NewNop().Sugar())
	err := id.Run(context.Background(), opts, cfgPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "confirmation required in non-interactive mode")
}

// TestDelete_VerboseAloneDoesNotForceNonInteractive proves --verbose alone
// (Config.Pretty false, JSON false) does not force non-interactive mode on a
// real TTY, only --json should. It runs Run itself rather than restating the
// gating rule against a hand-built confirm.Options, since that would pass
// even if Run went back to deriving JSON from Pretty. Without a real
// terminal, the prompt itself fails, so this only asserts we got past the
// gate, not that the prompt succeeded.
func TestDelete_VerboseAloneDoesNotForceNonInteractive(t *testing.T) {
	t.Parallel()

	srv := newDeleteServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}, nil)
	defer srv.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, newTestConfig(srv.URL).Save(cfgPath))

	opts := DeleteOptions{
		Name:       "my-db",
		Namespace:  "everest",
		Cluster:    "main",
		JSON:       false,
		IsTerminal: func() bool { return true },
		Yes:        true,
	}

	id := NewDeleter(Config{Pretty: false}, zap.NewNop().Sugar())
	err := id.Run(context.Background(), opts, cfgPath)
	assert.NotErrorIs(t, err, confirm.ErrNonInteractive, "verbose alone must not be treated as non-interactive")
}

func TestDelete_WaitUntilGone_Succeeds(t *testing.T) {
	t.Parallel()

	getCalls := 0
	srv := newDeleteServer(t,
		func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		},
		func(w http.ResponseWriter, _ *http.Request) {
			getCalls++
			w.WriteHeader(http.StatusNotFound)
		},
	)
	defer srv.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, newTestConfig(srv.URL).Save(cfgPath))

	opts := yesOpts()
	opts.Wait = true
	opts.Timeout = 0 // no timeout needed: the fake GET 404s on the very first poll

	id := NewDeleter(Config{}, zap.NewNop().Sugar())
	err := id.Run(context.Background(), opts, cfgPath)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, getCalls, 1)
}

func TestDelete_WaitTimesOut(t *testing.T) {
	t.Parallel()

	srv := newDeleteServer(t,
		func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		},
		func(w http.ResponseWriter, _ *http.Request) {
			// Instance stays present, so --wait times out.
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":{"phase":"Terminating"}}`))
		},
	)
	defer srv.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, newTestConfig(srv.URL).Save(cfgPath))

	opts := yesOpts()
	opts.Wait = true
	opts.Timeout = 50 * time.Millisecond // the immediate first poll sees the instance still present; the timeout then elapses before the 2s default poll interval ticks again

	id := NewDeleter(Config{}, zap.NewNop().Sugar())
	err := id.Run(context.Background(), opts, cfgPath)
	require.Error(t, err)
	assert.ErrorIs(t, err, wait.ErrTimeout)
}

// TestDelete_IgnoreNotFound_AlreadyGone_JSONOutput proves "deleted" is
// false when the instance was already gone.
//
//nolint:paralleltest // mutates global os.Stdout; must run serially
func TestDelete_IgnoreNotFound_AlreadyGone_JSONOutput(t *testing.T) {
	deleteCalled := false
	srv := newDeleteServer(t,
		func(w http.ResponseWriter, _ *http.Request) {
			deleteCalled = true
			w.WriteHeader(http.StatusNoContent)
		},
		func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		},
	)
	defer srv.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, newTestConfig(srv.URL).Save(cfgPath))

	opts := DeleteOptions{
		Name:           "my-db",
		Namespace:      "everest",
		Cluster:        "main",
		IgnoreNotFound: true,
		IsTerminal:     isTerminalFalse,
	}

	id := NewDeleter(Config{Pretty: false}, zap.NewNop().Sugar())
	var runErr error
	out := captureStdout(t, func() {
		runErr = id.Run(context.Background(), opts, cfgPath)
	})
	require.NoError(t, runErr)
	assert.False(t, deleteCalled)
	assert.JSONEq(t, `{"name":"my-db","namespace":"everest","deleted":false}`, out)
}

// TestBlastRadiusMessage covers explicit Cascade, explicit Orphan, and the
// unverified fallback, which must not name a policy as fact.
func TestBlastRadiusMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		policy   string
		verified bool
		want     string
	}{
		{
			name:     "explicit Cascade",
			policy:   "Cascade",
			verified: true,
			want:     `This permanently deletes instance "my-db" in namespace "everest" and, per policy Cascade, deletes all its Backup and Restore objects. Type the instance name to confirm.`,
		},
		{
			name:     "explicit Orphan",
			policy:   "Orphan",
			verified: true,
			want:     `This permanently deletes instance "my-db" in namespace "everest" and, per policy Orphan, leaves its Backup and Restore objects in place. Type the instance name to confirm.`,
		},
		{
			name:     "unverified fallback",
			policy:   "Cascade",
			verified: false,
			want:     `This permanently deletes instance "my-db" in namespace "everest" and, per the instance's own deletion policy, may delete all its Backup and Restore objects. Type the instance name to confirm.`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := blastRadiusMessage("my-db", "everest", tc.policy, tc.verified)
			assert.Equal(t, tc.want, got)
		})
	}
}
