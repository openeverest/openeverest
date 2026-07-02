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
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/openeverest/openeverest/v2/client"
	"github.com/openeverest/openeverest/v2/pkg/cli"
	"github.com/openeverest/openeverest/v2/pkg/cli/config"
)

// ---- helpers ----------------------------------------------------------------

func boolPtr(b bool) *bool   { return &b }
func strPtr(s string) *string { return &s }

// buildProvider builds a minimal Provider fixture for tests.
func buildProvider(name string, versions []struct {
	name     string
	isDefault bool
}, topologies map[string][]string) *client.Provider {
	meta := map[string]any{"name": name}

	var vers []struct {
		Components *map[string]string `json:"components,omitempty"`
		Default    *bool              `json:"default,omitempty"`
		Name       string             `json:"name"`
	}
	for _, v := range versions {
		v := v
		vers = append(vers, struct {
			Components *map[string]string `json:"components,omitempty"`
			Default    *bool              `json:"default,omitempty"`
			Name       string             `json:"name"`
		}{
			Name:    v.name,
			Default: boolPtr(v.isDefault),
		})
	}

	topos := map[string]struct {
		Components *map[string]struct {
			Optional *bool `json:"optional,omitempty"`
		} `json:"components,omitempty"`
		ConfigSchema *map[string]any `json:"configSchema,omitempty"`
	}{}
	for topo, comps := range topologies {
		compMap := map[string]struct {
			Optional *bool `json:"optional,omitempty"`
		}{}
		for _, c := range comps {
			compMap[c] = struct{ Optional *bool `json:"optional,omitempty"` }{}
		}
		topos[topo] = struct {
			Components *map[string]struct {
				Optional *bool `json:"optional,omitempty"`
			} `json:"components,omitempty"`
			ConfigSchema *map[string]any `json:"configSchema,omitempty"`
		}{Components: &compMap}
	}

	globalComps := map[string]struct {
		CustomSpecSchema *map[string]any `json:"customSpecSchema,omitempty"`
		Type             *string         `json:"type,omitempty"`
	}{}
	for _, comps := range topologies {
		for _, c := range comps {
			globalComps[c] = struct {
				CustomSpecSchema *map[string]any `json:"customSpecSchema,omitempty"`
				Type             *string         `json:"type,omitempty"`
			}{Type: strPtr(c + "-type")}
		}
	}

	prov := &client.Provider{
		Metadata: &meta,
	}
	prov.Spec.Versions = &vers
	prov.Spec.Topologies = &topos
	prov.Spec.Components = &globalComps
	return prov
}

// newTestConfig returns a config with a single context pointing at serverURL.
func newTestConfig(serverURL string) *config.Config {
	host := serverURL[len("http://"):]
	srvName := host
	userName := "admin@" + srvName
	return &config.Config{
		APIVersion:     "config.openeverest.io/v1alpha1",
		Kind:           "ClientConfig",
		CurrentContext: userName,
		Contexts: []config.NamedContext{
			{Name: userName, Context: config.Context{Server: srvName, User: userName}},
		},
		Servers: []config.NamedServer{
			{Name: srvName, Server: config.Server{URL: serverURL}},
		},
		Users: []config.NamedUser{
			{Name: userName, User: config.User{
				AccessToken:  "test-token",
				RefreshToken: "rt-test",
				ExpiresAt:    time.Now().Add(time.Hour),
			}},
		},
	}
}

// ---- defaultVersion ---------------------------------------------------------

func TestDefaultVersion_ReturnsDefaultBundle(t *testing.T) {
	t.Parallel()
	prov := buildProvider("psmdb", []struct {
		name      string
		isDefault bool
	}{
		{"7.0", false},
		{"8.0", true},
	}, nil)
	assert.Equal(t, "8.0", defaultVersion(prov))
}

func TestDefaultVersion_FallsBackToFirst(t *testing.T) {
	t.Parallel()
	prov := buildProvider("psmdb", []struct {
		name      string
		isDefault bool
	}{
		{"7.0", false},
		{"8.0", false},
	}, nil)
	assert.Equal(t, "7.0", defaultVersion(prov))
}

func TestDefaultVersion_NoVersions(t *testing.T) {
	t.Parallel()
	prov := &client.Provider{}
	assert.Equal(t, "", defaultVersion(prov))
}

// ---- firstTopology ----------------------------------------------------------

func TestFirstTopology_AlphabeticalOrder(t *testing.T) {
	t.Parallel()
	prov := buildProvider("psmdb", nil, map[string][]string{
		"standalone":  {"engine"},
		"replicaset":  {"engine", "proxy"},
	})
	assert.Equal(t, "replicaset", firstTopology(prov))
}

func TestFirstTopology_NoTopologies(t *testing.T) {
	t.Parallel()
	prov := &client.Provider{}
	assert.Equal(t, "", firstTopology(prov))
}

// ---- validateTopology -------------------------------------------------------

func TestValidateTopology_Valid(t *testing.T) {
	t.Parallel()
	prov := buildProvider("psmdb", nil, map[string][]string{"replicaset": {"engine"}})
	assert.NoError(t, validateTopology("replicaset", prov))
}

func TestValidateTopology_Invalid(t *testing.T) {
	t.Parallel()
	prov := buildProvider("psmdb", nil, map[string][]string{
		"standalone": {"engine"},
		"replicaset": {"engine"},
	})
	err := validateTopology("sharded", prov)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sharded")
	assert.Contains(t, err.Error(), "replicaset")
	assert.Contains(t, err.Error(), "standalone")
}

// ---- validateComponents -----------------------------------------------------

func TestValidateComponents_ValidPath(t *testing.T) {
	t.Parallel()
	prov := buildProvider("psmdb", nil, map[string][]string{"replicaset": {"engine", "proxy"}})
	err := validateComponents([]string{"components.engine.replicas=3"}, prov, "replicaset")
	assert.NoError(t, err)
}

func TestValidateComponents_InvalidComponent(t *testing.T) {
	t.Parallel()
	prov := buildProvider("psmdb", nil, map[string][]string{"replicaset": {"engine", "proxy"}})
	err := validateComponents([]string{"components.mongos.replicas=3"}, prov, "replicaset")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mongos")
	assert.Contains(t, err.Error(), "engine")
	assert.Contains(t, err.Error(), "proxy")
}

func TestValidateComponents_NonComponentPathSkipped(t *testing.T) {
	t.Parallel()
	// --set backup.enabled=true is not a components.* path — must not be rejected
	prov := buildProvider("psmdb", nil, map[string][]string{"replicaset": {"engine"}})
	err := validateComponents([]string{"backup.enabled=true"}, prov, "replicaset")
	assert.NoError(t, err)
}

func TestValidateComponents_EmptyTopologyFallsBackToGlobal(t *testing.T) {
	t.Parallel()
	// Topology has no components (simulates API stripping null entries).
	// Validation should fall back to spec.components.
	prov := buildProvider("psmdb", nil, map[string][]string{"replicaset": {"engine"}})
	// Override: topology map exists but components map is nil inside it.
	empty := map[string]struct {
		Components *map[string]struct {
			Optional *bool `json:"optional,omitempty"`
		} `json:"components,omitempty"`
		ConfigSchema *map[string]any `json:"configSchema,omitempty"`
	}{
		"replicaset": {Components: nil},
	}
	prov.Spec.Topologies = &empty
	// engine IS in spec.components (set by buildProvider), so this should pass.
	err := validateComponents([]string{"components.engine.replicas=3"}, prov, "replicaset")
	assert.NoError(t, err)
}

// ---- parseSetFlags ----------------------------------------------------------

func TestParseSetFlags_IntCoercion(t *testing.T) {
	t.Parallel()
	m, err := parseSetFlags([]string{"components.engine.replicas=3"})
	require.NoError(t, err)
	comps := m["components"].(map[string]any)
	engine := comps["engine"].(map[string]any)
	assert.Equal(t, int64(3), engine["replicas"])
}

func TestParseSetFlags_BoolCoercion(t *testing.T) {
	t.Parallel()
	m, err := parseSetFlags([]string{"backup.enabled=true"})
	require.NoError(t, err)
	backup := m["backup"].(map[string]any)
	assert.Equal(t, true, backup["enabled"])
}

func TestParseSetFlags_StringFallback(t *testing.T) {
	t.Parallel()
	m, err := parseSetFlags([]string{"components.engine.storage.size=50Gi"})
	require.NoError(t, err)
	comps := m["components"].(map[string]any)
	engine := comps["engine"].(map[string]any)
	storage := engine["storage"].(map[string]any)
	assert.Equal(t, "50Gi", storage["size"])
}

func TestParseSetFlags_MissingEquals(t *testing.T) {
	t.Parallel()
	_, err := parseSetFlags([]string{"components.engine.replicas"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be in the form")
}

func TestParseSetFlags_EmptyPath(t *testing.T) {
	t.Parallel()
	_, err := parseSetFlags([]string{"=value"})
	require.Error(t, err)
}

func TestParseSetFlags_ConflictingPaths(t *testing.T) {
	t.Parallel()
	// First sets engine to a scalar, second tries to descend into it.
	_, err := parseSetFlags([]string{"components.engine=5", "components.engine.replicas=3"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "conflicting")
}

func TestParseSetFlags_Empty(t *testing.T) {
	t.Parallel()
	m, err := parseSetFlags(nil)
	require.NoError(t, err)
	assert.Nil(t, m)
}

// ---- deepMerge --------------------------------------------------------------

func TestDeepMerge_ScalarOverride(t *testing.T) {
	t.Parallel()
	dst := map[string]any{"a": 1}
	src := map[string]any{"a": 2}
	deepMerge(dst, src)
	assert.Equal(t, 2, dst["a"])
}

func TestDeepMerge_NestedMerge(t *testing.T) {
	t.Parallel()
	dst := map[string]any{"engine": map[string]any{"replicas": 1, "size": "10Gi"}}
	src := map[string]any{"engine": map[string]any{"replicas": 3}}
	deepMerge(dst, src)
	eng := dst["engine"].(map[string]any)
	assert.Equal(t, 3, eng["replicas"])
	assert.Equal(t, "10Gi", eng["size"]) // preserved
}

func TestDeepMerge_NewKey(t *testing.T) {
	t.Parallel()
	dst := map[string]any{"a": 1}
	src := map[string]any{"b": 2}
	deepMerge(dst, src)
	assert.Equal(t, 1, dst["a"])
	assert.Equal(t, 2, dst["b"])
}

// ---- buildPayload -----------------------------------------------------------

func TestBuildPayload_BasicFields(t *testing.T) {
	t.Parallel()
	p := buildPayload("my-db", "psmdb", "8.0", "replicaset", nil)
	spec := p["spec"].(map[string]any)
	assert.Equal(t, "psmdb", spec["provider"])
	assert.Equal(t, "8.0", spec["version"])
	topo := spec["topology"].(map[string]any)
	assert.Equal(t, "replicaset", topo["type"])
	meta := p["metadata"].(map[string]any)
	assert.Equal(t, "my-db", meta["name"])
}

func TestBuildPayload_ExplicitFlagsWinOverOverrides(t *testing.T) {
	t.Parallel()
	overrides := map[string]any{"provider": "wrong", "version": "wrong"}
	p := buildPayload("db", "psmdb", "8.0", "standalone", overrides)
	spec := p["spec"].(map[string]any)
	assert.Equal(t, "psmdb", spec["provider"])
	assert.Equal(t, "8.0", spec["version"])
}

func TestBuildPayload_EmptyVersionAndTopologyOmitted(t *testing.T) {
	t.Parallel()
	p := buildPayload("db", "psmdb", "", "", nil)
	spec := p["spec"].(map[string]any)
	_, hasVersion := spec["version"]
	_, hasTopology := spec["topology"]
	assert.False(t, hasVersion)
	assert.False(t, hasTopology)
}

// ---- loadValuesFile ---------------------------------------------------------

func TestLoadValuesFile_Valid(t *testing.T) {
	t.Parallel()
	f := filepath.Join(t.TempDir(), "values.yaml")
	require.NoError(t, os.WriteFile(f, []byte("components:\n  engine:\n    replicas: 3\n"), 0600))
	m, err := loadValuesFile(f)
	require.NoError(t, err)
	comps := m["components"].(map[string]any)
	engine := comps["engine"].(map[string]any)
	assert.Equal(t, 3, engine["replicas"])
}

func TestLoadValuesFile_MissingFile(t *testing.T) {
	t.Parallel()
	_, err := loadValuesFile("/nonexistent/values.yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot read")
}

func TestLoadValuesFile_InvalidYAML(t *testing.T) {
	t.Parallel()
	f := filepath.Join(t.TempDir(), "bad.yaml")
	require.NoError(t, os.WriteFile(f, []byte(":\t:bad"), 0600))
	_, err := loadValuesFile(f)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot parse")
}

// ---- buildSpecOverrides -----------------------------------------------------

func TestBuildSpecOverrides_SetWinsOverFile(t *testing.T) {
	t.Parallel()
	f := filepath.Join(t.TempDir(), "values.yaml")
	require.NoError(t, os.WriteFile(f, []byte("components:\n  engine:\n    replicas: 1\n"), 0600))
	m, err := buildSpecOverrides(f, []string{"components.engine.replicas=5"})
	require.NoError(t, err)
	comps := m["components"].(map[string]any)
	engine := comps["engine"].(map[string]any)
	assert.Equal(t, int64(5), engine["replicas"])
}

func TestBuildSpecOverrides_FileOnly(t *testing.T) {
	t.Parallel()
	f := filepath.Join(t.TempDir(), "values.yaml")
	require.NoError(t, os.WriteFile(f, []byte("backup:\n  enabled: true\n"), 0600))
	m, err := buildSpecOverrides(f, nil)
	require.NoError(t, err)
	backup := m["backup"].(map[string]any)
	assert.Equal(t, true, backup["enabled"])
}

func TestBuildSpecOverrides_SetOnly(t *testing.T) {
	t.Parallel()
	m, err := buildSpecOverrides("", []string{"components.engine.replicas=3"})
	require.NoError(t, err)
	comps := m["components"].(map[string]any)
	engine := comps["engine"].(map[string]any)
	assert.Equal(t, int64(3), engine["replicas"])
}

// ---- validateServerURL / normalizeServerURL ---------------------------------

func TestValidateServerURL(t *testing.T) {
	t.Parallel()
	assert.NoError(t, cli.ValidateServerURL("http://localhost:8080"))
	assert.NoError(t, cli.ValidateServerURL("https://prod.example.com"))
	assert.Error(t, cli.ValidateServerURL("localhost:8080"))
	assert.Error(t, cli.ValidateServerURL("ftp://bad.example.com"))
}

func TestNormalizeServerURL(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "http://localhost:8080/v1", cli.NormalizeServerURL("http://localhost:8080"))
	assert.Equal(t, "http://localhost:8080/v1", cli.NormalizeServerURL("http://localhost:8080/"))
	assert.Equal(t, "http://localhost:8080/v1", cli.NormalizeServerURL("http://localhost:8080/v1"))
}

// ---- Run integration tests --------------------------------------------------

// newRunServer returns a test server that handles /v1/clusters/main/providers/{p}
// and /v1/clusters/main/namespaces/{ns}/instances with the provided handlers.
func newRunServer(t *testing.T, providerHandler, createHandler http.HandlerFunc) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/clusters/main/providers/psmdb", providerHandler)
	mux.HandleFunc("/v1/clusters/main/namespaces/", createHandler)
	return httptest.NewServer(mux)
}

func psmdbProvider() *client.Provider {
	return buildProvider("psmdb", []struct {
		name      string
		isDefault bool
	}{{"8.0", true}}, map[string][]string{
		"replicaset": {"engine", "proxy"},
		"standalone": {"engine"},
	})
}

func TestRun_HappyPath(t *testing.T) {
	t.Parallel()

	provHandler := func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(psmdbProvider())
	}
	createHandler := func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}

	srv := newRunServer(t, provHandler, createHandler)
	defer srv.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, newTestConfig(srv.URL).Save(cfgPath))

	ic := NewInstanceCreator(Config{}, zap.NewNop().Sugar())
	err := ic.Run(context.Background(), CreateOptions{
		Name:      "my-db",
		Namespace: "everest",
		Provider:  "psmdb",
		Cluster:   "main",
	}, cfgPath)
	assert.NoError(t, err)
}

func TestRun_ProviderNotFound(t *testing.T) {
	t.Parallel()

	srv := newRunServer(t,
		func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) },
		func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusCreated) },
	)
	defer srv.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, newTestConfig(srv.URL).Save(cfgPath))

	ic := NewInstanceCreator(Config{}, zap.NewNop().Sugar())
	err := ic.Run(context.Background(), CreateOptions{
		Name:      "my-db",
		Namespace: "everest",
		Provider:  "psmdb",
		Cluster:   "main",
	}, cfgPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRun_InstanceAlreadyExists(t *testing.T) {
	t.Parallel()

	provHandler := func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(psmdbProvider())
	}
	createHandler := func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
	}

	srv := newRunServer(t, provHandler, createHandler)
	defer srv.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, newTestConfig(srv.URL).Save(cfgPath))

	ic := NewInstanceCreator(Config{}, zap.NewNop().Sugar())
	err := ic.Run(context.Background(), CreateOptions{
		Name:      "my-db",
		Namespace: "everest",
		Provider:  "psmdb",
		Cluster:   "main",
	}, cfgPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestRun_InvalidTopology(t *testing.T) {
	t.Parallel()

	provHandler := func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(psmdbProvider())
	}

	srv := newRunServer(t, provHandler, func(w http.ResponseWriter, _ *http.Request) {})
	defer srv.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, newTestConfig(srv.URL).Save(cfgPath))

	ic := NewInstanceCreator(Config{}, zap.NewNop().Sugar())
	err := ic.Run(context.Background(), CreateOptions{
		Name:      "my-db",
		Namespace: "everest",
		Provider:  "psmdb",
		Cluster:   "main",
		Topology:  "sharded",
	}, cfgPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sharded")
	assert.Contains(t, err.Error(), "valid topologies")
}

func TestRun_InvalidComponent(t *testing.T) {
	t.Parallel()

	provHandler := func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(psmdbProvider())
	}

	srv := newRunServer(t, provHandler, func(w http.ResponseWriter, _ *http.Request) {})
	defer srv.Close()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, newTestConfig(srv.URL).Save(cfgPath))

	ic := NewInstanceCreator(Config{}, zap.NewNop().Sugar())
	err := ic.Run(context.Background(), CreateOptions{
		Name:      "my-db",
		Namespace: "everest",
		Provider:  "psmdb",
		Cluster:   "main",
		Set:       []string{"components.mongos.replicas=3"},
	}, cfgPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mongos")
	assert.Contains(t, err.Error(), "valid components")
}

func TestRun_NoActiveContext(t *testing.T) {
	t.Parallel()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, (&config.Config{
		APIVersion:     "config.openeverest.io/v1alpha1",
		Kind:           "ClientConfig",
		CurrentContext: "missing",
	}).Save(cfgPath))

	ic := NewInstanceCreator(Config{}, zap.NewNop().Sugar())
	err := ic.Run(context.Background(), CreateOptions{
		Name:      "my-db",
		Namespace: "everest",
		Provider:  "psmdb",
		Cluster:   "main",
	}, cfgPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no active context")
}

func TestRun_ContextFlag(t *testing.T) {
	t.Parallel()

	provHandler := func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(psmdbProvider())
	}
	createHandler := func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}

	srv := newRunServer(t, provHandler, createHandler)
	defer srv.Close()

	// Config has two contexts; currentContext points to "other".
	host := srv.URL[len("http://"):]
	cfg := &config.Config{
		APIVersion:     "config.openeverest.io/v1alpha1",
		Kind:           "ClientConfig",
		CurrentContext: "other@other",
		Contexts: []config.NamedContext{
			{Name: "other@other", Context: config.Context{Server: "other", User: "other@other"}},
			{Name: "admin@" + host, Context: config.Context{Server: host, User: "admin@" + host}},
		},
		Servers: []config.NamedServer{
			{Name: host, Server: config.Server{URL: srv.URL}},
		},
		Users: []config.NamedUser{
			{Name: "admin@" + host, User: config.User{
				AccessToken: "test-token",
				ExpiresAt:   time.Now().Add(time.Hour),
			}},
		},
	}
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, cfg.Save(cfgPath))

	ic := NewInstanceCreator(Config{}, zap.NewNop().Sugar())
	err := ic.Run(context.Background(), CreateOptions{
		Name:      "my-db",
		Namespace: "everest",
		Provider:  "psmdb",
		Cluster:   "main",
		Context:   "admin@" + host,
	}, cfgPath)
	assert.NoError(t, err)
}

func TestRun_UnknownContext(t *testing.T) {
	t.Parallel()

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, newTestConfig("http://localhost:9999").Save(cfgPath))

	ic := NewInstanceCreator(Config{}, zap.NewNop().Sugar())
	err := ic.Run(context.Background(), CreateOptions{
		Name:      "my-db",
		Namespace: "everest",
		Provider:  "psmdb",
		Cluster:   "main",
		Context:   "nonexistent",
	}, cfgPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent")
}
