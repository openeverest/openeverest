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

package versionservice

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	perconavs "github.com/Percona-Lab/percona-version-service/versionpb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestGetSupportedEngineVersions_SortOrder(t *testing.T) {
	t.Parallel()

	// Deliberately includes a two-digit patch ("8.0.32") so a lexicographic
	// sort ("8.0.32" < "8.0.4") would misorder it against a semver sort.
	resp := &perconavs.VersionResponse{
		Versions: []*perconavs.OperatorVersion{
			{
				Matrix: &perconavs.VersionMatrix{
					Mongod: map[string]*perconavs.Version{
						"8.0.4":  {},
						"8.0.32": {},
						"8.0.9":  {},
					},
				},
			},
		},
	}
	b, err := protojson.Marshal(resp)
	require.NoError(t, err)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Add("content-type", "application/json")
		_, err := w.Write(b)
		assert.NoError(t, err)
	}))
	defer ts.Close()

	c := New(ts.URL)
	versions, err := c.GetSupportedEngineVersions(context.Background(), PSMDBOperatorName, "")
	require.NoError(t, err)

	assert.Equal(t, []string{"8.0.4", "8.0.9", "8.0.32"}, versions)
}
