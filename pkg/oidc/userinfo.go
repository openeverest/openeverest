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

package oidc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

// UserInfo contains the user information returned by the OIDC provider's UserInfo endpoint.
type UserInfo struct {
	Subject string `json:"sub"`
	Email   string `json:"email"`
}

// FetchUserInfo calls the OIDC provider's UserInfo endpoint with the given access token
// and returns the user information. This works with both JWT and opaque access tokens.
func FetchUserInfo(ctx context.Context, userInfoURL, accessToken string) (*UserInfo, error) {
	if userInfoURL == "" {
		return nil, errors.New("userinfo endpoint URL is empty")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, userInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create userinfo request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call userinfo endpoint: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read userinfo response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("userinfo endpoint returned %s: %s", resp.Status, body)
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
