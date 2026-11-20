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
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"maps"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
			var claims idTokenClaims
			require.NoError(t, json.Unmarshal([]byte(tc.in), &claims))
			assert.Equal(t, tc.out, claims.Groups)
		})
	}
}

func TestNewIDTokenVerifier(t *testing.T) {
	t.Parallel()

	keyFunc := func(*jwt.Token) (any, error) { return nil, errors.New("unused") }

	_, err := NewIDTokenVerifier(keyFunc, "", "https://idp.example.com")
	require.Error(t, err, "client ID is required")

	_, err = NewIDTokenVerifier(keyFunc, "everest", "", "")
	require.Error(t, err, "an issuer is required")

	_, err = NewIDTokenVerifier(nil, "everest", "https://idp.example.com")
	require.Error(t, err, "a key function is required")
}

func TestIDTokenVerifier(t *testing.T) {
	t.Parallel()

	const (
		kid            = "idp-key"
		clientID       = "everest"
		issuer         = "https://idp.example.com/realms/everest"
		originalIssuer = "https://idp.example.com/realms/everest/"
	)

	idpKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	keyFunc := func(token *jwt.Token) (any, error) {
		if token.Header["kid"] != kid {
			return nil, errors.New("unknown kid")
		}
		return idpKey.Public(), nil
	}
	verifier, err := NewIDTokenVerifier(keyFunc, clientID, issuer, originalIssuer)
	require.NoError(t, err)

	sign := func(t *testing.T, method jwt.SigningMethod, key any, claims jwt.MapClaims, header map[string]any) string {
		t.Helper()
		token := jwt.NewWithClaims(method, claims)
		token.Header["kid"] = kid
		maps.Copy(token.Header, header)
		s, err := token.SignedString(key)
		require.NoError(t, err)
		return s
	}
	validClaims := func() jwt.MapClaims {
		now := time.Now()
		return jwt.MapClaims{
			"iss":                issuer,
			"aud":                clientID,
			"sub":                "oidc-subject",
			"iat":                now.Unix(),
			"exp":                now.Add(time.Hour).Unix(),
			"email":              "user@example.com",
			"name":               "Example User",
			"preferred_username": "euser",
			"groups":             []string{"admins", "devs"},
		}
	}
	with := func(overrides jwt.MapClaims) jwt.MapClaims {
		claims := validClaims()
		for k, v := range overrides {
			if v == nil {
				delete(claims, k)
				continue
			}
			claims[k] = v
		}
		return claims
	}

	t.Run("ID token issued to Everest returns its identity", func(t *testing.T) {
		t.Parallel()
		identity, err := verifier.Verify(sign(t, jwt.SigningMethodRS256, idpKey, validClaims(), nil))
		require.NoError(t, err)
		want := &Identity{
			Subject:           "oidc-subject",
			Email:             "user@example.com",
			Name:              "Example User",
			PreferredUsername: "euser",
			Groups:            StringList{"admins", "devs"},
		}
		assert.Equal(t, want, identity)
	})

	accepted := map[string]jwt.MapClaims{
		"configured issuer differing from the advertised one": with(jwt.MapClaims{"iss": originalIssuer}),
		"azp naming Everest":                         with(jwt.MapClaims{"azp": clientID}),
		"multiple audiences with azp naming Everest": with(jwt.MapClaims{"aud": []string{clientID, "api"}, "azp": clientID}),
		"iat slightly in the future (clock skew)":    with(jwt.MapClaims{"iat": time.Now().Add(30 * time.Second).Unix()}),
		"token issued just within the max age":       with(jwt.MapClaims{"iat": time.Now().Add(-maxIDTokenAge).Unix()}),
	}
	for name, claims := range accepted {
		t.Run("accepts "+name, func(t *testing.T) {
			t.Parallel()
			_, err := verifier.Verify(sign(t, jwt.SigningMethodRS256, idpKey, claims, nil))
			require.NoError(t, err)
		})
	}

	// Tokens that the same provider issued to other applications.
	rejected := map[string]jwt.MapClaims{
		"token issued to another client":      with(jwt.MapClaims{"aud": "other-app"}),
		"token without audience":              with(jwt.MapClaims{"aud": nil}),
		"token authorized for another client": with(jwt.MapClaims{"aud": []string{clientID, "other-app"}, "azp": "other-app"}),
		"multiple audiences without azp":      with(jwt.MapClaims{"aud": []string{clientID, "other-app"}}),
		"token from another issuer":           with(jwt.MapClaims{"iss": "https://evil.example.com"}),
		"token without issuer":                with(jwt.MapClaims{"iss": nil}),
		"expired token":                       with(jwt.MapClaims{"exp": time.Now().Add(-2 * time.Minute).Unix()}),
		"token without exp":                   with(jwt.MapClaims{"exp": nil}),
		"token issued in the future":          with(jwt.MapClaims{"iat": time.Now().Add(time.Hour).Unix()}),
		"token without sub":                   with(jwt.MapClaims{"sub": nil}),
		// A held ID token must not be exchangeable until it expires (hours on some providers).
		"unexpired token issued too long ago": with(jwt.MapClaims{
			"iat": time.Now().Add(-maxIDTokenAge - clockSkewLeeway - time.Minute).Unix(),
			"exp": time.Now().Add(10 * time.Hour).Unix(),
		}),
		"token without iat": with(jwt.MapClaims{"iat": nil}),
	}
	for name, claims := range rejected {
		t.Run("rejects "+name, func(t *testing.T) {
			t.Parallel()
			_, err := verifier.Verify(sign(t, jwt.SigningMethodRS256, idpKey, claims, nil))
			require.Error(t, err)
		})
	}

	t.Run("rejects RFC 9068 access token", func(t *testing.T) {
		t.Parallel()
		_, err := verifier.Verify(sign(t, jwt.SigningMethodRS256, idpKey, validClaims(), map[string]any{"typ": "at+jwt"}))
		require.Error(t, err)
	})

	t.Run("rejects token signed by another key", func(t *testing.T) {
		t.Parallel()
		_, err := verifier.Verify(sign(t, jwt.SigningMethodRS256, otherKey, validClaims(), nil))
		require.Error(t, err)
	})

	t.Run("rejects symmetric token", func(t *testing.T) {
		t.Parallel()
		_, err := verifier.Verify(sign(t, jwt.SigningMethodHS256, []byte("client-secret"), validClaims(), nil))
		require.Error(t, err)
	})

	t.Run("rejects opaque token", func(t *testing.T) {
		t.Parallel()
		_, err := verifier.Verify("opaque-random-string")
		require.Error(t, err)
	})
}
