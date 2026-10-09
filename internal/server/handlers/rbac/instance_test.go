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
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	apicommon "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	api "github.com/openeverest/openeverest/v2/internal/server/api"
	"github.com/openeverest/openeverest/v2/internal/server/handlers"
	valhandler "github.com/openeverest/openeverest/v2/internal/server/handlers/validation"
	"github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

func TestRBAC_Instance(t *testing.T) {
	t.Parallel()

	mockInstances := func() *handlers.MockHandler {
		h := &handlers.MockHandler{}
		h.On("ListInstances", mock.Anything, mock.Anything, mock.Anything).Return(
			&corev1alpha1.InstanceList{
				Items: []corev1alpha1.Instance{
					{ObjectMeta: metav1.ObjectMeta{Name: "db1", Namespace: "ns1"}},
					{ObjectMeta: metav1.ObjectMeta{Name: "db2", Namespace: "ns1"}},
					{ObjectMeta: metav1.ObjectMeta{Name: "db3", Namespace: "ns1"}},
				},
			}, nil,
		)
		h.On("GetInstance", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
			&corev1alpha1.Instance{ObjectMeta: metav1.ObjectMeta{Name: "db1", Namespace: "ns1"}},
			nil,
		)
		h.On("CreateInstance", mock.Anything, mock.Anything, mock.Anything).Return(
			&corev1alpha1.Instance{ObjectMeta: metav1.ObjectMeta{Name: "db1", Namespace: "ns1"}},
			nil,
		)
		h.On("UpdateInstance", mock.Anything, mock.Anything, mock.Anything).Return(
			&corev1alpha1.Instance{ObjectMeta: metav1.ObjectMeta{Name: "db1", Namespace: "ns1"}},
			nil,
		)
		h.On("PatchInstance", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
			&corev1alpha1.Instance{ObjectMeta: metav1.ObjectMeta{Name: "db1", Namespace: "ns1"}},
			nil,
		)
		h.On("DeleteInstance", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		host := "db1.ns1.svc"
		h.On("GetInstanceConnection", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
			&api.InstanceConnectionDetails{Host: &host},
			nil,
		)
		return h
	}

	t.Run("ListInstances", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			desc    string
			cluster string
			ns      string
			policy  string
			assert  func(list *corev1alpha1.InstanceList) bool
		}{
			{
				desc:    "admin",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"g, bob, role:admin",
				),
				assert: func(list *corev1alpha1.InstanceList) bool {
					return len(list.Items) == 3
				},
			},
			{
				desc:    "all instances on cluster and namespace",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"p, role:test, instances, read, prod/ns1/*",
					"g, bob, role:test",
				),
				assert: func(list *corev1alpha1.InstanceList) bool {
					return len(list.Items) == 3
				},
			},
			{
				desc:    "specific instance",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"p, role:test, instances, read, prod/ns1/db1",
					"g, bob, role:test",
				),
				assert: func(list *corev1alpha1.InstanceList) bool {
					return len(list.Items) == 1 && list.Items[0].Name == "db1"
				},
			},
			{
				desc:    "two specific instances",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"p, role:test, instances, read, prod/ns1/db1",
					"p, role:test, instances, read, prod/ns1/db3",
					"g, bob, role:test",
				),
				assert: func(list *corev1alpha1.InstanceList) bool {
					return len(list.Items) == 2 &&
						slices.ContainsFunc(list.Items, func(i corev1alpha1.Instance) bool { return i.Name == "db1" }) &&
						slices.ContainsFunc(list.Items, func(i corev1alpha1.Instance) bool { return i.Name == "db3" })
				},
			},
			{
				desc:    "wrong cluster",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"p, role:test, instances, read, staging/ns1/*",
					"g, bob, role:test",
				),
				assert: func(list *corev1alpha1.InstanceList) bool {
					return len(list.Items) == 0
				},
			},
			{
				desc:    "wrong namespace",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"p, role:test, instances, read, prod/ns2/*",
					"g, bob, role:test",
				),
				assert: func(list *corev1alpha1.InstanceList) bool {
					return len(list.Items) == 0
				},
			},
			{
				desc:    "all clusters wildcard",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"p, role:test, instances, read, */*/*",
					"g, bob, role:test",
				),
				assert: func(list *corev1alpha1.InstanceList) bool {
					return len(list.Items) == 3
				},
			},
			{
				desc:    "no permissions",
				cluster: "prod",
				ns:      "ns1",
				policy: newPolicy(
					"g, bob, role:test",
				),
				assert: func(list *corev1alpha1.InstanceList) bool {
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
				next := mockInstances()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				list, err := h.ListInstances(ctx, tc.cluster, tc.ns)
				require.NoError(t, err)
				assert.Condition(t, func() bool {
					return tc.assert(list)
				})
			})
		}
	})

	t.Run("GetInstance", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			desc    string
			cluster string
			ns      string
			name    string
			policy  string
			wantErr error
		}{
			{
				desc:    "admin",
				cluster: "prod",
				ns:      "ns1",
				name:    "db1",
				policy: newPolicy(
					"g, bob, role:admin",
				),
			},
			{
				desc:    "exact match",
				cluster: "prod",
				ns:      "ns1",
				name:    "db1",
				policy: newPolicy(
					"p, role:test, instances, read, prod/ns1/db1",
					"g, bob, role:test",
				),
			},
			{
				desc:    "namespace wildcard",
				cluster: "prod",
				ns:      "ns1",
				name:    "db1",
				policy: newPolicy(
					"p, role:test, instances, read, prod/ns1/*",
					"g, bob, role:test",
				),
			},
			{
				desc:    "wrong cluster",
				cluster: "prod",
				ns:      "ns1",
				name:    "db1",
				policy: newPolicy(
					"p, role:test, instances, read, staging/ns1/db1",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc:    "wrong namespace",
				cluster: "prod",
				ns:      "ns1",
				name:    "db1",
				policy: newPolicy(
					"p, role:test, instances, read, prod/ns2/db1",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc:    "wrong name",
				cluster: "prod",
				ns:      "ns1",
				name:    "db1",
				policy: newPolicy(
					"p, role:test, instances, read, prod/ns1/db2",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc:    "no permissions",
				cluster: "prod",
				ns:      "ns1",
				name:    "db1",
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
				next := mockInstances()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				result, err := h.GetInstance(ctx, tc.cluster, tc.ns, tc.name)
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
				} else {
					require.NoError(t, err)
					assert.Equal(t, "db1", result.Name)
				}
			})
		}
	})

	t.Run("CreateInstance", func(t *testing.T) {
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
					"p, role:test, instances, create, prod/ns1/db1",
					"g, bob, role:test",
				),
			},
			{
				desc:    "namespace wildcard create",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, instances, create, prod/ns1/*",
					"g, bob, role:test",
				),
			},
			{
				desc:    "has read but not create",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, instances, read, prod/ns1/db1",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc:    "wrong cluster",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, instances, create, staging/ns1/db1",
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

		ctx := context.WithValue(context.Background(), common.UserCtxKey, rbac.User{Subject: "bob"})
		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				t.Parallel()
				k8sMock := newConfigMapMock(tc.policy)
				enf, err := rbac.NewEnforcer(ctx, k8sMock, zap.NewNop().Sugar())
				require.NoError(t, err)
				next := mockInstances()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				instance := &corev1alpha1.Instance{
					ObjectMeta: metav1.ObjectMeta{Name: "db1", Namespace: "ns1"},
				}
				result, err := h.CreateInstance(ctx, tc.cluster, instance)
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
				} else {
					require.NoError(t, err)
					assert.Equal(t, "db1", result.Name)
				}
			})
		}
	})

	t.Run("CreateInstance with UserSecretRef", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			desc    string
			cluster string
			policy  string
			wantErr error
		}{
			{
				desc:    "create instance and read secret",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, instances, create, prod/ns1/db1",
					"p, role:test, secrets, read, prod/ns1/my-secret",
					"g, bob, role:test",
				),
			},
			{
				desc:    "create instance but no secret read permission",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, instances, create, prod/ns1/db1",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
		}

		ctx := context.WithValue(context.Background(), common.UserCtxKey, rbac.User{Subject: "bob"}) //nolint:staticcheck // for testing only
		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				t.Parallel()
				k8sMock := newConfigMapMock(tc.policy)
				enf, err := rbac.NewEnforcer(ctx, k8sMock, zap.NewNop().Sugar())
				require.NoError(t, err)
				next := mockInstances()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				instance := &corev1alpha1.Instance{
					ObjectMeta: metav1.ObjectMeta{Name: "db1", Namespace: "ns1"},
					Spec: corev1alpha1.InstanceSpec{
						UserSecretRef: &apicommon.SecretRef{Name: "my-secret"},
					},
				}
				result, err := h.CreateInstance(ctx, tc.cluster, instance)
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
				} else {
					require.NoError(t, err)
					assert.Equal(t, "db1", result.Name)
				}
			})
		}
	})

	t.Run("CreateInstance with InstancePreset", func(t *testing.T) {
		t.Parallel()

		testPreset := &corev1alpha1.InstancePreset{
			ObjectMeta: metav1.ObjectMeta{Name: "standard-mysql"},
			Spec: corev1alpha1.InstancePresetSpec{
				InstanceSpec: corev1alpha1.InstanceSpec{
					ProviderRef: apicommon.ObjectRef{Name: "mysql"},
					Version:     "8.0",
				},
			},
		}

		testCases := []struct {
			desc     string
			cluster  string
			instance *corev1alpha1.Instance
			policy   string
			wantErr  error
		}{
			{
				desc:    "admin",
				cluster: "prod",
				instance: &corev1alpha1.Instance{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "db1",
						Namespace:   "ns1",
						Annotations: map[string]string{"openeverest.io/instance-preset": "standard-mysql"},
					},
					Spec: corev1alpha1.InstanceSpec{
						ProviderRef: apicommon.ObjectRef{Name: "mysql"},
						Version:     "9.0", // Different from preset!
					},
				},
				policy: newPolicy(
					"g, bob, role:admin",
				),
			},
			{
				desc:    "has create",
				cluster: "prod",
				instance: &corev1alpha1.Instance{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "db1",
						Namespace:   "ns1",
						Annotations: map[string]string{"openeverest.io/instance-preset": "standard-mysql"},
					},
					Spec: corev1alpha1.InstanceSpec{
						ProviderRef: apicommon.ObjectRef{Name: "mysql"},
						Version:     "9.0", // Different from preset!
					},
				},
				policy: newPolicy(
					"p, role:developer, instances, create, prod/ns1/*",
					"p, role:developer, instance-presets, read, prod/*",
					"p, role:developer, providers, read, prod/*",
					"g, bob, role:developer",
				),
			},
			{
				desc:    "has create without preset annotation",
				cluster: "prod",
				instance: &corev1alpha1.Instance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "db1",
						Namespace: "ns1",
					},
					Spec: corev1alpha1.InstanceSpec{
						ProviderRef: apicommon.ObjectRef{Name: "mysql"},
						Version:     "9.0",
					},
				},
				policy: newPolicy(
					"p, role:developer, instances, create, prod/ns1/*",
					"p, role:developer, instance-presets, read, prod/*",
					"p, role:developer, providers, read, prod/*",
					"g, bob, role:developer",
				),
			},
			{
				desc:    "has deploy",
				cluster: "prod",
				instance: &corev1alpha1.Instance{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "db1",
						Namespace:   "ns1",
						Annotations: map[string]string{"openeverest.io/instance-preset": "standard-mysql"},
					},
					Spec: corev1alpha1.InstanceSpec{
						ProviderRef: apicommon.ObjectRef{Name: "mysql"},
						Version:     "8.0", // Matches preset exactly
					},
				},
				policy: newPolicy(
					"p, role:developer, instances, deploy, prod/ns1/*",
					"p, role:developer, instance-presets, read, prod/*",
					"g, bob, role:developer",
				),
			},
			{
				desc:    "has deploy without preset annotation",
				cluster: "prod",
				instance: &corev1alpha1.Instance{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "db1",
						Namespace: "ns1",
					},
					Spec: corev1alpha1.InstanceSpec{
						ProviderRef: apicommon.ObjectRef{Name: "mysql"},
						Version:     "8.0", // Matches preset exactly
					},
				},
				policy: newPolicy(
					"p, role:developer, instances, deploy, prod/ns1/*",
					"p, role:developer, instance-presets, read, prod/*",
					"g, bob, role:developer",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc:    "has deploy with customization",
				cluster: "prod",
				instance: &corev1alpha1.Instance{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "db1",
						Namespace:   "ns1",
						Annotations: map[string]string{"openeverest.io/instance-preset": "standard-mysql"},
					},
					Spec: corev1alpha1.InstanceSpec{
						ProviderRef: apicommon.ObjectRef{Name: "mysql"},
						Version:     "9.0", // Different from preset!
					},
				},
				policy: newPolicy(
					"p, role:developer, instances, deploy, prod/ns1/*",
					"p, role:developer, instance-presets, read, prod/*",
					"g, bob, role:developer",
				),
				wantErr: ErrInsufficientPermissions,
			},
			{
				desc:    "has deploy without preset read permission",
				cluster: "prod",
				instance: &corev1alpha1.Instance{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "db1",
						Namespace:   "ns1",
						Annotations: map[string]string{"openeverest.io/instance-preset": "standard-mysql"},
					},
					Spec: corev1alpha1.InstanceSpec{
						ProviderRef: apicommon.ObjectRef{Name: "mysql"},
						Version:     "8.0", // Matches preset exactly
					},
				},
				policy: newPolicy(
					"p, role:developer, instances, deploy, prod/ns1/*",
					"g, bob, role:developer",
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

				next := &handlers.MockHandler{}
				next.On("CreateInstance", mock.Anything, mock.Anything, mock.Anything).Return(tc.instance, nil)
				next.On("GetPreset", mock.Anything, mock.Anything, mock.Anything).Return(testPreset, nil)
				next.On("ResolveInstancePreset", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(testPreset, nil)

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				result, err := h.CreateInstance(ctx, tc.cluster, tc.instance)
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)

					return
				}

				require.NoError(t, err)
				assert.Equal(t, tc.instance.Name, result.Name)
			})
		}
	})

	t.Run("UpdateInstance", func(t *testing.T) {
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
				desc:    "has update permission",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, instances, update, prod/ns1/db1",
					"g, bob, role:test",
				),
			},
			{
				desc:    "has read but not update",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, instances, read, prod/ns1/db1",
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

		ctx := context.WithValue(context.Background(), common.UserCtxKey, rbac.User{Subject: "bob"})
		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				t.Parallel()
				k8sMock := newConfigMapMock(tc.policy)
				enf, err := rbac.NewEnforcer(ctx, k8sMock, zap.NewNop().Sugar())
				require.NoError(t, err)
				next := mockInstances()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				instance := &corev1alpha1.Instance{
					ObjectMeta: metav1.ObjectMeta{Name: "db1", Namespace: "ns1"},
				}
				result, err := h.UpdateInstance(ctx, tc.cluster, instance)
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
				} else {
					require.NoError(t, err)
					assert.Equal(t, "db1", result.Name)
				}
			})
		}
	})

	// A patch is authorised as an update, not as a permission of its own, so
	// read alone must not be enough to patch.
	t.Run("PatchInstance", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			desc    string
			policy  string
			wantErr error
		}{
			{
				desc: "has update permission",
				policy: newPolicy(
					"p, role:test, instances, update, prod/ns1/db1",
					"g, bob, role:test",
				),
			},
			{
				desc: "has read but not update",
				policy: newPolicy(
					"p, role:test, instances, read, prod/ns1/db1",
					"g, bob, role:test",
				),
				wantErr: ErrInsufficientPermissions,
			},
		}

		ctx := testUserContext(rbac.User{Subject: "bob"})
		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				t.Parallel()
				k8sMock := newConfigMapMock(tc.policy)
				enf, err := rbac.NewEnforcer(ctx, k8sMock, zap.NewNop().Sugar())
				require.NoError(t, err)

				h := &rbacHandler{
					next:       mockInstances(),
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				result, err := h.PatchInstance(ctx, "prod", "ns1", "db1", []byte(`{"spec":{"version":"8.1"}}`))
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
					return
				}
				require.NoError(t, err)
				assert.Equal(t, "db1", result.Name)
			})
		}
	})

	t.Run("DeleteInstance", func(t *testing.T) {
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
					"p, role:test, instances, delete, prod/ns1/db1",
					"g, bob, role:test",
				),
			},
			{
				desc:    "has read but not delete",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, instances, read, prod/ns1/db1",
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

		ctx := context.WithValue(context.Background(), common.UserCtxKey, rbac.User{Subject: "bob"})
		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				t.Parallel()
				k8sMock := newConfigMapMock(tc.policy)
				enf, err := rbac.NewEnforcer(ctx, k8sMock, zap.NewNop().Sugar())
				require.NoError(t, err)
				next := mockInstances()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				err = h.DeleteInstance(ctx, tc.cluster, "ns1", "db1", nil)
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
				} else {
					require.NoError(t, err)
				}
			})
		}
	})

	t.Run("GetInstanceConnection", func(t *testing.T) {
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
				desc:    "has read-connection permission",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, instances, read-connection, prod/ns1/db1",
					"g, bob, role:test",
				),
			},
			{
				desc:    "read permission alone is not enough",
				cluster: "prod",
				policy: newPolicy(
					"p, role:test, instances, read, prod/ns1/db1",
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

		ctx := context.WithValue(context.Background(), common.UserCtxKey, rbac.User{Subject: "bob"})
		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				t.Parallel()
				k8sMock := newConfigMapMock(tc.policy)
				enf, err := rbac.NewEnforcer(ctx, k8sMock, zap.NewNop().Sugar())
				require.NoError(t, err)
				next := mockInstances()

				h := &rbacHandler{
					next:       next,
					log:        zap.NewNop().Sugar(),
					enforcer:   enf,
					userGetter: testUserGetter,
				}

				result, err := h.GetInstanceConnection(ctx, tc.cluster, "ns1", "db1")
				if tc.wantErr != nil {
					require.ErrorIs(t, err, tc.wantErr)
				} else {
					require.NoError(t, err)
					assert.Equal(t, "db1.ns1.svc", *result.Host)
				}
			})
		}
	})
}

// Writes that point an instance at other resources must also be authorized on
// those resources, but only for references the request adds or changes.
func TestRBAC_InstanceReferences(t *testing.T) {
	t.Parallel()

	const basePolicy = "p, role:test, instances, *, prod/ns1/*\ng, bob, role:test"

	withStorages := func(storages ...corev1alpha1.InstanceBackupStorage) *corev1alpha1.InstanceBackupSpec {
		return &corev1alpha1.InstanceBackupSpec{ClassRef: apicommon.ObjectRef{Name: "pbm"}, Storages: storages}
	}
	storage := func(name string) corev1alpha1.InstanceBackupStorage {
		return corev1alpha1.InstanceBackupStorage{StorageRef: apicommon.ObjectRef{Name: name}}
	}
	scheduled := func(name string) corev1alpha1.InstanceBackupStorage {
		s := storage(name)
		s.Schedules = []corev1alpha1.InstanceBackupSchedule{{Name: "daily", Enabled: true, Cron: "0 0 * * *"}}
		return s
	}
	instanceWith := func(spec corev1alpha1.InstanceSpec) *corev1alpha1.Instance {
		spec.ProviderRef = apicommon.ObjectRef{Name: "mysql"}
		return &corev1alpha1.Instance{ObjectMeta: metav1.ObjectMeta{Name: "db1", Namespace: "ns1"}, Spec: spec}
	}
	// stored is what GetInstance returns: already pointing at storage s1,
	// which the caller in most cases below cannot read.
	stored := func() *corev1alpha1.Instance {
		return instanceWith(corev1alpha1.InstanceSpec{Version: "8.0", Backup: withStorages(scheduled("s1"))})
	}

	newHandler := func(t *testing.T, extraPolicy ...string) *rbacHandler {
		t.Helper()
		ctx := testUserContext(rbac.User{Subject: "bob"})
		enf, err := rbac.NewEnforcer(ctx, newConfigMapMock(newPolicy(append([]string{basePolicy}, extraPolicy...)...)), zap.NewNop().Sugar())
		require.NoError(t, err)
		next := &handlers.MockHandler{}
		next.On("GetInstance", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(stored(), nil)
		next.On("CreateInstance", mock.Anything, mock.Anything, mock.Anything).Return(stored(), nil)
		next.On("UpdateInstance", mock.Anything, mock.Anything, mock.Anything).Return(stored(), nil)
		next.On("PatchInstance", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(stored(), nil)
		next.On("GetBackup", mock.Anything, mock.Anything, mock.Anything, "backup-1").Return(&backupv1alpha1.Backup{
			ObjectMeta: metav1.ObjectMeta{Name: "backup-1", Namespace: "ns1"},
			Spec: backupv1alpha1.BackupSpec{Origin: backupv1alpha1.BackupOrigin{
				Type:        backupv1alpha1.BackupOriginTypeInstance,
				InstanceRef: &apicommon.ObjectRef{Name: "source-db"},
			}},
		}, nil)
		return &rbacHandler{next: next, log: zap.NewNop().Sugar(), enforcer: enf, userGetter: testUserGetter}
	}

	fromBackup := &backupv1alpha1.DataSource{
		Type:   backupv1alpha1.DataSourceTypeBackup,
		Backup: &backupv1alpha1.DataSourceBackup{BackupRef: apicommon.ObjectRef{Name: "backup-1"}},
	}
	const (
		readProvider  = "p, role:test, providers, read, prod/mysql"
		readClass     = "p, role:test, backup-classes, read, prod/pbm"
		readS1        = "p, role:test, backup-storages, read, prod/ns1/s1"
		readS2        = "p, role:test, backup-storages, read, prod/ns1/s2"
		createBackups = "p, role:test, backups, create, prod/ns1/db1"
		createRestore = "p, role:test, restores, create, prod/ns1/db1"
		readSource    = "p, role:test, backups, read, prod/ns1/source-db"
	)

	createCases := []struct {
		desc     string
		spec     corev1alpha1.InstanceSpec
		policy   []string
		wantDeny bool
	}{
		{desc: "provider readable", policy: []string{readProvider}},
		{desc: "provider not readable", wantDeny: true},
		{desc: "storage readable", spec: corev1alpha1.InstanceSpec{Backup: withStorages(storage("s1"))}, policy: []string{readProvider, readClass, readS1}},
		{desc: "storage not readable", spec: corev1alpha1.InstanceSpec{Backup: withStorages(storage("s1"))}, policy: []string{readProvider, readClass}, wantDeny: true},
		{desc: "backup class not readable", spec: corev1alpha1.InstanceSpec{Backup: withStorages(storage("s1"))}, policy: []string{readProvider, readS1}, wantDeny: true},
		{desc: "schedules with backups create", spec: corev1alpha1.InstanceSpec{Backup: withStorages(scheduled("s1"))}, policy: []string{readProvider, readClass, readS1, createBackups}},
		{desc: "schedules without backups create", spec: corev1alpha1.InstanceSpec{Backup: withStorages(scheduled("s1"))}, policy: []string{readProvider, readClass, readS1}, wantDeny: true},
		{desc: "data source readable", spec: corev1alpha1.InstanceSpec{DataSource: fromBackup}, policy: []string{readProvider, createRestore, readSource}},
		{desc: "data source without restores create", spec: corev1alpha1.InstanceSpec{DataSource: fromBackup}, policy: []string{readProvider, readSource}, wantDeny: true},
		{desc: "data source backup not readable", spec: corev1alpha1.InstanceSpec{DataSource: fromBackup}, policy: []string{readProvider, createRestore}, wantDeny: true},
	}
	for _, tc := range createCases {
		t.Run("CreateInstance/"+tc.desc, func(t *testing.T) {
			t.Parallel()
			_, err := newHandler(t, tc.policy...).CreateInstance(testUserContext(rbac.User{Subject: "bob"}), "prod", instanceWith(tc.spec))
			if tc.wantDeny {
				require.ErrorIs(t, err, ErrInsufficientPermissions)
				return
			}
			require.NoError(t, err)
		})
	}

	updateCases := []struct {
		desc     string
		mutate   func(*corev1alpha1.Instance)
		policy   []string
		wantDeny bool
	}{
		{desc: "untouched unreadable storage does not block an unrelated edit", mutate: func(i *corev1alpha1.Instance) { i.Spec.Version = "8.4" }},
		{desc: "adding a storage requires reading it", mutate: func(i *corev1alpha1.Instance) {
			i.Spec.Backup.Storages = append(i.Spec.Backup.Storages, storage("s2"))
		}, wantDeny: true},
		{desc: "adding a readable storage", mutate: func(i *corev1alpha1.Instance) {
			i.Spec.Backup.Storages = append(i.Spec.Backup.Storages, storage("s2"))
		}, policy: []string{readS2}},
		{desc: "changing a schedule requires backups create", mutate: func(i *corev1alpha1.Instance) {
			i.Spec.Backup.Storages[0].Schedules[0].Cron = "0 * * * *"
		}, wantDeny: true},
		{desc: "changing a schedule with backups create", mutate: func(i *corev1alpha1.Instance) {
			i.Spec.Backup.Storages[0].Schedules[0].Cron = "0 * * * *"
		}, policy: []string{createBackups}},
		{desc: "removing schedules needs no backups create", mutate: func(i *corev1alpha1.Instance) {
			i.Spec.Backup.Storages[0].Schedules = nil
		}},
	}
	for _, tc := range updateCases {
		t.Run("UpdateInstance/"+tc.desc, func(t *testing.T) {
			t.Parallel()
			instance := stored()
			tc.mutate(instance)
			_, err := newHandler(t, tc.policy...).UpdateInstance(testUserContext(rbac.User{Subject: "bob"}), "prod", instance)
			if tc.wantDeny {
				require.ErrorIs(t, err, ErrInsufficientPermissions)
				return
			}
			require.NoError(t, err)
		})
	}

	patchCases := []struct {
		desc    string
		patch   string
		wantErr error
	}{
		{desc: "untouched unreadable storage does not block an unrelated patch", patch: `{"spec":{"version":"8.4"}}`},
		{
			desc:    "patch adding an unreadable storage",
			patch:   `{"spec":{"backup":{"classRef":{"name":"pbm"},"storages":[{"storageRef":{"name":"s1"}},{"storageRef":{"name":"s2"}}]}}}`,
			wantErr: ErrInsufficientPermissions,
		},
		{
			desc:    "patch adding a data source",
			patch:   `{"spec":{"dataSource":{"type":"Backup","backup":{"backupRef":{"name":"backup-1"}}}}}`,
			wantErr: ErrInsufficientPermissions,
		},
		{desc: "malformed patch", patch: `{"spec":`, wantErr: valhandler.ErrInvalidRequest},
	}
	for _, tc := range patchCases {
		t.Run("PatchInstance/"+tc.desc, func(t *testing.T) {
			t.Parallel()
			_, err := newHandler(t).PatchInstance(testUserContext(rbac.User{Subject: "bob"}), "prod", "ns1", "db1", []byte(tc.patch))
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}

	// The write must be conditional on the version the checks ran against,
	// or a reference added in between would be written unchecked.
	t.Run("writes are pinned to the checked resource version", func(t *testing.T) {
		t.Parallel()
		ctx := testUserContext(rbac.User{Subject: "bob"})
		current := stored()
		current.ResourceVersion = "42"
		newPinned := func() (*rbacHandler, *handlers.MockHandler) {
			h := newHandler(t)
			next := &handlers.MockHandler{}
			next.On("GetInstance", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(current, nil)
			next.On("UpdateInstance", mock.Anything, mock.Anything, mock.Anything).Return(current, nil)
			next.On("PatchInstance", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(current, nil)
			h.next = next
			return h, next
		}

		h, next := newPinned()
		_, err := h.UpdateInstance(ctx, "prod", stored())
		require.NoError(t, err)
		next.AssertCalled(t, "UpdateInstance", mock.Anything, "prod", mock.MatchedBy(func(i *corev1alpha1.Instance) bool {
			return i.ResourceVersion == "42"
		}))

		h, next = newPinned()
		_, err = h.PatchInstance(ctx, "prod", "ns1", "db1", []byte(`{"spec":{"version":"8.4"}}`))
		require.NoError(t, err)
		next.AssertCalled(t, "PatchInstance", mock.Anything, "prod", "ns1", "db1", mock.MatchedBy(func(patch []byte) bool {
			return assert.JSONEq(t, `{"spec":{"version":"8.4"},"metadata":{"resourceVersion":"42"}}`, string(patch))
		}))

		// A version the caller names is theirs to be rejected on, not overwritten.
		h, next = newPinned()
		_, err = h.PatchInstance(ctx, "prod", "ns1", "db1", []byte(`{"metadata":{"resourceVersion":"7"},"spec":{"version":"8.4"}}`))
		require.NoError(t, err)
		next.AssertCalled(t, "PatchInstance", mock.Anything, "prod", "ns1", "db1", mock.MatchedBy(func(patch []byte) bool {
			return assert.JSONEq(t, `{"spec":{"version":"8.4"},"metadata":{"resourceVersion":"7"}}`, string(patch))
		}))
	})

	// A preset-only caller inherits the preset's vetted references, but not
	// a user secret they supply.
	t.Run("CreateInstance from preset", func(t *testing.T) {
		t.Parallel()
		presetSpec := corev1alpha1.InstanceSpec{
			ProviderRef: apicommon.ObjectRef{Name: "mysql"},
			Backup:      withStorages(scheduled("s1")),
		}
		for _, tc := range []struct {
			desc     string
			secret   *apicommon.SecretRef
			policy   []string
			wantDeny bool
		}{
			{desc: "pinned storage needs no grant"},
			{desc: "user secret still checked", secret: &apicommon.SecretRef{Name: "creds"}, wantDeny: true},
			{desc: "readable user secret", secret: &apicommon.SecretRef{Name: "creds"}, policy: []string{"p, role:deployer, secrets, read, prod/ns1/creds"}},
		} {
			t.Run(tc.desc, func(t *testing.T) {
				t.Parallel()
				ctx := testUserContext(rbac.User{Subject: "alice"})
				policy := newPolicy(append([]string{
					"p, role:deployer, instances, deploy, prod/ns1/*",
					"p, role:deployer, instance-presets, read, prod/*",
					"g, alice, role:deployer",
				}, tc.policy...)...)
				enf, err := rbac.NewEnforcer(ctx, newConfigMapMock(policy), zap.NewNop().Sugar())
				require.NoError(t, err)

				spec := *presetSpec.DeepCopy()
				spec.UserSecretRef = tc.secret
				preset := &corev1alpha1.InstancePreset{Spec: corev1alpha1.InstancePresetSpec{InstanceSpec: spec}}
				next := &handlers.MockHandler{}
				next.On("ResolveInstancePreset", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(preset, nil)
				next.On("CreateInstance", mock.Anything, mock.Anything, mock.Anything).Return(stored(), nil)
				h := &rbacHandler{next: next, log: zap.NewNop().Sugar(), enforcer: enf, userGetter: testUserGetter}

				instance := &corev1alpha1.Instance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "db1", Namespace: "ns1",
						Annotations: map[string]string{"openeverest.io/instance-preset": "standard"},
					},
					Spec: spec,
				}
				_, err = h.CreateInstance(ctx, "prod", instance)
				if tc.wantDeny {
					require.ErrorIs(t, err, ErrInsufficientPermissions)
					return
				}
				require.NoError(t, err)
			})
		}
	})
}
