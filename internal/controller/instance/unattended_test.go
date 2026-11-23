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
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/managedfields"
	"k8s.io/kube-openapi/pkg/validation/spec"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/yaml"

	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/config/crd/bases"
	"github.com/openeverest/openeverest/v2/contract"
	rtmonitoring "github.com/openeverest/openeverest/v2/provider-runtime/monitoring"
)

//nolint:gochecknoglobals // fixed reference time
var t0 = time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

const testProvider = "psmdb"

func testInstance(mutate ...func(*corev1alpha1.Instance)) *corev1alpha1.Instance {
	in := &corev1alpha1.Instance{
		ObjectMeta: metav1.ObjectMeta{
			Name: "orders", Namespace: "team-a", UID: "uid-1", Generation: 1,
			CreationTimestamp: metav1.NewTime(t0),
		},
		Spec:   corev1alpha1.InstanceSpec{ProviderRef: common.ObjectRef{Name: testProvider}},
		Status: corev1alpha1.InstanceStatus{ObservedGeneration: -1},
	}
	for _, m := range mutate {
		m(in)
	}
	return in
}

func testProviderObject(name string, created time.Time) *corev1alpha1.Provider {
	return &corev1alpha1.Provider{ObjectMeta: metav1.ObjectMeta{Name: name, CreationTimestamp: metav1.NewTime(created)}}
}

// pickedUp marks an Instance as a provider-runtime marks the Instances it
// reconciles.
func pickedUp(in *corev1alpha1.Instance) {
	in.Finalizers = []string{contract.ProviderFinalizer}
	in.Labels = map[string]string{contract.ProviderLabel: testProvider}
}

// observed says that a provider reported on generation, and that the spec
// has been edited since.
func observed(observedGeneration, generation int64) func(*corev1alpha1.Instance) {
	return func(in *corev1alpha1.Instance) {
		in.Generation = generation
		in.Status.ObservedGeneration = observedGeneration
		in.CreationTimestamp = metav1.NewTime(t0.Add(-30 * 24 * time.Hour))
	}
}

func withCondition(status metav1.ConditionStatus, reason string) func(*corev1alpha1.Instance) {
	return func(in *corev1alpha1.Instance) {
		in.Status.Conditions = append(in.Status.Conditions, metav1.Condition{
			Type: corev1alpha1.ConditionUnattended, Status: status, Reason: reason, Message: "written before",
			ObservedGeneration: in.Generation, LastTransitionTime: metav1.NewTime(t0.Add(-time.Hour)),
		})
	}
}

// crdTypeConverter makes the fake client's server-side apply follow the
// Instance CRD, so conditions merge by type.
func crdTypeConverter(t *testing.T) managedfields.TypeConverter { //nolint:ireturn // the fake client takes the interface
	t.Helper()
	raw, err := bases.CRDs.ReadFile("core.openeverest.io_instances.yaml")
	require.NoError(t, err)
	crd := &apiextensionsv1.CustomResourceDefinition{}
	require.NoError(t, yaml.Unmarshal(raw, crd))
	models := make(map[string]*spec.Schema)
	for _, v := range crd.Spec.Versions {
		js, err := json.Marshal(v.Schema.OpenAPIV3Schema)
		require.NoError(t, err)
		s := &spec.Schema{}
		require.NoError(t, json.Unmarshal(js, s))
		s.AddExtension("x-kubernetes-group-version-kind", []any{
			map[string]any{"group": crd.Spec.Group, "version": v.Name, "kind": crd.Spec.Names.Kind},
		})
		models[crd.Spec.Group+"."+v.Name+"."+crd.Spec.Names.Kind] = s
	}
	tc, err := managedfields.NewTypeConverter(models, false)
	require.NoError(t, err)
	return tc
}

// harness drives the reconciler with a clock the test moves, and records
// the applies it makes.
type harness struct {
	t       *testing.T
	c       client.WithWatch
	r       *UnattendedReconciler
	now     time.Time
	applies []applied
}

// applied is what one status apply asked for.
type applied struct {
	opts            client.SubResourceApplyOptions
	resourceVersion string
}

func newHarness(t *testing.T, objs ...client.Object) *harness {
	t.Helper()
	h := &harness{t: t, now: t0}
	scheme := runtime.NewScheme()
	require.NoError(t, corev1alpha1.AddToScheme(scheme))
	h.c = fake.NewClientBuilder().
		WithScheme(scheme).
		WithTypeConverters(crdTypeConverter(t), managedfields.NewDeducedTypeConverter()).
		WithReturnManagedFields().
		WithObjects(objs...).
		WithStatusSubresource(&corev1alpha1.Instance{}).
		Build()
	// Only the reconciler's applies are recorded, not the stand-in
	// provider's.
	recorded := interceptor.NewClient(h.c, interceptor.Funcs{
		SubResourceApply: func(
			ctx context.Context, c client.Client, subResource string, obj runtime.ApplyConfiguration, opts ...client.SubResourceApplyOption,
		) error {
			h.applies = append(h.applies, recordApply(t, obj, opts))
			return c.SubResource(subResource).Apply(ctx, obj, opts...)
		},
	})
	h.r = &UnattendedReconciler{Client: recorded, Clock: func() time.Time { return h.now }}
	return h
}

func recordApply(t *testing.T, obj runtime.ApplyConfiguration, opts []client.SubResourceApplyOption) applied {
	t.Helper()
	raw, err := json.Marshal(obj)
	require.NoError(t, err)
	var body struct {
		Metadata struct {
			ResourceVersion string `json:"resourceVersion"`
		} `json:"metadata"`
	}
	require.NoError(t, json.Unmarshal(raw, &body))
	o := client.SubResourceApplyOptions{}
	o.ApplyOpts(opts)
	return applied{opts: o, resourceVersion: body.Metadata.ResourceVersion}
}

func (h *harness) key() client.ObjectKey {
	return client.ObjectKey{Namespace: "team-a", Name: "orders"}
}

func (h *harness) reconcile() (*corev1alpha1.Instance, ctrl.Result) {
	h.t.Helper()
	res, err := h.r.Reconcile(h.t.Context(), reconcile.Request{NamespacedName: h.key()})
	require.NoError(h.t, err)
	got := &corev1alpha1.Instance{}
	require.NoError(h.t, h.c.Get(h.t.Context(), h.key(), got))
	return got, res
}

// catchUp stands for the provider reporting on the current generation, as
// provider-runtime does: under its own field owner, with ownership forced.
func (h *harness) catchUp() {
	h.t.Helper()
	in := &corev1alpha1.Instance{}
	require.NoError(h.t, h.c.Get(h.t.Context(), h.key(), in))
	require.NoError(h.t, rtmonitoring.ApplyStatus(h.t.Context(), h.c, in, contract.ProviderStatusFieldOwner(testProvider),
		map[string]any{"observedGeneration": in.Generation, "phase": string(corev1alpha1.InstancePhasePending)},
	))
}

// editSpec stands for a change to the Instance's spec: a new generation.
func (h *harness) editSpec() {
	h.t.Helper()
	in := &corev1alpha1.Instance{}
	require.NoError(h.t, h.c.Get(h.t.Context(), h.key(), in))
	in.Generation++
	require.NoError(h.t, h.c.Update(h.t.Context(), in))
}

// seed stands for a condition an earlier run of core wrote.
func (h *harness) seed(status metav1.ConditionStatus, reason string) {
	h.t.Helper()
	in := &corev1alpha1.Instance{}
	require.NoError(h.t, h.c.Get(h.t.Context(), h.key(), in))
	withCondition(status, reason)(in)
	raw, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&in.Status.Conditions[len(in.Status.Conditions)-1])
	require.NoError(h.t, err)
	require.NoError(h.t, rtmonitoring.ApplyStatus(h.t.Context(), h.c, in, UnattendedFieldOwner,
		map[string]any{"conditions": []any{raw}},
	))
}

func (h *harness) installProvider() {
	h.t.Helper()
	require.NoError(h.t, h.c.Create(h.t.Context(), testProviderObject(testProvider, h.now)))
}

// want is the Unattended condition a step must leave, nil for none.
type want struct {
	status     metav1.ConditionStatus
	reason     string
	message    string
	generation int64
	since      time.Time
}

type step struct {
	name    string
	advance time.Duration
	do      func(h *harness)
	want    *want
	requeue time.Duration
	// writes is how many applies were made up to the end of the step.
	writes int
}

func TestUnattendedReconciler(t *testing.T) {
	t.Parallel()

	const (
		notInstalled = `Provider "psmdb" is not installed. Installed Providers: postgresql. ` +
			`spec.providerRef cannot be changed: delete the Instance and create it again with the right provider.`
		notPickedUp = `Provider "psmdb" has not picked this Instance up for 5 minutes; check that it is running.`
		notReported = `Provider "psmdb" picked this Instance up but has not reported on generation 3 for 5 minutes; check its logs.`
	)
	providerNotFound := func(since time.Duration, generation int64) *want {
		return &want{
			status: metav1.ConditionTrue, reason: corev1alpha1.ReasonProviderNotFound, message: notInstalled,
			generation: generation, since: t0.Add(since),
		}
	}
	notResponding := func(message string, since time.Duration, generation int64) *want {
		return &want{
			status: metav1.ConditionUnknown, reason: corev1alpha1.ReasonNotResponding, message: message,
			generation: generation, since: t0.Add(since),
		}
	}
	installedLongAgo := testProviderObject(testProvider, t0.Add(-24*time.Hour))
	otherProvider := testProviderObject("postgresql", t0.Add(-24*time.Hour))

	tests := []struct {
		name string
		objs []client.Object
		// stored is the condition an earlier run of core left, if any.
		stored *metav1.Condition
		steps  []step
	}{
		{
			name: "a missing Provider is reported once the grace has passed",
			objs: []client.Object{testInstance(), otherProvider},
			steps: []step{
				{name: "first seen", requeue: time.Minute},
				{name: "within the grace", advance: 59 * time.Second, requeue: time.Second},
				{name: "grace passed", advance: time.Second, want: providerNotFound(time.Minute, 1), writes: 1},
				{name: "nothing changes", advance: time.Hour, want: providerNotFound(time.Minute, 1), writes: 1},
			},
		},
		{
			name: "a Provider that appears withdraws the verdict and gets its own deadline",
			objs: []client.Object{testInstance(), otherProvider},
			steps: []step{
				{name: "first seen", requeue: time.Minute},
				{name: "not found", advance: time.Minute, want: providerNotFound(time.Minute, 1), writes: 1},
				{
					name: "installed", advance: time.Minute, do: (*harness).installProvider,
					requeue: 5 * time.Minute, writes: 2,
				},
				{name: "deadline passed", advance: 5 * time.Minute, want: notResponding(notPickedUp, 7*time.Minute, 1), writes: 3},
			},
		},
		{
			name: "a Provider that exists is given five minutes",
			objs: []client.Object{testInstance(), installedLongAgo},
			steps: []step{
				{name: "first seen", requeue: 5 * time.Minute},
				{name: "within the deadline", advance: 5*time.Minute - time.Second, requeue: time.Second},
				{name: "deadline passed", advance: time.Second, want: notResponding(notPickedUp, 5*time.Minute, 1), writes: 1},
				{name: "the provider catches up", advance: time.Minute, do: (*harness).catchUp, writes: 2},
				{name: "nothing changes", advance: time.Hour, writes: 2},
			},
		},
		{
			name: "a provider that catches up in time leaves nothing to write",
			objs: []client.Object{testInstance(), installedLongAgo},
			steps: []step{
				{name: "first seen", requeue: 5 * time.Minute},
				{name: "the provider catches up", advance: time.Minute, do: (*harness).catchUp},
			},
		},
		{
			name: "an edit nobody observes",
			objs: []client.Object{testInstance(observed(2, 3), pickedUp), installedLongAgo},
			steps: []step{
				{name: "first seen", requeue: 5 * time.Minute},
				{name: "deadline passed", advance: 5 * time.Minute, want: notResponding(notReported, 5*time.Minute, 3), writes: 1},
				{
					name: "edited again: the verdict stands while the clock runs again", advance: time.Minute, do: (*harness).editSpec,
					want: notResponding(notReported, 5*time.Minute, 3), requeue: 5 * time.Minute, writes: 1,
				},
				{
					name: "deadline passed for the new generation", advance: 5 * time.Minute,
					want: notResponding(
						`Provider "psmdb" picked this Instance up but has not reported on generation 4 for 5 minutes; check its logs.`,
						5*time.Minute, 4,
					),
					writes: 2,
				},
				{name: "the provider catches up", advance: time.Minute, do: (*harness).catchUp, writes: 3},
			},
		},
		{
			name:   "a restart keeps a verdict the facts still support",
			objs:   []client.Object{testInstance(observed(2, 3)), installedLongAgo},
			stored: &metav1.Condition{Status: metav1.ConditionUnknown, Reason: corev1alpha1.ReasonNotResponding},
			steps: []step{
				{
					name: "first seen by this process", requeue: 5 * time.Minute,
					want: &want{
						status: metav1.ConditionUnknown, reason: corev1alpha1.ReasonNotResponding, message: "written before",
						generation: 3, since: t0.Add(-time.Hour),
					},
				},
				{
					name: "deadline passed", advance: 5 * time.Minute,
					want: notResponding(
						`Provider "psmdb" has not picked this Instance up for 5 minutes; check that it is running.`, -time.Hour, 3,
					),
					writes: 1,
				},
			},
		},
		{
			name:   "a restart withdraws a verdict the facts no longer support",
			objs:   []client.Object{testInstance(), installedLongAgo},
			stored: &metav1.Condition{Status: metav1.ConditionTrue, Reason: corev1alpha1.ReasonProviderNotFound},
			steps:  []step{{name: "first seen by this process", requeue: 5 * time.Minute, writes: 1}},
		},
		{
			name:   "a status that is current carries no condition",
			objs:   []client.Object{testInstance(observed(3, 3))},
			stored: &metav1.Condition{Status: metav1.ConditionUnknown, Reason: corev1alpha1.ReasonNotResponding},
			steps:  []step{{name: "withdrawn", writes: 1}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			h := newHarness(t, tc.objs...)
			if tc.stored != nil {
				h.seed(tc.stored.Status, tc.stored.Reason)
			}
			for _, s := range tc.steps {
				h.now = h.now.Add(s.advance)
				if s.do != nil {
					s.do(h)
				}
				got, res := h.reconcile()

				cond := meta.FindStatusCondition(got.Status.Conditions, corev1alpha1.ConditionUnattended)
				if s.want == nil {
					assert.Nil(t, cond, "%s: no condition", s.name)
				} else if assert.NotNil(t, cond, "%s: the condition", s.name) {
					assert.Equal(t, s.want.status, cond.Status, "%s: status", s.name)
					assert.Equal(t, s.want.reason, cond.Reason, "%s: reason", s.name)
					assert.Equal(t, s.want.message, cond.Message, "%s: message", s.name)
					assert.Equal(t, s.want.generation, cond.ObservedGeneration, "%s: observedGeneration", s.name)
					assert.True(t, s.want.since.Equal(cond.LastTransitionTime.Time),
						"%s: lastTransitionTime: want %s, got %s", s.name, s.want.since, cond.LastTransitionTime,
					)
				}
				assert.Equal(t, s.requeue, res.RequeueAfter, "%s: requeue", s.name)
				assert.Len(t, h.applies, s.writes, "%s: writes", s.name)
			}
			for _, a := range h.applies {
				assert.Equal(t, UnattendedFieldOwner, a.opts.FieldManager)
				assert.Nil(t, a.opts.Force, "ownership is never forced")
				assert.NotEmpty(t, a.resourceVersion, "the resourceVersion read is a precondition")
			}
		})
	}
}

// Core never takes over what the provider writes: its applies leave the
// provider's fields alone, and the provider's leave core's condition.
func TestUnattendedReconciler_SharesTheStatus(t *testing.T) {
	t.Parallel()

	h := newHarness(t, testInstance(observed(2, 3), pickedUp), testProviderObject(testProvider, t0.Add(-time.Hour)))
	h.reconcile()
	h.now = h.now.Add(5 * time.Minute)
	got, _ := h.reconcile()
	require.NotNil(t, meta.FindStatusCondition(got.Status.Conditions, corev1alpha1.ConditionUnattended))
	assert.Equal(t, int64(2), got.Status.ObservedGeneration, "the provider's field is left alone")

	h.catchUp()
	got = &corev1alpha1.Instance{}
	require.NoError(t, h.c.Get(t.Context(), h.key(), got))
	require.NotNil(t, meta.FindStatusCondition(got.Status.Conditions, corev1alpha1.ConditionUnattended),
		"the provider's apply leaves core's condition alone",
	)

	got, _ = h.reconcile()
	assert.Nil(t, meta.FindStatusCondition(got.Status.Conditions, corev1alpha1.ConditionUnattended))
	assert.Equal(t, int64(3), got.Status.ObservedGeneration)
	assert.Equal(t, corev1alpha1.InstancePhasePending, got.Status.Phase)
}

func TestUnattendedReconciler_Conflict(t *testing.T) {
	t.Parallel()

	instance := testInstance()
	scheme := runtime.NewScheme()
	require.NoError(t, corev1alpha1.AddToScheme(scheme))
	c := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(instance).
		WithStatusSubresource(&corev1alpha1.Instance{}).
		WithInterceptorFuncs(interceptor.Funcs{
			SubResourceApply: func(context.Context, client.Client, string, runtime.ApplyConfiguration, ...client.SubResourceApplyOption) error {
				return apierrors.NewConflict(schema.GroupResource{Group: corev1alpha1.GroupVersion.Group, Resource: "instances"},
					instance.Name, nil,
				)
			},
		}).
		Build()
	now := t0
	r := &UnattendedReconciler{Client: c, Clock: func() time.Time { return now }}
	req := reconcile.Request{NamespacedName: client.ObjectKeyFromObject(instance)}

	_, err := r.Reconcile(t.Context(), req)
	require.NoError(t, err)
	now = now.Add(time.Minute)
	res, err := r.Reconcile(t.Context(), req)
	require.NoError(t, err)
	assert.Equal(t, conflictRequeue, res.RequeueAfter, "a write that lost is decided again soon")
}

func TestUnattendedReconciler_ForgetsDeletedInstances(t *testing.T) {
	t.Parallel()

	h := newHarness(t, testInstance())
	h.reconcile()
	require.Len(t, h.r.seen.instances, 1)

	require.NoError(t, h.c.Delete(t.Context(), testInstance()))
	_, err := h.r.Reconcile(t.Context(), reconcile.Request{NamespacedName: h.key()})
	require.NoError(t, err)
	assert.Empty(t, h.r.seen.instances)
}

func TestProviderNotFoundMessage(t *testing.T) {
	t.Parallel()

	const howTo = "spec.providerRef cannot be changed: delete the Instance and create it again with the right provider."
	tests := []struct {
		name      string
		installed []string
		want      string
	}{
		{
			name: "none installed",
			want: `Provider "psmdb" is not installed. No Provider is installed. ` + howTo,
		},
		{
			name:      "a few installed",
			installed: []string{"postgresql", "percona-xtradb-cluster"},
			want:      `Provider "psmdb" is not installed. Installed Providers: postgresql, percona-xtradb-cluster. ` + howTo,
		},
		{
			name:      "five installed",
			installed: []string{"a", "b", "c", "d", "e"},
			want:      `Provider "psmdb" is not installed. Installed Providers: a, b, c, d, e. ` + howTo,
		},
		{
			name:      "more than five installed",
			installed: []string{"a", "b", "c", "d", "e", "f", "g"},
			want:      `Provider "psmdb" is not installed. Installed Providers: a, b, c, d, e and 2 more. ` + howTo,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, providerNotFoundMessage(testProvider, tc.installed))
		})
	}
}

func TestNotRespondingMessage(t *testing.T) {
	t.Parallel()

	const (
		pickedUpMessage = `Provider "psmdb" picked this Instance up but has not reported on generation 3 for 5 minutes; check its logs.`
		notPickedUp     = `Provider "psmdb" has not picked this Instance up for 5 minutes; check that it is running.`
	)
	tests := []struct {
		name   string
		mutate func(*corev1alpha1.Instance)
		want   string
	}{
		{name: "neither finalizer nor label", mutate: func(*corev1alpha1.Instance) {}, want: notPickedUp},
		{name: "finalizer and label", mutate: pickedUp, want: pickedUpMessage},
		{
			name:   "finalizer only",
			mutate: func(in *corev1alpha1.Instance) { in.Finalizers = []string{contract.ProviderFinalizer} },
			want:   pickedUpMessage,
		},
		{
			name:   "label only",
			mutate: func(in *corev1alpha1.Instance) { in.Labels = map[string]string{contract.ProviderLabel: testProvider} },
			want:   pickedUpMessage,
		},
		{
			name:   "another finalizer",
			mutate: func(in *corev1alpha1.Instance) { in.Finalizers = []string{"example.com/keep"} },
			want:   notPickedUp,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, notRespondingMessage(testInstance(observed(2, 3), tc.mutate), DefaultResponseDeadline))
		})
	}
}

func TestSpellDuration(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "1 minute", spellDuration(time.Minute))
	assert.Equal(t, "5 minutes", spellDuration(5*time.Minute))
	assert.Equal(t, "1m30s", spellDuration(90*time.Second))
	assert.Equal(t, "30s", spellDuration(30*time.Second))
}

func TestUnattendedEvents(t *testing.T) {
	t.Parallel()

	current := testInstance(observed(3, 3))
	lagging := testInstance(observed(2, 3))
	stale := testInstance(observed(3, 3), withCondition(metav1.ConditionUnknown, corev1alpha1.ReasonNotResponding))

	tests := []struct {
		name     string
		old, new *corev1alpha1.Instance
		want     bool
	}{
		{name: "edited", old: current, new: lagging, want: true},
		{name: "caught up", old: lagging, new: current, want: true},
		{name: "still behind", old: lagging, new: lagging, want: true},
		{name: "current, carrying the condition", old: stale, new: stale, want: true},
		{name: "current, without the condition", old: current, new: current, want: false},
	}
	p := unattendedEvents()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, p.Update(event.UpdateEvent{ObjectOld: tc.old, ObjectNew: tc.new}))
		})
	}
	assert.True(t, p.Create(event.CreateEvent{Object: current}))
	assert.True(t, p.Delete(event.DeleteEvent{Object: current}))
}

func TestInstancesOfProvider(t *testing.T) {
	t.Parallel()

	named := func(name, provider string, mutate ...func(*corev1alpha1.Instance)) *corev1alpha1.Instance {
		return testInstance(append(mutate, func(in *corev1alpha1.Instance) {
			in.Name = name
			in.UID = types.UID("uid-" + name)
			in.Spec.ProviderRef.Name = provider
		})...)
	}
	h := newHarness(t,
		named("new", testProvider),
		named("lagging", testProvider, observed(1, 2)),
		named("current", testProvider, observed(2, 2)),
		named("stale", testProvider, observed(2, 2), withCondition(metav1.ConditionTrue, corev1alpha1.ReasonProviderNotFound)),
		named("other", "postgresql"),
		named("typo", "percona-server-mongdb", withCondition(metav1.ConditionTrue, corev1alpha1.ReasonProviderNotFound)),
	)

	reqs := h.r.instancesOfProvider(t.Context(), testProviderObject(testProvider, t0))

	names := make([]string, 0, len(reqs))
	for _, req := range reqs {
		names = append(names, req.Name)
	}
	assert.ElementsMatch(t, []string{"new", "lagging", "stale", "typo"}, names,
		"an Instance whose Provider was not found lists the installed ones, so any Provider change requeues it",
	)
}
