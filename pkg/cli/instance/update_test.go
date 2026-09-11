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
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// newUpdateServer serves the provider, the instance GET and the instance PATCH.
// A nil patchHandler fails the test if a write is attempted.
func newUpdateServer(t *testing.T, currentSpec map[string]any, patchHandler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return newUpdateServerWithProvider(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(psmdbProvider())
	}, currentSpec, patchHandler)
}

func newUpdateServerWithProvider(t *testing.T, providerHandler http.HandlerFunc, currentSpec map[string]any, patchHandler http.HandlerFunc) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/clusters/main/providers/psmdb", providerHandler)
	mux.HandleFunc("/v1/clusters/main/namespaces/everest/instances/my-db", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			writeInstance(w, currentSpec)
		case http.MethodPatch:
			if patchHandler == nil {
				assert.Fail(t, "PATCH should not have been called")
				return
			}
			patchHandler(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
	return httptest.NewServer(mux)
}

func writeInstance(w http.ResponseWriter, spec map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{ //nolint:errchkjson
		"metadata": map[string]any{"name": "my-db", "namespace": "everest"},
		"spec":     spec,
	})
}

func baseUpdateOpts() UpdateOptions {
	return UpdateOptions{Name: "my-db", Namespace: "everest", Cluster: "main"}
}

func psmdbSpec() map[string]any {
	return map[string]any{
		"providerRef": map[string]any{"name": "psmdb"},
		"version":     "8.0",
		"topology":    map[string]any{"type": "replicaset"},
		"components": map[string]any{
			"engine": map[string]any{"replicas": 3},
			"proxy":  map[string]any{"replicas": 1},
		},
	}
}

// decodePatch reads the merge patch document off a PATCH request.
func decodePatch(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	assert.Equal(t, "application/merge-patch+json", r.Header.Get("Content-Type"))
	b, err := io.ReadAll(r.Body)
	assert.NoError(t, err)
	var doc map[string]any
	assert.NoError(t, json.Unmarshal(b, &doc))
	spec, isObject := doc["spec"].(map[string]any)
	assert.True(t, isObject, "patch has no spec object: %s", string(b))
	return spec
}

func runUpdate(t *testing.T, srv *httptest.Server, opts UpdateOptions) error {
	t.Helper()
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, newTestConfig(srv.URL).Save(cfgPath))
	return NewUpdater(Config{}, zap.NewNop().Sugar()).Run(context.Background(), opts, cfgPath)
}

func TestUpdate_NoOpRejected(t *testing.T) {
	t.Parallel()

	iu := NewUpdater(Config{}, zap.NewNop().Sugar())
	err := iu.Run(context.Background(), baseUpdateOpts(), filepath.Join(t.TempDir(), "config.yaml"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--set")
}

// The patch must carry only what --set named: sending the whole spec back is the
// read-modify-write this command exists to avoid.
func TestUpdate_PatchCarriesOnlyTheNamedFields(t *testing.T) {
	t.Parallel()

	srv := newUpdateServer(t, psmdbSpec(), func(w http.ResponseWriter, r *http.Request) {
		spec := decodePatch(t, r)
		assert.Equal(t, map[string]any{
			"components": map[string]any{"engine": map[string]any{"replicas": float64(5)}},
		}, spec)
		writeInstance(w, psmdbSpec())
	})
	defer srv.Close()

	opts := baseUpdateOpts()
	opts.Set = []string{"components.engine.replicas=5"}
	require.NoError(t, runUpdate(t, srv, opts))
}

func TestUpdate_SetNullSendsJSONNull(t *testing.T) {
	t.Parallel()

	srv := newUpdateServer(t, psmdbSpec(), func(w http.ResponseWriter, r *http.Request) {
		spec := decodePatch(t, r)
		value, present := spec["version"]
		assert.True(t, present, "null must be sent, not dropped: it is what removes the member")
		assert.Nil(t, value)
		writeInstance(w, psmdbSpec())
	})
	defer srv.Close()

	opts := baseUpdateOpts()
	opts.Set = []string{"version=null"}
	require.NoError(t, runUpdate(t, srv, opts))
}

func TestUpdate_SetWinsOverValuesFile(t *testing.T) {
	t.Parallel()

	valuesFile := filepath.Join(t.TempDir(), "values.yaml")
	require.NoError(t, os.WriteFile(valuesFile, []byte("components:\n  engine:\n    replicas: 2\nbackup:\n  enabled: true\n"), 0o600))

	srv := newUpdateServer(t, psmdbSpec(), func(w http.ResponseWriter, r *http.Request) {
		spec := decodePatch(t, r)
		components, _ := spec["components"].(map[string]any)
		engine, _ := components["engine"].(map[string]any)
		assert.InDelta(t, 7, engine["replicas"], 0, "--set must win over -f")
		backup, _ := spec["backup"].(map[string]any)
		assert.Equal(t, true, backup["enabled"], "-f fields not named by --set must still be sent")
		writeInstance(w, psmdbSpec())
	})
	defer srv.Close()

	opts := baseUpdateOpts()
	opts.ValuesFile = valuesFile
	opts.Set = []string{"components.engine.replicas=7"}
	require.NoError(t, runUpdate(t, srv, opts))
}

// spec.components is a map, so fieldValidation=Strict cannot catch a misspelt
// component name; this is the one check the CLI still owns.
func TestUpdate_UnknownComponentRejected(t *testing.T) {
	t.Parallel()

	srv := newUpdateServer(t, psmdbSpec(), nil)
	defer srv.Close()

	opts := baseUpdateOpts()
	opts.Set = []string{"components.engien.replicas=5"}

	err := runUpdate(t, srv, opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "engien")
	assert.Contains(t, err.Error(), "engine")
}

func TestUpdate_UnknownComponentInValuesFileRejected(t *testing.T) {
	t.Parallel()

	valuesFile := filepath.Join(t.TempDir(), "values.yaml")
	require.NoError(t, os.WriteFile(valuesFile, []byte("components:\n  engien:\n    replicas: 5\n"), 0o600))

	srv := newUpdateServer(t, psmdbSpec(), nil)
	defer srv.Close()

	opts := baseUpdateOpts()
	opts.ValuesFile = valuesFile

	err := runUpdate(t, srv, opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "engien")
}

// The component check fails open on purpose: the server validates component names
// too, so a provider lookup failure must not block an otherwise valid update.
func TestUpdate_ProviderLookupFails_UpdateStillProceeds(t *testing.T) {
	t.Parallel()

	var patched atomic.Bool
	srv := newUpdateServerWithProvider(t,
		func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"message":"provider backend is down"}`))
		},
		psmdbSpec(),
		func(w http.ResponseWriter, r *http.Request) {
			patched.Store(true)
			assert.Equal(t, map[string]any{
				"components": map[string]any{"engine": map[string]any{"replicas": float64(5)}},
			}, decodePatch(t, r))
			writeInstance(w, psmdbSpec())
		},
	)
	defer srv.Close()

	opts := baseUpdateOpts()
	opts.Set = []string{"components.engine.replicas=5"}

	require.NoError(t, runUpdate(t, srv, opts))
	assert.True(t, patched.Load(), "the check must fail open, not swallow the update")
}

func TestUpdate_NotFound(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/clusters/main/namespaces/everest/instances/my-db", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	opts := baseUpdateOpts()
	opts.Set = []string{"version=8.1"}

	err := runUpdate(t, srv, opts)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// A patch naming no component skips the GET entirely: the write path is one call.
func TestUpdate_NoGetWhenNothingNeedsIt(t *testing.T) {
	t.Parallel()

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/clusters/main/namespaces/everest/instances/my-db", func(w http.ResponseWriter, r *http.Request) {
		if !assert.Equal(t, http.MethodPatch, r.Method, "only the PATCH should be sent") {
			return
		}
		writeInstance(w, psmdbSpec())
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	opts := baseUpdateOpts()
	opts.Set = []string{"version=8.1"}
	require.NoError(t, runUpdate(t, srv, opts))
}

func TestUpdate_DryRunDoesNotWrite(t *testing.T) {
	t.Parallel()

	srv := newUpdateServer(t, psmdbSpec(), nil)
	defer srv.Close()

	opts := baseUpdateOpts()
	opts.DryRun = true
	opts.Set = []string{"components.engine.replicas=5"}
	require.NoError(t, runUpdate(t, srv, opts))
}

// A path already at the value being set is labelled, not printed as "3 -> 3" and
// not hidden, so a --set that does nothing is visible before it is sent.
func TestRenderChanges_LabelsNoOpPaths(t *testing.T) {
	t.Parallel()

	out := renderChanges([]specChange{
		{path: "components.engine.replicas", from: "3", to: "5"},
		{path: "components.proxy.replicas", from: "1", to: "1"},
	})

	assert.Equal(t, "  components.engine.replicas: 3 -> 5\n  components.proxy.replicas: 1 (no change)\n", out)
	assert.NotContains(t, out, "Nothing would change", "a real change must not be summarised as a no-op")
}

func TestRenderChanges_AllNoOpsSaysNothingWouldChange(t *testing.T) {
	t.Parallel()

	out := renderChanges([]specChange{{path: "version", from: "8.0", to: "8.0"}})

	assert.Contains(t, out, "version: 8.0 (no change)")
	assert.Contains(t, out, "Nothing would change")
}

// An empty patch names no paths at all, which is still "nothing would change"
// rather than a bare header with no explanation under it.
func TestRenderChanges_NoPathsSaysNothingWouldChange(t *testing.T) {
	t.Parallel()

	assert.Contains(t, renderChanges(nil), "Nothing would change")
}

func TestChangedPaths_ReportsCurrentAndNewValues(t *testing.T) {
	t.Parallel()

	current := map[string]any{
		"components": map[string]any{"engine": map[string]any{"replicas": float64(3)}},
		"version":    "8.0",
	}
	patch := map[string]any{
		"components": map[string]any{"engine": map[string]any{"replicas": int64(5)}},
		"version":    nil,
		"backup":     map[string]any{"enabled": true},
	}

	assert.Equal(t, []specChange{
		{path: "backup.enabled", from: "<unset>", to: "true"},
		{path: "components.engine.replicas", from: "3", to: "5"},
		{path: "version", from: "8.0", to: "null"},
	}, changedPaths(current, patch, ""))
}

// A list is one leaf, because a merge patch replaces it wholesale rather than
// merging it element by element.
func TestChangedPaths_ListIsOneLeaf(t *testing.T) {
	t.Parallel()

	current := map[string]any{"backup": map[string]any{"storages": []any{"a", "b"}}}
	patch := map[string]any{"backup": map[string]any{"storages": []any{"a"}}}

	assert.Equal(t, []specChange{
		{path: "backup.storages", from: `["a","b"]`, to: `["a"]`},
	}, changedPaths(current, patch, ""))
}
