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

	api "github.com/openeverest/openeverest/v2/internal/server/api"
	"github.com/openeverest/openeverest/v2/internal/server/handlers"
	"github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

func TestRBAC_Kubernetes(t *testing.T) {
	t.Parallel()

	data := func() *handlers.MockHandler {
		next := handlers.MockHandler{}
		next.On("GetUserPermissions",
			mock.Anything,
		).Return(
			&api.UserPermissions{
				Enabled: true,
			},
			nil,
		)
		return &next
	}

	t.Run("GetUserPermissions", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			desc     string
			user     rbac.User
			policy   string
			outPerms [][]string
		}{
			{
				desc: "default admin permissions",
				user: rbac.User{
					Subject: "bob",
				},
				policy: newPolicy(
					"g, bob, role:admin",
				),
				outPerms: [][]string{
					{"bob", "monitoring-instances", "*", "*/*"},
					{"bob", "database-cluster-backups", "*", "*/*"},
					{"bob", "database-cluster-restores", "*", "*/*"},
					{"bob", "database-clusters", "*", "*/*"},
					{"bob", "database-cluster-credentials", "*", "*/*"},
					{"bob", "database-engines", "*", "*/*"},
					{"bob", "namespaces", "*", "*"},
					{"bob", "backup-storages", "*", "*/*"},
					{"bob", "enginefeatures/split-horizon-dns-configs", "*", "*/*"},
					{"bob", "pod-scheduling-policies", "*", "*"},
					{"bob", "load-balancer-configs", "*", "*"},
					{"bob", "data-importers", "*", "*"},
					{"bob", "data-import-jobs", "*", "*/*"},
					// FIXME: providers, instances, backup-classes and backups
					// are part of the new clusters API so the permissions need
					// to be different.  Adding them here for now to avoid
					// breaking tests until we implement RBAC for the new API.
					{"bob", "providers", "*", "*/*"},
					{"bob", "instances", "*", "*/*"},
					{"bob", "backup-classes", "*", "*/*"},
					{"bob", "backups", "*", "*/*"},
					{"bob", "monitoring-configs", "*", "*/*"},
				},
			},
			{
				desc: "permissions from different roles are merged",
				user: rbac.User{
					Subject: "bob",
				},
				policy: newPolicy(
					"p, bob, database-clusters, *, */*",
					"p, role:creater, database-clusters, create, */*",
					"p, role:reader, database-clusters, read, */*",
					"p, role:updater, database-clusters, update, */*",
					"p, role:deleter, database-clusters, delete, */*",
					"g, bob, role:creater",
					"g, bob, role:reader",
					"g, bob, role:updater",
					"g, another-user, role:deleter",
				),
				outPerms: [][]string{
					{"bob", "database-clusters", "*", "*/*"},
					{"bob", "database-clusters", "create", "*/*"},
					{"bob", "database-clusters", "read", "*/*"},
					{"bob", "database-clusters", "update", "*/*"},
				},
			},
			{
				desc: "permissions from different groups are merged",
				user: rbac.User{
					Subject: "bob",
					Groups:  []string{"test-group-1", "test-group-2"},
				},
				policy: newPolicy(
					"p, bob, database-clusters, read, */*",
					"p, test-group-1, database-clusters, create, */*",
					"p, test-group-2, database-clusters, update, */*",
					"p, test-group-3, database-clusters, delete, */*",
				),
				outPerms: [][]string{
					{"bob", "database-clusters", "read", "*/*"},
					{"bob", "database-clusters", "create", "*/*"},
					{"bob", "database-clusters", "update", "*/*"},
				},
			},
			{
				desc: "duplicate permissions are removed",
				user: rbac.User{
					Subject: "bob",
				},
				policy: newPolicy(
					"p, bob, database-clusters, *, */*",
					"p, role:test, database-clusters, *, */*",
					"g, bob, role:test",
				),
				outPerms: [][]string{
					{"bob", "database-clusters", "*", "*/*"},
				},
			},
			{
				desc: "no policy",
				user: rbac.User{
					Subject: "bob",
				},
				policy:   newPolicy(),
				outPerms: [][]string{},
			},
		}

		for _, tc := range testCases {
			ctx := context.WithValue(context.Background(), common.UserCtxKey, tc.user)
			t.Run(tc.desc, func(t *testing.T) {
				t.Parallel()
				k8sMock := newConfigMapMock(tc.policy)
				enf, err := rbac.NewEnforcer(ctx, k8sMock, zap.NewNop().Sugar())
				require.NoError(t, err)
				next := data()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				perms, err := h.GetUserPermissions(ctx)
				require.NoError(t, err)
				assert.True(t, perms.Enabled)
				assert.ElementsMatch(t, tc.outPerms, *perms.Permissions)
			})
		}
	})
}
