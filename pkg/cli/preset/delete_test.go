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

package preset

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func yesOpts() DeleteOptions {
	return DeleteOptions{
		Name:      "mongodb-production",
		Namespace: "everest",
		Cluster:   "main",
		Yes:       true,
	}
}

func TestPresetDelete_HappyPath(t *testing.T) {
	var method string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	cfgPath := newConfigPath(t, srv.URL)
	out := captureStdout(t, func() {
		require.NoError(t, NewDeleter(Config{Pretty: true}, zap.NewNop().Sugar()).Run(t.Context(), yesOpts(), cfgPath))
	})
	assert.Equal(t, http.MethodDelete, method)
	assert.Contains(t, out, "deleted")
}

func TestPresetDelete_NotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	cfgPath := newConfigPath(t, srv.URL)
	err := NewDeleter(Config{Pretty: true}, zap.NewNop().Sugar()).Run(t.Context(), yesOpts(), cfgPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestPresetDelete_IgnoreNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		t.Fatalf("delete must not be called when already gone")
	}))
	defer srv.Close()

	cfgPath := newConfigPath(t, srv.URL)
	opts := yesOpts()
	opts.IgnoreNotFound = true
	out := captureStdout(t, func() {
		require.NoError(t, NewDeleter(Config{Pretty: true}, zap.NewNop().Sugar()).Run(t.Context(), opts, cfgPath))
	})
	assert.Contains(t, out, "nothing to delete")
}

//nolint:paralleltest // captureStdout mutates global os.Stdout; must run serially
func TestPresetDelete_IgnoreNotFound_AlreadyGone_YAMLOutput(t *testing.T) {
	deleteCalled := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		deleteCalled = true
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	cfgPath := newConfigPath(t, srv.URL)
	opts := DeleteOptions{
		Name:           "mongodb-production",
		Namespace:      "everest",
		Cluster:        "main",
		IgnoreNotFound: true,
		IsTerminal:     func() bool { return false },
	}
	var runErr error
	out := captureStdout(t, func() {
		runErr = NewDeleter(Config{Pretty: false, YAML: true}, zap.NewNop().Sugar()).Run(t.Context(), opts, cfgPath)
	})
	require.NoError(t, runErr)
	assert.False(t, deleteCalled)
	assert.Equal(t, "deleted: false\nname: mongodb-production\nnamespace: everest\n", out)
}
