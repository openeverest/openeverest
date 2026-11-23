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

package instance

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/contract"
	rtmonitoring "github.com/openeverest/openeverest/v2/provider-runtime/monitoring"
)

// The tests in this file run against a real API server, as server-side
// apply, its preconditions and CRD defaulting are its behaviour. They are
// skipped unless KUBEBUILDER_ASSETS is set, as make test does.
type testAPIServer struct {
	once sync.Once
	env  *envtest.Environment
	cfg  *rest.Config
	err  error
}

// sharedAPIServer is started by the first test that needs it.
var sharedAPIServer testAPIServer //nolint:gochecknoglobals // one API server per test binary

func TestMain(m *testing.M) {
	code := m.Run()
	if sharedAPIServer.env != nil {
		_ = sharedAPIServer.env.Stop()
	}
	os.Exit(code)
}

// apiServerClient returns a client of the shared test API server, which
// serves the CRDs of config/crd/bases, and a namespace of its own.
func apiServerClient(t *testing.T) (client.WithWatch, string) { //nolint:ireturn // the client interface is what tests use
	t.Helper()
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		t.Skip("KUBEBUILDER_ASSETS is not set: run make test")
	}
	s := &sharedAPIServer
	s.once.Do(func() {
		s.env = &envtest.Environment{
			CRDDirectoryPaths:     []string{filepath.Join("..", "..", "..", "config", "crd", "bases")},
			ErrorIfCRDPathMissing: true,
		}
		s.cfg, s.err = s.env.Start()
	})
	require.NoError(t, s.err)
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, corev1alpha1.AddToScheme(scheme))
	c, err := client.NewWithWatch(s.cfg, client.Options{Scheme: scheme})
	require.NoError(t, err)
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{GenerateName: "test-"}}
	require.NoError(t, c.Create(t.Context(), ns))
	return c, ns.Name
}

func apiServerInstance(namespace, provider string) *corev1alpha1.Instance {
	return &corev1alpha1.Instance{
		ObjectMeta: metav1.ObjectMeta{Name: "orders", Namespace: namespace},
		Spec: corev1alpha1.InstanceSpec{
			ProviderRef: common.ObjectRef{Name: provider},
			Version:     "7.0",
			Components:  map[string]corev1alpha1.ComponentSpec{"engine": {Replicas: new(int32(3))}},
		},
	}
}

// standIn plays provider-runtime: it applies the status fields it owns
// under its own field owner, forcing ownership.
type standIn struct {
	t    *testing.T
	c    client.Client
	name string
}

// report applies a status that describes generation.
func (s standIn) report(in *corev1alpha1.Instance, generation int64, message string) {
	s.t.Helper()
	ready, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&metav1.Condition{
		Type: corev1alpha1.ConditionReady, Status: metav1.ConditionFalse, Reason: corev1alpha1.ReasonCreating,
		Message: message, ObservedGeneration: generation, LastTransitionTime: metav1.Now(),
	})
	require.NoError(s.t, err)
	require.NoError(s.t, rtmonitoring.ApplyStatus(s.t.Context(), s.c, in, contract.ProviderStatusFieldOwner(s.name), map[string]any{
		"observedGeneration": generation,
		"phase":              string(corev1alpha1.InstancePhasePending),
		"conditions":         []any{ready},
	}))
}

// statusOwners returns the field managers that own fields of obj's status,
// with their operation.
func statusOwners(obj client.Object) []string {
	var owners []string
	for _, mf := range obj.GetManagedFields() {
		if mf.FieldsV1 != nil && strings.Contains(mf.FieldsV1.GetRawString(), `"f:status"`) {
			owners = append(owners, mf.Manager+"/"+string(mf.Operation))
		}
	}
	slices.Sort(owners)
	return owners
}

// TestAPIServer_Unattended checks, on a real API server, that core's
// condition and the runtime's status fields are applied side by side, that
// the runtime catching up has core withdraw its condition, and that a
// verdict reached on a stale read is refused.
func TestAPIServer_Unattended(t *testing.T) {
	t.Parallel()
	c, ns := apiServerClient(t)

	provider := &corev1alpha1.Provider{
		ObjectMeta: metav1.ObjectMeta{Name: "envtest-unattended"},
		Spec: corev1alpha1.ProviderSpec{
			Components: map[string]corev1alpha1.Component{"engine": {Type: "db"}},
			Versions:   []corev1alpha1.VersionBundle{{Name: "7.0"}},
		},
	}
	require.NoError(t, c.Create(t.Context(), provider))
	t.Cleanup(func() { _ = c.Delete(context.Background(), provider) })
	runtimeOwner := contract.ProviderStatusFieldOwner(provider.Name)
	stand := standIn{t: t, c: c, name: provider.Name}

	in := apiServerInstance(ns, provider.Name)
	require.NoError(t, c.Create(t.Context(), in))
	key := client.ObjectKeyFromObject(in)
	get := func() *corev1alpha1.Instance {
		t.Helper()
		got := &corev1alpha1.Instance{}
		require.NoError(t, c.Get(t.Context(), key, got))
		return got
	}

	// The stand-in picks the Instance up and reports on it, then the spec
	// is edited and the stand-in stops reporting.
	base := get()
	pickedUp := base.DeepCopy()
	pickedUp.Labels = map[string]string{contract.ProviderLabel: provider.Name}
	pickedUp.Finalizers = []string{contract.ProviderFinalizer}
	require.NoError(t, c.Patch(t.Context(), pickedUp, client.MergeFrom(base)))
	stand.report(pickedUp, 1, "waiting for the replica set")
	edited := get()
	edited.Spec.Components["engine"] = corev1alpha1.ComponentSpec{Replicas: new(int32(5))}
	require.NoError(t, c.Update(t.Context(), edited))
	require.Equal(t, int64(2), get().Generation)

	// The reconciler reads through a cache that can lag behind.
	var lagging *corev1alpha1.Instance
	cached := interceptor.NewClient(c, interceptor.Funcs{
		Get: func(ctx context.Context, c client.WithWatch, k client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
			if got, ok := obj.(*corev1alpha1.Instance); ok && lagging != nil {
				lagging.DeepCopyInto(got)
				return nil
			}
			return c.Get(ctx, k, obj, opts...)
		},
	})
	now := time.Now()
	r := &UnattendedReconciler{Client: cached, Clock: func() time.Time { return now }}
	reconcileAt := func(at time.Time) reconcile.Result {
		t.Helper()
		now = at
		res, err := r.Reconcile(t.Context(), reconcile.Request{NamespacedName: key})
		require.NoError(t, err)
		return res
	}

	res := reconcileAt(now)
	assert.Equal(t, DefaultResponseDeadline, res.RequeueAfter, "first seen")

	// A verdict reached on a stale read is refused.
	lagging = get()
	stand.report(lagging, 1, "still waiting for the replica set")
	res = reconcileAt(now.Add(DefaultResponseDeadline))
	lagging = nil
	assert.Equal(t, conflictRequeue, res.RequeueAfter)
	assert.Nil(t, meta.FindStatusCondition(get().Status.Conditions, corev1alpha1.ConditionUnattended))

	// Core's condition and the runtime's fields are applied side by side.
	reconcileAt(now)
	got := get()
	cond := meta.FindStatusCondition(got.Status.Conditions, corev1alpha1.ConditionUnattended)
	require.NotNil(t, cond)
	assert.Equal(t, metav1.ConditionUnknown, cond.Status)
	assert.Equal(t, corev1alpha1.ReasonNotResponding, cond.Reason)
	assert.Equal(t, int64(2), cond.ObservedGeneration)
	assert.Contains(t, cond.Message, "has not reported on generation 2")
	assert.Equal(t, int64(1), got.Status.ObservedGeneration)
	assert.Equal(t, corev1alpha1.InstancePhasePending, got.Status.Phase)
	requireReadyMessage(t, got, "still waiting for the replica set")
	assert.Equal(t, []string{UnattendedFieldOwner + "/Apply", runtimeOwner + "/Apply"}, statusOwners(got))

	stand.report(got, 1, "the runtime reports again")
	got = get()
	require.NotNil(t, meta.FindStatusCondition(got.Status.Conditions, corev1alpha1.ConditionUnattended),
		"the runtime's forced apply leaves core's condition alone",
	)
	requireReadyMessage(t, got, "the runtime reports again")

	// The runtime catching up has core withdraw its condition.
	stand.report(got, 2, "scaling to five")
	require.NotNil(t, meta.FindStatusCondition(get().Status.Conditions, corev1alpha1.ConditionUnattended))
	res = reconcileAt(now.Add(time.Second))
	assert.Zero(t, res.RequeueAfter)
	got = get()
	assert.Nil(t, meta.FindStatusCondition(got.Status.Conditions, corev1alpha1.ConditionUnattended))
	assert.Equal(t, int64(2), got.Status.ObservedGeneration)
	requireReadyMessage(t, got, "scaling to five")
	assert.Equal(t, []string{runtimeOwner + "/Apply"}, statusOwners(got))
}

func requireReadyMessage(t *testing.T, in *corev1alpha1.Instance, message string) {
	t.Helper()
	ready := meta.FindStatusCondition(in.Status.Conditions, corev1alpha1.ConditionReady)
	require.NotNil(t, ready)
	assert.Equal(t, message, ready.Message)
}

// TestAPIServer_UnattendedKeepsDefaults checks that core's apply on an
// Instance no provider has written keeps the status the CRD defaults, so it
// still reads as never reconciled.
func TestAPIServer_UnattendedKeepsDefaults(t *testing.T) {
	t.Parallel()
	c, ns := apiServerClient(t)

	in := apiServerInstance(ns, "envtest-not-installed")
	require.NoError(t, c.Create(t.Context(), in))
	now := time.Now()
	r := &UnattendedReconciler{Client: c, Clock: func() time.Time { return now }}
	req := reconcile.Request{NamespacedName: client.ObjectKeyFromObject(in)}
	_, err := r.Reconcile(t.Context(), req)
	require.NoError(t, err)
	now = now.Add(DefaultProviderGrace)
	_, err = r.Reconcile(t.Context(), req)
	require.NoError(t, err)

	got := &corev1alpha1.Instance{}
	require.NoError(t, c.Get(t.Context(), client.ObjectKeyFromObject(in), got))
	cond := meta.FindStatusCondition(got.Status.Conditions, corev1alpha1.ConditionUnattended)
	require.NotNil(t, cond)
	assert.Equal(t, metav1.ConditionTrue, cond.Status)
	assert.Equal(t, corev1alpha1.ReasonProviderNotFound, cond.Reason)
	assert.Contains(t, cond.Message, `Provider "envtest-not-installed" is not installed.`)
	assert.Equal(t, int64(-1), got.Status.ObservedGeneration)
	assert.Equal(t, corev1alpha1.InstancePhasePending, got.Status.Phase)
}
