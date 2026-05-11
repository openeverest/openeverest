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

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	"github.com/openeverest/openeverest/v2/internal/server/handlers"
	"github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

func TestRBAC_Backup(t *testing.T) {
	t.Parallel()

	mockBackups := func() *handlers.MockHandler {
		h := &handlers.MockHandler{}
		h.On("GetBackup", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
			&backupv1alpha1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "backup-1", Namespace: "ns1"}},
			nil,
		)
		h.On("CreateBackup", mock.Anything, mock.Anything, mock.Anything).Return(
			&backupv1alpha1.Backup{ObjectMeta: metav1.ObjectMeta{Name: "backup-1", Namespace: "ns1"}},
			nil,
		)
		h.On("DeleteBackup", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		return h
	}

	t.Run("GetBackup", func(t *testing.T) {
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
				desc:    "exact match",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, backups, read, prod/ns1/backup-1",
					"g, bob, role:test",
				),
			},
			{
				desc:    "namespace wildcard",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, backups, read, prod/ns1/*",
					"g, bob, role:test",
				),
			},
			{
				desc:    "wrong cluster",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, backups, read, staging/ns1/backup-1",
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
				next := mockBackups()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				result, err := h.GetBackup(ctx, tc.cluster, "ns1", "backup-1")
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
				} else {
					require.NoError(t, err)
					assert.Equal(t, "backup-1", result.Name)
				}
			})
		}
	})

	t.Run("CreateBackup", func(t *testing.T) {
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
					"p, role:test, backups, create, prod/ns1/backup-1",
					"g, bob, role:test",
				),
			},
			{
				desc:    "namespace wildcard",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, backups, create, prod/ns1/*",
					"g, bob, role:test",
				),
			},
			{
				desc:    "has read but not create",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, backups, read, prod/ns1/backup-1",
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
				next := mockBackups()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				backup := &backupv1alpha1.Backup{
					ObjectMeta: metav1.ObjectMeta{Name: "backup-1", Namespace: "ns1"},
				}
				result, err := h.CreateBackup(ctx, tc.cluster, backup)
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
				} else {
					require.NoError(t, err)
					assert.Equal(t, "backup-1", result.Name)
				}
			})
		}
	})

	t.Run("DeleteBackup", func(t *testing.T) {
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
					"p, role:test, backups, delete, prod/ns1/backup-1",
					"g, bob, role:test",
				),
			},
			{
				desc:    "has read but not delete",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, backups, read, prod/ns1/backup-1",
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
				next := mockBackups()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				err = h.DeleteBackup(ctx, tc.cluster, "ns1", "backup-1")
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
				} else {
					require.NoError(t, err)
				}
			})
		}
	})
}
