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
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScarfTransport_SendsAllPropsAsQueryParams(t *testing.T) {
	var (
		mu    sync.Mutex
		query url.Values
		hits  int
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		hits++
		query = r.URL.Query()
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tr := NewScarfTransport(srv.URL)
	err := tr.Send(context.Background(), map[string]string{
		"event":           "heartbeat",
		"schema_version":  "1",
		"everest_version": "v0.0.0-test",
	})
	require.NoError(t, err)

	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, 1, hits)
	assert.Equal(t, "heartbeat", query.Get("event"))
	assert.Equal(t, "1", query.Get("schema_version"))
	assert.Equal(t, "v0.0.0-test", query.Get("everest_version"))
}

func TestScarfTransport_RespectsEnvOptOut(t *testing.T) {
	t.Setenv("DO_NOT_TRACK", "1")

	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits++
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tr := NewScarfTransport(srv.URL)
	// Scarf SDK returns a non-nil sentinel error when opted out; we only
	// require that no request is sent and no panic occurs.
	assert.NotPanics(t, func() {
		_ = tr.Send(context.Background(), map[string]string{"event": "heartbeat"})
	})
	assert.Equal(t, 0, hits, "no request should be sent when opted out via env")
}

func TestScarfTransport_HandlesNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	tr := NewScarfTransport(srv.URL)
	// We only assert that an error is returned, not its exact shape; the
	// orchestrator logs and continues regardless.
	err := tr.Send(context.Background(), map[string]string{"event": "heartbeat"})
	assert.Error(t, err)
}

// TestScarfTransport_IgnoresScarfNoAnalyticsEnv guards against the underlying
// SDK leaking a vendor-specific opt-out env var into our public surface.
// Only DO_NOT_TRACK and our own DISABLE_TELEMETRY should disable telemetry.
func TestScarfTransport_IgnoresScarfNoAnalyticsEnv(t *testing.T) {
	t.Setenv("SCARF_NO_ANALYTICS", "1")

	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits++
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tr := NewScarfTransport(srv.URL)
	err := tr.Send(context.Background(), map[string]string{"event": "heartbeat"})
	assert.NoError(t, err, "SCARF_NO_ANALYTICS must not disable our transport")
	assert.Equal(t, 1, hits, "request must still be sent despite SCARF_NO_ANALYTICS=1")
}
