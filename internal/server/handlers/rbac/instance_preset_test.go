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

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/internal/server/handlers"
	"github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

func TestRBAC_InstancePreset(t *testing.T) {
	t.Parallel()

	preset := func(name string) *corev1alpha1.InstancePreset {
		return &corev1alpha1.InstancePreset{ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: name}}
	}

	mockInstancePresets := func() *handlers.MockHandler {
		h := &handlers.MockHandler{}
		h.On("ListInstancePresets", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
			&corev1alpha1.InstancePresetList{
				Items: []corev1alpha1.InstancePreset{
					*preset("small"),
					*preset("medium"),
					*preset("large"),
				},
			}, nil,
		)
		h.On("GetInstancePreset", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(preset("small"), nil)
		h.On("CreateInstancePreset", mock.Anything, mock.Anything, mock.Anything).Return(preset("small"), nil)
		h.On("UpdateInstancePreset", mock.Anything, mock.Anything, mock.Anything).Return(preset("small"), nil)
		h.On("DeleteInstancePreset", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		return h
	}

	newHandler := func(t *testing.T, ctx context.Context, policy string) *rbacHandler {
		t.Helper()
		enf, err := rbac.NewEnforcer(ctx, newConfigMapMock(policy), zap.NewNop().Sugar())
		require.NoError(t, err)
		return &rbacHandler{
			next:       mockInstancePresets(),
			log:        zap.NewNop().Sugar(),
			enforcer:   enf,
			userGetter: testUserGetter,
		}
	}

	ctx := context.WithValue(context.Background(), common.UserCtxKey, rbac.User{Subject: "bob"})

	t.Run("ListInstancePresets", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			desc   string
			policy string
			want   []string
		}{
			{
				desc:   "admin",
				policy: newPolicy("g, bob, role:admin"),
				want:   []string{"small", "medium", "large"},
			},
			{
				desc: "all presets in namespace",
				policy: newPolicy(
					"p, role:test, instance-presets, read, prod/default/*",
					"g, bob, role:test",
				),
				want: []string{"small", "medium", "large"},
			},
			{
				desc: "specific preset",
				policy: newPolicy(
					"p, role:test, instance-presets, read, prod/default/small",
					"g, bob, role:test",
				),
				want: []string{"small"},
			},
			{
				desc: "wrong namespace",
				policy: newPolicy(
					"p, role:test, instance-presets, read, prod/other/*",
					"g, bob, role:test",
				),
				want: []string{},
			},
			{
				desc: "wrong cluster",
				policy: newPolicy(
					"p, role:test, instance-presets, read, staging/default/*",
					"g, bob, role:test",
				),
				want: []string{},
			},
			{
				desc:   "no permissions",
				policy: newPolicy("g, bob, role:test"),
				want:   []string{},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.desc, func(t *testing.T) {
				t.Parallel()
				h := newHandler(t, ctx, tc.policy)

				list, err := h.ListInstancePresets(ctx, "prod", "default", "")
				require.NoError(t, err)

				names := make([]string, 0, len(list.Items))
				for _, item := range list.Items {
					names = append(names, item.Name)
				}
				assert.Equal(t, tc.want, names)
			})
		}
	})

	testCases := []struct {
		desc    string
		policy  string
		wantErr error
	}{
		{
			desc:   "admin",
			policy: newPolicy("g, bob, role:admin"),
		},
		{
			desc: "exact match",
			policy: newPolicy(
				"p, role:test, instance-presets, *, prod/default/small",
				"g, bob, role:test",
			),
		},
		{
			desc: "wildcard in namespace",
			policy: newPolicy(
				"p, role:test, instance-presets, *, prod/default/*",
				"g, bob, role:test",
			),
		},
		{
			desc: "wrong preset",
			policy: newPolicy(
				"p, role:test, instance-presets, *, prod/default/medium",
				"g, bob, role:test",
			),
			wantErr: ErrInsufficientPermissions,
		},
		{
			desc: "wrong namespace",
			policy: newPolicy(
				"p, role:test, instance-presets, *, prod/other/small",
				"g, bob, role:test",
			),
			wantErr: ErrInsufficientPermissions,
		},
		{
			desc:    "no permissions",
			policy:  newPolicy("g, bob, role:test"),
			wantErr: ErrInsufficientPermissions,
		},
	}

	operations := map[string]func(h *rbacHandler) error{
		"GetInstancePreset": func(h *rbacHandler) error {
			_, err := h.GetInstancePreset(ctx, "prod", "default", "small")
			return err
		},
		"CreateInstancePreset": func(h *rbacHandler) error {
			_, err := h.CreateInstancePreset(ctx, "prod", preset("small"))
			return err
		},
		"UpdateInstancePreset": func(h *rbacHandler) error {
			_, err := h.UpdateInstancePreset(ctx, "prod", preset("small"))
			return err
		},
		"DeleteInstancePreset": func(h *rbacHandler) error {
			return h.DeleteInstancePreset(ctx, "prod", "default", "small")
		},
	}

	for opName, op := range operations {
		t.Run(opName, func(t *testing.T) {
			t.Parallel()

			for _, tc := range testCases {
				t.Run(tc.desc, func(t *testing.T) {
					t.Parallel()
					h := newHandler(t, ctx, tc.policy)

					err := op(h)
					if tc.wantErr != nil {
						require.ErrorIs(t, err, tc.wantErr)
						return
					}
					require.NoError(t, err)
				})
			}
		})
	}

	t.Run("action mismatch", func(t *testing.T) {
		t.Parallel()
		h := newHandler(t, ctx, newPolicy(
			"p, role:test, instance-presets, read, prod/default/*",
			"g, bob, role:test",
		))

		_, err := h.GetInstancePreset(ctx, "prod", "default", "small")
		require.NoError(t, err)

		err = h.DeleteInstancePreset(ctx, "prod", "default", "small")
		require.ErrorIs(t, err, ErrInsufficientPermissions)
	})
}
