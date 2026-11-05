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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
)

// UserInfo contains the subset of identity claims Everest reads from the OIDC provider,
// either from its UserInfo endpoint or from a verified JWT access token.
// Name and PreferredUsername are only present when the `profile` scope was granted.
//
//nolint:tagliatelle // preferred_username is an OIDC claim name, not a Go-style identifier.
type UserInfo struct {
	Subject           string     `json:"sub"`
	Email             string     `json:"email"`
	Name              string     `json:"name"`
	PreferredUsername string     `json:"preferred_username"`
	Groups            StringList `json:"groups"`
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

// UserInfoFromJWT verifies a JWT access token against the provider's signing keys and returns
// the identity claims it carries. It fails for opaque tokens and for JWTs that can't be verified
// locally; callers fall back to FetchUserInfo in that case.
func UserInfoFromJWT(accessToken string, keyFunc jwt.Keyfunc) (*UserInfo, error) {
	token, err := jwt.Parse(accessToken, keyFunc,
		jwt.WithValidMethods(asymmetricSigningMethods),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to verify JWT: %w", err)
	}

	// Round-trip through JSON so the claims are decoded exactly like a UserInfo response.
	raw, err := json.Marshal(token.Claims)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JWT claims: %w", err)
	}
	var info UserInfo
	if err := json.Unmarshal(raw, &info); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JWT claims: %w", err)
	}
	if info.Subject == "" {
		return nil, errors.New("JWT missing 'sub' claim")
	}

	return &info, nil
}

// FetchUserInfo calls the OIDC provider's UserInfo endpoint with the given access token
// and returns the user information. It works with both JWT and opaque access tokens, since
// the provider (not Everest) is responsible for interpreting the token.
func FetchUserInfo(ctx context.Context, userInfoURL, accessToken string) (*UserInfo, error) {
	if userInfoURL == "" {
		return nil, errors.New("userinfo endpoint URL is empty")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, userInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create userinfo request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{Timeout: defaultHTTPClientTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call userinfo endpoint: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read userinfo response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %s: %s", ErrUnexpectedSatusCode, resp.Status, body)
	}

	var info UserInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("failed to unmarshal userinfo response: %w", err)
	}

	if info.Subject == "" {
		return nil, errors.New("userinfo response missing 'sub' claim")
	}

	return &info, nil
}
