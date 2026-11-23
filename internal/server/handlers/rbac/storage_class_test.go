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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openeverest/openeverest/v2/internal/server/handlers"
	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

func TestRBAC_StorageClasses(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		desc   string
		policy string
		want   []string
		getErr error
	}{
		{desc: "admin", policy: newPolicy("g, bob, role:admin"), want: []string{"standard", "local-path"}},
		{
			desc:   "one class",
			policy: newPolicy("p, role:test, storage-classes, read, prod/standard", "g, bob, role:test"),
			want:   []string{"standard"},
		},
		{
			desc:   "every class in the cluster",
			policy: newPolicy("p, role:test, storage-classes, read, prod/*", "g, bob, role:test"),
			want:   []string{"standard", "local-path"},
		},
		{
			desc:   "other cluster",
			policy: newPolicy("p, role:test, storage-classes, read, staging/*", "g, bob, role:test"),
			want:   []string{},
			getErr: ErrInsufficientPermissions,
		},
		{
			desc:   "no grant",
			policy: newPolicy("p, role:test, instances, read, prod/*/*", "g, bob, role:test"),
			want:   []string{},
			getErr: ErrInsufficientPermissions,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()

			next := &handlers.MockHandler{}
			next.On("ListStorageClasses", mock.Anything, mock.Anything).Return(
				&storagev1.StorageClassList{Items: []storagev1.StorageClass{
					{ObjectMeta: metav1.ObjectMeta{Name: "standard"}},
					{ObjectMeta: metav1.ObjectMeta{Name: "local-path"}},
				}},
				nil,
			)
			next.On("GetStorageClass", mock.Anything, mock.Anything, mock.Anything).Return(
				&storagev1.StorageClass{ObjectMeta: metav1.ObjectMeta{Name: "standard"}}, nil,
			)
			ctx := testUserContext(rbac.User{Subject: "bob"})
			enf, err := rbac.NewEnforcer(ctx, newConfigMapMock(tc.policy), zap.NewNop().Sugar())
			require.NoError(t, err)
			h := &rbacHandler{next: next, log: zap.NewNop().Sugar(), enforcer: enf, userGetter: testUserGetter}

			list, err := h.ListStorageClasses(ctx, "prod")
			require.NoError(t, err)
			names := make([]string, 0, len(list.Items))
			for _, c := range list.Items {
				names = append(names, c.Name)
			}
			assert.ElementsMatch(t, tc.want, names)

			class, err := h.GetStorageClass(ctx, "prod", "standard")
			if tc.getErr != nil {
				require.ErrorIs(t, err, tc.getErr)
				next.AssertNotCalled(t, "GetStorageClass", mock.Anything, mock.Anything, mock.Anything)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "standard", class.Name)
			next.AssertCalled(t, "GetStorageClass", mock.Anything, "prod", "standard")
		})
	}
}
