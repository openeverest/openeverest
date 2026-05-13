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

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	"github.com/openeverest/openeverest/v2/internal/server/handlers"
	"github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

func TestRBAC_BackupStorageV2(t *testing.T) {
	t.Parallel()

	mockBackupStorages := func() *handlers.MockHandler {
		h := &handlers.MockHandler{}
		h.On("ListBackupStorages", mock.Anything, mock.Anything, mock.Anything).Return(
			&backupv1alpha1.BackupStorageList{
				Items: []backupv1alpha1.BackupStorage{
					{ObjectMeta: metav1.ObjectMeta{Name: "s3-primary", Namespace: "ns1"}},
					{ObjectMeta: metav1.ObjectMeta{Name: "gcs-backup", Namespace: "ns1"}},
					{ObjectMeta: metav1.ObjectMeta{Name: "azure-archive", Namespace: "ns1"}},
				},
			}, nil,
		)
		h.On("GetBackupStorage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
			&backupv1alpha1.BackupStorage{ObjectMeta: metav1.ObjectMeta{Name: "s3-primary", Namespace: "ns1"}},
			nil,
		)
		h.On("CreateBackupStorage", mock.Anything, mock.Anything, mock.Anything).Return(
			&backupv1alpha1.BackupStorage{ObjectMeta: metav1.ObjectMeta{Name: "s3-primary", Namespace: "ns1"}},
			nil,
		)
		h.On("UpdateBackupStorage", mock.Anything, mock.Anything, mock.Anything).Return(
			&backupv1alpha1.BackupStorage{ObjectMeta: metav1.ObjectMeta{Name: "s3-primary", Namespace: "ns1"}},
			nil,
		)
		h.On("DeleteBackupStorage", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		return h
	}

	t.Run("ListBackupStorages", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			desc    string
			cluster string
			ns      string
			policy  string
			assert  func(list *backupv1alpha1.BackupStorageList) bool
		}{
			{
				desc:    "admin",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"g, bob, role:admin",
				),
				assert: func(list *backupv1alpha1.BackupStorageList) bool {
					return len(list.Items) == 3
				},
			},
			{
				desc:    "all in namespace with wildcard",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"p, role:test, backup-storages, read, prod/ns1/*",
					"g, bob, role:test",
				),
				assert: func(list *backupv1alpha1.BackupStorageList) bool {
					return len(list.Items) == 3
				},
			},
			{
				desc:    "specific storage",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"p, role:test, backup-storages, read, prod/ns1/s3-primary",
					"g, bob, role:test",
				),
				assert: func(list *backupv1alpha1.BackupStorageList) bool {
					return len(list.Items) == 1 && list.Items[0].Name == "s3-primary"
				},
			},
			{
				desc:    "two specific storages",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"p, role:test, backup-storages, read, prod/ns1/s3-primary",
					"p, role:test, backup-storages, read, prod/ns1/gcs-backup",
					"g, bob, role:test",
				),
				assert: func(list *backupv1alpha1.BackupStorageList) bool {
					return len(list.Items) == 2 &&
						slices.ContainsFunc(list.Items, func(bs backupv1alpha1.BackupStorage) bool { return bs.Name == "s3-primary" }) &&
						slices.ContainsFunc(list.Items, func(bs backupv1alpha1.BackupStorage) bool { return bs.Name == "gcs-backup" })
				},
			},
			{
				desc:    "wrong cluster",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"p, role:test, backup-storages, read, staging/ns1/*",
					"g, bob, role:test",
				),
				assert: func(list *backupv1alpha1.BackupStorageList) bool {
					return len(list.Items) == 0
				},
			},
			{
				desc:    "wrong namespace",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"p, role:test, backup-storages, read, prod/ns2/*",
					"g, bob, role:test",
				),
				assert: func(list *backupv1alpha1.BackupStorageList) bool {
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
				assert: func(list *backupv1alpha1.BackupStorageList) bool {
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
				next := mockBackupStorages()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				list, err := h.ListBackupStorages(ctx, tc.cluster, tc.ns)
				require.NoError(t, err)
				assert.Condition(t, func() bool {
					return tc.assert(list)
				})
			})
		}
	})

	t.Run("GetBackupStorage", func(t *testing.T) {
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
					"p, role:test, backup-storages, read, prod/ns1/s3-primary",
					"g, bob, role:test",
				),
			},
			{
				desc:    "wrong cluster",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, backup-storages, read, staging/ns1/s3-primary",
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
				next := mockBackupStorages()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				result, err := h.GetBackupStorage(ctx, tc.cluster, "ns1", "s3-primary")
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
				} else {
					require.NoError(t, err)
					assert.Equal(t, "s3-primary", result.Name)
				}
			})
		}
	})

	t.Run("CreateBackupStorage", func(t *testing.T) {
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
					"p, role:test, backup-storages, create, prod/ns1/s3-primary",
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
				next := mockBackupStorages()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				bs := &backupv1alpha1.BackupStorage{
					ObjectMeta: metav1.ObjectMeta{Name: "s3-primary", Namespace: "ns1"},
				}
				result, err := h.CreateBackupStorage(ctx, tc.cluster, bs)
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
				} else {
					require.NoError(t, err)
					assert.Equal(t, "s3-primary", result.Name)
				}
			})
		}
	})

	t.Run("UpdateBackupStorage", func(t *testing.T) {
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
					"p, role:test, backup-storages, update, prod/ns1/s3-primary",
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
				next := mockBackupStorages()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				bs := &backupv1alpha1.BackupStorage{
					ObjectMeta: metav1.ObjectMeta{Name: "s3-primary", Namespace: "ns1"},
				}
				result, err := h.UpdateBackupStorage(ctx, tc.cluster, bs)
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
				} else {
					require.NoError(t, err)
					assert.Equal(t, "s3-primary", result.Name)
				}
			})
		}
	})

	t.Run("DeleteBackupStorage", func(t *testing.T) {
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
					"p, role:test, backup-storages, delete, prod/ns1/s3-primary",
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
				next := mockBackupStorages()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				err = h.DeleteBackupStorage(ctx, tc.cluster, "ns1", "s3-primary")
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
				} else {
					require.NoError(t, err)
				}
			})
		}
	})
}
