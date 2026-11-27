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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	perconavs "github.com/Percona-Lab/percona-version-service/versionpb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestNew(t *testing.T) {
	t.Parallel()
	client := New("https://check.percona.com")
	require.NotNil(t, client)
}

func TestGetSupportedEngineVersions(t *testing.T) {
	t.Parallel()

	validMatrix := &perconavs.VersionResponse{
		Versions: []*perconavs.OperatorVersion{
			{
				Matrix: &perconavs.VersionMatrix{
					Pxc: map[string]*perconavs.Version{
						"8.0.32": {},
						"8.0.30": {},
						"5.7.44": {},
						"5.6.35": {},
					},
					Mongod: map[string]*perconavs.Version{
						"6.0.4":  {},
						"5.0.14": {},
						"7.0.2":  {},
					},
					Postgresql: map[string]*perconavs.Version{
						"15.2": {},
						"14.6": {},
						"16.0": {},
					},
				},
			},
		},
	}

	validMatrixJSON, err := protojson.Marshal(validMatrix)
	require.NoError(t, err)

	t.Run("PXC operator versions with 5.x filtered and remaining sorted", func(t *testing.T) {
		t.Parallel()

		var requestedPath string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestedPath = r.URL.Path
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(validMatrixJSON)
		}))
		defer server.Close()

		client := New(server.URL)
		versions, err := client.GetSupportedEngineVersions(context.Background(), PXCOperatorName, "v1.13.0")
		require.NoError(t, err)

		assert.Equal(t, "/versions/v1/pxc-operator/1.13.0", requestedPath)
		assert.Equal(t, []string{"8.0.30", "8.0.32"}, versions)
	})

	t.Run("PSMDB operator versions sorted", func(t *testing.T) {
		t.Parallel()

		var requestedPath string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestedPath = r.URL.Path
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(validMatrixJSON)
		}))
		defer server.Close()

		client := New(server.URL)
		versions, err := client.GetSupportedEngineVersions(context.Background(), PSMDBOperatorName, "1.16.0")
		require.NoError(t, err)

		assert.Equal(t, "/versions/v1/psmdb-operator/1.16.0", requestedPath)
		assert.Equal(t, []string{"5.0.14", "6.0.4", "7.0.2"}, versions)
	})

	t.Run("PG operator versions sorted", func(t *testing.T) {
		t.Parallel()

		var requestedPath string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestedPath = r.URL.Path
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(validMatrixJSON)
		}))
		defer server.Close()

		client := New(server.URL)
		versions, err := client.GetSupportedEngineVersions(context.Background(), PGOperatorName, "v2.3.1")
		require.NoError(t, err)

		assert.Equal(t, "/versions/v1/pg-operator/2.3.1", requestedPath)
		assert.Equal(t, []string{"14.6", "15.2", "16.0"}, versions)
	})

	t.Run("empty versions in response returns error", func(t *testing.T) {
		t.Parallel()

		emptyResponse := &perconavs.VersionResponse{
			Versions: []*perconavs.OperatorVersion{},
		}
		emptyJSON, err := protojson.Marshal(emptyResponse)
		require.NoError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(emptyJSON)
		}))
		defer server.Close()

		client := New(server.URL)
		versions, err := client.GetSupportedEngineVersions(context.Background(), PXCOperatorName, "1.13.0")
		require.Error(t, err)
		assert.Nil(t, versions)
		assert.Contains(t, err.Error(), "no versions found")
	})

	t.Run("invalid semver in matrix returns error", func(t *testing.T) {
		t.Parallel()

		badVersionMatrix := &perconavs.VersionResponse{
			Versions: []*perconavs.OperatorVersion{
				{
					Matrix: &perconavs.VersionMatrix{
						Pxc: map[string]*perconavs.Version{
							"invalid-semver": {},
						},
					},
				},
			},
		}
		badJSON, err := protojson.Marshal(badVersionMatrix)
		require.NoError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(badJSON)
		}))
		defer server.Close()

		client := New(server.URL)
		versions, err := client.GetSupportedEngineVersions(context.Background(), PXCOperatorName, "1.13.0")
		require.Error(t, err)
		assert.Nil(t, versions)
		assert.Contains(t, err.Error(), "malformed version")
	})

	t.Run("non-200 HTTP response returns error", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		client := New(server.URL)
		versions, err := client.GetSupportedEngineVersions(context.Background(), PXCOperatorName, "1.13.0")
		require.Error(t, err)
		assert.Nil(t, versions)
		assert.Contains(t, err.Error(), "invalid response from version service endpoint http 404")
	})

	t.Run("malformed JSON payload returns decode error", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("not-a-valid-json"))
		}))
		defer server.Close()

		client := New(server.URL)
		versions, err := client.GetSupportedEngineVersions(context.Background(), PXCOperatorName, "1.13.0")
		require.Error(t, err)
		assert.Nil(t, versions)
		assert.Contains(t, err.Error(), "could not decode version response")
	})

	t.Run("invalid URL returns parse error", func(t *testing.T) {
		t.Parallel()

		client := New("http://invalid url with spaces")
		versions, err := client.GetSupportedEngineVersions(context.Background(), PXCOperatorName, "1.13.0")
		require.Error(t, err)
		assert.Nil(t, versions)
		assert.Contains(t, err.Error(), "could not parse version service URL")
	})

	t.Run("unreachable server returns network error", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
		server.Close()

		client := New(server.URL)
		versions, err := client.GetSupportedEngineVersions(context.Background(), PXCOperatorName, "1.13.0")
		require.Error(t, err)
		assert.Nil(t, versions)
		assert.Contains(t, err.Error(), "could not retrieve version response")
	})
}

func TestGetEverestMetadata(t *testing.T) {
	t.Parallel()

	expectedMetadata := &perconavs.MetadataResponse{
		Versions: []*perconavs.MetadataVersion{
			{
				Version: "0.4.0",
				Recommended: map[string]string{
					"pxc-operator": "1.13.0",
				},
				Supported: map[string]string{
					"pxc-operator": ">= 1.12.0",
				},
			},
		},
	}

	expectedJSON, err := json.Marshal(expectedMetadata)
	require.NoError(t, err)

	t.Run("successful metadata retrieval", func(t *testing.T) {
		t.Parallel()

		var requestedPath string
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestedPath = r.URL.Path
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(expectedJSON)
		}))
		defer server.Close()

		client := New(server.URL)
		metadata, err := client.GetEverestMetadata(context.Background())
		require.NoError(t, err)

		assert.Equal(t, "/metadata/v1/everest", requestedPath)
		require.NotNil(t, metadata)
		require.Len(t, metadata.GetVersions(), 1)
		assert.Equal(t, "0.4.0", metadata.GetVersions()[0].GetVersion())
		assert.Equal(t, "1.13.0", metadata.GetVersions()[0].GetRecommended()["pxc-operator"])
		assert.Equal(t, ">= 1.12.0", metadata.GetVersions()[0].GetSupported()["pxc-operator"])
	})

	t.Run("non-200 HTTP response returns error", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
		}))
		defer server.Close()

		client := New(server.URL)
		metadata, err := client.GetEverestMetadata(context.Background())
		require.Error(t, err)
		assert.Nil(t, metadata)
		assert.Contains(t, err.Error(), "invalid response from Everest metadata endpoint http 503")
	})

	t.Run("malformed JSON returns decode error", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("{invalid-json"))
		}))
		defer server.Close()

		client := New(server.URL)
		metadata, err := client.GetEverestMetadata(context.Background())
		require.Error(t, err)
		assert.Nil(t, metadata)
		assert.Contains(t, err.Error(), "could not decode requirements from Everest metadata")
	})

	t.Run("invalid URL returns parse error", func(t *testing.T) {
		t.Parallel()

		client := New("http://invalid url with spaces")
		metadata, err := client.GetEverestMetadata(context.Background())
		require.Error(t, err)
		assert.Nil(t, metadata)
		assert.Contains(t, err.Error(), "could not parse version Everest metadata URL")
	})

	t.Run("unreachable server returns network error", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
		server.Close()

		client := New(server.URL)
		metadata, err := client.GetEverestMetadata(context.Background())
		require.Error(t, err)
		assert.Nil(t, metadata)
		assert.Contains(t, err.Error(), "could not retrieve Everest metadata")
	})
}
