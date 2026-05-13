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

func TestRBAC_BackupClass(t *testing.T) {
	t.Parallel()

	mockBackupClasses := func() *handlers.MockHandler {
		h := &handlers.MockHandler{}
		h.On("ListBackupClasses", mock.Anything, mock.Anything).Return(
			&backupv1alpha1.BackupClassList{
				Items: []backupv1alpha1.BackupClass{
					{ObjectMeta: metav1.ObjectMeta{Name: "s3-standard"}},
					{ObjectMeta: metav1.ObjectMeta{Name: "gcs-nearline"}},
				},
			}, nil,
		)
		h.On("GetBackupClass", mock.Anything, mock.Anything, mock.Anything).Return(
			&backupv1alpha1.BackupClass{ObjectMeta: metav1.ObjectMeta{Name: "s3-standard"}},
			nil,
		)
		return h
	}

	t.Run("ListBackupClasses", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			desc    string
			cluster string
			policy  string
			assert  func(list *backupv1alpha1.BackupClassList) bool
		}{
			{
				desc:    "admin",
				cluster: "prod",
				policy: newPolicy(
					"g, bob, role:admin",
				),
				assert: func(list *backupv1alpha1.BackupClassList) bool {
					return len(list.Items) == 2
				},
			},
			{
				desc:    "all backup classes on cluster",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, backup-classes, read, prod/*",
					"g, bob, role:test",
				),
				assert: func(list *backupv1alpha1.BackupClassList) bool {
					return len(list.Items) == 2
				},
			},
			{
				desc:    "specific backup class",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, backup-classes, read, prod/s3-standard",
					"g, bob, role:test",
				),
				assert: func(list *backupv1alpha1.BackupClassList) bool {
					return len(list.Items) == 1 && list.Items[0].Name == "s3-standard"
				},
			},
			{
				desc:    "wrong cluster",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, backup-classes, read, staging/*",
					"g, bob, role:test",
				),
				assert: func(list *backupv1alpha1.BackupClassList) bool {
					return len(list.Items) == 0
				},
			},
			{
				desc:    "no permissions",
				cluster: "prod",
				policy: newPolicy(
					"g, bob, role:test",
				),
				assert: func(list *backupv1alpha1.BackupClassList) bool {
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
				next := mockBackupClasses()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				list, err := h.ListBackupClasses(ctx, tc.cluster)
				require.NoError(t, err)
				assert.Condition(t, func() bool {
					return tc.assert(list)
				})
			})
		}
	})

	t.Run("GetBackupClass", func(t *testing.T) {
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
				name:    "s3-standard",
				policy: newPolicy(
					"g, bob, role:admin",
				),
			},
			{
				desc:    "exact match",
				cluster: "prod",
				name:    "s3-standard",
				policy: newPolicy(
					"p, role:test, backup-classes, read, prod/s3-standard",
					"g, bob, role:test",
				),
			},
			{
				desc:    "wrong cluster",
				cluster: "prod",
				name:    "s3-standard",
				policy: newPolicy(
					"p, role:test, backup-classes, read, staging/s3-standard",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc:    "no permissions",
				cluster: "prod",
				name:    "s3-standard",
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
				next := mockBackupClasses()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				result, err := h.GetBackupClass(ctx, tc.cluster, tc.name)
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
				} else {
					require.NoError(t, err)
					assert.Equal(t, "s3-standard", result.Name)
				}
			})
		}
	})
}
