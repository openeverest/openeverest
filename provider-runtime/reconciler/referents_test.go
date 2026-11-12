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

package reconciler

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

func TestReconcile_ProviderNotFound(t *testing.T) {
	t.Parallel()
	p := &readyTestProvider{}
	r := newReadyTestReconciler(p, readyTestInstance(nil))

	got, err := reconcileAndGet(t, r)
	require.NoError(t, err)

	c := requireReady(t, got, metav1.ConditionFalse, v1alpha1.ReasonProviderNotFound)
	assert.Equal(t, "provider test-provider not found", c.Message)
	assert.Equal(t, v1alpha1.InstancePhasePending, got.Status.Phase)
	assert.Zero(t, p.calls, "the provider must not be called")
	requireCondition(t, got, v1alpha1.ConditionReferencesResolved, metav1.ConditionFalse, v1alpha1.ReasonProviderNotFound)
}

func TestReconcile_UserSecretNotFound(t *testing.T) {
	t.Parallel()
	p := &readyTestProvider{status: controller.Ready()}
	provider := &v1alpha1.Provider{ObjectMeta: metav1.ObjectMeta{Name: "test-provider"}}
	in := readyTestInstance(func(in *v1alpha1.Instance) {
		in.Spec.UserSecretRef = &common.SecretRef{Name: "orders-users"}
	})
	r := newReadyTestReconciler(p, provider, in)

	got, err := reconcileAndGet(t, r)
	require.NoError(t, err)
	c := requireReady(t, got, metav1.ConditionFalse, v1alpha1.ReasonUserSecretNotFound)
	assert.Equal(t, "secret orders-users named by spec.userSecretRef not found in team-a", c.Message)
	assert.Equal(t, v1alpha1.InstancePhasePending, got.Status.Phase)
	assert.Zero(t, p.calls, "the provider must not be called")
	requireCondition(t, got, v1alpha1.ConditionReferencesResolved, metav1.ConditionFalse, v1alpha1.ReasonUserSecretNotFound)

	require.NoError(t, r.Create(t.Context(), &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "orders-users", Namespace: "team-a"},
	}))
	got, err = reconcileAndGet(t, r)
	require.NoError(t, err)
	requireReady(t, got, metav1.ConditionTrue, v1alpha1.ReasonReady)
	assert.Equal(t, 1, p.calls)
	requireCondition(t, got, v1alpha1.ConditionReferencesResolved, metav1.ConditionTrue, v1alpha1.ReasonResolved)
}

func TestInstancesForSecret(t *testing.T) {
	t.Parallel()
	withSecret := func(ns, provider, secret string) func(*v1alpha1.Instance) {
		return func(in *v1alpha1.Instance) {
			in.Namespace = ns
			in.Spec.ProviderRef.Name = provider
			if secret != "" {
				in.Spec.UserSecretRef = &common.SecretRef{Name: secret}
			}
		}
	}
	named := func(name string, mutate func(*v1alpha1.Instance)) *v1alpha1.Instance {
		in := readyTestInstance(mutate)
		in.Name = name
		return in
	}
	r := newReadyTestReconciler(&readyTestProvider{},
		named("match", withSecret("team-a", "test-provider", "users")),
		named("other-secret", withSecret("team-a", "test-provider", "admins")),
		named("no-secret", withSecret("team-a", "test-provider", "")),
		named("other-provider", withSecret("team-a", "someone-else", "users")),
		named("other-namespace", withSecret("team-b", "test-provider", "users")),
	)

	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "users", Namespace: "team-a"}}
	reqs := r.instancesForSecret(t.Context(), secret)
	require.Len(t, reqs, 1)
	assert.Equal(t, "match", reqs[0].Name)
}
