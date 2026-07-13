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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openeverest/openeverest/v2/internal/server/handlers"
	"github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

func TestRBAC_ConfigMap(t *testing.T) {
	t.Parallel()

	mockConfigMap := func() *handlers.MockHandler {
		h := &handlers.MockHandler{}
		h.On("ListConfigMaps", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
			&corev1.ConfigMapList{
				Items: []corev1.ConfigMap{
					{ObjectMeta: metav1.ObjectMeta{Name: "configmap-prod", Namespace: "ns1"}},
					{ObjectMeta: metav1.ObjectMeta{Name: "configmap-staging", Namespace: "ns1"}},
				},
			}, nil,
		)
		h.On("GetConfigMap", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
			&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "configmap-prod", Namespace: "ns1"}},
			nil,
		)
		h.On("CreateConfigMap", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
			&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "configmap-prod", Namespace: "ns1"}},
			nil,
		)
		h.On("DeleteConfigMap", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		return h
	}

	t.Run("ListConfigMaps", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			desc    string
			cluster string
			ns      string
			policy  string
			assert  func(list *corev1.ConfigMapList) bool
		}{
			{
				desc:    "admin",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"g, bob, role:admin",
				),
				assert: func(list *corev1.ConfigMapList) bool {
					return len(list.Items) == 2
				},
			},
			{
				desc:    "namespace wildcard",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"p, role:test, config-maps, read, prod/ns1/*",
					"g, bob, role:test",
				),
				assert: func(list *corev1.ConfigMapList) bool {
					return len(list.Items) == 2
				},
			},
			{
				desc:    "specific configmap",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"p, role:test, config-maps, read, prod/ns1/configmap-prod",
					"g, bob, role:test",
				),
				assert: func(list *corev1.ConfigMapList) bool {
					return len(list.Items) == 1
				},
			},
			{
				desc:    "wrong cluster",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"p, role:test, config-maps, read, staging/ns1/configmap-prod",
					"g, bob, role:test",
				),
				assert: func(list *corev1.ConfigMapList) bool {
					return len(list.Items) == 0
				},
			},
			{
				desc:    "wrong namespace",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"p, role:test, config-maps, read, prod/ns2/configmap-prod",
					"g, bob, role:test",
				),
				assert: func(list *corev1.ConfigMapList) bool {
					return len(list.Items) == 0
				},
			},
			{
				desc:    "no permissions",
				cluster: "prod",
				ns:      "ns1",
				policy:  newPolicy("g, bob, role:test"),
				assert: func(list *corev1.ConfigMapList) bool {
					return len(list.Items) == 0
				},
			},
			{
				desc:    "multiple permissions",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"p, role:test, config-maps, read, prod/ns1/configmap-prod",
					"p, role:test, config-maps, read, prod/ns1/configmap-staging",
					"g, bob, role:test",
				),
				assert: func(list *corev1.ConfigMapList) bool {
					return len(list.Items) == 2
				},
			},
		}

		ctx := context.WithValue(context.Background(), common.UserCtxKey, rbac.User{Subject: "bob"}) //nolint:staticcheck
		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				t.Parallel()
				k8sMock := newConfigMapMock(tc.policy)
				enf, err := rbac.NewEnforcer(ctx, k8sMock, zap.NewNop().Sugar())
				require.NoError(t, err)
				next := mockConfigMap()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				list, err := h.ListConfigMaps(ctx, tc.cluster, tc.ns, "", "")
				require.NoError(t, err)
				assert.Condition(t, func() bool {
					return tc.assert(list)
				})
			})
		}
	})

	t.Run("GetConfigMap", func(t *testing.T) {
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
					"p, role:test, config-maps, read, prod/ns1/configmap-prod",
					"g, bob, role:test",
				),
			},
			{
				desc:    "wrong cluster",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, config-maps, read, staging/ns1/configmap-prod",
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

		ctx := context.WithValue(context.Background(), common.UserCtxKey, rbac.User{Subject: "bob"}) //nolint:staticcheck
		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				t.Parallel()
				k8sMock := newConfigMapMock(tc.policy)
				enf, err := rbac.NewEnforcer(ctx, k8sMock, zap.NewNop().Sugar())
				require.NoError(t, err)
				next := mockConfigMap()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				result, err := h.GetConfigMap(ctx, tc.cluster, "ns1", "configmap-prod")
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
				} else {
					require.NoError(t, err)
					assert.Equal(t, "configmap-prod", result.Name)
				}
			})
		}
	})

	t.Run("CreateConfigMap", func(t *testing.T) {
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
					"p, role:test, config-maps, create, prod/ns1/configmap-prod",
					"g, bob, role:test",
				),
			},
			{
				desc:    "namespace wildcard",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, config-maps, create, prod/ns1/*",
					"g, bob, role:test",
				),
			},
			{
				desc:    "has read but not create",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, config-maps, read, prod/ns1/configmap-prod",
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

		ctx := context.WithValue(context.Background(), common.UserCtxKey, rbac.User{Subject: "bob"}) //nolint:staticcheck
		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				t.Parallel()
				k8sMock := newConfigMapMock(tc.policy)
				enf, err := rbac.NewEnforcer(ctx, k8sMock, zap.NewNop().Sugar())
				require.NoError(t, err)
				next := mockConfigMap()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				req := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "configmap-prod",
						Namespace: "ns1",
					},
				}
				result, err := h.CreateConfigMap(ctx, tc.cluster, "ns1", req)
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
				} else {
					require.NoError(t, err)
					assert.Equal(t, "configmap-prod", result.Name)
				}
			})
		}
	})

	t.Run("DeleteConfigMap", func(t *testing.T) {
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
					"p, role:test, config-maps, delete, prod/ns1/configmap-prod",
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

		ctx := context.WithValue(context.Background(), common.UserCtxKey, rbac.User{Subject: "bob"}) //nolint:staticcheck
		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				t.Parallel()
				k8sMock := newConfigMapMock(tc.policy)
				enf, err := rbac.NewEnforcer(ctx, k8sMock, zap.NewNop().Sugar())
				require.NoError(t, err)
				next := mockConfigMap()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				err = h.DeleteConfigMap(ctx, tc.cluster, "ns1", "configmap-prod")
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
				} else {
					require.NoError(t, err)
				}
			})
		}
	})
}
