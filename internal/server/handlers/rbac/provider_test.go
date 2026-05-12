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

func TestRBAC_Provider(t *testing.T) {
	t.Parallel()

	mockProviders := func() *handlers.MockHandler {
		h := &handlers.MockHandler{}
		h.On("ListProviders", mock.Anything, mock.Anything).Return(
			&corev1alpha1.ProviderList{
				Items: []corev1alpha1.Provider{
					{ObjectMeta: metav1.ObjectMeta{Name: "psmdb"}},
					{ObjectMeta: metav1.ObjectMeta{Name: "pxc"}},
					{ObjectMeta: metav1.ObjectMeta{Name: "pg"}},
				},
			}, nil,
		)
		h.On("GetProvider", mock.Anything, mock.Anything, mock.Anything).Return(
			&corev1alpha1.Provider{ObjectMeta: metav1.ObjectMeta{Name: "psmdb"}},
			nil,
		)
		return h
	}

	t.Run("ListProviders", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			desc    string
			cluster string
			policy  string
			assert  func(list *corev1alpha1.ProviderList) bool
		}{
			{
				desc:    "admin",
				cluster: "prod",
				policy: newPolicy(
					"g, bob, role:admin",
				),
				assert: func(list *corev1alpha1.ProviderList) bool {
					return len(list.Items) == 3
				},
			},
			{
				desc:    "all providers on cluster with wildcard",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, providers, read, prod/*",
					"g, bob, role:test",
				),
				assert: func(list *corev1alpha1.ProviderList) bool {
					return len(list.Items) == 3
				},
			},
			{
				desc:    "specific provider on cluster",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, providers, read, prod/psmdb",
					"g, bob, role:test",
				),
				assert: func(list *corev1alpha1.ProviderList) bool {
					return len(list.Items) == 1 && list.Items[0].Name == "psmdb"
				},
			},
			{
				desc:    "wrong cluster",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, providers, read, staging/*",
					"g, bob, role:test",
				),
				assert: func(list *corev1alpha1.ProviderList) bool {
					return len(list.Items) == 0
				},
			},
			{
				desc:    "no permissions",
				cluster: "prod",
				policy: newPolicy(
					"g, bob, role:test",
				),
				assert: func(list *corev1alpha1.ProviderList) bool {
					return len(list.Items) == 0
				},
			},
			{
				desc:    "all clusters all providers wildcard",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, providers, read, */*",
					"g, bob, role:test",
				),
				assert: func(list *corev1alpha1.ProviderList) bool {
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
				next := mockProviders()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				list, err := h.ListProviders(ctx, tc.cluster)
				require.NoError(t, err)
				assert.Condition(t, func() bool {
					return tc.assert(list)
				})
			})
		}
	})

	t.Run("GetProvider", func(t *testing.T) {
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
				name:    "psmdb",
				policy: newPolicy(
					"g, bob, role:admin",
				),
			},
			{
				desc:    "exact match",
				cluster: "prod",
				name:    "psmdb",
				policy: newPolicy(
					"p, role:test, providers, read, prod/psmdb",
					"g, bob, role:test",
				),
			},
			{
				desc:    "wildcard on cluster",
				cluster: "prod",
				name:    "psmdb",
				policy: newPolicy(
					"p, role:test, providers, read, prod/*",
					"g, bob, role:test",
				),
			},
			{
				desc:    "wrong cluster",
				cluster: "prod",
				name:    "psmdb",
				policy: newPolicy(
					"p, role:test, providers, read, staging/psmdb",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc:    "wrong provider",
				cluster: "prod",
				name:    "psmdb",
				policy: newPolicy(
					"p, role:test, providers, read, prod/pxc",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc:    "no permissions",
				cluster: "prod",
				name:    "psmdb",
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
				next := mockProviders()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				result, err := h.GetProvider(ctx, tc.cluster, tc.name)
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
				} else {
					require.NoError(t, err)
					assert.Equal(t, "psmdb", result.Name)
				}
			})
		}
	})
}
