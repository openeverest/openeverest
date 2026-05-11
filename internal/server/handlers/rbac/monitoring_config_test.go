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
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	monitoringv1alpha2 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha2"
	api "github.com/openeverest/openeverest/v2/internal/server/api"
	"github.com/openeverest/openeverest/v2/internal/server/handlers"
	"github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

func TestRBAC_MonitoringConfig(t *testing.T) {
	t.Parallel()

	mockMonitoring := func() *handlers.MockHandler {
		h := &handlers.MockHandler{}
		h.On("ListMonitoringConfigs", mock.Anything, mock.Anything, mock.Anything).Return(
			&monitoringv1alpha2.MonitoringConfigList{
				Items: []monitoringv1alpha2.MonitoringConfig{
					{ObjectMeta: metav1.ObjectMeta{Name: "pmm-prod", Namespace: "ns1"}},
					{ObjectMeta: metav1.ObjectMeta{Name: "pmm-staging", Namespace: "ns1"}},
				},
			}, nil,
		)
		h.On("GetMonitoringConfig", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
			&monitoringv1alpha2.MonitoringConfig{ObjectMeta: metav1.ObjectMeta{Name: "pmm-prod", Namespace: "ns1"}},
			nil,
		)
		h.On("CreateMonitoringConfig", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
			&monitoringv1alpha2.MonitoringConfig{ObjectMeta: metav1.ObjectMeta{Name: "pmm-prod", Namespace: "ns1"}},
			nil,
		)
		h.On("UpdateMonitoringConfig", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
			&monitoringv1alpha2.MonitoringConfig{ObjectMeta: metav1.ObjectMeta{Name: "pmm-prod", Namespace: "ns1"}},
			nil,
		)
		h.On("DeleteMonitoringConfig", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		return h
	}

	t.Run("ListMonitoringConfigs", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			desc    string
			cluster string
			ns      string
			policy  string
			assert  func(list *monitoringv1alpha2.MonitoringConfigList) bool
		}{
			{
				desc:    "admin",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"g, bob, role:admin",
				),
				assert: func(list *monitoringv1alpha2.MonitoringConfigList) bool {
					return len(list.Items) == 2
				},
			},
			{
				desc:    "all in namespace with wildcard",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"p, role:test, monitoring-configs, read, prod/ns1/*",
					"g, bob, role:test",
				),
				assert: func(list *monitoringv1alpha2.MonitoringConfigList) bool {
					return len(list.Items) == 2
				},
			},
			{
				desc:    "specific config",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"p, role:test, monitoring-configs, read, prod/ns1/pmm-prod",
					"g, bob, role:test",
				),
				assert: func(list *monitoringv1alpha2.MonitoringConfigList) bool {
					return len(list.Items) == 1 && list.Items[0].Name == "pmm-prod"
				},
			},
			{
				desc:    "wrong cluster",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"p, role:test, monitoring-configs, read, staging/ns1/*",
					"g, bob, role:test",
				),
				assert: func(list *monitoringv1alpha2.MonitoringConfigList) bool {
					return len(list.Items) == 0
				},
			},
			{
				desc:    "wrong namespace",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"p, role:test, monitoring-configs, read, prod/ns2/*",
					"g, bob, role:test",
				),
				assert: func(list *monitoringv1alpha2.MonitoringConfigList) bool {
					return len(list.Items) == 0
				},
			},
			{
				desc:    "no permissions",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"g, bob, role:test",
				),
				assert: func(list *monitoringv1alpha2.MonitoringConfigList) bool {
					return len(list.Items) == 0
				},
			},
			{
				desc:    "two specific via multiple rules",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"p, role:test, monitoring-configs, read, prod/ns1/pmm-prod",
					"p, role:test, monitoring-configs, read, prod/ns1/pmm-staging",
					"g, bob, role:test",
				),
				assert: func(list *monitoringv1alpha2.MonitoringConfigList) bool {
					return len(list.Items) == 2 &&
						slices.ContainsFunc(list.Items, func(mc monitoringv1alpha2.MonitoringConfig) bool { return mc.Name == "pmm-prod" }) &&
						slices.ContainsFunc(list.Items, func(mc monitoringv1alpha2.MonitoringConfig) bool { return mc.Name == "pmm-staging" })
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
				next := mockMonitoring()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				list, err := h.ListMonitoringConfigs(ctx, tc.cluster, tc.ns)
				require.NoError(t, err)
				assert.Condition(t, func() bool {
					return tc.assert(list)
				})
			})
		}
	})

	t.Run("GetMonitoringConfig", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			desc    string
			cluster string
			policy  string
			wantErr error
		}{
			{
				desc:    "admin",
				cluster: "prod",
				policy:  newPolicy("g, bob, role:admin"),
			},
			{
				desc:    "exact match",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, monitoring-configs, read, prod/ns1/pmm-prod",
					"g, bob, role:test",
				),
			},
			{
				desc:    "wrong cluster",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, monitoring-configs, read, staging/ns1/pmm-prod",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc:    "no permissions",
				cluster: "prod",
				policy:  newPolicy("g, bob, role:test"),
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
				next := mockMonitoring()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				result, err := h.GetMonitoringConfig(ctx, tc.cluster, "ns1", "pmm-prod")
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
				} else {
					require.NoError(t, err)
					assert.Equal(t, "pmm-prod", result.Name)
				}
			})
		}
	})

	t.Run("CreateMonitoringConfig", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			desc    string
			cluster string
			policy  string
			wantErr error
		}{
			{
				desc:    "admin",
				cluster: "prod",
				policy:  newPolicy("g, bob, role:admin"),
			},
			{
				desc:    "has create permission",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, monitoring-configs, create, prod/ns1/pmm-prod",
					"g, bob, role:test",
				),
			},
			{
				desc:    "namespace wildcard",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, monitoring-configs, create, prod/ns1/*",
					"g, bob, role:test",
				),
			},
			{
				desc:    "has read but not create",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, monitoring-configs, read, prod/ns1/pmm-prod",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc:    "no permissions",
				cluster: "prod",
				policy:  newPolicy("g, bob, role:test"),
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
				next := mockMonitoring()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				req := &api.MonitoringConfigCreateParams{Name: "pmm-prod"}
				result, err := h.CreateMonitoringConfig(ctx, tc.cluster, "ns1", req)
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
				} else {
					require.NoError(t, err)
					assert.Equal(t, "pmm-prod", result.Name)
				}
			})
		}
	})

	t.Run("UpdateMonitoringConfig", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			desc    string
			cluster string
			policy  string
			wantErr error
		}{
			{
				desc:    "admin",
				cluster: "prod",
				policy:  newPolicy("g, bob, role:admin"),
			},
			{
				desc:    "has update permission",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, monitoring-configs, update, prod/ns1/pmm-prod",
					"g, bob, role:test",
				),
			},
			{
				desc:    "no permissions",
				cluster: "prod",
				policy:  newPolicy("g, bob, role:test"),
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
				next := mockMonitoring()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				req := &api.MonitoringConfigUpdateParams{}
				result, err := h.UpdateMonitoringConfig(ctx, tc.cluster, "ns1", "pmm-prod", req)
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
				} else {
					require.NoError(t, err)
					assert.Equal(t, "pmm-prod", result.Name)
				}
			})
		}
	})

	t.Run("DeleteMonitoringConfig", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			desc    string
			cluster string
			policy  string
			wantErr error
		}{
			{
				desc:    "admin",
				cluster: "prod",
				policy:  newPolicy("g, bob, role:admin"),
			},
			{
				desc:    "has delete permission",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, monitoring-configs, delete, prod/ns1/pmm-prod",
					"g, bob, role:test",
				),
			},
			{
				desc:    "no permissions",
				cluster: "prod",
				policy:  newPolicy("g, bob, role:test"),
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
				next := mockMonitoring()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				err = h.DeleteMonitoringConfig(ctx, tc.cluster, "ns1", "pmm-prod")
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
				} else {
					require.NoError(t, err)
				}
			})
		}
	})
}
