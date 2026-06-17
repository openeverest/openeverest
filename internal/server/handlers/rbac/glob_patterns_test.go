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

// TestRBAC_GlobPatterns tests edge cases for the globMatch-based RBAC matcher.
// The matcher uses: g(r.sub, p.sub) && globMatch(r.res, p.res) && globMatch(r.act, p.act) && globMatch(r.obj, p.obj)
func TestRBAC_GlobPatterns(t *testing.T) {
	t.Parallel()

	// mockHandler provides a simple instance list for verifying filter behavior.
	mockHandler := func() *handlers.MockHandler {
		h := &handlers.MockHandler{}
		h.On("ListInstances", mock.Anything, mock.Anything, mock.Anything).Return(
			&corev1alpha1.InstanceList{
				Items: []corev1alpha1.Instance{
					{ObjectMeta: metav1.ObjectMeta{Name: "db1", Namespace: "ns1"}},
					{ObjectMeta: metav1.ObjectMeta{Name: "db2", Namespace: "ns1"}},
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
		return h
	}

	t.Run("wildcard patterns", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			desc      string
			cluster   string
			ns        string
			policy    string
			wantCount int
		}{
			{
				desc:    "triple wildcard matches all clusters/namespaces/names",
				cluster: "any-cluster",
				ns:      "ns1",
				policy: newPolicy(
					"p, role:test, instances, read, */*/*",
					"g, bob, role:test",
				),
				wantCount: 2,
			},
			{
				desc:    "cluster wildcard with exact namespace and name",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"p, role:test, instances, read, */ns1/db1",
					"g, bob, role:test",
				),
				wantCount: 1,
			},
			{
				desc:    "cluster wildcard with namespace wildcard",
				cluster: "staging",
				ns:      "ns1",
				policy: newPolicy(
					"p, role:test, instances, read, */*/db1",
					"g, bob, role:test",
				),
				wantCount: 1,
			},
			{
				desc:    "exact cluster with namespace wildcard and name wildcard",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"p, role:test, instances, read, prod/*/*",
					"g, bob, role:test",
				),
				wantCount: 2,
			},
			{
				desc:    "glob prefix on name",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"p, role:test, instances, read, prod/ns1/db*",
					"g, bob, role:test",
				),
				wantCount: 2,
			},
			{
				desc:    "glob prefix on cluster",
				cluster: "prod-us",
				ns:      "ns1",
				policy: newPolicy(
					"p, role:test, instances, read, prod-*/ns1/*",
					"g, bob, role:test",
				),
				wantCount: 2,
			},
			{
				desc:    "single character wildcard (?)",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"p, role:test, instances, read, prod/ns1/db?",
					"g, bob, role:test",
				),
				wantCount: 2,
			},
			{
				desc:    "single char doesn't match multi-char",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"p, role:test, instances, read, prod/ns1/d?",
					"g, bob, role:test",
				),
				wantCount: 0,
			},
		}

		ctx := context.WithValue(context.Background(), common.UserCtxKey, rbac.User{Subject: "bob"})
		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				t.Parallel()
				k8sMock := newConfigMapMock(tc.policy)
				enf, err := rbac.NewEnforcer(ctx, k8sMock, zap.NewNop().Sugar())
				require.NoError(t, err)
				next := mockHandler()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				list, err := h.ListInstances(ctx, tc.cluster, tc.ns)
				require.NoError(t, err)
				assert.Equal(t, tc.wantCount, len(list.Items))
			})
		}
	})

	t.Run("action wildcards", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			desc    string
			policy  string
			wantErr error
		}{
			{
				desc: "wildcard action grants all operations",
				policy: newPolicy(
					"p, role:test, instances, *, prod/ns1/db1",
					"g, bob, role:test",
				),
			},
			{
				desc: "read action does not grant create",
				policy: newPolicy(
					"p, role:test, instances, read, prod/ns1/db1",
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
				next := mockHandler()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				instance := &corev1alpha1.Instance{
					ObjectMeta: metav1.ObjectMeta{Name: "db1", Namespace: "ns1"},
				}
				_, err = h.CreateInstance(ctx, "prod", instance)
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
				} else {
					require.NoError(t, err)
				}
			})
		}
	})

	t.Run("resource wildcards", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			desc    string
			policy  string
			wantErr error
		}{
			{
				desc: "wildcard resource grants all resource types",
				policy: newPolicy(
					"p, role:test, *, read, prod/ns1/db1",
					"g, bob, role:test",
				),
			},
			{
				desc: "wrong resource type denies access",
				policy: newPolicy(
					"p, role:test, backups, read, prod/ns1/db1",
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
				next := mockHandler()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				_, err = h.GetInstance(ctx, "prod", "ns1", "db1")
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
				} else {
					require.NoError(t, err)
				}
			})
		}
	})

	t.Run("role inheritance", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			desc      string
			policy    string
			wantCount int
		}{
			{
				desc: "user inherits from role",
				policy: newPolicy(
					"p, role:viewer, instances, read, prod/ns1/*",
					"g, bob, role:viewer",
				),
				wantCount: 2,
			},
			{
				desc: "role inherits from another role",
				policy: newPolicy(
					"p, role:viewer, instances, read, prod/ns1/*",
					"g, role:editor, role:viewer",
					"g, bob, role:editor",
				),
				wantCount: 2,
			},
			{
				desc: "multiple roles combine permissions",
				policy: newPolicy(
					"p, role:ns1-viewer, instances, read, prod/ns1/db1",
					"p, role:ns1-extra, instances, read, prod/ns1/db2",
					"g, bob, role:ns1-viewer",
					"g, bob, role:ns1-extra",
				),
				wantCount: 2,
			},
		}

		ctx := context.WithValue(context.Background(), common.UserCtxKey, rbac.User{Subject: "bob"})
		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				t.Parallel()
				k8sMock := newConfigMapMock(tc.policy)
				enf, err := rbac.NewEnforcer(ctx, k8sMock, zap.NewNop().Sugar())
				require.NoError(t, err)
				next := mockHandler()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				list, err := h.ListInstances(ctx, "prod", "ns1")
				require.NoError(t, err)
				assert.Equal(t, tc.wantCount, len(list.Items))
			})
		}
	})

	t.Run("deny rules not supported by adapter", func(t *testing.T) {
		t.Parallel()

		// NOTE: The Casbin model defines: e = some(where (p.eft == allow)) && !some(where (p.eft == deny))
		// However, the current ConfigMap adapter only supports 4-field policy rules (sub, res, act, obj).
		// 5-field rules with explicit deny are rejected. This test documents that behavior.
		ctx := context.WithValue(context.Background(), common.UserCtxKey, rbac.User{Subject: "bob"})
		k8sMock := newConfigMapMock(newPolicy(
			"p, role:test, instances, read, prod/ns1/*",
			"p, role:test, instances, read, prod/ns1/db2, deny",
			"g, bob, role:test",
		))
		_, err := rbac.NewEnforcer(ctx, k8sMock, zap.NewNop().Sugar())
		require.Error(t, err, "5-field deny rules are not supported by the adapter")
	})
}
