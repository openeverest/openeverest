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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	api "github.com/openeverest/openeverest/v2/internal/server/api"
	"github.com/openeverest/openeverest/v2/internal/server/handlers"
	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

func TestRBAC_Plugin(t *testing.T) {
	t.Parallel()

	mockPlugins := func() *handlers.MockHandler {
		h := &handlers.MockHandler{}
		h.On("ListPlugins", mock.Anything, mock.Anything).Return(
			api.PluginDescriptorList{
				{Name: "sql-explorer"},
				{Name: "ai-copilot"},
				{Name: "proxysql"},
			}, nil,
		)
		h.On("GetPluginContext", mock.Anything, mock.Anything).Return(
			&api.PluginContext{User: "bob", Namespaces: []string{"prod-ns"}},
			nil,
		)
		return h
	}

	t.Run("ListPlugins", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			desc    string
			cluster string
			policy  string
			assert  func(list api.PluginDescriptorList) bool
		}{
			{
				desc:    "admin",
				cluster: "prod",
				policy: newPolicy(
					"g, bob, role:admin",
				),
				assert: func(list api.PluginDescriptorList) bool {
					return len(list) == 3
				},
			},
			{
				desc:    "all plugins on cluster with wildcard",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, plugins, read, prod/*",
					"g, bob, role:test",
				),
				assert: func(list api.PluginDescriptorList) bool {
					return len(list) == 3
				},
			},
			{
				desc:    "specific plugin on cluster",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, plugins, read, prod/sql-explorer",
					"g, bob, role:test",
				),
				assert: func(list api.PluginDescriptorList) bool {
					return len(list) == 1 && list[0].Name == "sql-explorer"
				},
			},
			{
				desc:    "wrong cluster",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, plugins, read, staging/*",
					"g, bob, role:test",
				),
				assert: func(list api.PluginDescriptorList) bool {
					return len(list) == 0
				},
			},
			{
				desc:    "no permissions",
				cluster: "prod",
				policy: newPolicy(
					"g, bob, role:test",
				),
				assert: func(list api.PluginDescriptorList) bool {
					return len(list) == 0
				},
			},
			{
				desc:    "all clusters all plugins wildcard",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, plugins, read, */*",
					"g, bob, role:test",
				),
				assert: func(list api.PluginDescriptorList) bool {
					return len(list) == 3
				},
			},
		}

		ctx := testUserContext(rbac.User{Subject: "bob"})
		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				t.Parallel()
				k8sMock := newConfigMapMock(tc.policy)
				enf, err := rbac.NewEnforcer(ctx, k8sMock, zap.NewNop().Sugar())
				require.NoError(t, err)
				next := mockPlugins()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				list, err := h.ListPlugins(ctx, tc.cluster)
				require.NoError(t, err)
				assert.Condition(t, func() bool {
					return tc.assert(list)
				})
			})
		}
	})

	// GetPluginContext is not gated on a per-plugin object — any authenticated
	// user may fetch it — but the namespace list is filtered to those the caller
	// can read. mockPlugins returns a single namespace "prod-ns".
	t.Run("GetPluginContext", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			desc       string
			policy     string
			namespaces []string
		}{
			{
				desc:       "admin sees the namespace",
				policy:     newPolicy("g, bob, role:admin"),
				namespaces: []string{"prod-ns"},
			},
			{
				desc:       "user without namespace access sees none",
				policy:     newPolicy("g, bob, role:test"),
				namespaces: []string{},
			},
		}

		ctx := testUserContext(rbac.User{Subject: "bob"})
		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				t.Parallel()
				k8sMock := newConfigMapMock(tc.policy)
				enf, err := rbac.NewEnforcer(ctx, k8sMock, zap.NewNop().Sugar())
				require.NoError(t, err)

				h := &rbacHandler{
					next:       mockPlugins(),
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				result, err := h.GetPluginContext(ctx, "prod")
				require.NoError(t, err)
				assert.Equal(t, "bob", result.User)
				assert.Equal(t, tc.namespaces, result.Namespaces)
			})
		}
	})
}
