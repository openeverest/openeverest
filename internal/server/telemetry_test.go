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

package server

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func observedLogger() (*zap.SugaredLogger, *observer.ObservedLogs) {
	core, logs := observer.New(zapcore.DebugLevel)
	return zap.New(core).Sugar(), logs
}

func TestPostTelemetryPayload(t *testing.T) {
	t.Parallel()

	t.Run("non-OK response returns an error and logs a truncated body", func(t *testing.T) {
		t.Parallel()

		// Longer than the read limit, so the truncation path is exercised.
		body := strings.Repeat("x", telemetryMaxErrorBodyLogBytes*2)
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(body))
		}))
		defer ts.Close()

		l, logs := observedLogger()
		err := postTelemetryPayload(context.Background(), ts.Client(), ts.URL, []byte(`{}`), l)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "500")

		truncated := logs.FilterMessage("telemetry error response body truncated; original response is longer")
		assert.Equal(t, 1, truncated.Len(), "expected a truncation warning")

		nonOK := logs.FilterMessage("telemetry non-OK response").All()
		require.Len(t, nonOK, 1)
		fields := nonOK[0].ContextMap()
		assert.Equal(t, int64(http.StatusInternalServerError), fields["status"])
		snippet, ok := fields["bodySnippet"].(string)
		require.True(t, ok, "bodySnippet should be a string")
		assert.Len(t, snippet, telemetryMaxErrorBodyLogBytes, "body should be capped at the read limit")
	})

	t.Run("non-OK response with a short body is not reported as truncated", func(t *testing.T) {
		t.Parallel()

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("bad payload"))
		}))
		defer ts.Close()

		l, logs := observedLogger()
		err := postTelemetryPayload(context.Background(), ts.Client(), ts.URL, []byte(`{}`), l)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "400")
		assert.Equal(t, 0, logs.FilterMessageSnippet("truncated").Len())

		nonOK := logs.FilterMessage("telemetry non-OK response").All()
		require.Len(t, nonOK, 1)
		assert.Equal(t, "bad payload", nonOK[0].ContextMap()["bodySnippet"])
	})

	t.Run("OK response returns no error and logs nothing", func(t *testing.T) {
		t.Parallel()

		var gotPath, gotContentType string
		var gotBody []byte
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotPath = r.URL.Path
			gotContentType = r.Header.Get("Content-Type")
			gotBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"ok":true}`))
		}))
		defer ts.Close()

		l, logs := observedLogger()
		err := postTelemetryPayload(context.Background(), ts.Client(), ts.URL, []byte(`{"reports":[]}`), l)

		require.NoError(t, err)
		assert.Equal(t, "/v1/telemetry/GenericReport", gotPath)
		assert.Equal(t, "application/json", gotContentType)
		assert.JSONEq(t, `{"reports":[]}`, string(gotBody))
		assert.Equal(t, 0, logs.Len(), "a successful report should not log")
	})

	t.Run("transport error is returned to the caller", func(t *testing.T) {
		t.Parallel()

		// Closed immediately, so the connection is refused.
		ts := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
		url := ts.URL
		client := ts.Client()
		ts.Close()

		l, _ := observedLogger()
		err := postTelemetryPayload(context.Background(), client, url, []byte(`{}`), l)
		require.Error(t, err)
	})
}

func TestDefaultTelemetryHTTPClientHasTimeout(t *testing.T) {
	t.Parallel()

	assert.Equal(t, telemetryHTTPTimeout, defaultTelemetryHTTPClient.Timeout,
		"telemetry must not use an unbounded client")
	assert.Positive(t, defaultTelemetryHTTPClient.Timeout)
}
