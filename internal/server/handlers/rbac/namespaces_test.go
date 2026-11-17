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

	"github.com/openeverest/openeverest/v2/internal/server/handlers"
	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

func TestRBAC_ListNamespaces(t *testing.T) {
	t.Parallel()

	next := func() *handlers.MockHandler {
		h := &handlers.MockHandler{}
		h.On("ListNamespaces", mock.Anything, mock.Anything).Return(
			[]string{"default", "team-a", "team-b"}, nil,
		)
		return h
	}

	testCases := []struct {
		desc   string
		policy string
		want   []string
	}{
		{
			desc:   "admin sees all namespaces",
			policy: newPolicy("g, bob, role:admin"),
			want:   []string{"default", "team-a", "team-b"},
		},
		{
			desc: "namespace-scoped grant exposes only that namespace",
			policy: newPolicy(
				"p, role:test, namespaces, read, prod/team-a",
				"g, bob, role:test",
			),
			want: []string{"team-a"},
		},
		{
			desc: "cluster-wide grant exposes all namespaces",
			policy: newPolicy(
				"p, role:test, namespaces, read, prod/*",
				"g, bob, role:test",
			),
			want: []string{"default", "team-a", "team-b"},
		},
		{
			desc: "grant on another cluster exposes none",
			policy: newPolicy(
				"p, role:test, namespaces, read, staging/*",
				"g, bob, role:test",
			),
			want: []string{},
		},
		{
			desc:   "no grant exposes none",
			policy: newPolicy("g, bob, role:test"),
			want:   []string{},
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
				next:       next(),
				log:        zap.NewNop().Sugar(),
				enforcer:   enf,
				userGetter: testUserGetter,
			}

			result, err := h.ListNamespaces(ctx, "prod")
			require.NoError(t, err)
			assert.Equal(t, tc.want, result)
		})
	}
}
