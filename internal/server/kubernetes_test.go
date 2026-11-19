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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	api "github.com/openeverest/openeverest/v2/internal/server/api"
	"github.com/openeverest/openeverest/v2/internal/server/handlers"
)

func TestKubernetesClusterEndpoints(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		cluster       string
		canonical     bool
		expectedLink  string
		expectedUsage string
	}{
		{
			name:          "cluster scoped resources",
			cluster:       "prod-eu",
			canonical:     true,
			expectedUsage: "resources",
		},
		{
			name:          "legacy resources",
			cluster:       legacyDefaultCluster,
			expectedLink:  "</v1/clusters/main/resources>; rel=\"successor-version\"",
			expectedUsage: "resources",
		},
		{
			name:          "cluster scoped info",
			cluster:       "prod-us",
			canonical:     true,
			expectedUsage: "info",
		},
		{
			name:          "legacy info",
			cluster:       legacyDefaultCluster,
			expectedLink:  "</v1/clusters/main/cluster-info>; rel=\"successor-version\"",
			expectedUsage: "info",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			next := &handlers.MockHandler{}
			switch tc.expectedUsage {
			case "resources":
				next.On(
					"GetKubernetesClusterResources",
					mock.Anything,
					tc.cluster,
				).Return(&api.KubernetesClusterResources{}, nil).Once()
			case "info":
				next.On(
					"GetKubernetesClusterInfo",
					mock.Anything,
					tc.cluster,
				).Return(&api.KubernetesClusterInfo{}, nil).Once()
			}

			server := &EverestServer{handler: next, l: zap.NewNop().Sugar()}
			rec := httptest.NewRecorder()
			ctx := echo.New().NewContext(
				httptest.NewRequest(http.MethodGet, "/", nil),
				rec,
			)

			var err error
			switch {
			case tc.expectedUsage == "resources" && tc.canonical:
				err = server.GetKubernetesClusterResourcesForCluster(ctx, tc.cluster)
			case tc.expectedUsage == "resources":
				err = server.GetKubernetesClusterResources(ctx)
			case tc.canonical:
				err = server.GetKubernetesClusterInfoForCluster(ctx, tc.cluster)
			default:
				err = server.GetKubernetesClusterInfo(ctx)
			}

			require.NoError(t, err)
			require.Equal(t, http.StatusOK, rec.Code)
			require.Equal(t, tc.expectedLink, rec.Header().Get("Link"))
			if !tc.canonical {
				require.Equal(t, "true", rec.Header().Get("Deprecation"))
			}
			next.AssertExpectations(t)
		})
	}
}
