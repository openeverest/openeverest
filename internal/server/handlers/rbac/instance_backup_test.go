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

func TestRBAC_InstanceBackup(t *testing.T) {
	t.Parallel()

	mockInstanceBackups := func() *handlers.MockHandler {
		h := &handlers.MockHandler{}
		h.On("ListInstanceBackups", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
			&backupv1alpha1.BackupList{
				Items: []backupv1alpha1.Backup{
					{ObjectMeta: metav1.ObjectMeta{Name: "backup-daily-1", Namespace: "ns1"}},
					{ObjectMeta: metav1.ObjectMeta{Name: "backup-daily-2", Namespace: "ns1"}},
					{ObjectMeta: metav1.ObjectMeta{Name: "backup-ondemand-1", Namespace: "ns1"}},
				},
			}, nil,
		)
		return h
	}

	// ListInstanceBackups is gated on the instance ("db1") it's scoped to,
	// not on each backup's own generateName-produced name.
	// Access is all-or-nothing per case rather than a partial filter.
	t.Run("ListInstanceBackups", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			desc    string
			cluster string
			ns      string
			policy  string
			wantErr error
		}{
			{
				desc:    "admin",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"g, bob, role:admin",
				),
			},
			{
				desc:    "exact match on the instance",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"p, role:test, backups, read, prod/ns1/db1",
					"g, bob, role:test",
				),
			},
			{
				desc:    "namespace wildcard",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"p, role:test, backups, read, prod/ns1/*",
					"g, bob, role:test",
				),
			},
			{
				desc:    "wrong cluster",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"p, role:test, backups, read, staging/ns1/db1",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc:    "wrong namespace",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"p, role:test, backups, read, prod/ns2/db1",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc:    "no permissions",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc:    "has instance read but not backup read",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"p, role:test, instances, read, prod/ns1/*",
					"g, bob, role:test",
				),
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
				next := mockInstanceBackups()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				list, err := h.ListInstanceBackups(ctx, tc.cluster, tc.ns, "db1")
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
				} else {
					require.NoError(t, err)
					assert.Len(t, list.Items, 3)
				}
			})
		}
	})
}
