// everest
// Copyright (C) 2023 Percona LLC
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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetSupportedEngineVersions_NilMatrix ensures that a Version Service
// response with a missing/omitted matrix returns a clear error instead of
// panicking with a nil pointer dereference. See issue #2593.
func TestGetSupportedEngineVersions_NilMatrix(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"versions": [{}]}`))
	}))
	defer srv.Close()

	c := New(srv.URL)

	require.NotPanics(t, func() {
		versions, err := c.GetSupportedEngineVersions(context.Background(), PXCOperatorName, "1.0.0")
		require.Error(t, err)
		assert.Nil(t, versions)
	})
}

// TestGetSupportedEngineVersions_UnsupportedOperator ensures that an
// unrecognized operator name returns an explicit error instead of silently
// succeeding with an empty result.
func TestGetSupportedEngineVersions_UnsupportedOperator(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"versions": [{"matrix": {"pxc": {"1.0.0": {}}}}]}`))
	}))
	defer srv.Close()

	c := New(srv.URL)

	versions, err := c.GetSupportedEngineVersions(context.Background(), "not-a-real-operator", "1.0.0")
	require.Error(t, err)
	assert.Nil(t, versions)
}
