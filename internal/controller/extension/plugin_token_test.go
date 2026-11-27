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

package extension

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	pluginv1alpha1 "github.com/openeverest/openeverest/v2/api/extensions/v1alpha1"
	"github.com/openeverest/openeverest/v2/pkg/plugintoken"
)

//nolint:gochecknoglobals // fixed reference time
var tokenT0 = time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

var tokenSecretKey = types.NamespacedName{Namespace: "everest-system", Name: "audit-token"} //nolint:gochecknoglobals // test fixture

func daemonPlugin() *pluginv1alpha1.Plugin {
	return &pluginv1alpha1.Plugin{
		ObjectMeta: metav1.ObjectMeta{Name: "audit", UID: "uid-1"},
		Spec: pluginv1alpha1.PluginSpec{
			DisplayName: "Audit Log",
			Enabled:     true,
			Backend: &pluginv1alpha1.PluginBackend{
				ServiceRef: &pluginv1alpha1.PluginBackendServiceRef{Namespace: "everest-system", Name: "audit", Port: 8080},
			},
			Permissions: []pluginv1alpha1.PluginPermission{{Verb: "read-events", Resource: "instances"}},
		},
	}
}

type tokenFixture struct {
	t   *testing.T
	c   client.Client
	r   *PluginReconciler
	now time.Time
}

func newTokenFixture(t *testing.T, signing bool, objs ...client.Object) *tokenFixture {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, pluginv1alpha1.AddToScheme(scheme))
	c := fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(objs...).WithStatusSubresource(&pluginv1alpha1.Plugin{}).Build()
	f := &tokenFixture{t: t, c: c, now: tokenT0}
	f.r = &PluginReconciler{Client: c, Scheme: scheme, Clock: func() time.Time { return f.now }}
	if signing {
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		require.NoError(t, err)
		f.r.Tokens = plugintoken.NewServiceWithKey(key)
	}
	return f
}

func (f *tokenFixture) reconcile() ctrl.Result {
	f.t.Helper()
	res, err := f.r.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Name: "audit"}})
	require.NoError(f.t, err)
	return res
}

func (f *tokenFixture) token() string {
	f.t.Helper()
	s := &corev1.Secret{}
	require.NoError(f.t, f.c.Get(context.Background(), tokenSecretKey, s))
	return string(s.Data[plugintoken.SecretKey])
}

func (f *tokenFixture) plugin() *pluginv1alpha1.Plugin {
	f.t.Helper()
	p := &pluginv1alpha1.Plugin{}
	require.NoError(f.t, f.c.Get(context.Background(), types.NamespacedName{Name: "audit"}, p))
	return p
}

func (f *tokenFixture) condition() *metav1.Condition {
	f.t.Helper()
	return meta.FindStatusCondition(f.plugin().Status.Conditions, ConditionTypeTokenIssued)
}

func TestPluginTokenIsIssuedAndRenewed(t *testing.T) {
	t.Parallel()
	f := newTokenFixture(t, true, daemonPlugin())

	res := f.reconcile()
	first := f.token()
	require.NotEmpty(t, first)
	assert.Equal(t, 16*time.Hour, res.RequeueAfter)
	assert.Equal(t, metav1.ConditionTrue, f.condition().Status)

	secret := &corev1.Secret{}
	require.NoError(t, f.c.Get(context.Background(), tokenSecretKey, secret))
	assert.True(t, metav1.IsControlledBy(secret, f.plugin()), "the Secret is garbage collected with its Plugin")

	f.now = tokenT0.Add(time.Hour)
	res = f.reconcile()
	assert.Equal(t, first, f.token(), "a current token is kept")
	assert.Equal(t, 15*time.Hour, res.RequeueAfter)

	f.now = tokenT0.Add(16 * time.Hour)
	f.reconcile()
	assert.NotEqual(t, first, f.token(), "the token is renewed with a third of its lifetime left")
}

func TestPluginTokenFollowsPermissions(t *testing.T) {
	t.Parallel()
	f := newTokenFixture(t, true, daemonPlugin())
	f.reconcile()
	first := f.token()

	p := f.plugin()
	p.Spec.Permissions = append(p.Spec.Permissions, pluginv1alpha1.PluginPermission{Verb: "read", Resource: "platform-events"})
	require.NoError(t, f.c.Update(context.Background(), p))
	f.reconcile()
	assert.NotEqual(t, first, f.token())

	p = f.plugin()
	p.Spec.Enabled = false
	require.NoError(t, f.c.Update(context.Background(), p))
	f.reconcile()
	err := f.c.Get(context.Background(), tokenSecretKey, &corev1.Secret{})
	assert.True(t, apierrors.IsNotFound(err), "a disabled plugin has no token")
	assert.Nil(t, f.condition())
}

func TestPluginTokenMovesWithTheBackend(t *testing.T) {
	t.Parallel()
	f := newTokenFixture(t, true, daemonPlugin())
	f.reconcile()

	p := f.plugin()
	p.Spec.Backend.ServiceRef.Namespace = "audit"
	require.NoError(t, f.c.Update(context.Background(), p))
	f.reconcile()

	err := f.c.Get(context.Background(), tokenSecretKey, &corev1.Secret{})
	assert.True(t, apierrors.IsNotFound(err))
	require.NoError(t, f.c.Get(context.Background(), types.NamespacedName{Namespace: "audit", Name: "audit-token"}, &corev1.Secret{}))
}

func TestPluginTokenWithoutSigningKey(t *testing.T) {
	t.Parallel()
	f := newTokenFixture(t, false, daemonPlugin())

	res := f.reconcile()
	assert.Zero(t, res.RequeueAfter)
	cond := f.condition()
	require.NotNil(t, cond)
	assert.Equal(t, metav1.ConditionFalse, cond.Status)
	assert.Equal(t, reasonSigningKeyUnavailable, cond.Reason)
}

func TestPluginTokenLeavesForeignSecretsAlone(t *testing.T) {
	t.Parallel()
	foreign := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Namespace: tokenSecretKey.Namespace, Name: tokenSecretKey.Name},
		Data:       map[string][]byte{"token": []byte("mine")},
	}
	f := newTokenFixture(t, true, daemonPlugin(), foreign)

	f.reconcile()
	assert.Equal(t, "mine", f.token())
	assert.Equal(t, reasonSecretConflict, f.condition().Reason)
}
