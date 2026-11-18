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

package lifecycleevents

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	toolscache "k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	commonv1alpha1 "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/pkg/common"
	"github.com/openeverest/openeverest/v2/pkg/events"
)

type recordingEmitter struct {
	mu      sync.Mutex
	records []events.Record
}

func (e *recordingEmitter) Emit(_ context.Context, r events.Record) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.records = append(e.records, r)
	return nil
}

func (e *recordingEmitter) reasons() []events.Reason {
	e.mu.Lock()
	defer e.mu.Unlock()
	reasons := make([]events.Reason, 0, len(e.records))
	for _, r := range e.records {
		reasons = append(reasons, r.Reason)
	}
	return reasons
}

//nolint:gochecknoglobals // fixed reference time
var now = time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)

func newTestRecorder(t *testing.T) (*Recorder, *recordingEmitter) {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, clientgoscheme.AddToScheme(scheme))
	require.NoError(t, corev1alpha1.AddToScheme(scheme))
	require.NoError(t, backupv1alpha1.AddToScheme(scheme))
	emitter := &recordingEmitter{}
	r := NewRecorder(nil, scheme, emitter, DefaultKinds(), logr.Discard())
	r.now = func() time.Time { return now }
	return r, emitter
}

// instance returns an Instance whose status word, the reason of its Ready
// condition, is status. An empty status means no provider reported on it.
func instance(generation int64, status string, createdAgo time.Duration) *corev1alpha1.Instance {
	in := &corev1alpha1.Instance{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "team-a", Name: "db1", UID: "uid-1", Generation: generation,
			ResourceVersion:   "100",
			CreationTimestamp: metav1.NewTime(now.Add(-createdAgo)),
			Annotations:       map[string]string{events.AnnotationActorType: "user", events.AnnotationActorID: "alice"},
		},
	}
	if status != "" {
		ready := metav1.ConditionFalse
		if status == corev1alpha1.ReasonReady {
			ready = metav1.ConditionTrue
		}
		meta.SetStatusCondition(&in.Status.Conditions, metav1.Condition{Type: corev1alpha1.ConditionReady, Status: ready, Reason: status})
	}
	return in
}

// needingAttention marks in as needing attention.
func needingAttention(in *corev1alpha1.Instance) *corev1alpha1.Instance {
	meta.SetStatusCondition(&in.Status.Conditions, metav1.Condition{
		Type: corev1alpha1.ConditionNeedsAttention, Status: metav1.ConditionTrue, Reason: "CrashLoopBackOff",
	})
	return in
}

//nolint:ireturn // informers take client-go event handlers
func instanceHandler(t *testing.T, r *Recorder) toolscache.ResourceEventHandler {
	t.Helper()
	return r.handlerFor(context.Background(), DefaultKinds()[0])
}

func TestRecorder_Add(t *testing.T) {
	t.Parallel()

	t.Run("a new object is recorded as created, with its creator", func(t *testing.T) {
		t.Parallel()
		r, emitter := newTestRecorder(t)
		obj := instance(1, "", 0)
		obj.ManagedFields = []metav1.ManagedFieldsEntry{{
			Manager:    "everest-api",
			Operation:  metav1.ManagedFieldsOperationUpdate,
			FieldsType: "FieldsV1",
			FieldsV1:   metav1.NewFieldsV1(`{"f:metadata":{"f:annotations":{"f:openeverest.io/last-actor-id":{}}}}`),
		}}
		instanceHandler(t, r).OnAdd(obj, false)

		require.Len(t, emitter.records, 1)
		rec := emitter.records[0]
		assert.Equal(t, events.ReasonCreated, rec.Reason)
		assert.Equal(t, events.Actor{Type: "user", ID: "alice"}, rec.Actor)
		wantRegarding := corev1.ObjectReference{
			APIVersion: "core.openeverest.io/v1alpha1", Kind: "Instance",
			Namespace: "team-a", Name: "db1", UID: "uid-1", ResourceVersion: "100",
		}
		assert.Equal(t, wantRegarding, rec.Regarding)
	})

	t.Run("the initial list only records objects created within the lookback", func(t *testing.T) {
		t.Parallel()
		r, emitter := newTestRecorder(t)
		handler := instanceHandler(t, r)
		handler.OnAdd(instance(1, "Ready", 2*DefaultLookback), true)
		assert.Empty(t, emitter.records, "objects older than the lookback were recorded already, or long ago")

		handler.OnAdd(instance(1, "Ready", 10*time.Minute), true)
		assert.Equal(t, []events.Reason{events.ReasonCreated}, emitter.reasons(),
			"an object created while no recorder ran is recorded",
		)
	})

	t.Run("the initial list records deletions requested within the lookback", func(t *testing.T) {
		t.Parallel()
		r, emitter := newTestRecorder(t)
		obj := instance(1, "Deleting", 2*DefaultLookback)
		deletion := metav1.NewTime(now.Add(-time.Minute))
		obj.DeletionTimestamp = &deletion
		instanceHandler(t, r).OnAdd(obj, true)
		assert.Equal(t, []events.Reason{events.ReasonDeletionRequested}, emitter.reasons())
	})
}

func TestRecorder_Update(t *testing.T) {
	t.Parallel()

	t.Run("a spec change is recorded as updated", func(t *testing.T) {
		t.Parallel()
		r, emitter := newTestRecorder(t)
		instanceHandler(t, r).OnUpdate(instance(1, "Ready", time.Hour), instance(2, "Ready", time.Hour))

		require.Len(t, emitter.records, 1)
		assert.Equal(t, events.ReasonUpdated, emitter.records[0].Reason)
		assert.Equal(t, int64(2), emitter.records[0].Generation)
		assert.Equal(t, "2", emitter.records[0].Key)
	})

	t.Run("a phase change is recorded with both phases", func(t *testing.T) {
		t.Parallel()
		r, emitter := newTestRecorder(t)
		newObj := instance(1, "Ready", time.Hour)
		newObj.ResourceVersion = "101"
		instanceHandler(t, r).OnUpdate(instance(1, "Creating", time.Hour), newObj)

		require.Len(t, emitter.records, 1)
		rec := emitter.records[0]
		assert.Equal(t, events.ReasonPhaseChanged, rec.Reason)
		assert.Equal(t, "Creating", rec.PhaseFrom)
		assert.Equal(t, "Ready", rec.PhaseTo)
		assert.Equal(t, "101", rec.Key)
		assert.Equal(t, events.ControllerActor, rec.Actor)
		assert.False(t, rec.Warning)
	})

	t.Run("a failure is recorded as a warning", func(t *testing.T) {
		t.Parallel()
		r, emitter := newTestRecorder(t)
		instanceHandler(t, r).OnUpdate(instance(1, "Ready", time.Hour), needingAttention(instance(1, "CrashLoopBackOff", time.Hour)))
		require.Len(t, emitter.records, 1)
		assert.True(t, emitter.records[0].Warning)
		assert.Equal(t, "CrashLoopBackOff", emitter.records[0].PhaseTo)
	})

	t.Run("a resync without changes records nothing", func(t *testing.T) {
		t.Parallel()
		r, emitter := newTestRecorder(t)
		instanceHandler(t, r).OnUpdate(instance(3, "Ready", time.Hour), instance(3, "Ready", time.Hour))
		assert.Empty(t, emitter.records)
	})

	t.Run("the generation bump of a deletion is not an update", func(t *testing.T) {
		t.Parallel()
		r, emitter := newTestRecorder(t)
		deleting := instance(3, "Ready", time.Hour)
		deletion := metav1.NewTime(now)
		deleting.DeletionTimestamp = &deletion
		instanceHandler(t, r).OnUpdate(instance(2, "Ready", time.Hour), deleting)
		assert.Equal(t, []events.Reason{events.ReasonDeletionRequested}, emitter.reasons())
	})

	t.Run("a deletion request is recorded once", func(t *testing.T) {
		t.Parallel()
		r, emitter := newTestRecorder(t)
		deleting := instance(1, "Ready", time.Hour)
		deletion := metav1.NewTime(now)
		deleting.DeletionTimestamp = &deletion
		handler := instanceHandler(t, r)
		handler.OnUpdate(instance(1, "Ready", time.Hour), deleting)
		handler.OnUpdate(deleting, deleting.DeepCopy())
		assert.Equal(t, []events.Reason{events.ReasonDeletionRequested}, emitter.reasons())
	})
}

func TestRecorder_Delete(t *testing.T) {
	t.Parallel()

	r, emitter := newTestRecorder(t)
	handler := instanceHandler(t, r)
	handler.OnDelete(instance(1, "Deleting", time.Hour))
	// A delete missed by the watch is delivered as a tombstone.
	handler.OnDelete(toolscache.DeletedFinalStateUnknown{Key: "team-a/db1", Obj: instance(1, "Deleting", time.Hour)})
	assert.Equal(t, []events.Reason{events.ReasonDeleted, events.ReasonDeleted}, emitter.reasons())
}

func TestRecorder_Include(t *testing.T) {
	t.Parallel()

	r, emitter := newTestRecorder(t)
	var namespaceKind Kind
	for _, k := range DefaultKinds() {
		if _, ok := k.New().(*corev1.Namespace); ok {
			namespaceKind = k
		}
	}
	handler := r.handlerFor(context.Background(), namespaceKind)
	namespace := func(labels map[string]string) client.Object {
		return &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "team-a", UID: "ns-uid", Labels: labels}}
	}

	handler.OnAdd(namespace(nil), false)
	assert.Empty(t, emitter.records, "namespaces not managed by OpenEverest are ignored")

	managed := namespace(map[string]string{common.KubernetesManagedByLabel: common.Everest})
	handler.OnAdd(managed, false)
	assert.Equal(t, []events.Reason{events.ReasonCreated}, emitter.reasons())

	// A namespace labeled after its creation starts being managed then, and stops being
	// managed when the label is removed.
	handler.OnUpdate(namespace(nil), managed)
	handler.OnUpdate(managed, namespace(nil))
	assert.Equal(t, []events.Reason{events.ReasonCreated, events.ReasonCreated, events.ReasonDeleted}, emitter.reasons())
}

func kindOf[T client.Object](t *testing.T) Kind {
	t.Helper()
	for _, k := range DefaultKinds() {
		if _, ok := k.New().(T); ok {
			return k
		}
	}
	var zero T
	t.Fatalf("no kind for %T", zero)
	return Kind{}
}

func TestRecorder_Owner(t *testing.T) {
	t.Parallel()

	backup := func(origin backupv1alpha1.BackupOrigin) *backupv1alpha1.Backup {
		return &backupv1alpha1.Backup{
			ObjectMeta: metav1.ObjectMeta{Namespace: "team-a", Name: "b1", UID: "uid-b1"},
			Spec:       backupv1alpha1.BackupSpec{Origin: origin},
		}
	}
	fromDB1 := backupv1alpha1.BackupOrigin{
		Type: backupv1alpha1.BackupOriginTypeInstance, InstanceRef: &commonv1alpha1.ObjectRef{Name: "db1"},
	}
	db1 := instance(1, "Ready", time.Hour)
	db1.Spec.ProviderRef = commonv1alpha1.ObjectRef{Name: "percona-server-mysql"}

	t.Run("an Instance carries its own name and provider, even once deleted", func(t *testing.T) {
		t.Parallel()
		r, emitter := newTestRecorder(t)
		instanceHandler(t, r).OnDelete(db1.DeepCopy())
		require.Len(t, emitter.records, 1)
		assert.Equal(t, "db1", emitter.records[0].Instance)
		assert.Equal(t, "percona-server-mysql", emitter.records[0].Provider)
	})

	t.Run("a Backup carries its Instance and the provider read from it", func(t *testing.T) {
		t.Parallel()
		r, emitter := newTestRecorder(t)
		r.reader = fake.NewClientBuilder().WithScheme(r.scheme).WithObjects(db1.DeepCopy()).Build()
		r.handlerFor(context.Background(), kindOf[*backupv1alpha1.Backup](t)).OnAdd(backup(fromDB1), false)
		require.Len(t, emitter.records, 1)
		assert.Equal(t, "db1", emitter.records[0].Instance)
		assert.Equal(t, "percona-server-mysql", emitter.records[0].Provider)
	})

	t.Run("a Restore of a deleted Instance carries the Instance only", func(t *testing.T) {
		t.Parallel()
		r, emitter := newTestRecorder(t)
		r.reader = fake.NewClientBuilder().WithScheme(r.scheme).Build()
		restore := &backupv1alpha1.Restore{
			ObjectMeta: metav1.ObjectMeta{Namespace: "team-a", Name: "r1", UID: "uid-r1"},
			Spec:       backupv1alpha1.RestoreSpec{InstanceRef: commonv1alpha1.ObjectRef{Name: "db1"}},
		}
		r.handlerFor(context.Background(), kindOf[*backupv1alpha1.Restore](t)).OnAdd(restore, false)
		require.Len(t, emitter.records, 1)
		assert.Equal(t, "db1", emitter.records[0].Instance)
		assert.Empty(t, emitter.records[0].Provider)
	})

	t.Run("an imported Backup has no Instance", func(t *testing.T) {
		t.Parallel()
		r, emitter := newTestRecorder(t)
		r.reader = fake.NewClientBuilder().WithScheme(r.scheme).WithObjects(db1.DeepCopy()).Build()
		external := backupv1alpha1.BackupOrigin{Type: backupv1alpha1.BackupOriginTypeExternal}
		r.handlerFor(context.Background(), kindOf[*backupv1alpha1.Backup](t)).OnAdd(backup(external), false)
		require.Len(t, emitter.records, 1)
		assert.Empty(t, emitter.records[0].Instance)
		assert.Empty(t, emitter.records[0].Provider)
	})
}
