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
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	objectref "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	"github.com/openeverest/openeverest/v2/internal/server/handlers"
	"github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

func TestRBAC_Backup(t *testing.T) {
	t.Parallel()

	instanceBackupFixture := func() *backupv1alpha1.Backup {
		return &backupv1alpha1.Backup{
			ObjectMeta: metav1.ObjectMeta{Name: "backup-1", Namespace: "ns1"},
			Spec: backupv1alpha1.BackupSpec{
				StorageRef: objectref.ObjectRef{Name: "storage-a"},
				Origin: backupv1alpha1.BackupOrigin{
					Type:        backupv1alpha1.BackupOriginTypeInstance,
					InstanceRef: &objectref.ObjectRef{Name: "instance-1"},
				},
			},
		}
	}

	externalBackupFixture := func() *backupv1alpha1.Backup {
		return &backupv1alpha1.Backup{
			ObjectMeta: metav1.ObjectMeta{Name: "backup-external", Namespace: "ns1"},
			Spec: backupv1alpha1.BackupSpec{
				StorageRef: objectref.ObjectRef{Name: "storage-a"},
				Origin: backupv1alpha1.BackupOrigin{
					Type:     backupv1alpha1.BackupOriginTypeExternal,
					External: &backupv1alpha1.BackupOriginExternal{Path: "path/to/backup"},
				},
			},
		}
	}

	mockBackups := func(backup *backupv1alpha1.Backup) *handlers.MockHandler {
		h := &handlers.MockHandler{}
		h.On("GetBackup", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
			backup, nil,
		)
		h.On("CreateBackup", mock.Anything, mock.Anything, mock.Anything).Return(
			backup, nil,
		)
		h.On("DeleteBackup", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		return h
	}

	t.Run("ListBackups", func(t *testing.T) {
		t.Parallel()

		backupList := &backupv1alpha1.BackupList{
			Items: []backupv1alpha1.Backup{
				{
					ObjectMeta: metav1.ObjectMeta{Name: "backup-1", Namespace: "ns1"},
					Spec: backupv1alpha1.BackupSpec{
						StorageRef: objectref.ObjectRef{Name: "storage-a"},
						Origin: backupv1alpha1.BackupOrigin{
							Type:        backupv1alpha1.BackupOriginTypeInstance,
							InstanceRef: &objectref.ObjectRef{Name: "instance-1"},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: "backup-2", Namespace: "ns1"},
					Spec: backupv1alpha1.BackupSpec{
						StorageRef: objectref.ObjectRef{Name: "storage-b"},
						Origin: backupv1alpha1.BackupOrigin{
							Type:        backupv1alpha1.BackupOriginTypeInstance,
							InstanceRef: &objectref.ObjectRef{Name: "instance-2"},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{Name: "backup-external", Namespace: "ns1"},
					Spec: backupv1alpha1.BackupSpec{
						StorageRef: objectref.ObjectRef{Name: "storage-a"},
						Origin: backupv1alpha1.BackupOrigin{
							Type:     backupv1alpha1.BackupOriginTypeExternal,
							External: &backupv1alpha1.BackupOriginExternal{Path: "path/to/backup"},
						},
					},
				},
			},
		}

		testCases := []struct {
			desc        string
			cluster     string
			policy      string
			wantBackups []string
		}{
			{
				desc:    "admin",
				cluster: "prod",
				policy: newPolicy(
					"g, bob, role:admin",
				),
				wantBackups: []string{"backup-1", "backup-2", "backup-external"},
			},
			{
				desc:    "namespace wildcard on backups and storages",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, backups, read, prod/ns1/*",
					"p, role:test, backup-storages, read, prod/ns1/*",
					"g, bob, role:test",
				),
				wantBackups: []string{"backup-1", "backup-2", "backup-external"},
			},
			{
				desc:    "backups access shows only instance backups",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, backups, read, prod/ns1/*",
					"g, bob, role:test",
				),
				wantBackups: []string{"backup-1", "backup-2"},
			},
			{
				desc:    "storage access alone shows no external backups",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, backup-storages, read, prod/ns1/storage-a",
					"g, bob, role:test",
				),
				wantBackups: []string{},
			},
			{
				desc:    "namespace-wide backups and specific storage",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, backups, read, prod/ns1/*",
					"p, role:test, backup-storages, read, prod/ns1/storage-a",
					"g, bob, role:test",
				),
				wantBackups: []string{"backup-1", "backup-2", "backup-external"},
			},
			{
				desc:    "specific instance and storage",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, backups, read, prod/ns1/instance-1",
					"p, role:test, backup-storages, read, prod/ns1/storage-a",
					"g, bob, role:test",
				),
				wantBackups: []string{"backup-1"},
			},
			{
				desc:    "instance backup does not need storage access",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, backups, read, prod/ns1/instance-2",
					"g, bob, role:test",
				),
				wantBackups: []string{"backup-2"},
			},
			{
				desc:    "external backup in other storage",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, backup-storages, read, prod/ns1/storage-b",
					"g, bob, role:test",
				),
				wantBackups: []string{},
			},
			{
				desc:    "wrong namespace",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, backups, read, prod/ns2/*",
					"p, role:test, backup-storages, read, prod/ns2/*",
					"g, bob, role:test",
				),
				wantBackups: []string{},
			},
			{
				desc:    "wrong cluster",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, backups, read, staging/ns1/*",
					"p, role:test, backup-storages, read, staging/ns1/*",
					"g, bob, role:test",
				),
				wantBackups: []string{},
			},
			{
				desc:    "no permissions",
				cluster: "prod",
				policy: newPolicy(
					"g, bob, role:test",
				),
				wantBackups: []string{},
			},
		}

		ctx := context.WithValue(context.Background(), common.UserCtxKey, rbac.User{Subject: "bob"}) //nolint:staticcheck // test only
		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				t.Parallel()
				k8sMock := newConfigMapMock(tc.policy)
				enf, err := rbac.NewEnforcer(ctx, k8sMock, zap.NewNop().Sugar())
				require.NoError(t, err)

				next := &handlers.MockHandler{}
				next.On("ListBackups", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
					backupList.DeepCopy(), nil,
				)

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				result, err := h.ListBackups(ctx, tc.cluster, "ns1", nil)
				require.NoError(t, err)

				names := make([]string, 0, len(result.Items))
				for _, b := range result.Items {
					names = append(names, b.Name)
				}
				assert.Equal(t, tc.wantBackups, names)
			})
		}
	})

	t.Run("GetBackup", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			desc    string
			cluster string
			backup  *backupv1alpha1.Backup
			policy  string
			wantErr error
		}{
			{
				desc:    "admin",
				cluster: "prod",
				backup:  instanceBackupFixture(),
				policy: newPolicy(
					"g, bob, role:admin",
				),
			},
			{
				desc:    "exact match",
				cluster: "prod",
				backup:  instanceBackupFixture(),
				policy: newPolicy(
					"p, role:test, backups, read, prod/ns1/instance-1",
					"g, bob, role:test",
				),
			},
			{
				desc:    "external: namespace-wide backups and storage",
				cluster: "prod",
				backup:  externalBackupFixture(),
				policy: newPolicy(
					"p, role:test, backups, read, prod/ns1/*",
					"p, role:test, backup-storages, read, prod/ns1/storage-a",
					"g, bob, role:test",
				),
			},
			{
				desc:    "external: no namespace-wide backups access",
				cluster: "prod",
				backup:  externalBackupFixture(),
				policy: newPolicy(
					"p, role:test, backup-storages, read, prod/ns1/storage-a",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc:    "external: no storage access",
				cluster: "prod",
				backup:  externalBackupFixture(),
				policy: newPolicy(
					"p, role:test, backups, read, prod/ns1/*",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc:    "external: instance grant and storage",
				cluster: "prod",
				backup:  externalBackupFixture(),
				policy: newPolicy(
					"p, role:test, backups, read, prod/ns1/instance-1",
					"p, role:test, backup-storages, read, prod/ns1/storage-a",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc:    "namespace wildcard",
				cluster: "prod",
				backup:  instanceBackupFixture(),
				policy: newPolicy(
					"p, role:test, backups, read, prod/ns1/*",
					"g, bob, role:test",
				),
			},
			{
				desc:    "wrong cluster",
				cluster: "prod",
				backup:  instanceBackupFixture(),
				policy: newPolicy(
					"p, role:test, backups, read, staging/ns1/instance-1",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc:    "no permissions",
				cluster: "prod",
				backup:  instanceBackupFixture(),
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
				next := mockBackups(tc.backup)

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				result, err := h.GetBackup(ctx, tc.cluster, "ns1", tc.backup.Name)
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
				} else {
					require.NoError(t, err)
					assert.Equal(t, tc.backup.Name, result.Name)
				}
			})
		}
	})

	t.Run("CreateBackup", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			desc    string
			cluster string
			backup  *backupv1alpha1.Backup
			policy  string
			wantErr error
		}{
			{
				desc:    "admin",
				cluster: "prod",
				backup:  instanceBackupFixture(),
				policy: newPolicy(
					"g, bob, role:admin",
				),
			},
			{
				desc:    "external: namespace-wide create and storage",
				cluster: "prod",
				backup:  externalBackupFixture(),
				policy: newPolicy(
					"p, role:test, backups, create, prod/ns1/*",
					"p, role:test, backup-storages, read, prod/ns1/storage-a",
					"g, bob, role:test",
				),
			},
			{
				desc:    "external: no storage access",
				cluster: "prod",
				backup:  externalBackupFixture(),
				policy: newPolicy(
					"p, role:test, backups, create, prod/ns1/*",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc:    "external: storage without create",
				cluster: "prod",
				backup:  externalBackupFixture(),
				policy: newPolicy(
					"p, role:test, backup-storages, read, prod/ns1/storage-a",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc:    "external: instance grant and storage",
				cluster: "prod",
				backup:  externalBackupFixture(),
				policy: newPolicy(
					"p, role:test, backups, create, prod/ns1/instance-1",
					"p, role:test, backup-storages, read, prod/ns1/storage-a",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc:    "instance backup has create permission",
				cluster: "prod",
				backup:  instanceBackupFixture(),
				policy: newPolicy(
					"p, role:test, backups, create, prod/ns1/instance-1",
					"g, bob, role:test",
				),
			},
			{
				desc:    "namespace wildcard",
				cluster: "prod",
				backup:  instanceBackupFixture(),
				policy: newPolicy(
					"p, role:test, backups, create, prod/ns1/*",
					"g, bob, role:test",
				),
			},
			{
				desc:    "has read but not create",
				cluster: "prod",
				backup:  instanceBackupFixture(),
				policy: newPolicy(
					"p, role:test, backups, read, prod/ns1/instance-1",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc:    "no permissions",
				cluster: "prod",
				backup:  instanceBackupFixture(),
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
				next := mockBackups(tc.backup)

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				result, err := h.CreateBackup(ctx, tc.cluster, tc.backup)
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
				} else {
					require.NoError(t, err)
					assert.Equal(t, tc.backup.Name, result.Name)
				}
			})
		}
	})

	t.Run("DeleteBackup", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			desc    string
			cluster string
			backup  *backupv1alpha1.Backup
			policy  string
			wantErr error
		}{
			{
				desc:    "admin",
				cluster: "prod",
				backup:  instanceBackupFixture(),
				policy: newPolicy(
					"g, bob, role:admin",
				),
			},
			{
				desc:    "external: namespace-wide delete",
				cluster: "prod",
				backup:  externalBackupFixture(),
				policy: newPolicy(
					"p, role:test, backups, delete, prod/ns1/*",
					"g, bob, role:test",
				),
			},
			{
				desc:    "external: instance grant",
				cluster: "prod",
				backup:  externalBackupFixture(),
				policy: newPolicy(
					"p, role:test, backups, delete, prod/ns1/instance-1",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc:    "has delete permission",
				cluster: "prod",
				backup:  instanceBackupFixture(),
				policy: newPolicy(
					"p, role:test, backups, delete, prod/ns1/instance-1",
					"g, bob, role:test",
				),
			},
			{
				desc:    "has read but not delete",
				cluster: "prod",
				backup:  instanceBackupFixture(),
				policy: newPolicy(
					"p, role:test, backups, read, prod/ns1/instance-1",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc:    "no permissions",
				cluster: "prod",
				backup:  instanceBackupFixture(),
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
				next := mockBackups(tc.backup)

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				err = h.DeleteBackup(ctx, tc.cluster, "ns1", tc.backup.Name, nil)
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
				} else {
					require.NoError(t, err)
				}
			})
		}
	})

	t.Run("missing backup collapses to the same error as denied", func(t *testing.T) {
		t.Parallel()

		notFound := k8serrors.NewNotFound(schema.GroupResource{Resource: "backups"}, "backup-1")
		next := &handlers.MockHandler{}
		next.On("GetBackup", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
			(*backupv1alpha1.Backup)(nil), notFound,
		)

		// Admin policy: admin can never legitimately fail the enforce check
		// below, so if the not-found collapse above were ever removed, this
		// is the subject that would expose it by getting a different error.
		ctx := context.WithValue(context.Background(), common.UserCtxKey, rbac.User{Subject: "bob"}) //nolint:staticcheck
		enf, err := rbac.NewEnforcer(ctx, newConfigMapMock(newPolicy("g, bob, role:admin")), zap.NewNop().Sugar())
		require.NoError(t, err)
		h := &rbacHandler{next: next, log: zap.NewNop().Sugar(), enforcer: enf, userGetter: testUserGetter}

		_, err = h.GetBackup(ctx, "prod", "ns1", "backup-1")
		require.ErrorIs(t, err, ErrInsufficientPermissions)

		err = h.DeleteBackup(ctx, "prod", "ns1", "backup-1", nil)
		require.ErrorIs(t, err, ErrInsufficientPermissions)
	})
}
