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

	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
	"github.com/openeverest/openeverest/v2/internal/server/handlers"
	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

func TestRBAC_MonitoringDestinations(t *testing.T) {
	t.Parallel()

	destination := func(namespace, name string) *monitoringv1alpha1.MonitoringDestination {
		return &monitoringv1alpha1.MonitoringDestination{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace}}
	}
	mockDestinations := func() *handlers.MockHandler {
		h := &handlers.MockHandler{}
		h.On("ListMonitoringDestinations", mock.Anything, mock.Anything, mock.Anything).Return(
			&monitoringv1alpha1.MonitoringDestinationList{Items: []monitoringv1alpha1.MonitoringDestination{
				*destination("ns1", "pmm-prod"),
				*destination("ns1", "prometheus"),
			}},
			nil,
		)
		h.On("CreateMonitoringDestination", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
			func(_ context.Context, _, namespace string, d *monitoringv1alpha1.MonitoringDestination) *monitoringv1alpha1.MonitoringDestination {
				return destination(namespace, d.Name)
			},
			nil,
		)
		h.On("UpdateMonitoringDestination", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(destination("ns1", "pmm-prod"), nil)
		h.On("DeleteMonitoringDestination", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		return h
	}
	newHandler := func(t *testing.T, policy string) *rbacHandler {
		t.Helper()
		enf, err := rbac.NewEnforcer(testUserContext(rbac.User{Subject: "bob"}), newConfigMapMock(policy), zap.NewNop().Sugar())
		require.NoError(t, err)
		return &rbacHandler{next: mockDestinations(), log: zap.NewNop().Sugar(), enforcer: enf, userGetter: testUserGetter}
	}
	ctx := testUserContext(rbac.User{Subject: "bob"})

	t.Run("ListMonitoringDestinations", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			desc   string
			policy string
			want   []string
		}{
			{desc: "admin", policy: newPolicy("g, bob, role:admin"), want: []string{"pmm-prod", "prometheus"}},
			{
				desc:   "one destination",
				policy: newPolicy("p, role:test, monitoring-destinations, read, prod/ns1/pmm-prod", "g, bob, role:test"),
				want:   []string{"pmm-prod"},
			},
			{
				desc:   "other namespace",
				policy: newPolicy("p, role:test, monitoring-destinations, read, prod/ns2/*", "g, bob, role:test"),
				want:   []string{},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				t.Parallel()

				list, err := newHandler(t, tc.policy).ListMonitoringDestinations(ctx, "prod", "ns1")
				require.NoError(t, err)

				names := make([]string, 0, len(list.Items))
				for _, d := range list.Items {
					names = append(names, d.Name)
				}
				assert.ElementsMatch(t, tc.want, names)
			})
		}
	})

	t.Run("CreateMonitoringDestination is authorised on the path namespace", func(t *testing.T) {
		t.Parallel()

		h := newHandler(t, newPolicy("p, role:test, monitoring-destinations, create, prod/ns1/*", "g, bob, role:test"))

		created, err := h.CreateMonitoringDestination(ctx, "prod", "ns1", destination("ns2", "pmm-prod"))
		require.NoError(t, err)
		assert.Equal(t, "ns1", created.Namespace)

		_, err = h.CreateMonitoringDestination(ctx, "prod", "ns2", destination("ns1", "pmm-prod"))
		require.ErrorIs(t, err, ErrInsufficientPermissions, "a namespace in the body must not borrow a grant")
	})

	t.Run("writes need their own verb", func(t *testing.T) {
		t.Parallel()

		h := newHandler(t, newPolicy("p, role:test, monitoring-destinations, read, prod/ns1/*", "g, bob, role:test"))

		_, err := h.CreateMonitoringDestination(ctx, "prod", "ns1", destination("ns1", "pmm-prod"))
		require.ErrorIs(t, err, ErrInsufficientPermissions)
		_, err = h.UpdateMonitoringDestination(ctx, "prod", "ns1", "pmm-prod", destination("ns1", "pmm-prod"))
		require.ErrorIs(t, err, ErrInsufficientPermissions)
		err = h.DeleteMonitoringDestination(ctx, "prod", "ns1", "pmm-prod")
		require.ErrorIs(t, err, ErrInsufficientPermissions)
	})

	t.Run("update and delete are authorised on the path name", func(t *testing.T) {
		t.Parallel()

		h := newHandler(t, newPolicy(
			"p, role:test, monitoring-destinations, update, prod/ns1/pmm-prod",
			"p, role:test, monitoring-destinations, delete, prod/ns1/pmm-prod",
			"g, bob, role:test",
		))

		updated, err := h.UpdateMonitoringDestination(ctx, "prod", "ns1", "pmm-prod", destination("ns1", "prometheus"))
		require.NoError(t, err)
		assert.Equal(t, "pmm-prod", updated.Name)
		_, err = h.UpdateMonitoringDestination(ctx, "prod", "ns1", "prometheus", destination("ns1", "pmm-prod"))
		require.ErrorIs(t, err, ErrInsufficientPermissions)

		require.NoError(t, h.DeleteMonitoringDestination(ctx, "prod", "ns1", "pmm-prod"))
		require.ErrorIs(t, h.DeleteMonitoringDestination(ctx, "prod", "ns1", "prometheus"), ErrInsufficientPermissions)
	})
}

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
