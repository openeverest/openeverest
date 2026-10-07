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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/internal/server/handlers"
	"github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

func TestRBAC_ClusterInstancePreset(t *testing.T) {
	t.Parallel()

	mockClusterInstancePresets := func() *handlers.MockHandler {
		h := &handlers.MockHandler{}
		h.On("ListClusterInstancePresets", mock.Anything, mock.Anything, mock.Anything).Return(
			&corev1alpha1.ClusterInstancePresetList{
				Items: []corev1alpha1.ClusterInstancePreset{
					{ObjectMeta: metav1.ObjectMeta{Name: "small"}},
					{ObjectMeta: metav1.ObjectMeta{Name: "medium"}},
					{ObjectMeta: metav1.ObjectMeta{Name: "large"}},
				},
			}, nil,
		)
		h.On("GetClusterInstancePreset", mock.Anything, mock.Anything, mock.Anything).Return(
			&corev1alpha1.ClusterInstancePreset{ObjectMeta: metav1.ObjectMeta{Name: "small"}},
			nil,
		)
		h.On("ResolveClusterInstancePreset", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
			&corev1alpha1.ClusterInstancePreset{ObjectMeta: metav1.ObjectMeta{Name: "small"}},
			nil,
		)
		return h
	}

	t.Run("ListClusterInstancePresets", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			desc    string
			cluster string
			policy  string
			assert  func(list *corev1alpha1.ClusterInstancePresetList) bool
		}{
			{
				desc:    "admin",
				cluster: "prod",
				policy: newPolicy(
					"g, bob, role:admin",
				),
				assert: func(list *corev1alpha1.ClusterInstancePresetList) bool {
					return len(list.Items) == 3
				},
			},
			{
				desc:    "all cluster-instance-presets on cluster with wildcard",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, cluster-instance-presets, read, prod/*",
					"g, bob, role:test",
				),
				assert: func(list *corev1alpha1.ClusterInstancePresetList) bool {
					return len(list.Items) == 3
				},
			},
			{
				desc:    "specific instance-preset on cluster",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, cluster-instance-presets, read, prod/small",
					"g, bob, role:test",
				),
				assert: func(list *corev1alpha1.ClusterInstancePresetList) bool {
					return len(list.Items) == 1 && list.Items[0].Name == "small"
				},
			},
			{
				desc:    "wrong cluster",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, cluster-instance-presets, read, staging/*",
					"g, bob, role:test",
				),
				assert: func(list *corev1alpha1.ClusterInstancePresetList) bool {
					return len(list.Items) == 0
				},
			},
			{
				desc:    "no permissions",
				cluster: "prod",
				policy: newPolicy(
					"g, bob, role:test",
				),
				assert: func(list *corev1alpha1.ClusterInstancePresetList) bool {
					return len(list.Items) == 0
				},
			},
			{
				desc:    "all clusters all cluster-instance-presets wildcard",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, cluster-instance-presets, read, */*",
					"g, bob, role:test",
				),
				assert: func(list *corev1alpha1.ClusterInstancePresetList) bool {
					return len(list.Items) == 3
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
				next := mockClusterInstancePresets()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				list, err := h.ListClusterInstancePresets(ctx, tc.cluster, "")
				require.NoError(t, err)
				assert.Condition(t, func() bool {
					return tc.assert(list)
				})
			})
		}
	})

	t.Run("GetClusterInstancePreset", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			desc    string
			cluster string
			name    string
			policy  string
			wantErr error
		}{
			{
				desc:    "admin",
				cluster: "prod",
				name:    "small",
				policy: newPolicy(
					"g, bob, role:admin",
				),
			},
			{
				desc:    "exact match",
				cluster: "prod",
				name:    "small",
				policy: newPolicy(
					"p, role:test, cluster-instance-presets, read, prod/small",
					"g, bob, role:test",
				),
			},
			{
				desc:    "wildcard on cluster",
				cluster: "prod",
				name:    "small",
				policy: newPolicy(
					"p, role:test, cluster-instance-presets, read, prod/*",
					"g, bob, role:test",
				),
			},
			{
				desc:    "wrong cluster",
				cluster: "prod",
				name:    "small",
				policy: newPolicy(
					"p, role:test, cluster-instance-presets, read, staging/small",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc:    "wrong instance-preset",
				cluster: "prod",
				name:    "small",
				policy: newPolicy(
					"p, role:test, cluster-instance-presets, read, prod/medium",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc:    "no permissions",
				cluster: "prod",
				name:    "small",
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
				next := mockClusterInstancePresets()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				result, err := h.GetClusterInstancePreset(ctx, tc.cluster, tc.name)
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
				} else {
					require.NoError(t, err)
					assert.Equal(t, "small", result.Name)
				}
			})
		}
	})

	t.Run("ResolveClusterInstancePreset", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			desc      string
			cluster   string
			name      string
			namespace string
			policy    string
			wantErr   error
		}{
			{
				desc:      "admin",
				cluster:   "prod",
				name:      "small",
				namespace: "default",
				policy: newPolicy(
					"g, bob, role:admin",
				),
			},
			{
				desc:      "exact match with namespace access",
				cluster:   "prod",
				name:      "small",
				namespace: "default",
				policy: newPolicy(
					"p, role:test, cluster-instance-presets, read, prod/small",
					"p, role:test, namespaces, read, prod/default",
					"g, bob, role:test",
				),
			},
			{
				desc:      "wildcard on cluster and namespace",
				cluster:   "prod",
				name:      "small",
				namespace: "default",
				policy: newPolicy(
					"p, role:test, cluster-instance-presets, read, prod/*",
					"p, role:test, namespaces, read, prod/*",
					"g, bob, role:test",
				),
			},
			{
				desc:      "no instance-preset access",
				cluster:   "prod",
				name:      "small",
				namespace: "default",
				policy: newPolicy(
					"p, role:test, namespaces, read, prod/default",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc:      "no namespace access",
				cluster:   "prod",
				name:      "small",
				namespace: "default",
				policy: newPolicy(
					"p, role:test, cluster-instance-presets, read, prod/small",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc:      "wrong cluster for instance-preset",
				cluster:   "prod",
				name:      "small",
				namespace: "default",
				policy: newPolicy(
					"p, role:test, cluster-instance-presets, read, staging/small",
					"p, role:test, namespaces, read, prod/default",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc:      "wrong cluster for namespace",
				cluster:   "prod",
				name:      "small",
				namespace: "default",
				policy: newPolicy(
					"p, role:test, cluster-instance-presets, read, prod/small",
					"p, role:test, namespaces, read, staging/default",
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
				next := mockClusterInstancePresets()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				result, err := h.ResolveClusterInstancePreset(ctx, tc.cluster, tc.name, tc.namespace)
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
				} else {
					require.NoError(t, err)
					assert.Equal(t, "small", result.Name)
				}
			})
		}
	})
}
