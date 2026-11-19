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

func TestKubernetesClusterInfoForCluster(t *testing.T) {
	t.Parallel()

	const cluster = "prod-us"
	next := &handlers.MockHandler{}
	next.On(
		"GetKubernetesClusterInfo",
		mock.Anything,
		cluster,
	).Return(&api.KubernetesClusterInfo{}, nil).Once()

	server := &EverestServer{handler: next, l: zap.NewNop().Sugar()}
	rec := httptest.NewRecorder()
	ctx := echo.New().NewContext(
		httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil),
		rec,
	)

	err := server.GetKubernetesClusterInfoForCluster(ctx, cluster)

	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)
	next.AssertExpectations(t)
}
