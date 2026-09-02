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

package backupstorage

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

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

// newDeleteServer fakes the backup storage DELETE (and optional GET) endpoint.
func newDeleteServer(t *testing.T, deleteHandler, getHandler http.HandlerFunc) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	path := "/v1/clusters/main/namespaces/everest/backup-storages/my-s3"
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
		Name:      "my-s3",
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

	d := NewDeleter(Config{}, zap.NewNop().Sugar())
	err := d.Run(context.Background(), yesOpts(), cfgPath)
	assert.NoError(t, err)
}

func TestDelete_NotFound_Errors(t *testing.T) {
	t.Parallel()

	srv := newDeleteServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}, nil)
	defer srv.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, newTestConfig(srv.URL).Save(cfgPath))

	d := NewDeleter(Config{}, zap.NewNop().Sugar())
	err := d.Run(context.Background(), yesOpts(), cfgPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `backup storage "my-s3" not found in namespace "everest"`)
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

	d := NewDeleter(Config{}, zap.NewNop().Sugar())
	err := d.Run(context.Background(), opts, cfgPath)
	assert.NoError(t, err)
}

//nolint:paralleltest // captureStdout mutates global os.Stdout; must run serially
func TestDelete_IgnoreNotFound_DeleteRaces404_ReportsNotDeleted(t *testing.T) {
	srv := newDeleteServer(t,
		func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		},
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"metadata":{"name":"my-s3","namespace":"everest"},"spec":{"type":"s3"}}`))
		},
	)
	defer srv.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, newTestConfig(srv.URL).Save(cfgPath))

	opts := yesOpts()
	opts.IgnoreNotFound = true

	d := NewDeleter(Config{Pretty: false}, zap.NewNop().Sugar())
	var runErr error
	out := captureStdout(t, func() {
		runErr = d.Run(context.Background(), opts, cfgPath)
	})
	require.NoError(t, runErr)
	assert.JSONEq(t, `{"name":"my-s3","namespace":"everest","deleted":false}`, out)
}

// TestDelete_IgnoreNotFound_AlreadyGone_SkipsConfirmationAndWait proves the
// --ignore-not-found short-circuit: when the backup storage is already gone,
// both the confirmation prompt and --wait are skipped entirely (no --yes, no
// TTY, and --wait: true all set here, none of it should matter), and the
// DELETE endpoint itself is never called.
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
		Name:           "my-s3",
		Namespace:      "everest",
		Cluster:        "main",
		IgnoreNotFound: true,
		Wait:           true,
		IsTerminal:     isTerminalFalse,
	}

	d := NewDeleter(Config{}, zap.NewNop().Sugar())
	err := d.Run(context.Background(), opts, cfgPath)
	require.NoError(t, err)
	assert.False(t, deleteCalled, "delete must not be issued when the backup storage is already gone")
}

func TestDelete_ServerError_ReturnsMessage(t *testing.T) {
	t.Parallel()

	srv := newDeleteServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{"message": "backup storage still referenced by an instance"})
	}, nil)
	defer srv.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, newTestConfig(srv.URL).Save(cfgPath))

	d := NewDeleter(Config{}, zap.NewNop().Sugar())
	err := d.Run(context.Background(), yesOpts(), cfgPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "backup storage still referenced by an instance")
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
		Name:       "my-s3",
		Namespace:  "everest",
		Cluster:    "main",
		IsTerminal: isTerminalFalse,
	}

	d := NewDeleter(Config{}, zap.NewNop().Sugar())
	err := d.Run(context.Background(), opts, cfgPath)
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
		Name:      "my-s3",
		Namespace: "everest",
		Cluster:   "main",
		JSON:      true,
		// IsTerminal is true here to prove --json alone forces non-interactive,
		// independent of whether stdin is actually a real terminal.
		IsTerminal: func() bool { return true },
	}

	d := NewDeleter(Config{Pretty: false}, zap.NewNop().Sugar())
	err := d.Run(context.Background(), opts, cfgPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "confirmation required in non-interactive mode")
}

func TestDelete_VerboseAloneDoesNotForceNonInteractive(t *testing.T) {
	t.Parallel()

	srv := newDeleteServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}, nil)
	defer srv.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, newTestConfig(srv.URL).Save(cfgPath))

	opts := DeleteOptions{
		Name:       "my-s3",
		Namespace:  "everest",
		Cluster:    "main",
		JSON:       false,
		IsTerminal: func() bool { return true },
		Yes:        true,
	}

	d := NewDeleter(Config{Pretty: false}, zap.NewNop().Sugar())
	err := d.Run(context.Background(), opts, cfgPath)
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

	d := NewDeleter(Config{}, zap.NewNop().Sugar())
	err := d.Run(context.Background(), opts, cfgPath)
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
			// Backup storage stays present (still referenced), so --wait times out.
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"metadata":{"name":"my-s3","namespace":"everest"},"spec":{"type":"s3"}}`))
		},
	)
	defer srv.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, newTestConfig(srv.URL).Save(cfgPath))

	opts := yesOpts()
	opts.Wait = true
	opts.Timeout = 50 * time.Millisecond // the immediate first poll sees the storage still present; the timeout then elapses before the 2s default poll interval ticks again

	d := NewDeleter(Config{}, zap.NewNop().Sugar())
	err := d.Run(context.Background(), opts, cfgPath)
	require.Error(t, err)
	require.ErrorIs(t, err, wait.ErrTimeout)
}

// TestDelete_IgnoreNotFound_AlreadyGone_JSONOutput proves the short-circuit
// output is honest: "deleted" is false, since this run never deleted
// anything — the backup storage was already gone.
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
		Name:           "my-s3",
		Namespace:      "everest",
		Cluster:        "main",
		IgnoreNotFound: true,
		IsTerminal:     isTerminalFalse,
	}

	d := NewDeleter(Config{Pretty: false}, zap.NewNop().Sugar())
	var runErr error
	out := captureStdout(t, func() {
		runErr = d.Run(context.Background(), opts, cfgPath)
	})
	require.NoError(t, runErr)
	assert.False(t, deleteCalled)
	assert.JSONEq(t, `{"name":"my-s3","namespace":"everest","deleted":false}`, out)
}
