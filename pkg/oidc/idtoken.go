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
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// clockSkewLeeway tolerates clock drift between Everest and the OIDC provider when checking
// the exp, iat and nbf claims.
const clockSkewLeeway = time.Minute

// maxIDTokenAge bounds how long after issuance an ID token can be exchanged. The exchange is
// verified offline, so without this bound a held ID token could be exchanged again until it
// expires (10 hours by default on Auth0), keeping a user disabled in the provider signed in.
// Clients exchange the ID token right after login or a silent renewal, so it is always fresh.
const maxIDTokenAge = 5 * time.Minute

// rfc9068AccessTokenType is the JOSE "typ" header of JWT access tokens (RFC 9068 §2.1).
// It lets a provider-signed access token be told apart from an ID token. Media type names are
// case-insensitive and may carry the "application/" prefix (RFC 7515 §4.1.9).
const rfc9068AccessTokenType = "at+jwt"

// Identity contains the subset of ID token claims Everest reads from the OIDC provider.
// Name and PreferredUsername are only present when the `profile` scope was granted.
type Identity struct {
	Subject           string
	Email             string
	Name              string
	PreferredUsername string
	Groups            StringList
}

// StringList decodes a claim that IdPs emit either as a single string or as an array.
// Non-string array items are skipped, so an unexpected shape never fails the login.
type StringList []string

// UnmarshalJSON implements json.Unmarshaler.
func (s *StringList) UnmarshalJSON(data []byte) error {
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*s = nil
	switch v := raw.(type) {
	case string:
		*s = StringList{v}
	case []any:
		for _, item := range v {
			if str, ok := item.(string); ok {
				*s = append(*s, str)
			}
		}
	}
	return nil
}

// asymmetricSigningMethods are the algorithms accepted for provider-signed JWTs. Symmetric
// (HS*) tokens are signed with the client secret and can't be verified against the JWKS.
var asymmetricSigningMethods = []string{ //nolint:gochecknoglobals
	"RS256", "RS384", "RS512",
	"PS256", "PS384", "PS512",
	"ES256", "ES384", "ES512",
	"EdDSA",
}

//nolint:tagliatelle // azp/preferred_username are OIDC claim names, not Go-style identifiers.
type idTokenClaims struct {
	jwt.RegisteredClaims

	AuthorizedParty   string     `json:"azp"`
	Email             string     `json:"email"`
	Name              string     `json:"name"`
	PreferredUsername string     `json:"preferred_username"`
	Groups            StringList `json:"groups"`
}

// IDTokenVerifier validates ID tokens that the OIDC provider issued to the Everest client.
//
// Only ID tokens are accepted: they are the one OIDC artifact whose audience is guaranteed to
// be the client the user signed in to. Access tokens (JWT or opaque) and UserInfo responses
// don't prove which application they were issued to, so accepting them would let any other
// application of the same provider sign in to Everest as its users.
type IDTokenVerifier struct {
	keyFunc  jwt.Keyfunc
	clientID string
	issuers  []string
}

// NewIDTokenVerifier returns a verifier for ID tokens signed with a key from keyFunc, issued by
// one of issuers and addressed to clientID.
func NewIDTokenVerifier(keyFunc jwt.Keyfunc, clientID string, issuers ...string) (*IDTokenVerifier, error) {
	if keyFunc == nil {
		return nil, errors.New("key function is required")
	}
	if clientID == "" {
		return nil, errors.New("OIDC client ID is not configured")
	}
	issuers = slices.DeleteFunc(slices.Clone(issuers), func(iss string) bool { return iss == "" })
	if len(issuers) == 0 {
		return nil, errors.New("OIDC issuer is not configured")
	}
	return &IDTokenVerifier{keyFunc: keyFunc, clientID: clientID, issuers: slices.Compact(issuers)}, nil
}

// Verify validates rawIDToken as described in OpenID Connect Core 1.0 §3.1.3.7 and returns the
// identity it carries. Nonce validation is left to the frontend that started the login. The
// token must also have been issued within maxIDTokenAge (§3.1.3.7 step 10).
func (v *IDTokenVerifier) Verify(rawIDToken string) (*Identity, error) {
	var claims idTokenClaims
	token, err := jwt.ParseWithClaims(rawIDToken, &claims, v.keyFunc,
		jwt.WithValidMethods(asymmetricSigningMethods),
		jwt.WithExpirationRequired(),
		jwt.WithIssuedAt(),
		jwt.WithAudience(v.clientID),
		jwt.WithLeeway(clockSkewLeeway),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to verify ID token: %w", err)
	}

	typ, _ := token.Header["typ"].(string)
	if strings.TrimPrefix(strings.ToLower(typ), "application/") == rfc9068AccessTokenType {
		return nil, errors.New("access tokens are not accepted, an ID token is required")
	}
	if !slices.Contains(v.issuers, claims.Issuer) {
		return nil, fmt.Errorf("unexpected ID token issuer %q", claims.Issuer)
	}
	// An ID token addressed to several audiences must name the client it was issued to.
	if len(claims.Audience) > 1 && claims.AuthorizedParty == "" {
		return nil, errors.New("ID token with multiple audiences is missing the 'azp' claim")
	}
	if claims.AuthorizedParty != "" && claims.AuthorizedParty != v.clientID {
		return nil, fmt.Errorf("ID token was issued to another client %q", claims.AuthorizedParty)
	}
	if claims.Subject == "" {
		return nil, errors.New("ID token is missing the 'sub' claim")
	}
	if claims.IssuedAt == nil {
		return nil, errors.New("ID token is missing the 'iat' claim")
	}
	if age := time.Since(claims.IssuedAt.Time); age > maxIDTokenAge+clockSkewLeeway {
		return nil, fmt.Errorf(
			"ID token was issued %s ago, only tokens issued within %s are accepted",
			age.Round(time.Second), maxIDTokenAge,
		)
	}

	return &Identity{
		Subject:           claims.Subject,
		Email:             claims.Email,
		Name:              claims.Name,
		PreferredUsername: claims.PreferredUsername,
		Groups:            claims.Groups,
	}, nil
}
