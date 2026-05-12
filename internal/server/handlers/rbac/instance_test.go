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

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	api "github.com/openeverest/openeverest/v2/internal/server/api"
	"github.com/openeverest/openeverest/v2/internal/server/handlers"
	"github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

func TestRBAC_Instance(t *testing.T) {
	t.Parallel()

	mockInstances := func() *handlers.MockHandler {
		h := &handlers.MockHandler{}
		h.On("ListInstances", mock.Anything, mock.Anything, mock.Anything).Return(
			&corev1alpha1.InstanceList{
				Items: []corev1alpha1.Instance{
					{ObjectMeta: metav1.ObjectMeta{Name: "db1", Namespace: "ns1"}},
					{ObjectMeta: metav1.ObjectMeta{Name: "db2", Namespace: "ns1"}},
					{ObjectMeta: metav1.ObjectMeta{Name: "db3", Namespace: "ns1"}},
				},
			}, nil,
		)
		h.On("GetInstance", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
			&corev1alpha1.Instance{ObjectMeta: metav1.ObjectMeta{Name: "db1", Namespace: "ns1"}},
			nil,
		)
		h.On("CreateInstance", mock.Anything, mock.Anything, mock.Anything).Return(
			&corev1alpha1.Instance{ObjectMeta: metav1.ObjectMeta{Name: "db1", Namespace: "ns1"}},
			nil,
		)
		h.On("UpdateInstance", mock.Anything, mock.Anything, mock.Anything).Return(
			&corev1alpha1.Instance{ObjectMeta: metav1.ObjectMeta{Name: "db1", Namespace: "ns1"}},
			nil,
		)
		h.On("DeleteInstance", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		host := "db1.ns1.svc"
		h.On("GetInstanceConnection", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
			&api.InstanceConnectionDetails{Host: &host},
			nil,
		)
		return h
	}

	t.Run("ListInstances", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			desc    string
			cluster string
			ns      string
			policy  string
			assert  func(list *corev1alpha1.InstanceList) bool
		}{
			{
				desc:    "admin",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"g, bob, role:admin",
				),
				assert: func(list *corev1alpha1.InstanceList) bool {
					return len(list.Items) == 3
				},
			},
			{
				desc:    "all instances on cluster and namespace",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"p, role:test, instances, read, prod/ns1/*",
					"g, bob, role:test",
				),
				assert: func(list *corev1alpha1.InstanceList) bool {
					return len(list.Items) == 3
				},
			},
			{
				desc:    "specific instance",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"p, role:test, instances, read, prod/ns1/db1",
					"g, bob, role:test",
				),
				assert: func(list *corev1alpha1.InstanceList) bool {
					return len(list.Items) == 1 && list.Items[0].Name == "db1"
				},
			},
			{
				desc:    "two specific instances",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"p, role:test, instances, read, prod/ns1/db1",
					"p, role:test, instances, read, prod/ns1/db3",
					"g, bob, role:test",
				),
				assert: func(list *corev1alpha1.InstanceList) bool {
					return len(list.Items) == 2 &&
						slices.ContainsFunc(list.Items, func(i corev1alpha1.Instance) bool { return i.Name == "db1" }) &&
						slices.ContainsFunc(list.Items, func(i corev1alpha1.Instance) bool { return i.Name == "db3" })
				},
			},
			{
				desc:    "wrong cluster",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"p, role:test, instances, read, staging/ns1/*",
					"g, bob, role:test",
				),
				assert: func(list *corev1alpha1.InstanceList) bool {
					return len(list.Items) == 0
				},
			},
			{
				desc:    "wrong namespace",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"p, role:test, instances, read, prod/ns2/*",
					"g, bob, role:test",
				),
				assert: func(list *corev1alpha1.InstanceList) bool {
					return len(list.Items) == 0
				},
			},
			{
				desc:    "all clusters wildcard",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"p, role:test, instances, read, */*/*",
					"g, bob, role:test",
				),
				assert: func(list *corev1alpha1.InstanceList) bool {
					return len(list.Items) == 3
				},
			},
			{
				desc:    "no permissions",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"g, bob, role:test",
				),
				assert: func(list *corev1alpha1.InstanceList) bool {
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
				next := mockInstances()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				list, err := h.ListInstances(ctx, tc.cluster, tc.ns)
				require.NoError(t, err)
				assert.Condition(t, func() bool {
					return tc.assert(list)
				})
			})
		}
	})

	t.Run("GetInstance", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			desc    string
			cluster string
			ns      string
			name    string
			policy  string
			wantErr error
		}{
			{
				desc:    "admin",
				cluster: "prod",
				ns:      "ns1",
				name:    "db1",
				policy: newPolicy(
					"g, bob, role:admin",
				),
			},
			{
				desc:    "exact match",
				cluster: "prod",
				ns:      "ns1",
				name:    "db1",
				policy: newPolicy(
					"p, role:test, instances, read, prod/ns1/db1",
					"g, bob, role:test",
				),
			},
			{
				desc:    "namespace wildcard",
				cluster: "prod",
				ns:      "ns1",
				name:    "db1",
				policy: newPolicy(
					"p, role:test, instances, read, prod/ns1/*",
					"g, bob, role:test",
				),
			},
			{
				desc:    "wrong cluster",
				cluster: "prod",
				ns:      "ns1",
				name:    "db1",
				policy: newPolicy(
					"p, role:test, instances, read, staging/ns1/db1",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc:    "wrong namespace",
				cluster: "prod",
				ns:      "ns1",
				name:    "db1",
				policy: newPolicy(
					"p, role:test, instances, read, prod/ns2/db1",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc:    "wrong name",
				cluster: "prod",
				ns:      "ns1",
				name:    "db1",
				policy: newPolicy(
					"p, role:test, instances, read, prod/ns1/db2",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc:    "no permissions",
				cluster: "prod",
				ns:      "ns1",
				name:    "db1",
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
				next := mockInstances()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				result, err := h.GetInstance(ctx, tc.cluster, tc.ns, tc.name)
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
				} else {
					require.NoError(t, err)
					assert.Equal(t, "db1", result.Name)
				}
			})
		}
	})

	t.Run("CreateInstance", func(t *testing.T) {
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
				policy: newPolicy(
					"g, bob, role:admin",
				),
			},
			{
				desc:    "has create permission",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, instances, create, prod/ns1/db1",
					"g, bob, role:test",
				),
			},
			{
				desc:    "namespace wildcard create",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, instances, create, prod/ns1/*",
					"g, bob, role:test",
				),
			},
			{
				desc:    "has read but not create",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, instances, read, prod/ns1/db1",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc:    "wrong cluster",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, instances, create, staging/ns1/db1",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc:    "no permissions",
				cluster: "prod",
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
				next := mockInstances()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				instance := &corev1alpha1.Instance{
					ObjectMeta: metav1.ObjectMeta{Name: "db1", Namespace: "ns1"},
				}
				result, err := h.CreateInstance(ctx, tc.cluster, instance)
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
				} else {
					require.NoError(t, err)
					assert.Equal(t, "db1", result.Name)
				}
			})
		}
	})

	t.Run("UpdateInstance", func(t *testing.T) {
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
				policy: newPolicy(
					"g, bob, role:admin",
				),
			},
			{
				desc:    "has update permission",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, instances, update, prod/ns1/db1",
					"g, bob, role:test",
				),
			},
			{
				desc:    "has read but not update",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, instances, read, prod/ns1/db1",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc:    "no permissions",
				cluster: "prod",
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
				next := mockInstances()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				instance := &corev1alpha1.Instance{
					ObjectMeta: metav1.ObjectMeta{Name: "db1", Namespace: "ns1"},
				}
				result, err := h.UpdateInstance(ctx, tc.cluster, instance)
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
				} else {
					require.NoError(t, err)
					assert.Equal(t, "db1", result.Name)
				}
			})
		}
	})

	t.Run("DeleteInstance", func(t *testing.T) {
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
				policy: newPolicy(
					"g, bob, role:admin",
				),
			},
			{
				desc:    "has delete permission",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, instances, delete, prod/ns1/db1",
					"g, bob, role:test",
				),
			},
			{
				desc:    "has read but not delete",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, instances, read, prod/ns1/db1",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc:    "no permissions",
				cluster: "prod",
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
				next := mockInstances()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				err = h.DeleteInstance(ctx, tc.cluster, "ns1", "db1")
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
				} else {
					require.NoError(t, err)
				}
			})
		}
	})

	t.Run("GetInstanceConnection", func(t *testing.T) {
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
				policy: newPolicy(
					"g, bob, role:admin",
				),
			},
			{
				desc:    "has read permission",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, instances, read, prod/ns1/db1",
					"g, bob, role:test",
				),
			},
			{
				desc:    "no permissions",
				cluster: "prod",
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
				next := mockInstances()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				result, err := h.GetInstanceConnection(ctx, tc.cluster, "ns1", "db1")
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
				} else {
					require.NoError(t, err)
					assert.Equal(t, "db1.ns1.svc", *result.Host)
				}
			})
		}
	})
}
