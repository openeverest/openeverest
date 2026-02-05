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
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	enginefeatureseverestv1alpha1 "github.com/percona/everest-operator/api/enginefeatures.everest/v1alpha1"
	"github.com/percona/everest/api"
	"github.com/percona/everest/internal/server/handlers"
	"github.com/percona/everest/pkg/common"
	"github.com/percona/everest/pkg/rbac"
)

const (
	namespace = "default"
	shdcName  = "shdc-test"
)

func Test_rbacHandler_CreateSplitHorizonDNSConfig(t *testing.T) {
	t.Parallel()
	data := func() *handlers.MockHandler {
		next := handlers.MockHandler{}
		next.On("CreateSplitHorizonDNSConfig",
			mock.Anything,
			mock.Anything,
		).
			Return(
				&enginefeatureseverestv1alpha1.SplitHorizonDNSConfig{}, nil,
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
			desc: "all actions for all everestfeatures/splithorizondnsconfigs in all namespaces",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, *, */*", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs),
				"g, bob, role:test",
			),
		},
		{
			desc: "all actions for default/shdc-test-1",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, *, %s/%s", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs, namespace, shdcName),
				"g, bob, role:test",
			),
		},
		{
			desc: "create only action for all everestfeatures/splithorizondnsconfigs in namespace",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, create, %s/*", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs, namespace),
				"g, bob, role:test",
			),
		},
		{
			desc: "create only action for all everestfeatures/splithorizondnsconfigs in all namespaces",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, create, */*", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs),
				"g, bob, role:test",
			),
		},
		{
			desc: "create only action for particular everestfeatures/splithorizondnsconfigs in namespace",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, create, %s/%s", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs, namespace, shdcName),
				"g, bob, role:test",
			),
		},
		{
			desc: "create only action for particular everestfeatures/splithorizondnsconfigs in another namespace",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, create, some/%s", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs, shdcName),
				"g, bob, role:test",
			),
			wantErr: ErrInsufficientPermissions,
		},
		{
			desc: "create only action for another everestfeatures/splithorizondnsconfigs in namespace",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, create, %s/some", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs, namespace),
				"g, bob, role:test",
			),
			wantErr: ErrInsufficientPermissions,
		},
		// the other methods
		{
			desc: "update only action for all everestfeatures/splithorizondnsconfigs in namespace",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, update, %s/*", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs, namespace),
				"g, bob, role:test",
			),
			wantErr: ErrInsufficientPermissions,
		},
		{
			desc: "update only action for everestfeatures/splithorizondnsconfigs in all namespaces",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, update, */*", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs),
				"g, bob, role:test",
			),
			wantErr: ErrInsufficientPermissions,
		},
		{
			desc: "read only action for all everestfeatures/splithorizondnsconfigs in namespace",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, read, %s/*", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs, namespace),
				"g, bob, role:test",
			),
			wantErr: ErrInsufficientPermissions,
		},
		{
			desc: "read only action for everestfeatures/splithorizondnsconfigs in all namespaces",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, read, */*", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs),
				"g, bob, role:test",
			),
			wantErr: ErrInsufficientPermissions,
		},
		{
			desc: "delete only action for all everestfeatures/splithorizondnsconfigs in namespace",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, delete, %s/*", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs, namespace),
				"g, bob, role:test",
			),
			wantErr: ErrInsufficientPermissions,
		},
		{
			desc: "delete only action for everestfeatures/splithorizondnsconfigs in all namespaces",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, delete, */*", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs),
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
			_, err = h.CreateSplitHorizonDNSConfig(ctx, &enginefeatureseverestv1alpha1.SplitHorizonDNSConfig{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      shdcName,
				},
			})
			assert.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func Test_rbacHandler_DeleteSplitHorizonDNSConfig(t *testing.T) {
	t.Parallel()
	data := func() *handlers.MockHandler {
		next := handlers.MockHandler{}
		next.On("DeleteSplitHorizonDNSConfig",
			mock.Anything,
			mock.Anything,
			mock.Anything,
		).
			Return(
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
			desc: "all actions for all everestfeatures/splithorizondnsconfigs in all namespaces",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, *, */*", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs),
				"g, bob, role:test",
			),
		},
		{
			desc: "all actions for default/shdc-test-1",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, *, %s/%s", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs, namespace, shdcName),
				"g, bob, role:test",
			),
		},
		{
			desc: "delete only action for all everestfeatures/splithorizondnsconfigs in namespace",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, delete, %s/*", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs, namespace),
				"g, bob, role:test",
			),
		},
		{
			desc: "delete only action for all everestfeatures/splithorizondnsconfigs in all namespaces",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, delete, */*", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs),
				"g, bob, role:test",
			),
		},
		{
			desc: "delete only action for particular everestfeatures/splithorizondnsconfigs in namespace",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, delete, %s/%s", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs, namespace, shdcName),
				"g, bob, role:test",
			),
		},
		{
			desc: "delete only action for particular everestfeatures/splithorizondnsconfigs in another namespace",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, delete, some/%s", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs, shdcName),
				"g, bob, role:test",
			),
			wantErr: ErrInsufficientPermissions,
		},
		{
			desc: "delete only action for another everestfeatures/splithorizondnsconfigs in namespace",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, delete, %s/some", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs, namespace),
				"g, bob, role:test",
			),
			wantErr: ErrInsufficientPermissions,
		},
		// the other methods
		{
			desc: "update only action for all everestfeatures/splithorizondnsconfigs in namespace",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, update, %s/*", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs, namespace),
				"g, bob, role:test",
			),
			wantErr: ErrInsufficientPermissions,
		},
		{
			desc: "update only action for everestfeatures/splithorizondnsconfigs in all namespaces",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, update, */*", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs),
				"g, bob, role:test",
			),
			wantErr: ErrInsufficientPermissions,
		},
		{
			desc: "read only action for all everestfeatures/splithorizondnsconfigs in namespace",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, read, %s/*", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs, namespace),
				"g, bob, role:test",
			),
			wantErr: ErrInsufficientPermissions,
		},
		{
			desc: "read only action for everestfeatures/splithorizondnsconfigs in all namespaces",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, read, */*", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs),
				"g, bob, role:test",
			),
			wantErr: ErrInsufficientPermissions,
		},
		{
			desc: "create only action for all everestfeatures/splithorizondnsconfigs in namespace",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, create, %s/*", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs, namespace),
				"g, bob, role:test",
			),
			wantErr: ErrInsufficientPermissions,
		},
		{
			desc: "create only action for everestfeatures/splithorizondnsconfigs in all namespaces",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, create, */*", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs),
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
			err = h.DeleteSplitHorizonDNSConfig(ctx, namespace, shdcName)
			assert.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func Test_rbacHandler_GetSplitHorizonDNSConfig(t *testing.T) {
	t.Parallel()
	data := func() *handlers.MockHandler {
		next := handlers.MockHandler{}
		next.On("GetSplitHorizonDNSConfig",
			mock.Anything,
			mock.Anything,
			mock.Anything,
		).Return(
			&enginefeatureseverestv1alpha1.SplitHorizonDNSConfig{},
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
			desc: "all actions for all everestfeatures/splithorizondnsconfigs in all namespaces",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, *, */*", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs),
				"g, bob, role:test",
			),
		},
		{
			desc: "all actions for default/shdc-test-1",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, *, %s/%s", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs, namespace, shdcName),
				"g, bob, role:test",
			),
		},
		{
			desc: "read only action for all everestfeatures/splithorizondnsconfigs in namespace",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, read, %s/*", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs, namespace),
				"g, bob, role:test",
			),
		},
		{
			desc: "read only action for all everestfeatures/splithorizondnsconfigs in all namespaces",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, read, */*", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs),
				"g, bob, role:test",
			),
		},
		{
			desc: "read only action for particular everestfeatures/splithorizondnsconfigs in namespace",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, read, %s/%s", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs, namespace, shdcName),
				"g, bob, role:test",
			),
		},
		{
			desc: "read only action for particular everestfeatures/splithorizondnsconfigs in another namespace",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, read, some/%s", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs, shdcName),
				"g, bob, role:test",
			),
			wantErr: ErrInsufficientPermissions,
		},
		{
			desc: "read only action for another everestfeatures/splithorizondnsconfigs in namespace",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, read, %s/some", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs, namespace),
				"g, bob, role:test",
			),
			wantErr: ErrInsufficientPermissions,
		},
		// the other methods
		{
			desc: "update only action for all everestfeatures/splithorizondnsconfigs in namespace",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, update, %s/*", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs, namespace),
				"g, bob, role:test",
			),
			wantErr: ErrInsufficientPermissions,
		},
		{
			desc: "update only action for everestfeatures/splithorizondnsconfigs in all namespaces",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, update, */*", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs),
				"g, bob, role:test",
			),
			wantErr: ErrInsufficientPermissions,
		},
		{
			desc: "create only action for all everestfeatures/splithorizondnsconfigs in namespace",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, create, %s/*", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs, namespace),
				"g, bob, role:test",
			),
			wantErr: ErrInsufficientPermissions,
		},
		{
			desc: "create only action for everestfeatures/splithorizondnsconfigs in all namespaces",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, create, */*", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs),
				"g, bob, role:test",
			),
			wantErr: ErrInsufficientPermissions,
		},
		{
			desc: "delete only action for all everestfeatures/splithorizondnsconfigs in namespace",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, delete, %s/*", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs, namespace),
				"g, bob, role:test",
			),
			wantErr: ErrInsufficientPermissions,
		},
		{
			desc: "delete only action for everestfeatures/splithorizondnsconfigs in all namespaces",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, delete, */*", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs),
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
			_, err = h.GetSplitHorizonDNSConfig(ctx, namespace, shdcName)
			assert.ErrorIs(t, err, tc.wantErr)
		})
	}
}

func Test_rbacHandler_ListSplitHorizonDNSConfigs(t *testing.T) {
	t.Parallel()

	data := func() *handlers.MockHandler {
		next := handlers.MockHandler{}
		next.On("ListSplitHorizonDNSConfigs",
			mock.Anything,
			mock.Anything,
		).Return(
			&enginefeatureseverestv1alpha1.SplitHorizonDNSConfigList{
				Items: []enginefeatureseverestv1alpha1.SplitHorizonDNSConfig{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      fmt.Sprintf("%s-1", shdcName),
							Namespace: namespace,
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      fmt.Sprintf("%s-2", shdcName),
							Namespace: namespace,
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      fmt.Sprintf("%s-3", shdcName),
							Namespace: namespace,
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
		assert func(list *enginefeatureseverestv1alpha1.SplitHorizonDNSConfigList) bool
	}
	testCases := []testCase{
		{
			desc: "admin",
			policy: newPolicy(
				"g, bob, role:admin",
			),
			assert: func(list *enginefeatureseverestv1alpha1.SplitHorizonDNSConfigList) bool {
				return len(list.Items) == 3 &&
					slices.ContainsFunc(list.Items, func(bs enginefeatureseverestv1alpha1.SplitHorizonDNSConfig) bool {
						return bs.GetName() == fmt.Sprintf("%s-1", shdcName)
					}) &&
					slices.ContainsFunc(list.Items, func(bs enginefeatureseverestv1alpha1.SplitHorizonDNSConfig) bool {
						return bs.GetName() == fmt.Sprintf("%s-2", shdcName)
					}) &&
					slices.ContainsFunc(list.Items, func(bs enginefeatureseverestv1alpha1.SplitHorizonDNSConfig) bool {
						return bs.GetName() == fmt.Sprintf("%s-3", shdcName)
					})
			},
		},
		{
			desc: "all actions for all everestfeatures/splithorizondnsconfigs in all namespaces",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, *, */*", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs),
				"g, bob, role:test",
			),
			assert: func(list *enginefeatureseverestv1alpha1.SplitHorizonDNSConfigList) bool {
				return len(list.Items) == 3 &&
					slices.ContainsFunc(list.Items, func(bs enginefeatureseverestv1alpha1.SplitHorizonDNSConfig) bool {
						return bs.GetName() == fmt.Sprintf("%s-1", shdcName)
					}) &&
					slices.ContainsFunc(list.Items, func(bs enginefeatureseverestv1alpha1.SplitHorizonDNSConfig) bool {
						return bs.GetName() == fmt.Sprintf("%s-2", shdcName)
					}) &&
					slices.ContainsFunc(list.Items, func(bs enginefeatureseverestv1alpha1.SplitHorizonDNSConfig) bool {
						return bs.GetName() == fmt.Sprintf("%s-3", shdcName)
					})
			},
		},
		{
			desc: "all actions for default/shdc-test-1",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, *, %s/%s-1", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs, namespace, shdcName),
				"g, bob, role:test",
			),
			assert: func(list *enginefeatureseverestv1alpha1.SplitHorizonDNSConfigList) bool {
				return len(list.Items) == 1 &&
					slices.ContainsFunc(list.Items, func(bs enginefeatureseverestv1alpha1.SplitHorizonDNSConfig) bool {
						return bs.GetName() == fmt.Sprintf("%s-1", shdcName)
					})
			},
		},
		{
			desc: "read only action for all everestfeatures/splithorizondnsconfigs in namespace",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, read, %s/*", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs, namespace),
				"g, bob, role:test",
			),
			assert: func(list *enginefeatureseverestv1alpha1.SplitHorizonDNSConfigList) bool {
				return len(list.Items) == 3 &&
					slices.ContainsFunc(list.Items, func(bs enginefeatureseverestv1alpha1.SplitHorizonDNSConfig) bool {
						return bs.GetName() == fmt.Sprintf("%s-1", shdcName)
					}) &&
					slices.ContainsFunc(list.Items, func(bs enginefeatureseverestv1alpha1.SplitHorizonDNSConfig) bool {
						return bs.GetName() == fmt.Sprintf("%s-2", shdcName)
					}) &&
					slices.ContainsFunc(list.Items, func(bs enginefeatureseverestv1alpha1.SplitHorizonDNSConfig) bool {
						return bs.GetName() == fmt.Sprintf("%s-3", shdcName)
					})
			},
		},
		{
			desc: "read only action for all everestfeatures/splithorizondnsconfigs in all namespaces",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, read, */*", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs),
				"g, bob, role:test",
			),
			assert: func(list *enginefeatureseverestv1alpha1.SplitHorizonDNSConfigList) bool {
				return len(list.Items) == 3 &&
					slices.ContainsFunc(list.Items, func(bs enginefeatureseverestv1alpha1.SplitHorizonDNSConfig) bool {
						return bs.GetName() == fmt.Sprintf("%s-1", shdcName)
					}) &&
					slices.ContainsFunc(list.Items, func(bs enginefeatureseverestv1alpha1.SplitHorizonDNSConfig) bool {
						return bs.GetName() == fmt.Sprintf("%s-2", shdcName)
					}) &&
					slices.ContainsFunc(list.Items, func(bs enginefeatureseverestv1alpha1.SplitHorizonDNSConfig) bool {
						return bs.GetName() == fmt.Sprintf("%s-3", shdcName)
					})
			},
		},
		{
			desc: "read only action for shdc-test-1 in namespace",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, read, %s/%s-1", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs, namespace, shdcName),
				"g, bob, role:test",
			),
			assert: func(list *enginefeatureseverestv1alpha1.SplitHorizonDNSConfigList) bool {
				return len(list.Items) == 1 &&
					slices.ContainsFunc(list.Items, func(bs enginefeatureseverestv1alpha1.SplitHorizonDNSConfig) bool {
						return bs.GetName() == fmt.Sprintf("%s-1", shdcName)
					})
			},
		},
		{
			desc: "read only action for shdc-test-1 in another namespace",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, read, some/%s", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs, shdcName),
				"g, bob, role:test",
			),
			assert: func(list *enginefeatureseverestv1alpha1.SplitHorizonDNSConfigList) bool {
				return len(list.Items) == 0
			},
		},
		{
			desc: "read only action for another everestfeatures/splithorizondnsconfigs in namespace",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, read, %s/some", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs, namespace),
				"g, bob, role:test",
			),
			assert: func(list *enginefeatureseverestv1alpha1.SplitHorizonDNSConfigList) bool {
				return len(list.Items) == 0
			},
		},
		{
			desc: "read only action for shdc-test-1 and shdc-test-2 in namespace",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, read, %s/%s-1", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs, namespace, shdcName),
				fmt.Sprintf("p, role:test, %s, read, %s/%s-2", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs, namespace, shdcName),
				"g, bob, role:test",
			),
			assert: func(list *enginefeatureseverestv1alpha1.SplitHorizonDNSConfigList) bool {
				return len(list.Items) == 2 &&
					slices.ContainsFunc(list.Items, func(bs enginefeatureseverestv1alpha1.SplitHorizonDNSConfig) bool {
						return bs.GetName() == fmt.Sprintf("%s-1", shdcName)
					}) &&
					slices.ContainsFunc(list.Items, func(bs enginefeatureseverestv1alpha1.SplitHorizonDNSConfig) bool {
						return bs.GetName() == fmt.Sprintf("%s-2", shdcName)
					})
			},
		},
		// the other methods
		{
			desc: "update only action for all everestfeatures/splithorizondnsconfigs in namespace",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, update, %s/*", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs, namespace),
				"g, bob, role:test",
			),
			assert: func(list *enginefeatureseverestv1alpha1.SplitHorizonDNSConfigList) bool {
				return len(list.Items) == 0
			},
		},
		{
			desc: "update only action for everestfeatures/splithorizondnsconfigs in all namespaces",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, update, */*", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs),
				"g, bob, role:test",
			),
			assert: func(list *enginefeatureseverestv1alpha1.SplitHorizonDNSConfigList) bool {
				return len(list.Items) == 0
			},
		},
		{
			desc: "create only action for all everestfeatures/splithorizondnsconfigs in namespace",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, create, %s/*", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs, namespace),
				"g, bob, role:test",
			),
			assert: func(list *enginefeatureseverestv1alpha1.SplitHorizonDNSConfigList) bool {
				return len(list.Items) == 0
			},
		},
		{
			desc: "create only action for everestfeatures/splithorizondnsconfigs in all namespaces",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, create, */*", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs),
				"g, bob, role:test",
			),
			assert: func(list *enginefeatureseverestv1alpha1.SplitHorizonDNSConfigList) bool {
				return len(list.Items) == 0
			},
		},
		{
			desc: "delete only action for all everestfeatures/splithorizondnsconfigs in namespace",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, delete, %s/*", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs, namespace),
				"g, bob, role:test",
			),
			assert: func(list *enginefeatureseverestv1alpha1.SplitHorizonDNSConfigList) bool {
				return len(list.Items) == 0
			},
		},
		{
			desc: "delete only action for everestfeatures/splithorizondnsconfigs in all namespaces",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, delete, */*", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs),
				"g, bob, role:test",
			),
			assert: func(list *enginefeatureseverestv1alpha1.SplitHorizonDNSConfigList) bool {
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

			list, err := h.ListSplitHorizonDNSConfigs(ctx, namespace)
			require.NoError(t, err)
			assert.Condition(t, func() bool {
				return tc.assert(list)
			})
		})
	}
}

func Test_rbacHandler_UpdateSplitHorizonDNSConfig(t *testing.T) {
	t.Parallel()

	data := func() *handlers.MockHandler {
		next := handlers.MockHandler{}
		next.On("UpdateSplitHorizonDNSConfig",
			mock.Anything,
			mock.Anything,
			mock.Anything,
			mock.Anything,
		).
			Return(
				&enginefeatureseverestv1alpha1.SplitHorizonDNSConfig{}, nil,
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
			desc: "all actions for all everestfeatures/splithorizondnsconfigs in all namespaces",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, *, */*", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs),
				"g, bob, role:test",
			),
		},
		{
			desc: "all actions for default/shdc-test-1",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, *, %s/%s", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs, namespace, shdcName),
				"g, bob, role:test",
			),
		},
		{
			desc: "update only action for all everestfeatures/splithorizondnsconfigs in namespace",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, update, %s/*", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs, namespace),
				"g, bob, role:test",
			),
		},
		{
			desc: "update only action for all everestfeatures/splithorizondnsconfigs in all namespaces",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, update, */*", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs),
				"g, bob, role:test",
			),
		},
		{
			desc: "update only action for particular everestfeatures/splithorizondnsconfigs in namespace",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, update, %s/%s", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs, namespace, shdcName),
				"g, bob, role:test",
			),
		},
		{
			desc: "update only action for particular everestfeatures/splithorizondnsconfigs in another namespace",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, update, some/%s", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs, shdcName),
				"g, bob, role:test",
			),
			wantErr: ErrInsufficientPermissions,
		},
		{
			desc: "update only action for another everestfeatures/splithorizondnsconfigs in namespace",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, update, %s/some", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs, namespace),
				"g, bob, role:test",
			),
			wantErr: ErrInsufficientPermissions,
		},
		// the other methods
		{
			desc: "create only action for all everestfeatures/splithorizondnsconfigs in namespace",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, create, %s/*", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs, namespace),
				"g, bob, role:test",
			),
			wantErr: ErrInsufficientPermissions,
		},
		{
			desc: "create only action for everestfeatures/splithorizondnsconfigs in all namespaces",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, create, */*", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs),
				"g, bob, role:test",
			),
			wantErr: ErrInsufficientPermissions,
		},
		{
			desc: "read only action for all everestfeatures/splithorizondnsconfigs in namespace",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, read, %s/*", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs, namespace),
				"g, bob, role:test",
			),
			wantErr: ErrInsufficientPermissions,
		},
		{
			desc: "read only action for everestfeatures/splithorizondnsconfigs in all namespaces",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, read, */*", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs),
				"g, bob, role:test",
			),
			wantErr: ErrInsufficientPermissions,
		},
		{
			desc: "delete only action for all everestfeatures/splithorizondnsconfigs in namespace",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, delete, %s/*", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs, namespace),
				"g, bob, role:test",
			),
			wantErr: ErrInsufficientPermissions,
		},
		{
			desc: "delete only action for everestfeatures/splithorizondnsconfigs in all namespaces",
			policy: newPolicy(
				fmt.Sprintf("p, role:test, %s, delete, */*", rbac.ResourceEngineFeatures_SplitHorizonDNSConfigs),
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
			_, err = h.UpdateSplitHorizonDNSConfig(ctx, namespace, shdcName, &api.SplitHorizonDNSConfigUpdateParams{})
			assert.ErrorIs(t, err, tc.wantErr)
		})
	}
}
