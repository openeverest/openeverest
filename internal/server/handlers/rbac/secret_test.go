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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openeverest/openeverest/v2/internal/server/handlers"
	"github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

func TestRBAC_Secret(t *testing.T) {
	t.Parallel()

	mockSecret := func() *handlers.MockHandler {
		secrets := &corev1.SecretList{
			Items: []corev1.Secret{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret-prod",
						Namespace: "ns1",
					},
					Type: corev1.SecretTypeOpaque,
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret-staging",
						Namespace: "ns1",
					},
					Type: corev1.SecretTypeOpaque,
				},
			},
		}

		h := &handlers.MockHandler{}
		h.On("ListSecrets", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(secrets, nil)
		h.On("GetSecret", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&secrets.Items[0], nil)
		h.On("CreateSecret", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&secrets.Items[0], nil)
		h.On("DeleteSecret", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

		return h
	}

	t.Run("ListSecrets", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			desc    string
			cluster string
			ns      string
			policy  string
			assert  func(*corev1.SecretList) bool
		}{
			{
				desc:    "admin",
				cluster: "prod",
				ns:      "ns1",
				policy:  newPolicy("g, bob, role:admin"),
				assert: func(list *corev1.SecretList) bool {
					return len(list.Items) == 2
				},
			},
			{
				desc:    "namespace wildcard",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"p, role:test, secrets, read, prod/ns1/*",
					"g, bob, role:test",
				),
				assert: func(list *corev1.SecretList) bool {
					return len(list.Items) == 2
				},
			},
			{
				desc:    "specific secret",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"p, role:test, secrets, read, prod/ns1/secret-prod",
					"g, bob, role:test",
				),
				assert: func(list *corev1.SecretList) bool {
					return len(list.Items) == 1
				},
			},
			{
				desc:    "wrong cluster",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"p, role:test, secrets, read, staging/ns1/secret-prod",
					"g, bob, role:test",
				),
				assert: func(list *corev1.SecretList) bool {
					return len(list.Items) == 0
				},
			},
			{
				desc:    "wrong namespace",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"p, role:test, secrets, read, prod/ns2/secret-prod",
					"g, bob, role:test",
				),
				assert: func(list *corev1.SecretList) bool {
					return len(list.Items) == 0
				},
			},
			{
				desc:    "no permissions",
				cluster: "prod",
				ns:      "ns1",
				policy:  newPolicy("g, bob, role:test"),
				assert: func(list *corev1.SecretList) bool {
					return len(list.Items) == 0
				},
			},
			{
				desc:    "multiple permissions",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"p, role:test, secrets, read, prod/ns1/secret-prod",
					"p, role:test, secrets, read, prod/ns1/secret-staging",
					"g, bob, role:test",
				),
				assert: func(list *corev1.SecretList) bool {
					return len(list.Items) == 2 &&
						slices.ContainsFunc(list.Items, func(s corev1.Secret) bool { return s.Name == "secret-prod" }) &&
						slices.ContainsFunc(list.Items, func(s corev1.Secret) bool { return s.Name == "secret-staging" })
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
				next := mockSecret()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				list, err := h.ListSecrets(ctx, tc.cluster, tc.ns, "test-provider", "test-definition")
				require.NoError(t, err)
				assert.Condition(t, func() bool {
					return tc.assert(list)
				})
			})
		}
	})

	t.Run("GetSecret", func(t *testing.T) {
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
					"p, role:test, secrets, read, prod/ns1/secret-prod",
					"g, bob, role:test",
				),
			},
			{
				desc:    "wrong cluster",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, secrets, read, staging/ns1/secret-prod",
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
				next := mockSecret()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				result, err := h.GetSecret(ctx, tc.cluster, "ns1", "secret-prod")
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
				} else {
					require.NoError(t, err)
					assert.Equal(t, "secret-prod", result.Name)
				}
			})
		}
	})

	t.Run("CreateSecret", func(t *testing.T) {
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
					"p, role:test, secrets, create, prod/ns1/secret-prod",
					"g, bob, role:test",
				),
			},
			{
				desc:    "namespace wildcard",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, secrets, create, prod/ns1/*",
					"g, bob, role:test",
				),
			},
			{
				desc:    "has read but not create",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, secrets, read, prod/ns1/secret-prod",
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
				next := mockSecret()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				req := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret-prod",
						Namespace: "ns1",
					},
					Type: corev1.SecretTypeOpaque,
				}
				result, err := h.CreateSecret(ctx, tc.cluster, "ns1", req)
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
				} else {
					require.NoError(t, err)
					assert.Equal(t, "secret-prod", result.Name)
				}
			})
		}
	})

	t.Run("DeleteSecret", func(t *testing.T) {
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
					"p, role:test, secrets, delete, prod/ns1/secret-prod",
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
				next := mockSecret()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				err = h.DeleteSecret(ctx, tc.cluster, "ns1", "secret-prod")
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
				} else {
					require.NoError(t, err)
				}
			})
		}
	})
}
