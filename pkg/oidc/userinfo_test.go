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

package oidc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchUserInfo(t *testing.T) {
	t.Parallel()

	t.Run("valid opaque token returns claims", func(t *testing.T) {
		t.Parallel()
		var gotAuth string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotAuth = r.Header.Get("Authorization")
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"sub":"oidc-subject-uuid","email":"user@example.com"}`))
		}))
		defer srv.Close()

		info, err := FetchUserInfo(context.Background(), srv.URL, "opaque-random-string")
		require.NoError(t, err)
		assert.Equal(t, "oidc-subject-uuid", info.Subject)
		assert.Equal(t, "user@example.com", info.Email)
		// The opaque token must be forwarded as a bearer credential to the IdP.
		assert.Equal(t, "Bearer opaque-random-string", gotAuth)
	})

	t.Run("empty url errors", func(t *testing.T) {
		t.Parallel()
		_, err := FetchUserInfo(context.Background(), "", "tok")
		require.Error(t, err)
	})

	t.Run("non-200 response errors", func(t *testing.T) {
		t.Parallel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":"invalid_token"}`))
		}))
		defer srv.Close()

		_, err := FetchUserInfo(context.Background(), srv.URL, "bad-token")
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrUnexpectedSatusCode)
	})

	t.Run("missing sub errors", func(t *testing.T) {
		t.Parallel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"email":"user@example.com"}`))
		}))
		defer srv.Close()

		_, err := FetchUserInfo(context.Background(), srv.URL, "tok")
		require.Error(t, err)
	})

	t.Run("invalid json errors", func(t *testing.T) {
		t.Parallel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`not-json`))
		}))
		defer srv.Close()

		_, err := FetchUserInfo(context.Background(), srv.URL, "tok")
		require.Error(t, err)
	})
}
