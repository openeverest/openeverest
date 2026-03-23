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

// Package pmm provides utilities to interact with PMM API.
package pmm

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	goversion "github.com/hashicorp/go-version"
)

// PMMServerVersion represents the version of PMM server.
type PMMServerVersion string

// pmmErrorMessage represents the error message returned by PMM API when an error occurs.
type pmmErrorMessage struct {
	Message string `json:"message"`
}

// versionResponse represents the response from PMM when requesting the version.
type versionResponse struct {
	Version string `json:"version"`
}

// iAuth an interface to apply auth to a request.
type iAuth interface {
	apply(req *http.Request)
}

// basicAuth represents basic auth with User/Password
type basicAuth struct {
	user     string
	password string
}

func (a basicAuth) apply(req *http.Request) {
	req.SetBasicAuth(a.user, a.password)
}

// bearerAuth represents bearer auth with a token
type bearerAuth struct {
	token string
}

func (a bearerAuth) apply(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+a.token)
}

// GetPMMServerVersion gets the PMM server version using the provided credentials secret.
func GetPMMServerVersion(ctx context.Context, url string, token string, skipVerifyTLS bool) (PMMServerVersion, error) {
	return getPMMVersion(ctx, url, bearerAuth{token: token}, skipVerifyTLS)
}

// getPMMVersion makes an API request to the PMM server to figure out the current version
func getPMMVersion(ctx context.Context, baseURL string, auth iAuth, skipTLSVerify bool) (PMMServerVersion, error) {
	url := fmt.Sprintf("%s/v1/version", baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}

	resp, err := doJSONRequest[versionResponse](req, auth, skipTLSVerify)
	if err != nil {
		return "", err
	}

	return PMMServerVersion(resp.Version), nil
}

// isLegacyAuth returns true if the instance uses legacy auth (PMM2) otherwise it returns false
func isLegacyAuth(version PMMServerVersion) bool {
	ver, err := goversion.NewVersion(string(version))
	if err != nil {
		return false
	}
	segments := ver.Segments()
	return len(segments) > 0 && segments[0] == 2
}

// postJSONRequest makes an HTTP POST request with a JSON body and returns the decoded response.
func postJSONRequest[T any](ctx context.Context, url string, auth iAuth, body any, skipTLSVerify bool) (T, error) {
	var zero T
	b, err := json.Marshal(body)
	if err != nil {
		return zero, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return zero, fmt.Errorf("build request: %w", err)
	}

	return doJSONRequest[T](req, auth, skipTLSVerify)
}

// doJSONRequest sets HTTP headers and makes a request and decodes the response.
func doJSONRequest[T any](req *http.Request, auth iAuth, skipTLSVerify bool) (T, error) {
	var zero T

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	if auth != nil {
		auth.apply(req)
	}

	req.Close = true

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: skipTLSVerify,
			},
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return zero, fmt.Errorf("do request: %w", err)
	}

	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return zero, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		var pmmErr *pmmErrorMessage
		if err := json.Unmarshal(data, &pmmErr); err != nil {
			return zero, errors.Join(err, fmt.Errorf("PMM returned an unknown error. HTTP %d", resp.StatusCode))
		}

		var errMsg string
		if pmmErr != nil {
			errMsg = pmmErr.Message
		}

		return zero, fmt.Errorf("PMM returned an error: %s", errMsg)
	}

	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		return zero, fmt.Errorf("unmarshal response: %w", err)
	}

	return result, nil
}
