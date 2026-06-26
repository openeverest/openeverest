// everest
// Copyright (C) 2025 Percona LLC
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

package telemetry

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/percona/everest/pkg/kubernetes"
)

// stubProbe lets tests script per-probe outcomes without touching K8s.
type stubProbe struct {
	name string
	out  map[string]string
	err  error
}

func (s stubProbe) Name() string { return s.name }
func (s stubProbe) Collect(_ context.Context, _ kubernetes.KubernetesConnector) (map[string]string, error) {
	return s.out, s.err
}

// stubTransport captures the last props sent and can be made to error.
type stubTransport struct {
	mu   sync.Mutex
	got  map[string]string
	err  error
	hits int
}

func (s *stubTransport) Send(_ context.Context, props map[string]string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hits++
	s.got = props
	return s.err
}

func (s *stubTransport) snapshot() (int, map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := make(map[string]string, len(s.got))
	for k, v := range s.got {
		cp[k] = v
	}
	return s.hits, cp
}

func newTestRunner(t *testing.T, transport Transport, probes []Probe) *Runner {
	t.Helper()
	// kubeConnector is nil because stub probes don't touch it and the only
	// always-on call is GetEverestID, which we tolerate failing.
	return NewRunner(nil, zap.NewNop().Sugar(), transport, time.Minute, probes)
}

func TestRunner_CollectAndSend_MergesProbeOutputs(t *testing.T) {
	tr := &stubTransport{}
	r := newTestRunner(t, tr, []Probe{
		stubProbe{name: "a", out: map[string]string{"foo": "1", "bar": "x"}},
		stubProbe{name: "b", out: map[string]string{"baz": "y"}},
	})

	r.collectAndSend(context.Background())

	hits, got := tr.snapshot()
	require.Equal(t, 1, hits)
	assert.Equal(t, "1", got["foo"])
	assert.Equal(t, "x", got["bar"])
	assert.Equal(t, "y", got["baz"])
	assert.Equal(t, "heartbeat", got["event"])
	assert.Equal(t, "1", got["schema_version"])
	assert.Equal(t, "dev", got["build_channel"], "default build channel should be dev in tests")
}

func TestRunner_CollectAndSend_ProbeFailureIsIsolated(t *testing.T) {
	tr := &stubTransport{}
	r := newTestRunner(t, tr, []Probe{
		stubProbe{name: "ok", out: map[string]string{"good": "yes"}},
		stubProbe{name: "boom", err: errors.New("kaboom")},
		stubProbe{name: "also_ok", out: map[string]string{"more": "data"}},
	})

	r.collectAndSend(context.Background())

	hits, got := tr.snapshot()
	require.Equal(t, 1, hits, "send should still occur when one probe fails")
	assert.Equal(t, "yes", got["good"])
	assert.Equal(t, "data", got["more"])
	// Always-on props must still be present.
	assert.Equal(t, "heartbeat", got["event"])
}

func TestRunner_CollectAndSend_AllProbesFailStillEmitsAlwaysOn(t *testing.T) {
	tr := &stubTransport{}
	r := newTestRunner(t, tr, []Probe{
		stubProbe{name: "p1", err: errors.New("fail1")},
		stubProbe{name: "p2", err: errors.New("fail2")},
	})

	r.collectAndSend(context.Background())

	hits, got := tr.snapshot()
	require.Equal(t, 1, hits)
	assert.Equal(t, "heartbeat", got["event"])
	assert.Equal(t, "1", got["schema_version"])
}

func TestRunner_CollectAndSend_TransportErrorDoesNotPanic(t *testing.T) {
	tr := &stubTransport{err: errors.New("network down")}
	r := newTestRunner(t, tr, []Probe{
		stubProbe{name: "p", out: map[string]string{"k": "v"}},
	})

	assert.NotPanics(t, func() {
		r.collectAndSend(context.Background())
	})
}

func TestNewRunner_NilProbesUsesDefaults(t *testing.T) {
	r := NewRunner(nil, zap.NewNop().Sugar(), &stubTransport{}, time.Minute, nil)
	require.NotNil(t, r)
	assert.NotEmpty(t, r.probes, "nil probes should fall back to defaults")
}
