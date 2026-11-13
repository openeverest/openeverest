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
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/openeverest/openeverest/v2/client"
	"github.com/openeverest/openeverest/v2/pkg/cli/config"
)

// testInstance mirrors just the JSON shape the list runner reads off of
// client.Instance, so tests can build fixtures without the generated type's
// large anonymous Spec/Status structs.
type testInstance struct {
	Metadata map[string]any `json:"metadata"`
	Spec     struct {
		ProviderRef struct {
			Name string `json:"name,omitempty"`
		} `json:"providerRef"`
	} `json:"spec"`
	Status struct {
		Phase   string `json:"phase,omitempty"`
		Version string `json:"version,omitempty"`
	} `json:"status"`
}

func newTestInstance(name, namespace, provider, phase, version string) testInstance {
	inst := testInstance{
		Metadata: map[string]any{
			"name":              name,
			"namespace":         namespace,
			"creationTimestamp": time.Now().Add(-time.Hour).UTC().Format(time.RFC3339),
		},
	}
	inst.Spec.ProviderRef.Name = provider
	inst.Status.Phase = phase
	inst.Status.Version = version
	return inst
}

// The rows come from the fixtures shared with the other readers of an
// Instance's status. STATUS says what is happening and AVAILABLE whether
// clients can connect: a database can be up with a problem, or down while
// nothing is wrong.
func TestPrintInstanceTable_Fixtures(t *testing.T) {
	t.Parallel()

	names := []string{
		"healthy",
		"rolling-upgrade",
		"in-place-restore",
		"creating",
		"change-availability-unknown",
		"backup-storage-missing-stalled",
		"crash-loop-all-replicas-stalled",
		"reason-the-client-has-never-seen",
		"never-reconciled-old",
		"deleting",
	}
	instances := make([]client.Instance, 0, len(names))
	var now time.Time
	for _, name := range names {
		fixture := displayFixtureNamed(t, name)
		instances = append(instances, fixture.Instance)
		now = fixture.Now
	}
	var out bytes.Buffer

	printInstanceTable(&out, instances, now)

	want := `NAME                              NAMESPACE  PROVIDER  VERSION  STATUS                     AVAILABLE  AGE
healthy                           demo       -         -        Ready                      Yes        40d
rolling-upgrade                   demo       -         -        Upgrading                  Yes        40d
in-place-restore                  demo       -         -        Restoring                  No         40d
creating                          demo       -         -        Creating                   No         40d
change-availability-unknown       demo       -         -        Applying changes           -          40d
backup-storage-missing-stalled    demo       -         -        Backup storage not found   Yes        40d
crash-loop-all-replicas-stalled   demo       -         -        Crash loop                 No         40d
reason-the-client-has-never-seen  demo       -         -        Rebalancing shards         Yes        40d
never-reconciled-old              demo       -         -        Not picked up by provider  -          6m
deleting                          demo       -         -        Deleting                   -          40d
`
	assert.Equal(t, want, trimLineEnds(out.String()))
}

// trimLineEnds drops the padding the table writer leaves after the last
// column.
func trimLineEnds(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " ")
	}
	return strings.Join(lines, "\n")
}

func TestInstanceList_HappyPath(t *testing.T) {
	t.Parallel()

	items := []testInstance{
		newTestInstance("my-mongo", "everest", "percona-server-mongodb", "Provisioned", "8.0.12"),
		newTestInstance("my-postgres", "everest", "postgresql", "Pending", "16.1"),
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"items": items})
	}))
	defer srv.Close()

	cfg := newTestConfig(srv.URL)
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, cfg.Save(cfgPath))

	runner := NewListRunner(Config{Pretty: true}, zap.NewNop().Sugar())
	err := runner.Run(t.Context(), ListOptions{
		Namespace: "everest",
		Cluster:   "main",
	}, cfgPath)
	require.NoError(t, err)
}

func TestInstanceList_AllNamespaces(t *testing.T) {
	t.Parallel()

	nsInstances := map[string][]testInstance{
		"everest":  {newTestInstance("my-mongo", "everest", "percona-server-mongodb", "Provisioned", "8.0.12")},
		"everest2": {newTestInstance("my-postgres", "everest2", "postgresql", "Provisioned", "16.1")},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if strings.HasSuffix(r.URL.Path, "/namespaces") {
			_ = json.NewEncoder(w).Encode([]string{"everest", "everest2"})
			return
		}

		for ns, items := range nsInstances {
			if strings.Contains(r.URL.Path, "/namespaces/"+ns+"/instances") {
				_ = json.NewEncoder(w).Encode(map[string]any{"items": items})
				return
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"items": []testInstance{}})
	}))
	defer srv.Close()

	cfg := newTestConfig(srv.URL)
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, cfg.Save(cfgPath))

	runner := NewListRunner(Config{Pretty: true}, zap.NewNop().Sugar())
	err := runner.Run(t.Context(), ListOptions{
		AllNamespaces: true,
		Cluster:       "main",
	}, cfgPath)
	require.NoError(t, err)
}

func TestInstanceList_EmptyResult(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"items": []testInstance{}})
	}))
	defer srv.Close()

	cfg := newTestConfig(srv.URL)
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, cfg.Save(cfgPath))

	runner := NewListRunner(Config{Pretty: true}, zap.NewNop().Sugar())
	err := runner.Run(t.Context(), ListOptions{
		Namespace: "everest",
		Cluster:   "main",
	}, cfgPath)
	require.NoError(t, err)
}

func TestInstanceList_ServerError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	cfg := newTestConfig(srv.URL)
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, cfg.Save(cfgPath))

	runner := NewListRunner(Config{Pretty: true}, zap.NewNop().Sugar())
	err := runner.Run(t.Context(), ListOptions{
		Namespace: "everest",
		Cluster:   "main",
	}, cfgPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected response")
}

func TestInstanceList_NoActiveContext(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		APIVersion: "config.openeverest.io/v1alpha1",
		Kind:       "ClientConfig",
	}
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, cfg.Save(cfgPath))

	runner := NewListRunner(Config{Pretty: true}, zap.NewNop().Sugar())
	err := runner.Run(t.Context(), ListOptions{
		Namespace: "everest",
		Cluster:   "main",
	}, cfgPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no active context")
}

func TestInstanceList_JSONOutput(t *testing.T) {
	t.Parallel()

	items := []testInstance{
		newTestInstance("my-mongo", "everest", "percona-server-mongodb", "Provisioned", "8.0.12"),
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{"items": items})
	}))
	defer srv.Close()

	cfg := newTestConfig(srv.URL)
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, cfg.Save(cfgPath))

	// Pretty=false → JSON path
	runner := NewListRunner(Config{Pretty: false}, zap.NewNop().Sugar())
	err := runner.Run(t.Context(), ListOptions{
		Namespace: "everest",
		Cluster:   "main",
	}, cfgPath)
	require.NoError(t, err)
}
