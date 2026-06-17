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

package rbac

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	api "github.com/openeverest/openeverest/v2/internal/server/api"
	"github.com/openeverest/openeverest/v2/internal/server/handlers"
	"github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

func TestRBAC_Cluster(t *testing.T) {
	t.Parallel()

	mockClusters := func() *handlers.MockHandler {
		h := &handlers.MockHandler{}
		h.On("ListClusters", mock.Anything).Return(
			&api.ClusterList{
				Items: []api.Cluster{
					{Name: "prod-us"},
					{Name: "prod-eu"},
					{Name: "staging"},
				},
			}, nil,
		)
		h.On("GetCluster", mock.Anything, mock.Anything).Return(
			&api.Cluster{Name: "prod-us", Server: "https://prod-us.example.com"},
			nil,
		)
		return h
	}

	t.Run("ListClusters", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			desc   string
			policy string
			assert func(list *api.ClusterList) bool
		}{
			{
				desc: "admin",
				policy: newPolicy(
					"g, bob, role:admin",
				),
				assert: func(list *api.ClusterList) bool {
					return len(list.Items) == 3
				},
			},
			{
				desc: "full read access with wildcard",
				policy: newPolicy(
					"p, role:test, clusters, read, *",
					"g, bob, role:test",
				),
				assert: func(list *api.ClusterList) bool {
					return len(list.Items) == 3
				},
			},
			{
				desc: "specific cluster only",
				policy: newPolicy(
					"p, role:test, clusters, read, prod-us",
					"g, bob, role:test",
				),
				assert: func(list *api.ClusterList) bool {
					return len(list.Items) == 1 && list.Items[0].Name == "prod-us"
				},
			},
			{
				desc: "glob prefix match",
				policy: newPolicy(
					"p, role:test, clusters, read, prod-*",
					"g, bob, role:test",
				),
				assert: func(list *api.ClusterList) bool {
					return len(list.Items) == 2
				},
			},
			{
				desc: "no permissions",
				policy: newPolicy(
					"g, bob, role:test",
				),
				assert: func(list *api.ClusterList) bool {
					return len(list.Items) == 0
				},
			},
			{
				desc: "wrong action does not grant read",
				policy: newPolicy(
					"p, role:test, clusters, create, *",
					"g, bob, role:test",
				),
				assert: func(list *api.ClusterList) bool {
					return len(list.Items) == 0
				},
			},
		}

		ctx := context.WithValue(context.Background(), common.UserCtxKey, rbac.User{Subject: "bob"})
		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				t.Parallel()
				k8sMock := newConfigMapMock(tc.policy)
				enf, err := rbac.NewEnforcer(ctx, k8sMock, zap.NewNop().Sugar())
				require.NoError(t, err)
				next := mockClusters()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				list, err := h.ListClusters(ctx)
				require.NoError(t, err)
				assert.Condition(t, func() bool {
					return tc.assert(list)
				})
			})
		}
	})

	t.Run("GetCluster", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			desc    string
			policy  string
			wantErr error
		}{
			{
				desc: "admin",
				policy: newPolicy(
					"g, bob, role:admin",
				),
			},
			{
				desc: "has permission",
				policy: newPolicy(
					"p, role:test, clusters, read, prod-us",
					"g, bob, role:test",
				),
			},
			{
				desc: "wildcard permission",
				policy: newPolicy(
					"p, role:test, clusters, read, *",
					"g, bob, role:test",
				),
			},
			{
				desc: "wrong cluster",
				policy: newPolicy(
					"p, role:test, clusters, read, staging",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc: "no permissions",
				policy: newPolicy(
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
		}

		ctx := context.WithValue(context.Background(), common.UserCtxKey, rbac.User{Subject: "bob"})
		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				t.Parallel()
				k8sMock := newConfigMapMock(tc.policy)
				enf, err := rbac.NewEnforcer(ctx, k8sMock, zap.NewNop().Sugar())
				require.NoError(t, err)
				next := mockClusters()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				result, err := h.GetCluster(ctx, "prod-us")
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
				} else {
					require.NoError(t, err)
					assert.Equal(t, "prod-us", result.Name)
				}
			})
		}
	})
}
