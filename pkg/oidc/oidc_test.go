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

package oidc_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/stretchr/testify/require"

	"github.com/percona/everest/pkg/oidc"
)

// wellKnownServer starts an httptest server that serves the given body as
// JSON at oidc.WellKnownPath, and anything else with the given status code.
func wellKnownServer(t *testing.T, status int, body any) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc(oidc.WellKnownPath, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		if body != nil {
			require.NoError(t, json.NewEncoder(w).Encode(body))
		}
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestNewProviderConfig_OriginalIssuerDiffersFromResponseIssuer(t *testing.T) {
	t.Parallel()

	// This models the Microsoft Entra case: the issuer the admin configures
	// (the server URL below) is not the same as the issuer advertised in the
	// well-known document. internal/server/middlewares.go only adds the
	// second CSP connect-src entry when OriginalIssuer != Issuer, so this is
	// the behavior most worth locking down.
	const advertisedIssuer = "https://login.microsoftonline.com/some-tenant-id/v2.0"

	srv := wellKnownServer(t, http.StatusOK, map[string]any{
		"issuer":                 advertisedIssuer,
		"authorization_endpoint": advertisedIssuer + "/oauth2/v2.0/authorize",
		"token_endpoint":         advertisedIssuer + "/oauth2/v2.0/token",
		"jwks_uri":               advertisedIssuer + "/discovery/v2.0/keys",
	})

	cfg, err := oidc.NewProviderConfig(context.Background(), srv.URL)
	require.NoError(t, err)

	// OriginalIssuer must be the URL the caller passed in...
	require.Equal(t, srv.URL, cfg.OriginalIssuer)
	// ...while Issuer must be whatever the provider's own document advertised.
	require.Equal(t, advertisedIssuer, cfg.Issuer)
	require.NotEqual(t, cfg.OriginalIssuer, cfg.Issuer)
}

func TestNewProviderConfig_OriginalIssuerMatchesWhenProviderAgrees(t *testing.T) {
	t.Parallel()

	// The common case: the provider's well-known document echoes back the
	// same issuer the caller configured. OriginalIssuer and Issuer should
	// then be equal, so middlewares.go does not add a redundant CSP entry.
	// The handler needs to know its own URL to echo it back as "issuer",
	// which isn't available until after the server starts, so build the
	// server directly here instead of via the wellKnownServer helper.
	mux := http.NewServeMux()
	mux.HandleFunc(oidc.WellKnownPath, func(w http.ResponseWriter, r *http.Request) {
		issuer := "http://" + r.Host
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer":   issuer,
			"jwks_uri": issuer + "/keys",
		})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	cfg, err := oidc.NewProviderConfig(context.Background(), srv.URL)
	require.NoError(t, err)
	require.Equal(t, srv.URL, cfg.OriginalIssuer)
	require.Equal(t, cfg.OriginalIssuer, cfg.Issuer)
}

func TestNewProviderConfig_NonOKStatusReturnsUnexpectedStatusCode(t *testing.T) {
	t.Parallel()

	srv := wellKnownServer(t, http.StatusInternalServerError, nil)

	_, err := oidc.NewProviderConfig(context.Background(), srv.URL)
	require.Error(t, err)
	// getOIDCProviderConfigureSteps (pkg/oidc/configure.go) branches on this
	// specific sentinel to give the user a distinct message, so the wrapping
	// must be preserved.
	require.ErrorIs(t, err, oidc.ErrUnexpectedSatusCode)
}

func TestNewProviderConfig_UnreachableIssuerReturnsError(t *testing.T) {
	t.Parallel()

	// Nothing is listening on this address.
	_, err := oidc.NewProviderConfig(context.Background(), "http://127.0.0.1:0")
	require.Error(t, err)
	require.NotErrorIs(t, err, oidc.ErrUnexpectedSatusCode)
}

// ---- NewKeyFunc ----

func TestNewKeyFunc_NoJWKSURI(t *testing.T) {
	t.Parallel()

	cfg := &oidc.ProviderConfig{} // JWKSURL left empty
	_, err := cfg.NewKeyFunc(context.Background())
	require.Error(t, err)
	require.Contains(t, err.Error(), "jwks_uri")
}

func TestNewKeyFunc_MalformedJWKSURLFailsOnFirstUse(t *testing.T) {
	t.Parallel()

	// httprc's queue.Register (v1.0.6, jwk.Cache's underlying implementation)
	// does not parse or otherwise validate the URL string at all -- it is
	// simply stored as a map key, and Register() only returns an error for
	// unrelated reasons (e.g. an explicit RefreshInterval option smaller than
	// the cache's refresh window). Since NewKeyFunc calls Register with no
	// options, registration cannot fail for a malformed URL in this version.
	// The failure instead surfaces on first use, when the returned Keyfunc
	// calls keyCache.Get and the queue actually attempts to build/issue the
	// HTTP request for the malformed URL.
	cfg := &oidc.ProviderConfig{JWKSURL: "://not-a-valid-url"}

	keyFunc, err := cfg.NewKeyFunc(context.Background())
	require.NoError(t, err, "registration itself does not validate the URL")

	tok := &jwt.Token{Header: map[string]any{"kid": "any"}}
	_, err = keyFunc(tok)
	require.Error(t, err, "fetching the malformed URL should fail on first use")
}

// rsaJWKS starts a server that publishes a single RSA public key under the
// given kid, and returns the private key so tests can build tokens.
func rsaJWKS(t *testing.T, kid string) (*httptest.Server, *rsa.PrivateKey) {
	t.Helper()

	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	pubJWK, err := jwk.FromRaw(priv.Public())
	require.NoError(t, err)
	require.NoError(t, pubJWK.Set(jwk.KeyIDKey, kid))
	require.NoError(t, pubJWK.Set(jwk.AlgorithmKey, "RS256"))

	set := jwk.NewSet()
	require.NoError(t, set.AddKey(pubJWK))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		require.NoError(t, json.NewEncoder(w).Encode(set))
	}))
	t.Cleanup(srv.Close)
	return srv, priv
}

func TestNewKeyFunc_TokenMissingKid(t *testing.T) {
	t.Parallel()

	srv, _ := rsaJWKS(t, "key-1")
	cfg := &oidc.ProviderConfig{JWKSURL: srv.URL}

	keyFunc, err := cfg.NewKeyFunc(context.Background())
	require.NoError(t, err)

	tok := &jwt.Token{Header: map[string]any{}} // no "kid"
	_, err = keyFunc(tok)
	require.Error(t, err)
	require.Contains(t, err.Error(), "kid")
}

func TestNewKeyFunc_KidNotInKeySet(t *testing.T) {
	t.Parallel()

	srv, _ := rsaJWKS(t, "key-1")
	cfg := &oidc.ProviderConfig{JWKSURL: srv.URL}

	keyFunc, err := cfg.NewKeyFunc(context.Background())
	require.NoError(t, err)

	tok := &jwt.Token{Header: map[string]any{"kid": "does-not-exist"}}
	_, err = keyFunc(tok)
	require.Error(t, err)
	require.Contains(t, err.Error(), "unable to find key")
}

func TestNewKeyFunc_HappyPath(t *testing.T) {
	t.Parallel()

	srv, priv := rsaJWKS(t, "key-1")
	cfg := &oidc.ProviderConfig{JWKSURL: srv.URL}

	keyFunc, err := cfg.NewKeyFunc(context.Background())
	require.NoError(t, err)

	tok := &jwt.Token{Header: map[string]any{"kid": "key-1"}}
	got, err := keyFunc(tok)
	require.NoError(t, err)

	gotPub, ok := got.(*rsa.PublicKey)
	require.True(t, ok, "expected *rsa.PublicKey, got %T", got)
	require.True(t, priv.PublicKey.Equal(gotPub))
}

func TestNewKeyFunc_KeySetFailsToUnmarshal(t *testing.T) {
	t.Parallel()

	// Serve a JWKS document where the one key's "n" field is not valid
	// base64url, so the whole set fails to parse when the cache fetches it.
	// This surfaces at keyCache.Get() inside the returned jwt.Keyfunc, distinct
	// from the "kid not found" and "missing kid" cases above.
	malformed := `{"keys":[{"kty":"RSA","kid":"key-1","alg":"RS256","n":"not-valid-base64!!!","e":"AQAB"}]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, malformed)
	}))
	t.Cleanup(srv.Close)

	cfg := &oidc.ProviderConfig{JWKSURL: srv.URL}
	keyFunc, err := cfg.NewKeyFunc(context.Background())
	require.NoError(t, err) // registration itself succeeds; the URL is valid

	tok := &jwt.Token{Header: map[string]any{"kid": "key-1"}}
	_, err = keyFunc(tok)
	require.Error(t, err)
}
