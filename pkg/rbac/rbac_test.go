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
	"sync/atomic"
	"testing"

	"github.com/casbin/casbin/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/percona/everest/pkg/common"
	configmapadapter "github.com/percona/everest/pkg/rbac/configmap-adapter"
)

func TestGetScopeValues(t *testing.T) {
	t.Parallel()
	testcases := []struct {
		desc   string
		claims jwt.MapClaims
		scopes []string
		out    []string
	}{
		{
			desc:   "empty claims",
			claims: jwt.MapClaims{},
			scopes: []string{"groups"},
			out:    []string{},
		},
		{
			desc:   "empty scopes",
			claims: jwt.MapClaims{"groups": []string{"my-org:my-team"}},
			scopes: nil,
			out:    []string{},
		},
		{
			desc:   "empty groups",
			claims: jwt.MapClaims{"groups": []string{}},
			scopes: []string{"groups"},
			out:    []string{},
		},
		{
			desc:   "single group",
			claims: jwt.MapClaims{"groups": []string{"my-org:my-team"}},
			scopes: []string{"groups"},
			out:    []string{"my-org:my-team"},
		},
		{
			desc:   "multiple groups",
			claims: jwt.MapClaims{"groups": []string{"my-org:my-team1", "my-org:my-team2"}},
			scopes: []string{"groups"},
			out:    []string{"my-org:my-team1", "my-org:my-team2"},
		},
		{
			desc:   "multiple groups and other",
			claims: jwt.MapClaims{"groups": []string{"my-org:my-team1", "my-org:my-team2"}, "other": []string{"other1", "other2"}},
			scopes: []string{"groups"},
			out:    []string{"my-org:my-team1", "my-org:my-team2"},
		},
		{
			desc:   "multiple groups and other with all scopes",
			claims: jwt.MapClaims{"groups": []string{"my-org:my-team1", "my-org:my-team2"}, "other": []string{"other1", "other2"}},
			scopes: []string{"groups", "other"},
			out:    []string{"my-org:my-team1", "my-org:my-team2", "other1", "other2"},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.out, getScopeValues(tc.claims, tc.scopes))
		})
	}
}

// fakeConfigMapGetter is a fake whose GetConfigMap returns a configurable
// policy and counts how many times it is called.
type fakeConfigMapGetter struct {
	policy atomic.Pointer[string]
	calls  atomic.Int64
}

func newFakeConfigMapGetter(policy string) *fakeConfigMapGetter {
	f := &fakeConfigMapGetter{}
	f.policy.Store(&policy)
	return f
}

func (f *fakeConfigMapGetter) GetConfigMap(_ context.Context, _ ctrlclient.ObjectKey) (*corev1.ConfigMap, error) {
	f.calls.Add(1)
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      common.EverestRBACConfigMapName,
			Namespace: common.SystemNamespace,
		},
		Data: map[string]string{
			"enabled":    "true",
			"policy.csv": *f.policy.Load(),
		},
	}, nil
}

func (f *fakeConfigMapGetter) setPolicy(policy string) {
	f.policy.Store(&policy)
}

// newFakeEnforcer builds a enforcer backed by the fake fakeConfigMapGetter.
func newFakeEnforcer(t *testing.T, getter *fakeConfigMapGetter) *casbin.Enforcer {
	t.Helper()
	adapter := configmapadapter.New(
		zap.NewNop().Sugar(),
		getter,
		types.NamespacedName{
			Namespace: common.SystemNamespace,
			Name:      common.EverestRBACConfigMapName,
		},
	)
	enf, err := newEnforcer(adapter, false)
	require.NoError(t, err)
	return enf
}

// cmWithPolicy returns the RBAC ConfigMap object handed to the informer's
// OnUpdate callback for the given policy.
func cmWithPolicy(policy string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      common.EverestRBACConfigMapName,
			Namespace: common.SystemNamespace,
		},
		Data: map[string]string{
			"enabled":    "true",
			"policy.csv": policy,
		},
	}
}

// TestReloadEnforcerFromConfigMap exercises the real reloadEnforcerFromConfigMap
// function (the body of refreshEnforcerInBackground's OnUpdate callback) using a
// fake, latency-injecting ConfigMap adapter.
func TestReloadEnforcerFromConfigMap(t *testing.T) {
	t.Parallel()

	t.Run("reloads the policy on update", func(t *testing.T) {
		t.Parallel()
		getter := newFakeConfigMapGetter("")
		enf := newFakeEnforcer(t, getter)
		getter.calls.Store(0) // ignore the initial construction load.

		// Initially alice has no permissions.
		ok, err := enf.Enforce("alice", "database-clusters", "read", "my-ns/my-cluster")
		require.NoError(t, err)
		require.False(t, ok)

		// Update the backing policy and fire the real reload.
		policy := "p, alice, database-clusters, read, my-ns/my-cluster"
		getter.setPolicy(policy)
		reloadEnforcerFromConfigMap(enf, cmWithPolicy(policy), zap.NewNop().Sugar())

		ok, err = enf.Enforce("alice", "database-clusters", "read", "my-ns/my-cluster")
		require.NoError(t, err)
		assert.True(t, ok)
		assert.LessOrEqual(t, getter.calls.Load(), int64(1))
	})

	t.Run("invalid policy is rejected", func(t *testing.T) {
		t.Parallel()

		initialPolicy := "p, alice, database-clusters, read, my-ns/my-cluster"
		getter := newFakeConfigMapGetter(initialPolicy)
		enf := newFakeEnforcer(t, getter)
		getter.calls.Store(0) // ignore the initial construction load.

		ok, err := enf.Enforce("alice", "database-clusters", "read", "my-ns/my-cluster")
		require.NoError(t, err)
		assert.True(t, ok)

		// Invalid policy keeps the previous valid policy.
		newInvalidPolicy := "p, alice, not-a-real-resource, read, my-ns/my-cluster"
		getter.setPolicy(newInvalidPolicy)
		reloadEnforcerFromConfigMap(enf, cmWithPolicy(newInvalidPolicy), zap.NewNop().Sugar())

		ok, err = enf.Enforce("alice", "database-clusters", "read", "my-ns/my-cluster")
		require.NoError(t, err)
		assert.True(t, ok)
		assert.LessOrEqual(t, getter.calls.Load(), int64(1))
	})
}
