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

package reconciler

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
	"github.com/openeverest/openeverest/v2/provider-runtime/server"
)

// readinessTestProvider is a no-op provider used to construct a reconciler.
type readinessTestProvider struct{}

func (p *readinessTestProvider) Name() string                       { return "readiness-test" }
func (p *readinessTestProvider) Types() func(*runtime.Scheme) error { return nil }
func (p *readinessTestProvider) Validate(*controller.Context) error { return nil }
func (p *readinessTestProvider) Sync(*controller.Context) error     { return nil }
func (p *readinessTestProvider) Cleanup(*controller.Context) error  { return nil }
func (p *readinessTestProvider) Status(*controller.Context) (controller.Status, error) {
	return controller.Status{}, nil
}

// unreachableKubeconfig points at a port with nothing listening, so the
// manager's caches can never sync.
const unreachableKubeconfig = `apiVersion: v1
kind: Config
clusters:
- name: c
  cluster:
    server: https://127.0.0.1:59999
contexts:
- name: c
  context:
    cluster: c
    user: u
current-context: c
users:
- name: u
  user: {}
`

func freePort(t *testing.T) int {
	t.Helper()
	l, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer l.Close() //nolint:errcheck
	addr, ok := l.Addr().(*net.TCPAddr)
	require.True(t, ok)
	return addr.Port
}

func statusCode(url string) int {
	resp, err := http.Get(url) //nolint:noctx,gosec
	if err != nil {
		return -1
	}
	defer resp.Body.Close() //nolint:errcheck
	return resp.StatusCode
}

// TestServerReadinessGatedOnManager is a regression test for #2598.
//
// The reconciler's validation server must not report ready before the manager's
// caches have synced (before the fix, SetReady(true) ran before manager.Start,
// so /readyz returned 200 immediately and /validate could be called against a
// not-yet-usable cache-backed client).
//
// Pointing the manager at an unreachable API server keeps its caches from ever
// syncing, so a correctly gated /readyz must never return 200, while /healthz
// still becomes OK because the HTTP server itself is up.
func TestServerReadinessGatedOnManager(t *testing.T) {
	dir := t.TempDir()
	kc := filepath.Join(dir, "kubeconfig")
	require.NoError(t, os.WriteFile(kc, []byte(unreachableKubeconfig), 0o600))
	t.Setenv("KUBECONFIG", kc)

	port := freePort(t)
	ctx := t.Context()

	r, err := New(ctx, &readinessTestProvider{},
		WithServer(server.ServerConfig{Port: port}),
		WithMetrics("0"),
	)
	require.NoError(t, err)

	go func() { _ = r.Start(ctx) }()

	base := fmt.Sprintf("http://127.0.0.1:%d", port)

	// The HTTP server comes up even though the manager cannot sync.
	require.Eventually(t, func() bool {
		return statusCode(base+"/healthz") == http.StatusOK
	}, 3*time.Second, 20*time.Millisecond, "server did not start listening")

	// Readiness must stay false for the whole observation window because the
	// manager's caches never sync.
	require.Never(t, func() bool {
		return statusCode(base+"/readyz") == http.StatusOK
	}, time.Second, 50*time.Millisecond, "/readyz reported ready before the manager was ready")
}
