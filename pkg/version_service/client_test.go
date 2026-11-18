// everest
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

package versionservice_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	versionservice "github.com/openeverest/openeverest/v2/pkg/version_service"
)

func TestGetSupportedEngineVersions(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, operator, engine, versions string
		want                             []string
		wantError                        bool
	}{
		{name: "pxc numeric order and 5.x exclusion", operator: versionservice.PXCOperatorName, engine: "pxc", versions: `"5.7.44":{},"8.0.10":{},"8.0.9":{},"8.4.1":{}`, want: []string{"8.0.9", "8.0.10", "8.4.1"}},
		{name: "mongodb numeric order", operator: versionservice.PSMDBOperatorName, engine: "mongod", versions: `"7.0.10":{},"7.0.9":{},"8.0.1":{}`, want: []string{"7.0.9", "7.0.10", "8.0.1"}},
		{name: "postgres numeric order", operator: versionservice.PGOperatorName, engine: "postgresql", versions: `"17.10":{},"17.9":{},"18.1":{}`, want: []string{"17.9", "17.10", "18.1"}},
		{name: "prereleases precede releases", operator: versionservice.PGOperatorName, engine: "postgresql", versions: `"18.0":{},"18.0-rc.2":{},"18.0-rc.10":{}`, want: []string{"18.0-rc.2", "18.0-rc.10", "18.0"}},
		{name: "equivalent versions have stable order", operator: versionservice.PGOperatorName, engine: "postgresql", versions: `"17.0":{},"17.0.0":{}`, want: []string{"17.0", "17.0.0"}},
		{name: "invalid version", operator: versionservice.PGOperatorName, engine: "postgresql", versions: `"invalid":{}`, wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/versions/v1/"+tc.operator+"/1.0.0" {
					t.Errorf("unexpected path: %s", r.URL.Path)
				}
				_, _ = fmt.Fprintf(w, `{"versions":[{"matrix":{"%s":{%s}}}]}`, tc.engine, tc.versions)
			}))
			defer server.Close()
			got, err := versionservice.New(server.URL).GetSupportedEngineVersions(t.Context(), tc.operator, "v1.0.0")
			if tc.wantError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}
