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

// Package oidc provides functionality related OIDC based IDPs.
package oidc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/golang-jwt/jwt/v5"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

// ProviderConfig contains the configuration of an OIDC provider.
//
//nolint:tagliatelle
type ProviderConfig struct {
	OriginalIssuer string   `json:"OriginalIssuer"`
	Issuer         string   `json:"issuer"`
	AuthURL        string   `json:"authorization_endpoint"`
	TokenURL       string   `json:"token_endpoint"`
	DeviceAuthURL  string   `json:"device_authorization_endpoint"`
	JWKSURL        string   `json:"jwks_uri"`
	UserInfoURL    string   `json:"userinfo_endpoint"`
	Algorithms     []string `json:"id_token_signing_alg_values_supported"`
}

const (
	// WellKnownPath is the path to the well-known OIDC configuration.
	WellKnownPath = "/.well-known/openid-configuration"
)

// ErrUnexpectedSatusCode is returned when HTTP 200 is not returned.
var ErrUnexpectedSatusCode = fmt.Errorf("unexpected status code")

// NewProviderConfig fetches the OIDC provider configuration from the issuer's
// well-known endpoint and returns it as a ProviderConfig.
func NewProviderConfig(ctx context.Context, issuer string) (ProviderConfig, error) {
	wellKnown, err := url.JoinPath(issuer, WellKnownPath)
	if err != nil {
		return ProviderConfig{}, errors.Join(err, errors.New("failed to join issuer url and well-known path"))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, wellKnown, nil)
	if err != nil {
		return ProviderConfig{}, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ProviderConfig{}, err
	}
	defer resp.Body.Close() //nolint:errcheck

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ProviderConfig{}, fmt.Errorf("unable to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return ProviderConfig{}, errors.Join(ErrUnexpectedSatusCode, fmt.Errorf("%s: %s", resp.Status, body))
	}

	var result ProviderConfig
	if err := json.Unmarshal(body, &result); err != nil {
		return ProviderConfig{}, fmt.Errorf("failed to unmarshal json response: %w", err)
	}

	// It appears that issuerUrl provided by user is not always
	// the same as the one fetched from the OIDC provider's .well-known/openid-configuration (Microsoft Entra case).
	// Need to store the original issuerUrl too.
	result.OriginalIssuer = issuer
	return result, nil
}

// NewKeyFunc returns a new function for getting the public JWK keys
// from the OIDC provider at the given issuer URL.
func (c *ProviderConfig) NewKeyFunc(ctx context.Context) (jwt.Keyfunc, error) {
	if c.JWKSURL == "" {
		return nil, errors.New("did not find jwks_uri in oidc config")
	}

	keyCache := jwk.NewCache(ctx)
	if err := keyCache.Register(c.JWKSURL); err != nil {
		return nil, errors.Join(err, errors.New("failed to register jwk cache"))
	}

	return func(token *jwt.Token) (any, error) {
		keySet, err := keyCache.Get(ctx, c.JWKSURL)
		if err != nil {
			return nil, err
		}

		keyID, ok := token.Header["kid"].(string)
		if !ok {
			return nil, errors.New("expecting JWT header to have a key ID in the kid field")
		}

		key, found := keySet.LookupKeyID(keyID)
		if !found {
			return nil, fmt.Errorf("unable to find key %q", keyID)
		}

		var pubkey any
		if err := key.Raw(&pubkey); err != nil {
			return nil, errors.Join(err, errors.New("failed to get the public key"))
		}
		return pubkey, nil
	}, nil
}
