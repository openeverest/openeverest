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
	"strings"
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
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
	configmapadapter "github.com/openeverest/openeverest/v2/pkg/rbac/configmap-adapter"
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

func newTestEnforcer(t *testing.T, policy string) *rbacTestEnforcer {
	t.Helper()
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-ns",
			Name:      common.EverestRBACConfigMapName,
		},
		Data: map[string]string{
			"enabled":    "true",
			"policy.csv": policy,
		},
	}
	mockClient := fakeclient.NewClientBuilder().
		WithScheme(kubernetes.CreateScheme()).
		WithObjects(cm)
	k := kubernetes.NewEmpty(zap.NewNop().Sugar(), "test-ns").WithKubernetesClient(mockClient.Build())
	enf, err := NewEnforcer(context.Background(), k, zap.NewNop().Sugar())
	require.NoError(t, err)
	return &rbacTestEnforcer{enf: enf, t: t}
}

type rbacTestEnforcer struct {
	enf interface {
		Enforce(rvals ...interface{}) (bool, error)
	}
	t *testing.T
}

func (e *rbacTestEnforcer) assertAllowed(sub, resource, action, object string) {
	e.t.Helper()
	ok, err := e.enf.Enforce(sub, resource, action, object)
	require.NoError(e.t, err)
	assert.True(e.t, ok, "expected ALLOWED: sub=%s res=%s act=%s obj=%s", sub, resource, action, object)
}

func (e *rbacTestEnforcer) assertDenied(sub, resource, action, object string) {
	e.t.Helper()
	ok, err := e.enf.Enforce(sub, resource, action, object)
	require.NoError(e.t, err)
	assert.False(e.t, ok, "expected DENIED: sub=%s res=%s act=%s obj=%s", sub, resource, action, object)
}

func TestLoadAdminPolicy(t *testing.T) {
	t.Parallel()

	policy := strings.Join([]string{
		"g, admin-user, role:admin",
	}, "\n")

	enf := newTestEnforcer(t, policy)

	t.Run("admin can access global resources", func(t *testing.T) {
		t.Parallel()
		enf.assertAllowed("admin-user", ResourceClusters, ActionRead, "any-cluster")
		enf.assertAllowed("admin-user", ResourceClusters, ActionCreate, "new-cluster")
	})

	t.Run("admin can access cluster-scoped resources", func(t *testing.T) {
		t.Parallel()
		enf.assertAllowed("admin-user", ResourceNamespaces, ActionRead, "any-cluster/any-ns")
		enf.assertAllowed("admin-user", ResourceProviders, ActionRead, "prod/psmdb")
		enf.assertAllowed("admin-user", ResourceBackupClasses, ActionRead, "staging/my-class")
	})

	t.Run("admin can access cluster-namespaced resources", func(t *testing.T) {
		t.Parallel()
		enf.assertAllowed("admin-user", ResourceInstances, ActionRead, "prod/ns1/my-db")
		enf.assertAllowed("admin-user", ResourceInstances, ActionCreate, "prod/ns1/new-db")
		enf.assertAllowed("admin-user", ResourceBackups, ActionRead, "prod/default/backup-1")
		enf.assertAllowed("admin-user", ResourceRestores, ActionCreate, "staging/ns2/restore-1")
		enf.assertAllowed("admin-user", ResourceBackupStorages, ActionUpdate, "prod/default/my-storage")
		enf.assertAllowed("admin-user", ResourceMonitoringConfigs, ActionDelete, "prod/ns1/mc-1")
	})

	t.Run("admin cannot access cluster-namespaced resources via v1 format", func(t *testing.T) {
		t.Parallel()
		// backup-storages is v2-only (cluster/namespace/name), so v1 namespace/name must fail.
		enf.assertDenied("admin-user", ResourceBackupStorages, ActionRead, "default/bs1")
	})

	t.Run("non-admin without policy is denied", func(t *testing.T) {
		t.Parallel()
		enf.assertDenied("random-user", ResourceClusters, ActionRead, "any-cluster")
		enf.assertDenied("random-user", ResourceInstances, ActionRead, "prod/ns1/db1")
		enf.assertDenied("random-user", ResourceProviders, ActionRead, "prod/psmdb")
	})
}

func TestEnforceWithSpecificPolicies(t *testing.T) {
	t.Parallel()

	t.Run("cluster-scoped with specific cluster", func(t *testing.T) {
		t.Parallel()
		policy := strings.Join([]string{
			"p, role:dev, providers, read, prod/*",
			"g, bob, role:dev",
		}, "\n")
		enf := newTestEnforcer(t, policy)
		enf.assertAllowed("bob", ResourceProviders, ActionRead, "prod/psmdb")
		enf.assertAllowed("bob", ResourceProviders, ActionRead, "prod/pxc")
		enf.assertDenied("bob", ResourceProviders, ActionRead, "staging/psmdb")
	})

	t.Run("cluster-namespaced with specific cluster and namespace", func(t *testing.T) {
		t.Parallel()
		policy := strings.Join([]string{
			"p, role:dev, instances, read, prod/ns1/*",
			"g, bob, role:dev",
		}, "\n")
		enf := newTestEnforcer(t, policy)
		enf.assertAllowed("bob", ResourceInstances, ActionRead, "prod/ns1/my-db")
		enf.assertDenied("bob", ResourceInstances, ActionRead, "prod/ns2/my-db")
		enf.assertDenied("bob", ResourceInstances, ActionRead, "staging/ns1/my-db")
	})

	t.Run("global resource with specific name", func(t *testing.T) {
		t.Parallel()
		policy := strings.Join([]string{
			"p, role:dev, clusters, read, prod",
			"g, bob, role:dev",
		}, "\n")
		enf := newTestEnforcer(t, policy)
		enf.assertAllowed("bob", ResourceClusters, ActionRead, "prod")
		enf.assertDenied("bob", ResourceClusters, ActionRead, "staging")
	})

	t.Run("action-specific permissions", func(t *testing.T) {
		t.Parallel()
		policy := strings.Join([]string{
			"p, role:reader, instances, read, prod/*/*",
			"g, bob, role:reader",
		}, "\n")
		enf := newTestEnforcer(t, policy)
		enf.assertAllowed("bob", ResourceInstances, ActionRead, "prod/ns1/db1")
		enf.assertDenied("bob", ResourceInstances, ActionCreate, "prod/ns1/db1")
		enf.assertDenied("bob", ResourceInstances, ActionDelete, "prod/ns1/db1")
	})
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
			Namespace: "everest-system",
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
			Namespace: "everest-system",
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
			Name: common.EverestRBACConfigMapName,
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
		ok, err := enf.Enforce("alice", "instances", "read", "prod/ns/my-inst")
		require.NoError(t, err)
		require.False(t, ok)

		// Update the backing policy and fire the real reload.
		policy := "p, alice, instances, read, prod/ns/my-inst"
		getter.setPolicy(policy)
		reloadEnforcerFromConfigMap(enf, cmWithPolicy(policy), zap.NewNop().Sugar())

		ok, err = enf.Enforce("alice", "instances", "read", "prod/ns/my-inst")
		require.NoError(t, err)
		assert.True(t, ok)
		assert.LessOrEqual(t, getter.calls.Load(), int64(1))
	})

	t.Run("invalid policy is rejected", func(t *testing.T) {
		t.Parallel()

		initialPolicy := "p, alice, instances, read, prod/ns/my-inst"
		getter := newFakeConfigMapGetter(initialPolicy)
		enf := newFakeEnforcer(t, getter)
		getter.calls.Store(0) // ignore the initial construction load.

		ok, err := enf.Enforce("alice", "instances", "read", "prod/ns/my-inst")
		require.NoError(t, err)
		assert.True(t, ok)

		// Invalid policy keeps the previous valid policy.
		newInvalidPolicy := "p, alice, not-a-real-resource, read, prod/ns/my-inst"
		getter.setPolicy(newInvalidPolicy)
		reloadEnforcerFromConfigMap(enf, cmWithPolicy(newInvalidPolicy), zap.NewNop().Sugar())

		ok, err = enf.Enforce("alice", "instances", "read", "prod/ns/my-inst")
		require.NoError(t, err)
		assert.True(t, ok)
		assert.LessOrEqual(t, getter.calls.Load(), int64(1))
	})
}
