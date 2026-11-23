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

package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/openeverest/openeverest/v2/contract"
	"github.com/openeverest/openeverest/v2/pkg/version"
)

func TestVersionInfo(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	ctx := echo.New().NewContext(httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/v1/version", nil), rec)
	require.NoError(t, (&EverestServer{}).VersionInfo(ctx))
	require.Equal(t, http.StatusOK, rec.Code)

	var got map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	upgradableFrom := make([]any, 0, len(contract.UpgradableFrom))
	for _, v := range contract.UpgradableFrom {
		upgradableFrom = append(upgradableFrom, v)
	}
	want := map[string]any{
		"projectName":        version.ProjectName,
		"version":            version.Version,
		"fullCommit":         version.FullCommit,
		"contractVersion":    contract.Version,
		"minContractVersion": contract.MinVersion,
		"upgradableFrom":     upgradableFrom,
	}
	assert.Equal(t, want, got)
}
