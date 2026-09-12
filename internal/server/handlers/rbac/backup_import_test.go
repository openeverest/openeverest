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
	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	"github.com/openeverest/openeverest/v2/internal/server/handlers"
	pkgcommon "github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

func TestRBAC_BackupImport(t *testing.T) {
	t.Parallel()

	mockBackupImports := func() *handlers.MockHandler {
		h := &handlers.MockHandler{}
		h.On("ListBackupImports", mock.Anything, mock.Anything, mock.Anything).Return(
			&backupv1alpha1.BackupImportList{
				Items: []backupv1alpha1.BackupImport{
					{ObjectMeta: metav1.ObjectMeta{Name: "import-1", Namespace: "ns1"}},
					{ObjectMeta: metav1.ObjectMeta{Name: "import-2", Namespace: "ns1"}},
				},
			}, nil,
		)
		h.On("GetBackupImport", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
			&backupv1alpha1.BackupImport{ObjectMeta: metav1.ObjectMeta{Name: "import-1", Namespace: "ns1"}},
			nil,
		)
		h.On("CreateBackupImport", mock.Anything, mock.Anything, mock.Anything).Return(
			&backupv1alpha1.BackupImport{ObjectMeta: metav1.ObjectMeta{Name: "import-1", Namespace: "ns1"}},
			nil,
		)
		h.On("DeleteBackupImport", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		return h
	}

	newHandler := func(policy string) *rbacHandler {
		ctx := context.WithValue(context.Background(), pkgcommon.UserCtxKey, rbac.User{Subject: "bob"}) //nolint:staticcheck // test use only
		k8sMock := newConfigMapMock(policy)
		enf, err := rbac.NewEnforcer(ctx, k8sMock, zap.NewNop().Sugar())
		require.NoError(t, err)
		return &rbacHandler{
			next:       mockBackupImports(),
			log:        zap.NewNop().Sugar(),
			enforcer:   enf,
			userGetter: testUserGetter,
		}
	}

	ctx := context.WithValue(context.Background(), pkgcommon.UserCtxKey, rbac.User{Subject: "bob"}) //nolint:staticcheck // test use only

	t.Run("ListBackupImports", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			desc    string
			cluster string
			policy  string
			assert  func(list *backupv1alpha1.BackupImportList) bool
		}{
			{
				desc:    "admin",
				cluster: "prod",
				policy: newPolicy(
					"g, bob, role:admin",
				),
				assert: func(list *backupv1alpha1.BackupImportList) bool {
					return len(list.Items) == 2
				},
			},
			{
				desc:    "all backup imports in namespace",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, backup-imports, read, prod/ns1/*",
					"g, bob, role:test",
				),
				assert: func(list *backupv1alpha1.BackupImportList) bool {
					return len(list.Items) == 2
				},
			},
			{
				desc:    "specific backup import",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, backup-imports, read, prod/ns1/import-1",
					"g, bob, role:test",
				),
				assert: func(list *backupv1alpha1.BackupImportList) bool {
					return len(list.Items) == 1 && list.Items[0].Name == "import-1"
				},
			},
			{
				desc:    "wrong cluster",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, backup-imports, read, staging/ns1/*",
					"g, bob, role:test",
				),
				assert: func(list *backupv1alpha1.BackupImportList) bool {
					return len(list.Items) == 0
				},
			},
			{
				desc:    "no permissions",
				cluster: "prod",
				policy: newPolicy(
					"g, bob, role:test",
				),
				assert: func(list *backupv1alpha1.BackupImportList) bool {
					return len(list.Items) == 0
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				t.Parallel()
				h := newHandler(tc.policy)

				list, err := h.ListBackupImports(ctx, tc.cluster, "ns1")
				require.NoError(t, err)
				assert.Condition(t, func() bool {
					return tc.assert(list)
				})
			})
		}
	})

	t.Run("GetBackupImport", func(t *testing.T) {
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
					"p, role:test, backup-imports, read, prod/ns1/import-1",
					"g, bob, role:test",
				),
			},
			{
				desc:    "namespace wildcard",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, backup-imports, read, prod/ns1/*",
					"g, bob, role:test",
				),
			},
			{
				desc:    "wrong cluster",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, backup-imports, read, staging/ns1/import-1",
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

		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				t.Parallel()
				h := newHandler(tc.policy)

				result, err := h.GetBackupImport(ctx, tc.cluster, "ns1", "import-1")
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
				} else {
					require.NoError(t, err)
					assert.Equal(t, "import-1", result.Name)
				}
			})
		}
	})

	t.Run("CreateBackupImport", func(t *testing.T) {
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
					"p, role:test, backup-imports, create, prod/ns1/import-1",
					"g, bob, role:test",
				),
			},
			{
				desc:    "has read but not create",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, backup-imports, read, prod/ns1/import-1",
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

		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				t.Parallel()
				h := newHandler(tc.policy)

				backupImport := &backupv1alpha1.BackupImport{
					ObjectMeta: metav1.ObjectMeta{Name: "import-1", Namespace: "ns1"},
					Spec: backupv1alpha1.BackupImportSpec{
						ClassRef:   common.ObjectRef{Name: "s3-standard"},
						StorageRef: common.ObjectRef{Name: "storage-1"},
					},
				}
				result, err := h.CreateBackupImport(ctx, tc.cluster, backupImport)
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
				} else {
					require.NoError(t, err)
					assert.Equal(t, "import-1", result.Name)
				}
			})
		}
	})

	t.Run("DeleteBackupImport", func(t *testing.T) {
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
					"p, role:test, backup-imports, delete, prod/ns1/import-1",
					"g, bob, role:test",
				),
			},
			{
				desc:    "has read but not delete",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, backup-imports, read, prod/ns1/import-1",
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

		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				t.Parallel()
				h := newHandler(tc.policy)

				err := h.DeleteBackupImport(ctx, tc.cluster, "ns1", "import-1")
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
				} else {
					require.NoError(t, err)
				}
			})
		}
	})
}
