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
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
	"github.com/openeverest/openeverest/v2/provider-runtime/summary"
)

// readyTestProvider is a provider that does what the test tells it to.
type readyTestProvider struct {
	validateErrs field.ErrorList
	status       controller.Status
	// calls counts the calls of Validate.
	calls int

	// sync runs inside Sync, before syncErr is returned.
	sync       func(*controller.Context)
	syncErr    error
	statusErr  error
	cleanupErr error
	syncs      int
}

func (p *readyTestProvider) Name() string                       { return "test-provider" }
func (p *readyTestProvider) Types() func(*runtime.Scheme) error { return nil }
func (p *readyTestProvider) Cleanup(*controller.Context) error  { return p.cleanupErr }

func (p *readyTestProvider) Sync(c *controller.Context) error {
	p.syncs++
	if p.sync != nil {
		p.sync(c)
	}

	return p.syncErr
}

func (p *readyTestProvider) Validate(controller.ValidationInput) controller.ValidationResult {
	p.calls++
	return controller.ValidationResult{Errors: p.validateErrs}
}

func (p *readyTestProvider) Status(*controller.Context) (controller.Status, error) {
	return p.status, p.statusErr
}

func readyTestInstance(mutate func(*v1alpha1.Instance)) *v1alpha1.Instance {
	in := &v1alpha1.Instance{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "orders",
			Namespace:  "team-a",
			Generation: 2,
			Labels:     map[string]string{controller.ProviderLabel: "test-provider"},
			Finalizers: []string{finalizerName},
		},
		Spec: v1alpha1.InstanceSpec{ProviderRef: common.ObjectRef{Name: "test-provider"}},
	}
	if mutate != nil {
		mutate(in)
	}
	return in
}

func readyTestProviderObject() *v1alpha1.Provider {
	return &v1alpha1.Provider{ObjectMeta: metav1.ObjectMeta{Name: "test-provider"}}
}

// readyTestClient returns a client that holds the objects and has the field
// indexes a backup provider's reconciler relies on.
func readyTestClient(objs ...client.Object) *fake.ClientBuilder {
	scheme := newTestScheme()
	_ = corev1.AddToScheme(scheme)
	return fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(objs...).
		WithStatusSubresource(&v1alpha1.Instance{}).
		WithIndex(&backupv1alpha1.Backup{}, controller.IndexBackupInstanceName, func(obj client.Object) []string {
			b, ok := obj.(*backupv1alpha1.Backup)
			if !ok || b.Spec.Origin.InstanceRef == nil {
				return nil
			}
			return []string{b.Spec.Origin.InstanceRef.Name}
		}).
		WithIndex(&backupv1alpha1.Restore{}, controller.IndexRestoreInstanceName, func(obj client.Object) []string {
			rs, ok := obj.(*backupv1alpha1.Restore)
			if !ok || rs.Spec.InstanceRef.Name == "" {
				return nil
			}
			return []string{rs.Spec.InstanceRef.Name}
		})
}

func newReadyTestReconciler(p providerAdapter, objs ...client.Object) *ProviderReconciler {
	return &ProviderReconciler{provider: p, Client: readyTestClient(objs...).Build()}
}

func reconcileAndGet(t *testing.T, r *ProviderReconciler) (*v1alpha1.Instance, error) {
	t.Helper()
	key := client.ObjectKey{Namespace: "team-a", Name: "orders"}
	_, err := r.Reconcile(t.Context(), reconcile.Request{NamespacedName: key})
	got := &v1alpha1.Instance{}
	require.NoError(t, r.Get(t.Context(), key, got))
	return got, err
}

func requireReady(t *testing.T, in *v1alpha1.Instance, status metav1.ConditionStatus, reason string) *metav1.Condition {
	t.Helper()
	c := meta.FindStatusCondition(in.Status.Conditions, v1alpha1.ConditionReady)
	require.NotNil(t, c, "Ready condition missing")
	assert.Equal(t, status, c.Status)
	assert.Equal(t, reason, c.Reason)
	assert.Equal(t, in.Generation, in.Status.ObservedGeneration)
	return c
}

func requireCondition(t *testing.T, in *v1alpha1.Instance, condType string, status metav1.ConditionStatus, reason string) *metav1.Condition {
	t.Helper()
	c := meta.FindStatusCondition(in.Status.Conditions, condType)
	require.NotNil(t, c, "%s condition missing", condType)
	assert.Equal(t, status, c.Status, condType)
	assert.Equal(t, reason, c.Reason, condType)
	return c
}

// statusTest drives a reconciler whose clock the test moves, and records what
// the reconciler asks for besides its result: status writes, and requeues it
// schedules itself because the pass ends in an error.
type statusTest struct {
	t        *testing.T
	r        *ProviderReconciler
	now      time.Time
	writes   int
	requeued []time.Duration
}

func newStatusTest(t *testing.T, p providerAdapter, objs ...client.Object) *statusTest {
	t.Helper()
	st := &statusTest{t: t, now: time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)}
	c := readyTestClient(objs...).WithInterceptorFuncs(interceptor.Funcs{
		SubResourceUpdate: func(ctx context.Context, c client.Client, subResource string, obj client.Object, opts ...client.SubResourceUpdateOption) error {
			st.writes++
			return c.SubResource(subResource).Update(ctx, obj, opts...)
		},
	}).Build()
	st.r = &ProviderReconciler{provider: p, Client: c, clock: func() time.Time { return st.now }}
	var record requeuer = func(_ reconcile.Request, after time.Duration) { st.requeued = append(st.requeued, after) }
	st.r.requeue.Store(&record)

	return st
}

func (st *statusTest) key() client.ObjectKey {
	return client.ObjectKey{Namespace: "team-a", Name: "orders"}
}

// reconcile runs one pass and returns the Instance as stored afterwards.
func (st *statusTest) reconcile() (*v1alpha1.Instance, reconcile.Result, error) {
	st.t.Helper()
	res, err := st.r.Reconcile(st.t.Context(), reconcile.Request{NamespacedName: st.key()})
	got := &v1alpha1.Instance{}
	require.NoError(st.t, st.r.Get(st.t.Context(), st.key(), got))

	return got, res, err
}

// pass runs one pass that must not fail.
func (st *statusTest) pass() (*v1alpha1.Instance, reconcile.Result) {
	st.t.Helper()
	got, res, err := st.reconcile()
	require.NoError(st.t, err)

	return got, res
}

func (st *statusTest) advance(d time.Duration) { st.now = st.now.Add(d) }

// editSpec stands for a change to the Instance's spec: a new generation.
func (st *statusTest) editSpec() {
	st.t.Helper()
	stored := &v1alpha1.Instance{}
	require.NoError(st.t, st.r.Get(st.t.Context(), st.key(), stored))
	generation := stored.Generation + 1
	stored.Generation = generation
	require.NoError(st.t, st.r.Update(st.t.Context(), stored))
	require.NoError(st.t, st.r.Get(st.t.Context(), st.key(), stored))
	require.Equal(st.t, generation, stored.Generation)
}

// lastRequeue returns the requeue the reconciler scheduled itself in the
// last pass, and forgets it.
func (st *statusTest) lastRequeue() time.Duration {
	st.t.Helper()
	require.NotEmpty(st.t, st.requeued, "no requeue was scheduled")
	last := st.requeued[len(st.requeued)-1]
	st.requeued = nil

	return last
}

// wantStatus is what a pass must leave on the Instance.
type wantStatus struct {
	phase       v1alpha1.InstancePhase
	available   metav1.ConditionStatus
	ready       metav1.ConditionStatus
	reason      string
	message     string
	reconciling bool
	stalled     bool
	attention   bool
}

func requireStatus(t *testing.T, in *v1alpha1.Instance, w wantStatus) {
	t.Helper()
	status := func(b bool) metav1.ConditionStatus {
		if b {
			return metav1.ConditionTrue
		}
		return metav1.ConditionFalse
	}
	assert.Equal(t, w.phase, in.Status.Phase, "phase")
	assert.Equal(t, in.Generation, in.Status.ObservedGeneration, "status.observedGeneration")
	for _, condType := range []string{
		v1alpha1.ConditionAvailable, v1alpha1.ConditionReady, v1alpha1.ConditionReconciling,
		v1alpha1.ConditionStalled, v1alpha1.ConditionNeedsAttention,
	} {
		c := meta.FindStatusCondition(in.Status.Conditions, condType)
		require.NotNil(t, c, "%s condition missing", condType)
		assert.Equal(t, in.Generation, c.ObservedGeneration, "%s must carry the generation it describes", condType)
	}
	assert.Equal(t, w.available, meta.FindStatusCondition(in.Status.Conditions, v1alpha1.ConditionAvailable).Status, "Available")
	ready := meta.FindStatusCondition(in.Status.Conditions, v1alpha1.ConditionReady)
	assert.Equal(t, w.ready, ready.Status, "Ready")
	assert.Equal(t, w.reason, ready.Reason, "the status word")
	if w.message != "" {
		assert.Equal(t, w.message, ready.Message, "Ready's message")
	}
	assert.Equal(t, status(w.reconciling), meta.FindStatusCondition(in.Status.Conditions, v1alpha1.ConditionReconciling).Status, "Reconciling")
	assert.Equal(t, status(w.stalled), meta.FindStatusCondition(in.Status.Conditions, v1alpha1.ConditionStalled).Status, "Stalled")
	assert.Equal(t, status(w.attention), meta.FindStatusCondition(in.Status.Conditions, v1alpha1.ConditionNeedsAttention).Status, "NeedsAttention")
	if w.ready != metav1.ConditionTrue {
		assert.NotEqual(t, w.reconciling, w.stalled, "exactly one of Reconciling and Stalled while not Ready")
	}
}

const (
	testPending     = v1alpha1.InstancePhasePending
	testProvisioned = v1alpha1.InstancePhaseProvisioned
)

func TestReconcile_FirstPassWritesNoStatus(t *testing.T) {
	t.Parallel()
	in := readyTestInstance(func(in *v1alpha1.Instance) {
		in.Labels = nil
		in.Finalizers = nil
	})
	st := newStatusTest(t, &readyTestProvider{status: controller.Ready()}, readyTestProviderObject(), in)

	got, res := st.pass()

	assert.Equal(t, []string{finalizerName}, got.Finalizers)
	assert.Zero(t, st.writes, "the status default says in progress until the next pass writes")
	assert.Empty(t, got.Status.Conditions)
	assert.True(t, res.Requeue) //nolint:staticcheck // the pass that follows is immediate
}

//nolint:tparallel // the subtests continue one story, in order
func TestReconcile_StatusReturns(t *testing.T) {
	t.Parallel()
	p := &readyTestProvider{status: controller.Progressing("Waiting for the replica set to initialise")}
	st := newStatusTest(t, p, readyTestProviderObject(), readyTestInstance(nil))

	got, res := st.pass()
	requireStatus(t, got, wantStatus{
		phase: testPending, available: metav1.ConditionFalse,
		ready: metav1.ConditionFalse, reason: v1alpha1.ReasonCreating, message: "Waiting for the replica set to initialise",
		reconciling: true,
	})
	requireCondition(t, got, v1alpha1.ConditionAvailable, metav1.ConditionFalse, v1alpha1.ReasonNotCreated)
	requireCondition(t, got, v1alpha1.ConditionReferencesResolved, metav1.ConditionTrue, v1alpha1.ReasonResolved)
	requireCondition(t, got, v1alpha1.ConditionSynced, metav1.ConditionTrue, v1alpha1.ReasonSynced)
	requireCondition(t, got, v1alpha1.ConditionEngineHealthy, metav1.ConditionTrue, v1alpha1.ReasonHealthy)
	assert.Equal(t, summary.DefaultProgressDeadline, res.RequeueAfter, "the unnamed wait is looked at again when its deadline passes")

	st.advance(time.Minute)
	p.status = controller.Ready()
	got, res = st.pass()
	requireStatus(t, got, wantStatus{
		phase: testProvisioned, available: metav1.ConditionTrue,
		ready: metav1.ConditionTrue, reason: v1alpha1.ReasonReady,
	})
	assert.Zero(t, res.RequeueAfter, "nothing is due")
	assert.Equal(t, 2, st.writes)

	t.Run("nothing is written when nothing changed", func(t *testing.T) {
		for range 3 {
			st.advance(time.Minute)
			st.pass()
		}
		assert.Equal(t, 2, st.writes)
	})
}

// Providers that still call the helpers that named a phase report nothing
// about availability. They must get what those phases meant, and no outage
// that they cannot see.
func TestReconcile_DeprecatedHelpers(t *testing.T) {
	t.Parallel()
	p := &readyTestProvider{status: controller.Provisioning("Cluster is being created")} //nolint:staticcheck // the deprecated helper is what is tested
	st := newStatusTest(t, p, readyTestProviderObject(), readyTestInstance(nil))

	got, _ := st.pass()
	requireStatus(t, got, wantStatus{
		phase: testPending, available: metav1.ConditionFalse,
		ready: metav1.ConditionFalse, reason: v1alpha1.ReasonCreating, message: "Cluster is being created", reconciling: true,
	})

	p.status = controller.Ready()
	got, _ = st.pass()
	requireStatus(t, got, wantStatus{
		phase: testProvisioned, available: metav1.ConditionTrue, ready: metav1.ConditionTrue, reason: v1alpha1.ReasonReady,
	})

	p.status = controller.Updating("Rolling out the change") //nolint:staticcheck // the deprecated helper is what is tested
	got, _ = st.pass()
	requireStatus(t, got, wantStatus{
		phase: testProvisioned, available: metav1.ConditionUnknown,
		ready: metav1.ConditionFalse, reason: v1alpha1.ReasonApplyingChanges, message: "Rolling out the change", reconciling: true,
	})
	requireCondition(t, got, v1alpha1.ConditionAvailable, metav1.ConditionUnknown, v1alpha1.ReasonNotReported)

	p.status = controller.Failed("replica set has no primary")
	got, res := st.pass()
	requireStatus(t, got, wantStatus{
		phase: testProvisioned, available: metav1.ConditionUnknown,
		ready: metav1.ConditionFalse, reason: v1alpha1.ReasonEngineError, message: "replica set has no primary",
		reconciling: true, attention: true,
	})
	requireCondition(t, got, v1alpha1.ConditionEngineHealthy, metav1.ConditionFalse, v1alpha1.ReasonEngineError)
	assert.Equal(t, summary.DefaultProblemDeadline, res.RequeueAfter)

	st.advance(summary.DefaultProblemDeadline)
	got, res = st.pass()
	requireStatus(t, got, wantStatus{
		phase: testProvisioned, available: metav1.ConditionUnknown,
		ready: metav1.ConditionFalse, reason: v1alpha1.ReasonEngineError, stalled: true, attention: true,
	})
	assert.Zero(t, res.RequeueAfter)

	p.status = controller.Ready()
	got, _ = st.pass()
	requireStatus(t, got, wantStatus{
		phase: testProvisioned, available: metav1.ConditionTrue, ready: metav1.ConditionTrue, reason: v1alpha1.ReasonReady,
	})
}

func TestReconcile_MissingReferentStallsAtTheDeadline(t *testing.T) {
	t.Parallel()
	p := &readyTestProvider{status: controller.Ready()}
	in := readyTestInstance(func(in *v1alpha1.Instance) {
		in.Spec.UserSecretRef = &common.SecretRef{Name: "orders-users"}
	})
	st := newStatusTest(t, p, readyTestProviderObject(), in)
	waiting := wantStatus{
		phase: testPending, available: metav1.ConditionFalse,
		ready: metav1.ConditionFalse, reason: v1alpha1.ReasonUserSecretNotFound,
		message:     "secret orders-users named by spec.userSecretRef not found in team-a",
		reconciling: true, attention: true,
	}

	got, res := st.pass()
	requireStatus(t, got, waiting)
	requireCondition(t, got, v1alpha1.ConditionReferencesResolved, metav1.ConditionFalse, v1alpha1.ReasonUserSecretNotFound)
	assert.Equal(t, summary.DefaultProblemDeadline, res.RequeueAfter, "no event comes when the deadline passes")
	assert.Zero(t, p.syncs, "the provider is not called")

	st.advance(summary.DefaultProblemDeadline - time.Second)
	got, res = st.pass()
	requireStatus(t, got, waiting)
	assert.Equal(t, time.Second, res.RequeueAfter)
	assert.Equal(t, 1, st.writes, "nothing changed in between")

	st.advance(time.Second)
	got, res = st.pass()
	stalled := waiting
	stalled.reconciling, stalled.stalled = false, true
	requireStatus(t, got, stalled)
	assert.Zero(t, res.RequeueAfter)

	// A change to the spec reads as in progress even if it does not help:
	// the deadline starts again with the new generation.
	st.advance(time.Hour)
	st.editSpec()
	got, res = st.pass()
	requireStatus(t, got, waiting)
	assert.Equal(t, summary.DefaultProblemDeadline, res.RequeueAfter)
	st.advance(summary.DefaultProblemDeadline)
	got, _ = st.pass()
	requireStatus(t, got, stalled)

	// The Secret arrives late. The Instance is created from scratch, with
	// the whole progress deadline ahead of it.
	require.NoError(t, st.r.Create(t.Context(), &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "orders-users", Namespace: "team-a"}}))
	p.status = controller.Progressing("Cluster is being created")
	st.advance(time.Hour)
	got, res = st.pass()
	requireStatus(t, got, wantStatus{
		phase: testPending, available: metav1.ConditionFalse,
		ready: metav1.ConditionFalse, reason: v1alpha1.ReasonCreating, reconciling: true,
	})
	requireCondition(t, got, v1alpha1.ConditionReferencesResolved, metav1.ConditionTrue, v1alpha1.ReasonResolved)
	assert.Equal(t, summary.DefaultProgressDeadline, res.RequeueAfter)
}

func TestReconcile_InvalidReference(t *testing.T) {
	t.Parallel()
	p := &referenceTestProvider{referenceErrs: field.ErrorList{field.Invalid(
		field.NewPath("spec", "userSecretRef", "name"), "orders-users", `missing "MONGODB_BACKUP_USER"`,
	)}}
	st := newStatusTest(t, p, readyTestProviderObject(), readyTestInstance(nil))

	got, res := st.pass()

	requireStatus(t, got, wantStatus{
		phase: testPending, available: metav1.ConditionFalse,
		ready: metav1.ConditionFalse, reason: v1alpha1.ReasonInvalidReference, reconciling: true, attention: true,
	})
	requireCondition(t, got, v1alpha1.ConditionReferencesResolved, metav1.ConditionFalse, v1alpha1.ReasonInvalidReference)
	assert.Equal(t, referenceRecheck, res.RequeueAfter)
	assert.Zero(t, p.syncCalls)
}

//nolint:tparallel // the subtests continue one story, in order
func TestReconcile_RejectedSpecStallsAtOnce(t *testing.T) {
	t.Parallel()
	replicas := field.NewPath("spec", "components").Key("engine").Child("replicas")
	p := &readyTestProvider{
		status:       controller.Ready(),
		validateErrs: field.ErrorList{field.Invalid(replicas, 2, "must be odd")},
	}
	st := newStatusTest(t, p, readyTestProviderObject(), readyTestInstance(nil))
	rejected := wantStatus{
		phase: testPending, available: metav1.ConditionFalse,
		ready: metav1.ConditionFalse, reason: v1alpha1.ReasonValidationFailed,
		message: "spec.components[engine].replicas: Invalid value: 2: must be odd",
		stalled: true, attention: true,
	}

	got, res, err := st.reconcile()
	require.NoError(t, err, "a rejected spec is not retried with backoff")
	requireStatus(t, got, rejected)
	assert.Zero(t, res.RequeueAfter, "only a change to the Instance or the Provider helps, and both enqueue it")
	assert.Zero(t, p.syncs)

	t.Run("a fix reads as in progress", func(t *testing.T) {
		st.advance(time.Hour)
		p.validateErrs = nil
		p.status = controller.Progressing("Cluster is being created")
		st.editSpec()

		got, _ := st.pass()
		requireStatus(t, got, wantStatus{
			phase: testPending, available: metav1.ConditionFalse,
			ready: metav1.ConditionFalse, reason: v1alpha1.ReasonCreating, reconciling: true,
		})
	})
}

func TestReconcile_RejectedSpecKeepsThePhase(t *testing.T) {
	t.Parallel()
	replicas := field.NewPath("spec", "components").Key("engine").Child("replicas")
	p := &readyTestProvider{status: controller.Ready()}
	st := newStatusTest(t, p, readyTestProviderObject(), readyTestInstance(nil))
	st.pass()

	p.validateErrs = field.ErrorList{field.Invalid(replicas, 2, "must be odd")}
	got, _ := st.pass()

	requireStatus(t, got, wantStatus{
		phase: testProvisioned, available: metav1.ConditionTrue,
		ready: metav1.ConditionFalse, reason: v1alpha1.ReasonValidationFailed, stalled: true, attention: true,
	})
}

func TestReconcile_BackupConfiguration(t *testing.T) {
	t.Parallel()

	class := func(maxStorages int32) *backupv1alpha1.BackupClass {
		bc := testBackupClass(backupv1alpha1.BackupExecutionModeProviderManaged, "test-provider")
		bc.Spec.ProviderManaged = &backupv1alpha1.ProviderManagedSpec{
			Limits: &backupv1alpha1.BackupClassLimits{MaxStorages: &maxStorages},
		}

		return bc
	}

	t.Run("a configuration the class does not allow stalls at once", func(t *testing.T) {
		t.Parallel()
		p := &backupTestProvider{readyTestProvider{status: controller.Ready()}}
		st := newStatusTest(t, p,
			readyTestProviderObject(), readyTestInstance(withBackup(true, "s3-prod", "s3-dr")), class(1),
			testBackupStorage("s3-prod", "keys"), testBackupStorage("s3-dr", "keys"), testSecret("keys"),
		)

		got, res := st.pass()

		requireStatus(t, got, wantStatus{
			phase: testPending, available: metav1.ConditionFalse,
			ready: metav1.ConditionFalse, reason: controller.LimitsExceededReason, stalled: true, attention: true,
		})
		requireCondition(t, got, v1alpha1.ConditionBackupConfigured, metav1.ConditionFalse, controller.LimitsExceededReason)
		assert.Zero(t, res.RequeueAfter)
		assert.Zero(t, p.syncs)
	})

	t.Run("a storage that is missing may still arrive", func(t *testing.T) {
		t.Parallel()
		p := &backupTestProvider{readyTestProvider{status: controller.Ready()}}
		st := newStatusTest(t, p,
			readyTestProviderObject(), readyTestInstance(withBackup(true, "s3-prod")), class(1), testSecret("keys"),
		)
		missing := wantStatus{
			phase: testPending, available: metav1.ConditionFalse,
			ready: metav1.ConditionFalse, reason: v1alpha1.ReasonBackupStorageNotFound, reconciling: true, attention: true,
		}

		got, res := st.pass()
		requireStatus(t, got, missing)
		assert.Equal(t, summary.DefaultProblemDeadline, res.RequeueAfter)

		st.advance(summary.DefaultProblemDeadline)
		got, _ = st.pass()
		missing.reconciling, missing.stalled = false, true
		requireStatus(t, got, missing)

		require.NoError(t, st.r.Create(t.Context(), testBackupStorage("s3-prod", "keys")))
		got, _ = st.pass()
		requireStatus(t, got, wantStatus{
			phase: testProvisioned, available: metav1.ConditionTrue, ready: metav1.ConditionTrue, reason: v1alpha1.ReasonReady,
		})
		requireCondition(t, got, v1alpha1.ConditionBackupConfigured, metav1.ConditionTrue, "Configured")
	})

	t.Run("a verdict of the runtime's own checks is withdrawn when they pass and Sync does not get to apply the configuration", func(t *testing.T) {
		t.Parallel()
		p := &backupTestProvider{readyTestProvider{status: controller.Ready()}}
		st := newStatusTest(t, p,
			readyTestProviderObject(), readyTestInstance(withBackup(true, "s3-prod")), class(1), testSecret("keys"),
		)
		got, _ := st.pass()
		requireCondition(t, got, v1alpha1.ConditionBackupConfigured, metav1.ConditionFalse, v1alpha1.ReasonBackupStorageNotFound)

		require.NoError(t, st.r.Create(t.Context(), testBackupStorage("s3-prod", "keys")))
		p.syncErr = controller.WaitFor("waiting for the operator")
		got, _ = st.pass()

		assert.Nil(t, meta.FindStatusCondition(got.Status.Conditions, v1alpha1.ConditionBackupConfigured), "the storage is there, and nothing else is known yet")
		requireStatus(t, got, wantStatus{
			phase: testPending, available: metav1.ConditionFalse,
			ready: metav1.ConditionFalse, reason: v1alpha1.ReasonCreating, message: "waiting for the operator", reconciling: true,
		})
	})

	t.Run("a configuration the provider cannot apply is a problem that may clear", func(t *testing.T) {
		t.Parallel()
		p := &backupTestProvider{readyTestProvider{
			status:  controller.Ready(),
			syncErr: &controller.BackupConfigError{Reason: "BucketUnreachable", Message: "bucket orders-backups cannot be reached"},
		}}
		st := newStatusTest(t, p,
			readyTestProviderObject(), readyTestInstance(withBackup(true, "s3-prod")), class(1),
			testBackupStorage("s3-prod", "keys"), testSecret("keys"),
		)

		got, res, err := st.reconcile()

		require.NoError(t, err)
		requireStatus(t, got, wantStatus{
			phase: testPending, available: metav1.ConditionFalse,
			ready: metav1.ConditionFalse, reason: "BucketUnreachable", message: "bucket orders-backups cannot be reached",
			reconciling: true, attention: true,
		})
		requireCondition(t, got, v1alpha1.ConditionSynced, metav1.ConditionTrue, v1alpha1.ReasonSynced)
		assert.Equal(t, summary.DefaultProblemDeadline, res.RequeueAfter)

		st.advance(time.Minute)
		st.pass()
		assert.Equal(t, 1, st.writes, "the same verdict is not written twice")
	})
}

//nolint:tparallel // the subtests continue one story, in order
func TestReconcile_ProviderWaits(t *testing.T) {
	t.Parallel()
	p := &readyTestProvider{
		status:  controller.Ready(),
		syncErr: controller.WaitForDuration("hidden members start once the volumes of earlier ones are deleted", 7*time.Second),
	}
	st := newStatusTest(t, p, readyTestProviderObject(), readyTestInstance(nil))

	got, res, err := st.reconcile()

	require.NoError(t, err)
	requireStatus(t, got, wantStatus{
		phase: testPending, available: metav1.ConditionFalse,
		ready: metav1.ConditionFalse, reason: v1alpha1.ReasonCreating,
		message:     "hidden members start once the volumes of earlier ones are deleted",
		reconciling: true,
	})
	requireCondition(t, got, v1alpha1.ConditionSynced, metav1.ConditionTrue, v1alpha1.ReasonSynced)
	assert.Equal(t, 7*time.Second, res.RequeueAfter)

	t.Run("on an Instance that was Ready", func(t *testing.T) {
		p.syncErr = nil
		got, _ := st.pass()
		requireReady(t, got, metav1.ConditionTrue, v1alpha1.ReasonReady)

		p.syncErr = controller.WaitFor("rotating certificates")
		got, res := st.pass()
		requireStatus(t, got, wantStatus{
			phase: testProvisioned, available: metav1.ConditionTrue,
			ready: metav1.ConditionFalse, reason: v1alpha1.ReasonApplyingChanges, message: "rotating certificates", reconciling: true,
		})
		assert.Equal(t, 10*time.Second, res.RequeueAfter)
	})
}

func TestReconcile_ProviderFailsWhileConverging(t *testing.T) {
	t.Parallel()

	for _, call := range []string{"Sync", "Status"} {
		t.Run(call, func(t *testing.T) {
			t.Parallel()
			failure := errors.New("admission webhook unreachable")
			p := &readyTestProvider{status: controller.Ready()}
			fail := func(err error) {
				if call == "Sync" {
					p.syncErr = err
				} else {
					p.statusErr = err
				}
			}
			fail(failure)
			st := newStatusTest(t, p, readyTestProviderObject(), readyTestInstance(nil))

			// One failed call is not a problem yet. The status word stays
			// what it is, and the error is the sentence, since the provider
			// gave none.
			converging := wantStatus{
				phase: testPending, available: metav1.ConditionFalse,
				ready: metav1.ConditionFalse, reason: v1alpha1.ReasonCreating, message: "admission webhook unreachable",
				reconciling: true,
			}
			got, _, err := st.reconcile()
			require.ErrorIs(t, err, failure)
			requireStatus(t, got, converging)
			requireCondition(t, got, v1alpha1.ConditionSynced, metav1.ConditionUnknown, v1alpha1.ReasonReconcileError)
			assert.Equal(t, syncGrace, st.lastRequeue(), "the retry backoff may be longer than the time left")

			st.advance(syncGrace - time.Second)
			got, _, err = st.reconcile()
			require.ErrorIs(t, err, failure)
			requireStatus(t, got, converging)
			assert.Equal(t, 1, st.writes, "nothing changed in between")
			assert.Equal(t, time.Second, st.lastRequeue())

			// A minute of failures is a problem.
			st.advance(time.Second)
			got, _, err = st.reconcile()
			require.ErrorIs(t, err, failure)
			failing := wantStatus{
				phase: testPending, available: metav1.ConditionFalse,
				ready: metav1.ConditionFalse, reason: v1alpha1.ReasonReconcileError, reconciling: true, attention: true,
			}
			requireStatus(t, got, failing)
			requireCondition(t, got, v1alpha1.ConditionSynced, metav1.ConditionFalse, v1alpha1.ReasonReconcileError)
			assert.Equal(t, summary.DefaultProblemDeadline, st.lastRequeue())

			// And stalls like any other.
			st.advance(summary.DefaultProblemDeadline)
			got, _, err = st.reconcile()
			require.ErrorIs(t, err, failure)
			failing.reconciling, failing.stalled = false, true
			requireStatus(t, got, failing)
			assert.Empty(t, st.requeued, "nothing is left to wait for")

			// The first success clears it.
			fail(nil)
			got, _ = st.pass()
			requireStatus(t, got, wantStatus{
				phase: testProvisioned, available: metav1.ConditionTrue, ready: metav1.ConditionTrue, reason: v1alpha1.ReasonReady,
			})
			requireCondition(t, got, v1alpha1.ConditionSynced, metav1.ConditionTrue, v1alpha1.ReasonSynced)
		})
	}
}

// One failed call must not make a healthy database read as not ready.
//
//nolint:tparallel // the subtests continue one story, in order
func TestReconcile_ProviderFailsAfterConverging(t *testing.T) {
	t.Parallel()
	failure := errors.New("the object has been modified")
	p := &readyTestProvider{status: controller.Ready().WithAvailable(controller.Available)}
	st := newStatusTest(t, p, readyTestProviderObject(), readyTestInstance(nil))
	ready := wantStatus{phase: testProvisioned, available: metav1.ConditionTrue, ready: metav1.ConditionTrue, reason: v1alpha1.ReasonReady}
	got, _ := st.pass()
	requireStatus(t, got, ready)
	require.Equal(t, 1, st.writes)

	p.syncErr = failure
	got, _, err := st.reconcile()
	require.ErrorIs(t, err, failure)
	requireStatus(t, got, ready)
	assert.Equal(t, 1, st.writes, "nothing changes")
	assert.Equal(t, syncGrace, st.lastRequeue(), "but the Instance is looked at again when the failures become a problem")

	st.advance(syncGrace - time.Second)
	got, _, err = st.reconcile()
	require.ErrorIs(t, err, failure)
	requireStatus(t, got, ready)
	assert.Equal(t, 1, st.writes)
	assert.Equal(t, time.Second, st.lastRequeue())

	st.advance(time.Second)
	got, _, err = st.reconcile()
	require.ErrorIs(t, err, failure)
	requireStatus(t, got, wantStatus{
		phase: testProvisioned, available: metav1.ConditionTrue,
		ready: metav1.ConditionFalse, reason: v1alpha1.ReasonReconcileError, message: "the object has been modified",
		reconciling: true, attention: true,
	})
	requireCondition(t, got, v1alpha1.ConditionAvailable, metav1.ConditionTrue, v1alpha1.ReasonAvailable)

	t.Run("a single success in between starts the minute again", func(t *testing.T) {
		p.syncErr = nil
		got, _ := st.pass()
		requireStatus(t, got, ready)
		writes := st.writes

		p.syncErr = failure
		st.advance(time.Hour)
		got, _, err := st.reconcile()
		require.ErrorIs(t, err, failure)
		requireStatus(t, got, ready)
		assert.Equal(t, writes, st.writes)
	})
}

func TestReconcile_Problems(t *testing.T) {
	t.Parallel()

	t.Run("a problem the provider names warns first and stalls at the deadline", func(t *testing.T) {
		t.Parallel()
		p := &readyTestProvider{status: controller.Progressing("electing").
			WithUnavailable("NoPrimary", "rs0 has no primary").
			WithProblem("ReplicaSetNoPrimary", "rs0 has no primary").
			WithProblem("ReplicaLagging", "rs0-2 is 40s behind")}
		st := newStatusTest(t, p, readyTestProviderObject(), readyTestInstance(nil))

		got, res := st.pass()
		requireStatus(t, got, wantStatus{
			phase: testPending, available: metav1.ConditionFalse,
			ready: metav1.ConditionFalse, reason: "ReplicaSetNoPrimary", message: "rs0 has no primary\nrs0-2 is 40s behind",
			reconciling: true, attention: true,
		})
		requireCondition(t, got, v1alpha1.ConditionAvailable, metav1.ConditionFalse, "NoPrimary")
		requireCondition(t, got, v1alpha1.ConditionEngineHealthy, metav1.ConditionFalse, "ReplicaSetNoPrimary")
		assert.Equal(t, summary.DefaultProblemDeadline, res.RequeueAfter)

		p.status = controller.Ready()
		got, _ = st.pass()
		requireCondition(t, got, v1alpha1.ConditionEngineHealthy, metav1.ConditionTrue, v1alpha1.ReasonHealthy)
		requireReady(t, got, metav1.ConditionTrue, v1alpha1.ReasonReady)
	})

	t.Run("a problem the provider calls terminal stalls at once, and holds while the provider cannot be asked", func(t *testing.T) {
		t.Parallel()
		p := &readyTestProvider{status: controller.Progressing("not applied").
			WithProblem("ReplicaLagging", "rs0-2 is 40s behind").
			WithTerminalProblem("TopologyChangeUnsupported", "a replica set cannot become sharded")}
		st := newStatusTest(t, p, readyTestProviderObject(), readyTestInstance(nil))
		blocked := wantStatus{
			phase: testPending, available: metav1.ConditionFalse,
			ready: metav1.ConditionFalse, reason: "TopologyChangeUnsupported", stalled: true, attention: true,
		}

		got, res := st.pass()
		requireStatus(t, got, blocked)
		requireCondition(t, got, v1alpha1.ConditionEngineHealthy, metav1.ConditionFalse, "TopologyChangeUnsupported")
		assert.Zero(t, res.RequeueAfter)

		p.syncErr = errors.New("conflict")
		got, _, err := st.reconcile()
		require.Error(t, err)
		requireStatus(t, got, blocked)
	})

	t.Run("a reason a condition cannot carry does not fail the status write", func(t *testing.T) {
		t.Parallel()
		p := &readyTestProvider{status: controller.Progressing("electing").WithProblem("no primary!", "rs0 has no primary")}
		st := newStatusTest(t, p, readyTestProviderObject(), readyTestInstance(nil))

		got, _ := st.pass()

		requireCondition(t, got, v1alpha1.ConditionEngineHealthy, metav1.ConditionFalse, v1alpha1.ReasonEngineError)
	})

	t.Run("exhausted maintenance retries stall at once", func(t *testing.T) {
		t.Parallel()
		p := &readyTestProvider{status: controller.Ready()}
		in := readyTestInstance(func(in *v1alpha1.Instance) {
			in.Status.Conditions = []metav1.Condition{{
				Type: v1alpha1.ConditionMaintenancePending, Status: metav1.ConditionTrue,
				Reason: v1alpha1.ReasonRetriesExhausted, Message: "1 action(s) held", LastTransitionTime: metav1.Now(),
			}}
		})
		// The provider keeps requesting the action, and the breaker keeps holding it.
		p.sync = func(c *controller.Context) {
			c.BlockMaintenance("upgrade-to-0.3")
			c.RequestMaintenance("upgrade-to-0.3", "rolling restart", controller.MaintenanceNonDisruptive)
		}
		st := newStatusTest(t, p, readyTestProviderObject(), in)

		got, _ := st.pass()

		requireStatus(t, got, wantStatus{
			phase: testProvisioned, available: metav1.ConditionTrue,
			ready: metav1.ConditionFalse, reason: v1alpha1.ReasonRetriesExhausted, stalled: true, attention: true,
		})
	})
}

func TestReconcile_Operations(t *testing.T) {
	t.Parallel()

	t.Run("an operation is True while reported and keeps when it finished", func(t *testing.T) {
		t.Parallel()
		p := &readyTestProvider{status: controller.Ready().WithAvailable(controller.Available)}
		st := newStatusTest(t, p, readyTestProviderObject(), readyTestInstance(nil))
		got, _ := st.pass()
		assert.Nil(t, meta.FindStatusCondition(got.Status.Conditions, v1alpha1.ConditionUpgrading), "an operation that never ran is absent")

		// A change to the spec that the provider carries out as an upgrade.
		st.editSpec()
		started := st.now.Add(time.Minute)
		st.advance(time.Minute)
		p.status = controller.Progressing("Rolling out 8.0.12").
			WithAvailable(controller.Available).
			WithOperation(v1alpha1.ConditionUpgrading, "MinorVersionRolling", "1 of 3 pods updated").
			WithOperation("Defragmenting", "Compacting", "an operation the runtime does not know")
		got, res := st.pass()
		requireStatus(t, got, wantStatus{
			phase: testProvisioned, available: metav1.ConditionTrue,
			ready: metav1.ConditionFalse, reason: v1alpha1.ConditionUpgrading, message: "1 of 3 pods updated", reconciling: true,
		})
		upgrading := requireCondition(t, got, v1alpha1.ConditionUpgrading, metav1.ConditionTrue, "MinorVersionRolling")
		assert.True(t, started.Equal(upgrading.LastTransitionTime.Time), "it started at %s, not at %s", started, upgrading.LastTransitionTime)
		assert.Nil(t, meta.FindStatusCondition(got.Status.Conditions, "Defragmenting"))
		assert.Zero(t, res.RequeueAfter, "a named operation has no deadline")

		st.advance(2 * time.Hour)
		got, _ = st.pass()
		requireReady(t, got, metav1.ConditionFalse, v1alpha1.ConditionUpgrading)
		assert.False(t, meta.IsStatusConditionTrue(got.Status.Conditions, v1alpha1.ConditionStalled), "however long it takes")

		p.status = controller.Ready().WithAvailable(controller.Available)
		got, _ = st.pass()
		requireStatus(t, got, wantStatus{
			phase: testProvisioned, available: metav1.ConditionTrue, ready: metav1.ConditionTrue, reason: v1alpha1.ReasonReady,
		})
		finished := requireCondition(t, got, v1alpha1.ConditionUpgrading, metav1.ConditionFalse, v1alpha1.ReasonSucceeded)
		assert.True(t, st.now.Equal(finished.LastTransitionTime.Time), "it finished at %s, not at %s", st.now, finished.LastTransitionTime)
	})

	t.Run("an operation the engine starts on an unchanged spec keeps the Instance Ready", func(t *testing.T) {
		t.Parallel()
		p := &readyTestProvider{status: controller.Ready()}
		st := newStatusTest(t, p, readyTestProviderObject(), readyTestInstance(nil))
		st.pass()

		p.status = controller.Ready().WithOperation(v1alpha1.ConditionStorageResizing, "ExpansionTriggered", "Expanding volumes to 50Gi")
		got, _ := st.pass()

		requireStatus(t, got, wantStatus{
			phase: testProvisioned, available: metav1.ConditionTrue,
			ready: metav1.ConditionTrue, reason: v1alpha1.ConditionStorageResizing, message: "Expanding volumes to 50Gi",
		})
	})

	t.Run("a Restore keeps the Instance from being Ready though its spec did not change", func(t *testing.T) {
		t.Parallel()
		p := &backupTestProvider{readyTestProvider{status: controller.Ready()}}
		restore := &backupv1alpha1.Restore{
			ObjectMeta: metav1.ObjectMeta{Name: "orders-0412", Namespace: "team-a"},
			Spec:       backupv1alpha1.RestoreSpec{InstanceRef: common.ObjectRef{Name: "orders"}},
			Status:     backupv1alpha1.RestoreStatus{State: backupv1alpha1.RestoreStateRunning},
		}
		st := newStatusTest(t, p, readyTestProviderObject(), readyTestInstance(nil), restore)
		st.pass()

		// The engine reports itself converged for a moment while it restores.
		p.status = controller.Ready().WithOperation(v1alpha1.ConditionRestoring, "Restoring", "Restore orders-0412 is running")
		got, _ := st.pass()
		requireStatus(t, got, wantStatus{
			phase: testProvisioned, available: metav1.ConditionTrue,
			ready: metav1.ConditionFalse, reason: v1alpha1.ConditionRestoring, reconciling: true,
		})

		restore.Status.State = backupv1alpha1.RestoreStateSucceeded
		require.NoError(t, st.r.Update(t.Context(), restore))
		p.status = controller.Ready()
		got, _ = st.pass()
		requireReady(t, got, metav1.ConditionTrue, v1alpha1.ReasonReady)
		requireCondition(t, got, v1alpha1.ConditionRestoring, metav1.ConditionFalse, v1alpha1.ReasonSucceeded)
	})
}

//nolint:tparallel,paralleltest // the subtests continue one story, in order
func TestReconcile_Seeding(t *testing.T) {
	t.Parallel()

	staged := controller.DataSourceStatus{}
	p := &readyTestProvider{sync: func(c *controller.Context) { c.SetDataSourceStatus(staged) }}
	st := newStatusTest(t, p, readyTestProviderObject(), readyTestInstance(nil))

	// The source cannot be read: a problem, not progress.
	staged = controller.DataSourceStatus{
		State: controller.DataSourceStateWaiting, Blocked: true,
		Reason: v1alpha1.ReasonDataSourceSourceBackupNotFound, Message: `source Backup "nightly" not found`,
	}
	p.status = controller.Progressing("Cluster is being created")
	got, _ := st.pass()
	requireStatus(t, got, wantStatus{
		phase: testPending, available: metav1.ConditionFalse,
		ready: metav1.ConditionFalse, reason: v1alpha1.ReasonDataSourceSourceBackupNotFound, reconciling: true, attention: true,
	})
	requireCondition(t, got, v1alpha1.ConditionDataSourceReady, metav1.ConditionFalse, v1alpha1.ReasonDataSourceSourceBackupNotFound)
	assert.Nil(t, meta.FindStatusCondition(got.Status.Conditions, v1alpha1.ConditionRestoring), "nothing is being restored")

	// The engine is not up yet, then the data is loaded. Both are seeding.
	// The provider reports nothing but that it has not converged.
	for _, step := range []controller.DataSourceStatus{
		{State: controller.DataSourceStateWaiting, Reason: v1alpha1.ReasonDataSourceWaitingForCluster, Message: "Waiting for the cluster"},
		{State: controller.DataSourceStateRestoring, Reason: v1alpha1.ReasonDataSourceRestoring, Message: "Restore orders-datasource is running"},
	} {
		staged = step
		st.advance(time.Minute)
		got, res := st.pass()
		requireStatus(t, got, wantStatus{
			phase: testPending, available: metav1.ConditionFalse,
			ready: metav1.ConditionFalse, reason: v1alpha1.ConditionRestoring, message: step.Message, reconciling: true,
		})
		requireCondition(t, got, v1alpha1.ConditionDataSourceReady, metav1.ConditionUnknown, step.Reason)
		requireCondition(t, got, v1alpha1.ConditionRestoring, metav1.ConditionTrue, v1alpha1.ReasonSeeding)
		requireCondition(t, got, v1alpha1.ConditionAvailable, metav1.ConditionFalse, v1alpha1.ReasonLoadingData)
		assert.Zero(t, res.RequeueAfter, "loading data has no deadline")
	}

	t.Run("it is still a restore on a pass that cannot ask the provider", func(t *testing.T) {
		p.syncErr = errors.New("conflict")
		got, _, err := st.reconcile()
		require.Error(t, err)
		requireCondition(t, got, v1alpha1.ConditionRestoring, metav1.ConditionTrue, v1alpha1.ReasonSeeding)
		requireReady(t, got, metav1.ConditionFalse, v1alpha1.ConditionRestoring)
		p.syncErr = nil
	})

	t.Run("the engine being reachable does not make it available while data loads", func(t *testing.T) {
		p.status = controller.Progressing("Restoring").WithAvailable(controller.Available)
		got, _ := st.pass()
		requireCondition(t, got, v1alpha1.ConditionAvailable, metav1.ConditionFalse, v1alpha1.ReasonLoadingData)
		assert.Equal(t, testPending, got.Status.Phase)
	})

	t.Run("seeded", func(t *testing.T) {
		staged = controller.DataSourceStatus{
			Done: true, State: controller.DataSourceStateSucceeded,
			Reason: v1alpha1.ReasonDataSourceSucceeded, Message: "Instance seeded",
		}
		p.status = controller.Ready()
		st.advance(time.Minute)
		got, _ := st.pass()
		requireStatus(t, got, wantStatus{
			phase: testProvisioned, available: metav1.ConditionTrue, ready: metav1.ConditionTrue, reason: v1alpha1.ReasonReady,
		})
		requireCondition(t, got, v1alpha1.ConditionDataSourceReady, metav1.ConditionTrue, v1alpha1.ReasonDataSourceSucceeded)
		finished := requireCondition(t, got, v1alpha1.ConditionRestoring, metav1.ConditionFalse, v1alpha1.ReasonSucceeded)
		assert.True(t, st.now.Equal(finished.LastTransitionTime.Time), "it finished at %s, not at %s", st.now, finished.LastTransitionTime)
	})
}

func TestReconcile_SeedingFailed(t *testing.T) {
	t.Parallel()
	staged := controller.DataSourceStatus{
		State: controller.DataSourceStateRestoring, Reason: v1alpha1.ReasonDataSourceRestoring, Message: "Restore is running",
	}
	p := &readyTestProvider{
		status: controller.Progressing("Restore is running"),
		sync:   func(c *controller.Context) { c.SetDataSourceStatus(staged) },
	}
	st := newStatusTest(t, p, readyTestProviderObject(), readyTestInstance(nil))
	got, _ := st.pass()
	requireCondition(t, got, v1alpha1.ConditionRestoring, metav1.ConditionTrue, v1alpha1.ReasonSeeding)

	staged = controller.DataSourceStatus{
		Done: true, State: controller.DataSourceStateFailed,
		Reason: v1alpha1.ReasonDataSourceUserSecretMismatch, Message: "the users in the backup differ from spec.userSecretRef",
	}
	p.status = controller.Progressing("Seeding failed")
	got, _ = st.pass()

	requireStatus(t, got, wantStatus{
		phase: testPending, available: metav1.ConditionFalse,
		ready: metav1.ConditionFalse, reason: v1alpha1.ReasonDataSourceUserSecretMismatch, reconciling: true, attention: true,
	})
	requireCondition(t, got, v1alpha1.ConditionDataSourceReady, metav1.ConditionFalse, v1alpha1.ReasonDataSourceUserSecretMismatch)
	requireCondition(t, got, v1alpha1.ConditionRestoring, metav1.ConditionFalse, v1alpha1.ReasonDataSourceFailed)
}

// A pod that keeps crashing is seen whether or not the provider can be
// called, and the Instance is looked at again when the verdict is due to
// change with time alone.
func TestReconcile_CrashLoop(t *testing.T) {
	t.Parallel()

	p := &readyTestProvider{status: controller.Ready().WithAvailable(controller.Available)}
	p.sync = func(c *controller.Context) { c.PodLabels("engine") }
	st := newStatusTest(t, p, readyTestProviderObject(), readyTestInstance(nil))
	started := st.now
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "orders-rs0-1", Namespace: "team-a",
			Labels: map[string]string{controller.InstanceLabel: "orders", controller.ComponentLabel: "engine"},
		},
		Status: corev1.PodStatus{
			Phase:      corev1.PodRunning,
			Conditions: []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionTrue}},
			ContainerStatuses: []corev1.ContainerStatus{{
				Name: "mongod", RestartCount: 5,
				State:                corev1.ContainerState{Running: &corev1.ContainerStateRunning{StartedAt: metav1.NewTime(started)}},
				LastTerminationState: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Reason: "Error", ExitCode: 14}},
			}},
		},
	}
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	st.r.pods.Store(client.Reader(fake.NewClientBuilder().WithScheme(scheme).WithObjects(pod).Build()))
	crashing := wantStatus{
		phase: testProvisioned, available: metav1.ConditionTrue,
		ready: metav1.ConditionFalse, reason: v1alpha1.ReasonCrashLoopBackOff,
		message:     "engine: 1 of 1 pods keep crashing: container mongod last exited with Error (exit code 14)",
		reconciling: true, attention: true,
	}

	// The container is up and its pod Ready, between two crashes.
	st.advance(30 * time.Second)
	got, res := st.pass()
	requireStatus(t, got, crashing)
	requireCondition(t, got, v1alpha1.ConditionPodsReady, metav1.ConditionFalse, v1alpha1.ReasonCrashLoopBackOff)
	assert.Equal(t, crashLoopQuiet-30*time.Second, res.RequeueAfter, "when the container has stayed up for long enough")

	// The provider cannot be called. The pods are still looked at.
	p.syncErr = errors.New("conflict")
	st.advance(time.Minute)
	got, _, err := st.reconcile()
	require.Error(t, err)
	requireStatus(t, got, crashing)
	assert.Equal(t, syncGrace, st.lastRequeue(), "the soonest of what is due: here the failing call, before the crash loop may clear")
	p.syncErr = nil

	// It stayed up: the crash loop is over, with no event from the pod.
	st.advance(crashLoopQuiet)
	got, res = st.pass()
	requireStatus(t, got, wantStatus{
		phase: testProvisioned, available: metav1.ConditionTrue, ready: metav1.ConditionTrue, reason: v1alpha1.ReasonReady,
	})
	requireCondition(t, got, v1alpha1.ConditionPodsReady, metav1.ConditionTrue, v1alpha1.ReasonReady)
	assert.Zero(t, res.RequeueAfter)
}

// Pods go not ready and come back while a change rolls. The status word must
// not follow them.
func TestReconcile_PodsStartingLeaveTheStatusWordAlone(t *testing.T) {
	t.Parallel()

	p := &readyTestProvider{status: controller.Progressing("Waiting for pods to roll").WithAvailable(controller.Available)}
	p.sync = func(c *controller.Context) { c.PodLabels("engine") }
	st := newStatusTest(t, p, readyTestProviderObject(), readyTestInstance(nil))
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "orders-rs0-1", Namespace: "team-a",
			Labels: map[string]string{controller.InstanceLabel: "orders", controller.ComponentLabel: "engine"},
		},
		Status: corev1.PodStatus{Phase: corev1.PodRunning},
	}
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	pods := fake.NewClientBuilder().WithScheme(scheme).WithObjects(pod).Build()
	st.r.pods.Store(client.Reader(pods))

	for pass := range 6 {
		podReady, wantPods, wantReason := corev1.ConditionFalse, metav1.ConditionUnknown, v1alpha1.ReasonNotReady
		if pass%2 == 1 {
			podReady, wantPods, wantReason = corev1.ConditionTrue, metav1.ConditionTrue, v1alpha1.ReasonReady
		}
		pod.Status.Conditions = []corev1.PodCondition{{Type: corev1.PodReady, Status: podReady}}
		require.NoError(t, pods.Status().Update(t.Context(), pod))
		st.advance(5 * time.Second)

		got, _ := st.pass()

		requireCondition(t, got, v1alpha1.ConditionPodsReady, wantPods, wantReason)
		requireStatus(t, got, wantStatus{
			phase: testProvisioned, available: metav1.ConditionTrue,
			ready: metav1.ConditionFalse, reason: v1alpha1.ReasonApplyingChanges, message: "Waiting for pods to roll",
			reconciling: true,
		})
		requireCondition(t, got, v1alpha1.ConditionReconciling, metav1.ConditionTrue, v1alpha1.ReasonApplyingChanges)
	}
}

func TestReconcile_Deletion(t *testing.T) {
	t.Parallel()

	deleting := func(mutate func(*v1alpha1.Instance)) *v1alpha1.Instance {
		return readyTestInstance(func(in *v1alpha1.Instance) {
			now := metav1.Now()
			in.DeletionTimestamp = &now
			in.Status.Phase = v1alpha1.InstancePhaseProvisioned
			if mutate != nil {
				mutate(in)
			}
		})
	}

	t.Run("every pass says what the deletion waits for, and it can stall", func(t *testing.T) {
		t.Parallel()
		p := &readyTestProvider{cleanupErr: controller.WaitFor("volumes are being released")}
		st := newStatusTest(t, p, readyTestProviderObject(), deleting(nil))
		waiting := wantStatus{
			phase: v1alpha1.InstancePhaseTerminating, available: metav1.ConditionUnknown,
			ready: metav1.ConditionFalse, reason: v1alpha1.ReasonDeleting, message: "volumes are being released", reconciling: true,
		}

		got, res := st.pass()
		requireStatus(t, got, waiting)
		requireCondition(t, got, v1alpha1.ConditionAvailable, metav1.ConditionUnknown, v1alpha1.ReasonDeleting)
		assert.Equal(t, 10*time.Second, res.RequeueAfter)
		writes := st.writes

		st.advance(10 * time.Second)
		got, _ = st.pass()
		requireStatus(t, got, waiting)
		assert.Equal(t, writes, st.writes, "nothing changed")

		st.advance(summary.DefaultProgressDeadline)
		got, _ = st.pass()
		requireStatus(t, got, wantStatus{
			phase: v1alpha1.InstancePhaseTerminating, available: metav1.ConditionUnknown,
			ready: metav1.ConditionFalse, reason: v1alpha1.ReasonProgressDeadlineExceeded,
			message: "No progress for 30m0s: volumes are being released",
			stalled: true, attention: true,
		})

		p.cleanupErr = nil
		_, err := st.r.Reconcile(t.Context(), reconcile.Request{NamespacedName: st.key()})
		require.NoError(t, err)
		err = st.r.Get(t.Context(), st.key(), &v1alpha1.Instance{})
		assert.True(t, apierrors.IsNotFound(err), "the finalizer is removed once cleanup is done")
	})

	t.Run("it waits for the Instance's backups first", func(t *testing.T) {
		t.Parallel()
		p := &backupTestProvider{}
		backup := &backupv1alpha1.Backup{
			ObjectMeta: metav1.ObjectMeta{Name: "nightly", Namespace: "team-a", Finalizers: []string{"example.com/purge"}},
			Spec: backupv1alpha1.BackupSpec{Origin: backupv1alpha1.BackupOrigin{
				Type: backupv1alpha1.BackupOriginTypeInstance, InstanceRef: &common.ObjectRef{Name: "orders"},
			}},
		}
		in := deleting(func(in *v1alpha1.Instance) { in.Spec.DeletionPolicy = v1alpha1.InstanceDeletionPolicyCascade })
		st := newStatusTest(t, p, readyTestProviderObject(), in, backup)

		got, res := st.pass()

		requireStatus(t, got, wantStatus{
			phase: v1alpha1.InstancePhaseTerminating, available: metav1.ConditionUnknown,
			ready: metav1.ConditionFalse, reason: v1alpha1.ReasonWaitingForBackups, message: "Waiting for 1 backup to be deleted",
			reconciling: true,
		})
		assert.Equal(t, childrenRecheck, res.RequeueAfter)
	})

	t.Run("a cleanup that keeps failing is a problem", func(t *testing.T) {
		t.Parallel()
		p := &readyTestProvider{cleanupErr: errors.New("delete cluster: forbidden")}
		st := newStatusTest(t, p, readyTestProviderObject(), deleting(nil))

		got, _, err := st.reconcile()
		require.Error(t, err)
		requireStatus(t, got, wantStatus{
			phase: v1alpha1.InstancePhaseTerminating, available: metav1.ConditionUnknown,
			ready: metav1.ConditionFalse, reason: v1alpha1.ReasonDeleting, reconciling: true,
		})
		assert.Equal(t, syncGrace, st.lastRequeue())

		st.advance(syncGrace)
		got, _, err = st.reconcile()
		require.Error(t, err)
		requireStatus(t, got, wantStatus{
			phase: v1alpha1.InstancePhaseTerminating, available: metav1.ConditionUnknown,
			ready: metav1.ConditionFalse, reason: v1alpha1.ReasonReconcileError, message: "delete cluster: forbidden",
			reconciling: true, attention: true,
		})
	})
}

func TestWaitingForChildren(t *testing.T) {
	t.Parallel()

	tests := []struct {
		backups, restores int
		want              summary.Cause
	}{
		{1, 0, summary.Cause{Reason: v1alpha1.ReasonWaitingForBackups, Message: "Waiting for 1 backup to be deleted"}},
		{2, 0, summary.Cause{Reason: v1alpha1.ReasonWaitingForBackups, Message: "Waiting for 2 backups to be deleted"}},
		{0, 1, summary.Cause{Reason: v1alpha1.ReasonWaitingForRestores, Message: "Waiting for 1 restore to be deleted"}},
		{3, 2, summary.Cause{Reason: v1alpha1.ReasonWaitingForBackups, Message: "Waiting for 3 backups and 2 restores to be deleted"}},
	}
	for _, tt := range tests {
		t.Run(tt.want.Message, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, *waitingForChildren(tt.backups, tt.restores))
		})
	}
}

// What is remembered about an Instance is kept per generation, and rebuilt
// from the stored status by a process that has just started.
func TestTracker(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	in := readyTestInstance(func(in *v1alpha1.Instance) { in.UID = "a" })
	var tr tracker

	first := tr.track(in, now)
	first.failingSince = now
	assert.Equal(t, now, first.start)
	assert.False(t, first.converged)
	assert.Same(t, first, tr.track(in, now.Add(time.Minute)), "the same generation")

	in.Generation++
	second := tr.track(in, now.Add(time.Hour))
	assert.Equal(t, now.Add(time.Hour), second.start, "a new generation starts the clocks again")
	assert.True(t, second.failingSince.IsZero())

	in.UID = "b"
	assert.NotSame(t, second, tr.track(in, now.Add(2*time.Hour)), "an Instance created again under the same name")

	tr.forget(client.ObjectKeyFromObject(in))
	in.Status.Conditions = []metav1.Condition{{
		Type: v1alpha1.ConditionReady, Status: metav1.ConditionTrue, ObservedGeneration: in.Generation,
	}}
	restarted := tr.track(in, now.Add(3*time.Hour))
	assert.True(t, restarted.converged, "a generation stored as Ready had converged")
	assert.Equal(t, now.Add(3*time.Hour), restarted.start, "the clocks start again, which only delays a stall")

	tr.forget(client.ObjectKeyFromObject(in))
	in.Generation++
	assert.False(t, tr.track(in, now).converged, "Ready describes the generation before")
}

func TestSettleConditions(t *testing.T) {
	t.Parallel()
	t0 := metav1.NewTime(time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC))
	t1 := metav1.NewTime(t0.Add(time.Hour))
	stored := []metav1.Condition{
		{Type: v1alpha1.ConditionPodsReady, Status: metav1.ConditionTrue, LastTransitionTime: t0},
		{Type: v1alpha1.ConditionReady, Status: metav1.ConditionTrue, LastTransitionTime: t0},
	}
	conditions := []metav1.Condition{
		{Type: "SomethingElse", Status: metav1.ConditionTrue, LastTransitionTime: t1},
		{Type: v1alpha1.ConditionUpgrading, Status: metav1.ConditionTrue, LastTransitionTime: t1},
		// Set to False and back to True during the pass.
		{Type: v1alpha1.ConditionPodsReady, Status: metav1.ConditionTrue, LastTransitionTime: t1},
		{Type: v1alpha1.ConditionReady, Status: metav1.ConditionFalse, LastTransitionTime: t1},
		{Type: v1alpha1.ConditionAvailable, Status: metav1.ConditionTrue, LastTransitionTime: t1},
	}

	settleConditions(stored, conditions)

	types := make([]string, 0, len(conditions))
	for _, c := range conditions {
		types = append(types, c.Type)
	}
	want := []string{
		v1alpha1.ConditionAvailable, v1alpha1.ConditionReady, v1alpha1.ConditionPodsReady,
		v1alpha1.ConditionUpgrading, "SomethingElse",
	}
	assert.Equal(t, want, types, "the summary first, then checks, operations and the rest")
	assert.Equal(t, t0, conditions[2].LastTransitionTime, "PodsReady is what it was")
	assert.Equal(t, t1, conditions[1].LastTransitionTime, "Ready changed")
}
