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
	"context"
	"fmt"
	"net/http"

	goversion "github.com/hashicorp/go-version"
)

// PMMServerVersion represents the version of PMM server.
type PMMServerVersion string

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

// getPMMVersion makes an API request to the PMM server to figure out the current version
func getPMMVersion(ctx context.Context, url string, auth iAuth, skipTLSVerify bool) (PMMServerVersion, error) {
	resp, err := doJSONRequest[struct {
		Version string `json:"version"`
	}](ctx, http.MethodGet, fmt.Sprintf("%s/v1/version", url), auth, "", skipTLSVerify)
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
