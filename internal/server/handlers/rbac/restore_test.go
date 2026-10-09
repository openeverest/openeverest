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

// TestRBAC_Restore covers the flat /restores routes. Before this, they had no
// RBAC enforcement at all (openeverest/openeverest#3189): any authenticated
// user could create, get, or delete a restore regardless of permissions.
func TestRBAC_Restore(t *testing.T) {
	t.Parallel()

	restoreFixture := func() *backupv1alpha1.Restore {
		return &backupv1alpha1.Restore{
			ObjectMeta: metav1.ObjectMeta{Name: "restore-1", Namespace: "ns1"},
			Spec: backupv1alpha1.RestoreSpec{
				InstanceRef: objectref.ObjectRef{Name: "instance-1"},
			},
		}
	}

	mockRestores := func() *handlers.MockHandler {
		h := &handlers.MockHandler{}
		h.On("GetRestore", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
			restoreFixture(), nil,
		)
		h.On("CreateRestore", mock.Anything, mock.Anything, mock.Anything).Return(
			restoreFixture(), nil,
		)
		h.On("DeleteRestore", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		return h
	}

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
			desc:    "exact match on the owning instance",
			cluster: "prod",
			policy: newPolicy(
				"p, role:test, restores, *, prod/ns1/instance-1",
				"g, bob, role:test",
			),
		},
		{
			desc:    "namespace wildcard",
			cluster: "prod",
			policy: newPolicy(
				"p, role:test, restores, *, prod/ns1/*",
				"g, bob, role:test",
			),
		},
		{
			desc:    "wrong cluster",
			cluster: "prod",
			policy: newPolicy(
				"p, role:test, restores, *, staging/ns1/instance-1",
				"g, bob, role:test",
			),
			wantErr: ErrInsufficientPermissions,
		},
		{
			desc:    "no permissions at all",
			cluster: "prod",
			policy: newPolicy(
				"g, bob, role:test",
			),
			wantErr: ErrInsufficientPermissions,
		},
	}

	ctx := context.WithValue(context.Background(), common.UserCtxKey, rbac.User{Subject: "bob"}) //nolint:staticcheck

	newHandler := func(t *testing.T, policy string, next *handlers.MockHandler) *rbacHandler {
		t.Helper()
		k8sMock := newConfigMapMock(policy)
		enf, err := rbac.NewEnforcer(ctx, k8sMock, zap.NewNop().Sugar())
		require.NoError(t, err)
		return &rbacHandler{next: next, log: zap.NewNop().Sugar(), enforcer: enf, userGetter: testUserGetter}
	}

	t.Run("GetRestore", func(t *testing.T) {
		t.Parallel()
		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				t.Parallel()
				h := newHandler(t, tc.policy, mockRestores())
				result, err := h.GetRestore(ctx, tc.cluster, "ns1", "restore-1")
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
				} else {
					require.NoError(t, err)
					assert.Equal(t, "restore-1", result.Name)
				}
			})
		}
	})

	t.Run("CreateRestore", func(t *testing.T) {
		t.Parallel()
		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				t.Parallel()
				h := newHandler(t, tc.policy, mockRestores())
				result, err := h.CreateRestore(ctx, tc.cluster, restoreFixture())
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
				} else {
					require.NoError(t, err)
					assert.Equal(t, "restore-1", result.Name)
				}
			})
		}
	})

	t.Run("DeleteRestore", func(t *testing.T) {
		t.Parallel()
		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				t.Parallel()
				h := newHandler(t, tc.policy, mockRestores())
				err := h.DeleteRestore(ctx, tc.cluster, "ns1", "restore-1")
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
				} else {
					require.NoError(t, err)
				}
			})
		}
	})

	t.Run("CreateRestore with no instanceRef fails closed under a namespace wildcard", func(t *testing.T) {
		t.Parallel()

		// InstanceRef is a value, not a pointer: an omitted instanceRef in the
		// request body decodes to an empty name, which would otherwise build
		// "prod/ns1/" and match a namespace-wide wildcard grant despite naming
		// no real instance. Regression test for the behavior of namespace-wide wildcard grants.
		restore := restoreFixture()
		restore.Spec.InstanceRef.Name = ""

		policy := newPolicy(
			"p, role:test, restores, *, prod/ns1/*",
			"g, bob, role:test",
		)
		h := newHandler(t, policy, mockRestores())

		_, err := h.CreateRestore(ctx, "prod", restore)
		require.ErrorIs(t, err, ErrInsufficientPermissions)
	})

	// A restore reads its source, so restores:create on the target is not
	// enough: the caller must also be able to read the backups it restores from.
	t.Run("CreateRestore requires reading the data source", func(t *testing.T) {
		t.Parallel()

		fromBackup := func() *backupv1alpha1.Restore {
			restore := restoreFixture()
			restore.Spec.DataSource = backupv1alpha1.DataSource{
				Type:   backupv1alpha1.DataSourceTypeBackup,
				Backup: &backupv1alpha1.DataSourceBackup{BackupRef: objectref.ObjectRef{Name: "backup-1"}},
			}
			return restore
		}
		fromPITR := func() *backupv1alpha1.Restore {
			restore := restoreFixture()
			restore.Spec.DataSource = backupv1alpha1.DataSource{
				Type: backupv1alpha1.DataSourceTypePointInTime,
				PointInTime: &backupv1alpha1.DataSourcePointInTime{Source: backupv1alpha1.StreamSource{
					InstanceRef: &objectref.ObjectRef{Name: "source-db"},
					StorageRef:  objectref.ObjectRef{Name: "s3"},
				}},
			}
			return restore
		}
		const (
			createRestore = "p, role:test, restores, create, prod/ns1/instance-1"
			readSource    = "p, role:test, backups, read, prod/ns1/source-db"
			readStorage   = "p, role:test, backup-storages, read, prod/ns1/s3"
		)

		for _, tc := range []struct {
			desc     string
			restore  *backupv1alpha1.Restore
			policy   []string
			wantDeny bool
		}{
			{desc: "backup readable", restore: fromBackup(), policy: []string{createRestore, readSource}},
			{desc: "backup not readable", restore: fromBackup(), policy: []string{createRestore}, wantDeny: true},
			{desc: "pitr source readable", restore: fromPITR(), policy: []string{createRestore, readSource, readStorage}},
			{desc: "pitr storage not readable", restore: fromPITR(), policy: []string{createRestore, readSource}, wantDeny: true},
			{desc: "pitr source backups not readable", restore: fromPITR(), policy: []string{createRestore, readStorage}, wantDeny: true},
		} {
			t.Run(tc.desc, func(t *testing.T) {
				t.Parallel()
				next := mockRestores()
				next.On("GetBackup", mock.Anything, mock.Anything, mock.Anything, "backup-1").Return(&backupv1alpha1.Backup{
					ObjectMeta: metav1.ObjectMeta{Name: "backup-1", Namespace: "ns1"},
					Spec: backupv1alpha1.BackupSpec{Origin: backupv1alpha1.BackupOrigin{
						Type:        backupv1alpha1.BackupOriginTypeInstance,
						InstanceRef: &objectref.ObjectRef{Name: "source-db"},
					}},
				}, nil)
				h := newHandler(t, newPolicy(append(tc.policy, "g, bob, role:test")...), next)

				_, err := h.CreateRestore(ctx, "prod", tc.restore)
				if tc.wantDeny {
					require.ErrorIs(t, err, ErrInsufficientPermissions)
					return
				}
				require.NoError(t, err)
			})
		}
	})

	t.Run("missing restore collapses to the same error as denied", func(t *testing.T) {
		t.Parallel()

		notFound := k8serrors.NewNotFound(schema.GroupResource{Resource: "restores"}, "restore-1")
		next := &handlers.MockHandler{}
		next.On("GetRestore", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
			(*backupv1alpha1.Restore)(nil), notFound,
		)

		// Admin policy: admin can never legitimately fail the enforce check
		// below, so if the not-found collapse above were ever removed, this
		// is the subject that would expose it by getting a different error.
		h := newHandler(t, newPolicy("g, bob, role:admin"), next)

		_, err := h.GetRestore(ctx, "prod", "ns1", "restore-1")
		require.ErrorIs(t, err, ErrInsufficientPermissions)

		err = h.DeleteRestore(ctx, "prod", "ns1", "restore-1")
		require.ErrorIs(t, err, ErrInsufficientPermissions)
	})
}
