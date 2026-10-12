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
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
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
			_, _ = w.Write([]byte(`{"sub":"oidc-subject-uuid","email":"user@example.com","name":"Example User","preferred_username":"euser"}`))
		}))
		defer srv.Close()

		info, err := FetchUserInfo(context.Background(), srv.URL, "opaque-random-string")
		require.NoError(t, err)
		assert.Equal(t, "oidc-subject-uuid", info.Subject)
		assert.Equal(t, "user@example.com", info.Email)
		assert.Equal(t, "Example User", info.Name)
		assert.Equal(t, "euser", info.PreferredUsername)
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

	t.Run("groups are returned", func(t *testing.T) {
		t.Parallel()
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"sub":"s","groups":["admins","devs"]}`))
		}))
		defer srv.Close()

		info, err := FetchUserInfo(context.Background(), srv.URL, "tok")
		require.NoError(t, err)
		assert.Equal(t, StringList{"admins", "devs"}, info.Groups)
	})
}

func TestStringListUnmarshal(t *testing.T) {
	t.Parallel()

	tcases := []struct {
		name string
		in   string
		out  StringList
	}{
		{name: "array", in: `{"groups":["a","b"]}`, out: StringList{"a", "b"}},
		{name: "single string", in: `{"groups":"a"}`, out: StringList{"a"}},
		{name: "non-string items are skipped", in: `{"groups":["a",{"id":1},2]}`, out: StringList{"a"}},
		{name: "unexpected shape is ignored", in: `{"groups":{"id":1}}`, out: nil},
		{name: "missing", in: `{}`, out: nil},
	}
	for _, tc := range tcases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var info UserInfo
			require.NoError(t, json.Unmarshal([]byte(tc.in), &info))
			assert.Equal(t, tc.out, info.Groups)
		})
	}
}

func TestUserInfoFromJWT(t *testing.T) {
	t.Parallel()

	idpKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	const kid = "idp-key"
	keyFunc := func(token *jwt.Token) (any, error) {
		if token.Header["kid"] != kid {
			return nil, errors.New("unknown kid")
		}
		return idpKey.Public(), nil
	}

	sign := func(t *testing.T, method jwt.SigningMethod, key any, claims jwt.MapClaims) string {
		t.Helper()
		token := jwt.NewWithClaims(method, claims)
		token.Header["kid"] = kid
		s, err := token.SignedString(key)
		require.NoError(t, err)
		return s
	}
	validClaims := func() jwt.MapClaims {
		return jwt.MapClaims{
			"iss":                "https://idp.example.com",
			"sub":                "oidc-subject",
			"exp":                time.Now().Add(time.Hour).Unix(),
			"email":              "user@example.com",
			"name":               "Example User",
			"preferred_username": "euser",
			"groups":             []string{"admins", "devs"},
		}
	}

	t.Run("provider-signed JWT returns its claims", func(t *testing.T) {
		t.Parallel()
		info, err := UserInfoFromJWT(sign(t, jwt.SigningMethodRS256, idpKey, validClaims()), keyFunc)
		require.NoError(t, err)
		assert.Equal(t, &UserInfo{
			Subject:           "oidc-subject",
			Email:             "user@example.com",
			Name:              "Example User",
			PreferredUsername: "euser",
			Groups:            StringList{"admins", "devs"},
		}, info)
	})

	t.Run("opaque token is rejected", func(t *testing.T) {
		t.Parallel()
		_, err := UserInfoFromJWT("opaque-random-string", keyFunc)
		require.Error(t, err)
	})

	t.Run("JWT signed by another key is rejected", func(t *testing.T) {
		t.Parallel()
		_, err := UserInfoFromJWT(sign(t, jwt.SigningMethodRS256, otherKey, validClaims()), keyFunc)
		require.Error(t, err)
	})

	t.Run("symmetric JWT is rejected", func(t *testing.T) {
		t.Parallel()
		_, err := UserInfoFromJWT(sign(t, jwt.SigningMethodHS256, []byte("client-secret"), validClaims()), keyFunc)
		require.Error(t, err)
	})

	t.Run("expired JWT is rejected", func(t *testing.T) {
		t.Parallel()
		claims := validClaims()
		claims["exp"] = time.Now().Add(-time.Minute).Unix()
		_, err := UserInfoFromJWT(sign(t, jwt.SigningMethodRS256, idpKey, claims), keyFunc)
		require.Error(t, err)
	})

	t.Run("JWT without exp is rejected", func(t *testing.T) {
		t.Parallel()
		claims := validClaims()
		delete(claims, "exp")
		_, err := UserInfoFromJWT(sign(t, jwt.SigningMethodRS256, idpKey, claims), keyFunc)
		require.Error(t, err)
	})

	t.Run("JWT without sub is rejected", func(t *testing.T) {
		t.Parallel()
		claims := validClaims()
		delete(claims, "sub")
		_, err := UserInfoFromJWT(sign(t, jwt.SigningMethodRS256, idpKey, claims), keyFunc)
		require.Error(t, err)
	})
}
