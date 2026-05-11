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

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	api "github.com/openeverest/openeverest/v2/internal/server/api"
	"github.com/openeverest/openeverest/v2/internal/server/handlers"
	"github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

func TestRBAC_DatabaseClusterBackup(t *testing.T) {
	t.Run("ListDatabaseClusterBackups", func(t *testing.T) {
		next := func() *handlers.MockHandler {
			h := &handlers.MockHandler{}
			h.On("ListDatabaseClusterBackups", mock.Anything, mock.Anything, mock.Anything).Return(&everestv1alpha1.DatabaseClusterBackupList{
				Items: []everestv1alpha1.DatabaseClusterBackup{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "backup1",
							Namespace: "default",
						},
						Spec: everestv1alpha1.DatabaseClusterBackupSpec{
							DBClusterName:     "cluster1",
							BackupStorageName: "bs1",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "backup2",
							Namespace: "default",
						},
						Spec: everestv1alpha1.DatabaseClusterBackupSpec{
							DBClusterName:     "cluster1",
							BackupStorageName: "bs2",
						},
					},
				},
			}, nil,
			)
			return h
		}

		testCases := []struct {
			desc   string
			policy string
			assert func(*everestv1alpha1.DatabaseClusterBackupList) bool
		}{
			{
				desc: "admin",
				policy: newPolicy(
					"g, bob, role:admin",
				),
				assert: func(list *everestv1alpha1.DatabaseClusterBackupList) bool {
					return len(list.Items) == 2 &&
						slices.ContainsFunc(list.Items, func(backup everestv1alpha1.DatabaseClusterBackup) bool {
							return backup.GetName() == "backup1"
						}) &&
						slices.ContainsFunc(list.Items, func(backup everestv1alpha1.DatabaseClusterBackup) bool {
							return backup.GetName() == "backup2"
						})
				},
			},
			{
				desc: "success",
				policy: newPolicy(
					"p, role:test, database-cluster-backups, read, default/cluster1",
					"g, bob, role:test",
				),
				assert: func(list *everestv1alpha1.DatabaseClusterBackupList) bool {
					return len(list.Items) == 2 &&
						slices.ContainsFunc(list.Items, func(backup everestv1alpha1.DatabaseClusterBackup) bool {
							return backup.GetName() == "backup1"
						}) &&
						slices.ContainsFunc(list.Items, func(backup everestv1alpha1.DatabaseClusterBackup) bool {
							return backup.GetName() == "backup2"
						})
				},
			},
			{
				desc: "missing read permissons for database-cluster-backups on cluster 'cluster1'",
				policy: newPolicy(
					"g, bob, role:test",
				),
				assert: func(list *everestv1alpha1.DatabaseClusterBackupList) bool {
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
				next := next()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				list, err := h.ListDatabaseClusterBackups(ctx, "default", "cluster1")
				require.NoError(t, err)
				assert.Condition(t, func() bool {
					return tc.assert(list)
				})
			})
		}
	})

	t.Run("GetDatabaseClusterBackup", func(t *testing.T) {
		next := func() *handlers.MockHandler {
			h := &handlers.MockHandler{}
			h.On("GetDatabaseClusterBackup", mock.Anything, mock.Anything, mock.Anything).Return(&everestv1alpha1.DatabaseClusterBackup{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "backup1",
					Namespace: "default",
				},
				Spec: everestv1alpha1.DatabaseClusterBackupSpec{
					DBClusterName:     "cluster1",
					BackupStorageName: "bs1",
				},
			}, nil,
			)
			return h
		}

		testCases := []struct {
			desc    string
			policy  string
			wantErr error
		}{
			{
				desc: "admin",
				policy: newPolicy(
					"g, bob, role:admin",
				),
			},
			{
				desc: "success",
				policy: newPolicy(
					"p, role:test, database-cluster-backups, read, default/cluster1",
					"g, bob, role:test",
				),
			},
			{
				desc: "missing read permissons for database-cluster-backups on cluster 'cluster1'",
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
				next := next()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				_, err = h.GetDatabaseClusterBackup(ctx, "default", "backup1")
				assert.ErrorIs(t, err, tc.wantErr)
			})
		}
	})

	t.Run("CreateDatabaseClusterBackup", func(t *testing.T) {
		next := func() *handlers.MockHandler {
			h := &handlers.MockHandler{}
			h.On("CreateDatabaseClusterBackup", mock.Anything, mock.Anything, mock.Anything).Return(&everestv1alpha1.DatabaseClusterBackup{}, nil)
			return h
		}

		testCases := []struct {
			desc    string
			policy  string
			wantErr error
		}{
			{
				desc: "admin",
				policy: newPolicy(
					"g, bob, role:admin",
				),
			},
			{
				desc: "success",
				policy: newPolicy(
					"p, role:test, database-cluster-backups, create, default/cluster1",
					"g, bob, role:test",
				),
			},
			{
				desc: "missing create permissons for database-cluster-backups on cluster 'cluster1'",
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
				next := next()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				_, err = h.CreateDatabaseClusterBackup(ctx, &everestv1alpha1.DatabaseClusterBackup{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "backup1",
						Namespace: "default",
					},
					Spec: everestv1alpha1.DatabaseClusterBackupSpec{
						DBClusterName:     "cluster1",
						BackupStorageName: "bs1",
					},
				})
				assert.ErrorIs(t, err, tc.wantErr)
			})
		}
	})

	t.Run("DeleteDatabaseClusterBackup", func(t *testing.T) {
		next := func() *handlers.MockHandler {
			h := &handlers.MockHandler{}
			h.On("GetDatabaseClusterBackup", mock.Anything, mock.Anything, mock.Anything).
				Return(&everestv1alpha1.DatabaseClusterBackup{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "backup1",
						Namespace: "default",
					},
					Spec: everestv1alpha1.DatabaseClusterBackupSpec{
						DBClusterName:     "cluster1",
						BackupStorageName: "bs1",
					},
				}, nil,
				)
			h.On("DeleteDatabaseClusterBackup", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
				Return(nil)
			return h
		}

		testCases := []struct {
			desc    string
			policy  string
			wantErr error
		}{
			{
				desc: "admin",
				policy: newPolicy(
					"g, bob, role:admin",
				),
			},
			{
				desc: "success",
				policy: newPolicy(
					"p, role:test, database-cluster-backups, delete, default/cluster1",
					"g, bob, role:test",
				),
			},
			{
				desc:    "missing delete permissons for database-cluster-backups on cluster 'cluster1'",
				policy:  newPolicy(),
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
				next := next()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				err = h.DeleteDatabaseClusterBackup(ctx, "default", "backup1", &api.DeleteDatabaseClusterBackupParams{})
				assert.ErrorIs(t, err, tc.wantErr)
			})
		}
	})
}
