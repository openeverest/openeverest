// everest
// Copyright (C) 2023 Percona LLC
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

package versionservice

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// slowServer returns a test server that delays every response by the given duration.
func slowServer(delay time.Duration) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(delay)
		w.WriteHeader(http.StatusOK)
	}))
}

func TestNew_HasTimeout(t *testing.T) {
	t.Parallel()
	c := New("http://example.com").(*versionServiceClient)
	if c.httpClient == nil {
		t.Fatal("httpClient must not be nil")
	}
	if c.httpClient.Timeout != defaultHTTPTimeout {
		t.Fatalf("expected timeout %v, got %v", defaultHTTPTimeout, c.httpClient.Timeout)
	}
}

func TestGetEverestMetadata_RespectsClientTimeout(t *testing.T) {
	t.Parallel()

	// Server responds slower than the client timeout we set below.
	srv := slowServer(200 * time.Millisecond)
	defer srv.Close()

	// Create a client whose HTTP timeout is shorter than the server delay.
	c := &versionServiceClient{
		url:        srv.URL,
		httpClient: &http.Client{Timeout: 50 * time.Millisecond},
	}

	_, err := c.GetEverestMetadata(context.Background())
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	// The error wraps a context deadline / timeout error from net/http.
	if !strings.Contains(err.Error(), "could not retrieve Everest metadata") {
		t.Fatalf("unexpected error text: %v", err)
	}
}

func TestGetSupportedEngineVersions_RespectsClientTimeout(t *testing.T) {
	t.Parallel()

	srv := slowServer(200 * time.Millisecond)
	defer srv.Close()

	c := &versionServiceClient{
		url:        srv.URL,
		httpClient: &http.Client{Timeout: 50 * time.Millisecond},
	}

	_, err := c.GetSupportedEngineVersions(context.Background(), PXCOperatorName, "1.0.0")
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "could not retrieve version response") {
		t.Fatalf("unexpected error text: %v", err)
	}
}
