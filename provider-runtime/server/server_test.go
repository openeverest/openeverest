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

package server

import (
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// freePort returns an available localhost TCP port.
func freePort(t *testing.T) int {
	t.Helper()
	l, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer l.Close() //nolint:errcheck
	addr, ok := l.Addr().(*net.TCPAddr)
	require.True(t, ok)
	return addr.Port
}

// statusCode issues a GET and returns the status code, or -1 if the request
// could not be made (e.g. the listener is not up yet).
func statusCode(url string) int {
	resp, err := http.Get(url) //nolint:noctx,gosec
	if err != nil {
		return -1
	}
	defer resp.Body.Close() //nolint:errcheck
	return resp.StatusCode
}

// TestServerReadinessEndpoint verifies that /readyz reflects the readiness flag
// set via SetReady, while /healthz is always OK once the server is listening.
func TestServerReadinessEndpoint(t *testing.T) {
	t.Parallel()

	port := freePort(t)
	srv := NewServer(ServerConfig{Port: port}, nil)

	ctx := t.Context()
	go func() { _ = srv.Start(ctx) }()

	base := fmt.Sprintf("http://127.0.0.1:%d", port)

	// Wait for the listener to come up. /healthz is independent of readiness.
	require.Eventually(t, func() bool {
		return statusCode(base+"/healthz") == http.StatusOK
	}, 3*time.Second, 20*time.Millisecond, "server did not start listening")

	// Before SetReady(true): server is healthy but not ready.
	assert.Equal(t, http.StatusServiceUnavailable, statusCode(base+"/readyz"))
	assert.Equal(t, http.StatusOK, statusCode(base+"/healthz"))

	// After SetReady(true): ready.
	srv.SetReady(true)
	assert.Equal(t, http.StatusOK, statusCode(base+"/readyz"))

	// After SetReady(false): not ready again.
	srv.SetReady(false)
	assert.Equal(t, http.StatusServiceUnavailable, statusCode(base+"/readyz"))
}
