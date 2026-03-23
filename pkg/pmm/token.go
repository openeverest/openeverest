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

package pmm

import (
	"context"
	"fmt"
)

// keysResponse represents the response from PMM when creating an API key in PMM2.
type keysResponse struct {
	Key string `json:"key"`
}

// serviceAccountsResponse represents the response from PMM when creating a service account in PMM3.
type serviceAccountsResponse struct {
	Uid string `json:"uid"`
}

// tokensResponse represents the response from PMM when creating a token in PMM3.
type tokensResponse struct {
	Key string `json:"key"`
}

// CreateAPIKey creates an API key in PMM and returns its value.
// It automatically detects the PMM version and uses the appropriate API to create the key.
func CreateAPIKey(ctx context.Context, urlBase, apiKeyName, user, password string, skipTLSVerify bool) (string, error) {
	auth := basicAuth{
		user:     user,
		password: password,
	}

	version, err := getPMMVersion(ctx, urlBase, auth, skipTLSVerify)
	if err != nil {
		return "", err
	}

	payload := map[string]string{
		"name": apiKeyName,
		"role": "Admin",
	}

	if isLegacyAuth(version) {
		// for PMM2, create a key directly
		url := fmt.Sprintf("%s/graph/api/auth/keys", urlBase)
		resp, err := postJSONRequest[keysResponse](ctx, url, auth, payload, skipTLSVerify)
		if err != nil {
			return "", err
		}

		return resp.Key, nil
	}

	// for PMM3, create a service account and then create a token for that account
	url := fmt.Sprintf("%s/graph/api/serviceaccounts", urlBase)
	account, err := postJSONRequest[serviceAccountsResponse](ctx, url, auth, payload, skipTLSVerify)
	if err != nil {
		return "", err
	}

	url = fmt.Sprintf("%s/graph/api/serviceaccounts/%s/tokens", urlBase, account.Uid)
	token, err := postJSONRequest[tokensResponse](ctx, url, auth, payload, skipTLSVerify)
	if err != nil {
		return "", err
	}

	return token.Key, nil
}
