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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
	"github.com/openeverest/openeverest/v2/internal/server/handlers"
	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

func TestRBAC_MonitoringClasses(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		desc   string
		policy string
		want   []string
		getErr error
	}{
		{desc: "admin", policy: newPolicy("g, bob, role:admin"), want: []string{"pmm", "prometheus"}},
		{
			desc:   "one class",
			policy: newPolicy("p, role:test, monitoring-classes, read, prod/pmm", "g, bob, role:test"),
			want:   []string{"pmm"},
		},
		{
			desc:   "other cluster",
			policy: newPolicy("p, role:test, monitoring-classes, read, staging/*", "g, bob, role:test"),
			want:   []string{},
			getErr: ErrInsufficientPermissions,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()

			next := &handlers.MockHandler{}
			next.On("ListMonitoringClasses", mock.Anything, mock.Anything).Return(
				&monitoringv1alpha1.MonitoringClassList{Items: []monitoringv1alpha1.MonitoringClass{
					{ObjectMeta: metav1.ObjectMeta{Name: "pmm"}},
					{ObjectMeta: metav1.ObjectMeta{Name: "prometheus"}},
				}},
				nil,
			)
			next.On("GetMonitoringClass", mock.Anything, mock.Anything, mock.Anything).Return(
				&monitoringv1alpha1.MonitoringClass{ObjectMeta: metav1.ObjectMeta{Name: "pmm"}}, nil,
			)
			ctx := testUserContext(rbac.User{Subject: "bob"})
			enf, err := rbac.NewEnforcer(ctx, newConfigMapMock(tc.policy), zap.NewNop().Sugar())
			require.NoError(t, err)
			h := &rbacHandler{next: next, log: zap.NewNop().Sugar(), enforcer: enf, userGetter: testUserGetter}

			list, err := h.ListMonitoringClasses(ctx, "prod")
			require.NoError(t, err)
			names := make([]string, 0, len(list.Items))
			for _, c := range list.Items {
				names = append(names, c.Name)
			}
			assert.ElementsMatch(t, tc.want, names)

			class, err := h.GetMonitoringClass(ctx, "prod", "pmm")
			if tc.getErr != nil {
				require.ErrorIs(t, err, tc.getErr)
				next.AssertNotCalled(t, "GetMonitoringClass", mock.Anything, mock.Anything, mock.Anything)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, "pmm", class.Name)
		})
	}
}
