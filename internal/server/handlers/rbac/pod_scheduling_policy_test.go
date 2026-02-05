// everest
// Copyright (C) 2025 Percona LLC
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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
	"github.com/percona/everest/internal/server/handlers"
	"github.com/percona/everest/pkg/common"
	"github.com/percona/everest/pkg/rbac"
)

func TestRBAC_ListPodSchedulingPolicies(t *testing.T) {
	t.Parallel()

	data := func() *handlers.MockHandler {
		next := handlers.MockHandler{}
		next.On("ListPodSchedulingPolicies",
			mock.Anything,
			mock.Anything,
		).Return(
			&everestv1alpha1.PodSchedulingPolicyList{
				Items: []everestv1alpha1.PodSchedulingPolicy{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:       "everest-default-mysql",
							Finalizers: []string{everestv1alpha1.ReadOnlyFinalizer},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:       "everest-default-postgresql",
							Finalizers: []string{everestv1alpha1.ReadOnlyFinalizer},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:       "everest-default-mongodb",
							Finalizers: []string{everestv1alpha1.ReadOnlyFinalizer},
						},
					},
				},
			},
			nil,
		)
		return &next
	}

	type testCase struct {
		desc   string
		policy string
		outLen int
		assert func(*everestv1alpha1.PodSchedulingPolicyList) bool
	}
	testCases := []testCase{
		{
			desc: "admin",
			policy: newPolicy(
				"g, bob, role:admin",
			),
			assert: func(list *everestv1alpha1.PodSchedulingPolicyList) bool {
				return len(list.Items) == 3 &&
					slices.ContainsFunc(list.Items, func(bs everestv1alpha1.PodSchedulingPolicy) bool {
						return bs.GetName() == "everest-default-mysql"
					}) &&
					slices.ContainsFunc(list.Items, func(bs everestv1alpha1.PodSchedulingPolicy) bool {
						return bs.GetName() == "everest-default-postgresql"
					}) &&
					slices.ContainsFunc(list.Items, func(bs everestv1alpha1.PodSchedulingPolicy) bool {
						return bs.GetName() == "everest-default-mongodb"
					})
			},
		},
		{
			desc: "all actions for all pod-scheduling-policies",
			policy: newPolicy(
				"p, role:test, pod-scheduling-policies, *, *",
				"g, bob, role:test",
			),
			assert: func(list *everestv1alpha1.PodSchedulingPolicyList) bool {
				return len(list.Items) == 3 &&
					slices.ContainsFunc(list.Items, func(bs everestv1alpha1.PodSchedulingPolicy) bool {
						return bs.GetName() == "everest-default-mysql"
					}) &&
					slices.ContainsFunc(list.Items, func(bs everestv1alpha1.PodSchedulingPolicy) bool {
						return bs.GetName() == "everest-default-postgresql"
					}) &&
					slices.ContainsFunc(list.Items, func(bs everestv1alpha1.PodSchedulingPolicy) bool {
						return bs.GetName() == "everest-default-mongodb"
					})
			},
		},
		{
			desc: "all actions for 'everest-default-mysql'",
			policy: newPolicy(
				"p, role:test, pod-scheduling-policies, *, everest-default-mysql",
				"g, bob, role:test",
			),
			assert: func(list *everestv1alpha1.PodSchedulingPolicyList) bool {
				return len(list.Items) == 1 &&
					slices.ContainsFunc(list.Items, func(bs everestv1alpha1.PodSchedulingPolicy) bool {
						return bs.GetName() == "everest-default-mysql"
					})
			},
		},
		{
			desc: "create only action for all pod-scheduling-policies",
			policy: newPolicy(
				"p, role:test, pod-scheduling-policies, create, *",
				"g, bob, role:test",
			),
			assert: func(list *everestv1alpha1.PodSchedulingPolicyList) bool {
				return len(list.Items) == 0
			},
		},
		{
			desc: "create only action for 'test-policy-1'",
			policy: newPolicy(
				"p, role:test, pod-scheduling-policies, create, test-policy-1",
				"g, bob, role:test",
			),
			assert: func(list *everestv1alpha1.PodSchedulingPolicyList) bool {
				return len(list.Items) == 0
			},
		},
		{
			desc: "update only action for all pod-scheduling-policies",
			policy: newPolicy(
				"p, role:test, pod-scheduling-policies, update, *",
				"g, bob, role:test",
			),
			assert: func(list *everestv1alpha1.PodSchedulingPolicyList) bool {
				return len(list.Items) == 0
			},
		},
		{
			desc: "update only action for 'test-policy-1'",
			policy: newPolicy(
				"p, role:test, pod-scheduling-policies, update, test-policy-1",
				"g, bob, role:test",
			),
			assert: func(list *everestv1alpha1.PodSchedulingPolicyList) bool {
				return len(list.Items) == 0
			},
		},
		{
			desc: "read only for all pod-scheduling-policies",
			policy: newPolicy(
				"p, role:test, pod-scheduling-policies, read, *",
				"g, bob, role:test",
			),
			assert: func(list *everestv1alpha1.PodSchedulingPolicyList) bool {
				return len(list.Items) == 3 &&
					slices.ContainsFunc(list.Items, func(bs everestv1alpha1.PodSchedulingPolicy) bool {
						return bs.GetName() == "everest-default-mysql"
					}) &&
					slices.ContainsFunc(list.Items, func(bs everestv1alpha1.PodSchedulingPolicy) bool {
						return bs.GetName() == "everest-default-postgresql"
					}) &&
					slices.ContainsFunc(list.Items, func(bs everestv1alpha1.PodSchedulingPolicy) bool {
						return bs.GetName() == "everest-default-mongodb"
					})
			},
		},
		{
			desc: "read only for everest-default-mysql",
			policy: newPolicy(
				"p, role:test, pod-scheduling-policies, read, everest-default-mysql",
				"g, bob, role:test",
			),
			assert: func(list *everestv1alpha1.PodSchedulingPolicyList) bool {
				return len(list.Items) == 1 &&
					slices.ContainsFunc(list.Items, func(bs everestv1alpha1.PodSchedulingPolicy) bool {
						return bs.GetName() == "everest-default-mysql"
					})
			},
		},
		{
			desc: "read only for everest-default-mysql and everest-default-postgresql",
			policy: newPolicy(
				"p, role:test, pod-scheduling-policies, read, everest-default-mysql",
				"p, role:test, pod-scheduling-policies, read, everest-default-postgresql",
				"g, bob, role:test",
			),
			assert: func(list *everestv1alpha1.PodSchedulingPolicyList) bool {
				return len(list.Items) == 2 &&
					slices.ContainsFunc(list.Items, func(bs everestv1alpha1.PodSchedulingPolicy) bool {
						return bs.GetName() == "everest-default-mysql"
					}) &&
					slices.ContainsFunc(list.Items, func(bs everestv1alpha1.PodSchedulingPolicy) bool {
						return bs.GetName() == "everest-default-postgresql"
					})
			},
		},
		{
			desc: "delete only action for all pod-scheduling-policies",
			policy: newPolicy(
				"p, role:test, pod-scheduling-policies, delete, *",
				"g, bob, role:test",
			),
			assert: func(list *everestv1alpha1.PodSchedulingPolicyList) bool {
				return len(list.Items) == 0
			},
		},
		{
			desc: "delete only action for 'everest-default-mysql'",
			policy: newPolicy(
				"p, role:test, pod-scheduling-policies, delete, everest-default-mysql",
				"g, bob, role:test",
			),
			assert: func(list *everestv1alpha1.PodSchedulingPolicyList) bool {
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
			next := data()

			h := &rbacHandler{
				next:       next,
				log:        zap.NewNop().Sugar(),
				enforcer:   enf,
				userGetter: testUserGetter,
			}

			list, err := h.ListPodSchedulingPolicies(ctx, nil)
			require.NoError(t, err)
			assert.Condition(t, func() bool {
				return tc.assert(list)
			})
		})
	}
}

func TestRBAC_GetPodSchedulingPolicy(t *testing.T) {
	t.Parallel()

	data := func() *handlers.MockHandler {
		next := handlers.MockHandler{}
		next.On("GetPodSchedulingPolicy",
			mock.Anything,
			mock.Anything,
		).Return(
			&everestv1alpha1.PodSchedulingPolicy{},
			nil,
		)
		return &next
	}

	type testCase struct {
		desc    string
		policy  string
		wantErr error
	}
	testCases := []testCase{
		{
			desc: "admin",
			policy: newPolicy(
				"g, bob, role:admin",
			),
		},
		{
			desc: "all actions for all pod-scheduling-policies",
			policy: newPolicy(
				"p, role:test, pod-scheduling-policies, *, *",
				"g, bob, role:test",
			),
		},
		{
			desc: "all actions for 'test-policy-1'",
			policy: newPolicy(
				"p, role:test, pod-scheduling-policies, *, test-policy-1",
				"g, bob, role:test",
			),
		},
		{
			desc: "all actions for some",
			policy: newPolicy(
				"p, role:test, pod-scheduling-policies, *, some",
				"g, bob, role:test",
			),
			wantErr: ErrInsufficientPermissions,
		},
		{
			desc: "create only action for all pod-scheduling-policies",
			policy: newPolicy(
				"p, role:test, pod-scheduling-policies, create, *",
				"g, bob, role:test",
			),
			wantErr: ErrInsufficientPermissions,
		},
		{
			desc: "create only action for 'test-policy-1'",
			policy: newPolicy(
				"p, role:test, pod-scheduling-policies, create, test-policy-1",
				"g, bob, role:test",
			),
			wantErr: ErrInsufficientPermissions,
		},
		{
			desc: "update only action for all pod-scheduling-policies",
			policy: newPolicy(
				"p, role:test, pod-scheduling-policies, update, *",
				"g, bob, role:test",
			),
			wantErr: ErrInsufficientPermissions,
		},
		{
			desc: "update only action for 'test-policy-1'",
			policy: newPolicy(
				"p, role:test, pod-scheduling-policies, update, test-policy-1",
				"g, bob, role:test",
			),
			wantErr: ErrInsufficientPermissions,
		},
		{
			desc: "read only action for all pod-scheduling-policies",
			policy: newPolicy(
				"p, role:test, pod-scheduling-policies, read, *",
				"g, bob, role:test",
			),
		},
		{
			desc: "read only action for 'test-policy-1'",
			policy: newPolicy(
				"p, role:test, pod-scheduling-policies, read, test-policy-1",
				"g, bob, role:test",
			),
		},
		{
			desc: "read only action for some",
			policy: newPolicy(
				"p, role:test, pod-scheduling-policies, read, some",
				"g, bob, role:test",
			),
			wantErr: ErrInsufficientPermissions,
		},
		{
			desc: "delete only action for all pod-scheduling-policies",
			policy: newPolicy(
				"p, role:test, pod-scheduling-policies, delete, *",
				"g, bob, role:test",
			),
			wantErr: ErrInsufficientPermissions,
		},
		{
			desc: "delete only action for 'test-policy-1'",
			policy: newPolicy(
				"p, role:test, pod-scheduling-policies, delete, test-policy-1",
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

			next := data()

			h := &rbacHandler{
				next:       next,
				log:        zap.NewNop().Sugar(),
				enforcer:   enf,
				userGetter: testUserGetter,
			}
			_, err = h.GetPodSchedulingPolicy(ctx, "test-policy-1")
			assert.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestRBAC_CreatePodSchedulingPolicy(t *testing.T) {
	t.Parallel()

	data := func() *handlers.MockHandler {
		next := handlers.MockHandler{}
		next.On("CreatePodSchedulingPolicy",
			mock.Anything,
			mock.Anything,
		).
			Return(
				&everestv1alpha1.PodSchedulingPolicy{}, nil,
			)
		return &next
	}

	type testCase struct {
		desc    string
		policy  string
		wantErr error
	}
	testCases := []testCase{
		{
			desc: "admin",
			policy: newPolicy(
				"g, bob, role:admin",
			),
		},
		{
			desc: "all actions for all pod-scheduling-policies",
			policy: newPolicy(
				"p, role:test, pod-scheduling-policies, *, *",
				"g, bob, role:test",
			),
		},
		{
			desc: "all actions for 'test-policy-1'",
			policy: newPolicy(
				"p, role:test, pod-scheduling-policies, *, test-policy-1",
				"g, bob, role:test",
			),
		},
		{
			desc: "create only action for all pod-scheduling-policies",
			policy: newPolicy(
				"p, role:test, pod-scheduling-policies, create, *",
				"g, bob, role:test",
			),
		},
		{
			desc: "create only action for 'test-policy-1'",
			policy: newPolicy(
				"p, role:test, pod-scheduling-policies, create, test-policy-1",
				"g, bob, role:test",
			),
		},
		{
			desc: "create only action for some",
			policy: newPolicy(
				"p, role:test, pod-scheduling-policies, create, some",
				"g, bob, role:test",
			),
			wantErr: ErrInsufficientPermissions,
		},
		{
			desc: "update only action for all pod-scheduling-policies",
			policy: newPolicy(
				"p, role:test, pod-scheduling-policies, update, *",
				"g, bob, role:test",
			),
			wantErr: ErrInsufficientPermissions,
		},
		{
			desc: "update only action for 'test-policy-1'",
			policy: newPolicy(
				"p, role:test, pod-scheduling-policies, update, test-policy-1",
				"g, bob, role:test",
			),
			wantErr: ErrInsufficientPermissions,
		},
		{
			desc: "read only action for all pod-scheduling-policies",
			policy: newPolicy(
				"p, role:test, pod-scheduling-policies, read, *",
				"g, bob, role:test",
			),
			wantErr: ErrInsufficientPermissions,
		},
		{
			desc: "read only action for 'test-policy-1'",
			policy: newPolicy(
				"p, role:test, pod-scheduling-policies, read, test-policy-1",
				"g, bob, role:test",
			),
			wantErr: ErrInsufficientPermissions,
		},
		{
			desc: "delete only action for all pod-scheduling-policies",
			policy: newPolicy(
				"p, role:test, pod-scheduling-policies, delete, *",
				"g, bob, role:test",
			),
			wantErr: ErrInsufficientPermissions,
		},
		{
			desc: "delete only action for 'test-policy-1'",
			policy: newPolicy(
				"p, role:test, pod-scheduling-policies, delete, test-policy-1",
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
			next := data()

			h := &rbacHandler{
				next:       next,
				log:        zap.NewNop().Sugar(),
				enforcer:   enf,
				userGetter: testUserGetter,
			}
			_, err = h.CreatePodSchedulingPolicy(ctx, &everestv1alpha1.PodSchedulingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-policy-1",
				},
			})
			assert.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestRBAC_UpdatePodSchedulingPolicy(t *testing.T) {
	t.Parallel()

	data := func() *handlers.MockHandler {
		next := handlers.MockHandler{}
		next.On("UpdatePodSchedulingPolicy",
			mock.Anything,
			mock.Anything,
		).
			Return(
				&everestv1alpha1.PodSchedulingPolicy{}, nil,
			)
		return &next
	}

	type testCase struct {
		desc    string
		policy  string
		wantErr error
	}
	testCases := []testCase{
		{
			desc: "admin",
			policy: newPolicy(
				"g, bob, role:admin",
			),
		},
		{
			desc: "all actions for all pod-scheduling-policies",
			policy: newPolicy(
				"p, role:test, pod-scheduling-policies, *, *",
				"g, bob, role:test",
			),
		},
		{
			desc: "all actions for 'test-policy-1'",
			policy: newPolicy(
				"p, role:test, pod-scheduling-policies, *, test-policy-1",
				"g, bob, role:test",
			),
		},
		{
			desc: "create only action for all pod-scheduling-policies",
			policy: newPolicy(
				"p, role:test, pod-scheduling-policies, create, *",
				"g, bob, role:test",
			),
			wantErr: ErrInsufficientPermissions,
		},
		{
			desc: "create only action for 'test-policy-1'",
			policy: newPolicy(
				"p, role:test, pod-scheduling-policies, create, test-policy-1",
				"g, bob, role:test",
			),
			wantErr: ErrInsufficientPermissions,
		},
		{
			desc: "update only action for all pod-scheduling-policies",
			policy: newPolicy(
				"p, role:test, pod-scheduling-policies, update, *",
				"g, bob, role:test",
			),
		},
		{
			desc: "update only action for 'test-policy-1'",
			policy: newPolicy(
				"p, role:test, pod-scheduling-policies, update, test-policy-1",
				"g, bob, role:test",
			),
		},
		{
			desc: "update only action for some",
			policy: newPolicy(
				"p, role:test, pod-scheduling-policies, update, some",
				"g, bob, role:test",
			),
			wantErr: ErrInsufficientPermissions,
		},
		{
			desc: "read only action for all pod-scheduling-policies",
			policy: newPolicy(
				"p, role:test, pod-scheduling-policies, read, *",
				"g, bob, role:test",
			),
			wantErr: ErrInsufficientPermissions,
		},
		{
			desc: "read only action for 'test-policy-1'",
			policy: newPolicy(
				"p, role:test, pod-scheduling-policies, read, test-policy-1",
				"g, bob, role:test",
			),
			wantErr: ErrInsufficientPermissions,
		},
		{
			desc: "delete only action for all pod-scheduling-policies",
			policy: newPolicy(
				"p, role:test, pod-scheduling-policies, delete, *",
				"g, bob, role:test",
			),
			wantErr: ErrInsufficientPermissions,
		},
		{
			desc: "delete only action for 'test-policy-1'",
			policy: newPolicy(
				"p, role:test, pod-scheduling-policies, delete, test-policy-1",
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
			next := data()

			h := &rbacHandler{
				next:       next,
				log:        zap.NewNop().Sugar(),
				enforcer:   enf,
				userGetter: testUserGetter,
			}
			_, err = h.UpdatePodSchedulingPolicy(ctx, &everestv1alpha1.PodSchedulingPolicy{ObjectMeta: metav1.ObjectMeta{
				Name:      "test-policy-1",
				Namespace: common.SystemNamespace,
			}})
			assert.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func TestRBAC_DeletePodSchedulingPolicy(t *testing.T) {
	t.Parallel()

	data := func() *handlers.MockHandler {
		next := handlers.MockHandler{}
		next.On("DeletePodSchedulingPolicy",
			mock.Anything,
			mock.Anything,
		).
			Return(
				nil,
			)
		return &next
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
			desc: "all actions for all pod-scheduling-policies",
			policy: newPolicy(
				"p, role:test, pod-scheduling-policies, *, *",
				"g, bob, role:test",
			),
		},
		{
			desc: "all actions for 'test-policy-1'",
			policy: newPolicy(
				"p, role:test, pod-scheduling-policies, *, test-policy-1",
				"g, bob, role:test",
			),
		},
		{
			desc: "create only action for all pod-scheduling-policies",
			policy: newPolicy(
				"p, role:test, pod-scheduling-policies, create, *",
				"g, bob, role:test",
			),
			wantErr: ErrInsufficientPermissions,
		},
		{
			desc: "create only action for 'test-policy-1'",
			policy: newPolicy(
				"p, role:test, pod-scheduling-policies, create, test-policy-1",
				"g, bob, role:test",
			),
			wantErr: ErrInsufficientPermissions,
		},
		{
			desc: "update only action for all pod-scheduling-policies",
			policy: newPolicy(
				"p, role:test, pod-scheduling-policies, update, *",
				"g, bob, role:test",
			),
			wantErr: ErrInsufficientPermissions,
		},
		{
			desc: "update only action for 'test-policy-1'",
			policy: newPolicy(
				"p, role:test, pod-scheduling-policies, update, test-policy-1",
				"g, bob, role:test",
			),
			wantErr: ErrInsufficientPermissions,
		},
		{
			desc: "read only action for all pod-scheduling-policies",
			policy: newPolicy(
				"p, role:test, pod-scheduling-policies, read, *",
				"g, bob, role:test",
			),
			wantErr: ErrInsufficientPermissions,
		},
		{
			desc: "read only action for 'test-policy-1'",
			policy: newPolicy(
				"p, role:test, pod-scheduling-policies, read, test-policy-1",
				"g, bob, role:test",
			),
			wantErr: ErrInsufficientPermissions,
		},
		{
			desc: "delete only action for all pod-scheduling-policies",
			policy: newPolicy(
				"p, role:test, pod-scheduling-policies, delete, *",
				"g, bob, role:test",
			),
		},
		{
			desc: "delete only action for 'test-policy-1'",
			policy: newPolicy(
				"p, role:test, pod-scheduling-policies, delete, test-policy-1",
				"g, bob, role:test",
			),
		},
		{
			desc: "delete only action for some",
			policy: newPolicy(
				"p, role:test, pod-scheduling-policies, delete, some",
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
			next := data()

			h := &rbacHandler{
				next:       next,
				log:        zap.NewNop().Sugar(),
				enforcer:   enf,
				userGetter: testUserGetter,
			}
			err = h.DeletePodSchedulingPolicy(ctx, "test-policy-1")
			assert.ErrorIs(t, err, tc.wantErr)
		})
	}
}
