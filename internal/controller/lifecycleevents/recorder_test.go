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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	toolscache "k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

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
	emitter := &recordingEmitter{}
	r := NewRecorder(nil, scheme, emitter, DefaultKinds(), logr.Discard())
	r.now = func() time.Time { return now }
	return r, emitter
}

func instance(generation int64, phase corev1alpha1.InstancePhase, createdAgo time.Duration) *corev1alpha1.Instance {
	return &corev1alpha1.Instance{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "team-a", Name: "db1", UID: "uid-1", Generation: generation,
			ResourceVersion:   "100",
			CreationTimestamp: metav1.NewTime(now.Add(-createdAgo)),
			Annotations:       map[string]string{events.AnnotationActorType: "user", events.AnnotationActorID: "alice"},
		},
		Status: corev1alpha1.InstanceStatus{Phase: phase},
	}
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
		instanceHandler(t, r).OnAdd(instance(1, "", 0), false)

		require.Len(t, emitter.records, 1)
		rec := emitter.records[0]
		assert.Equal(t, events.ReasonCreated, rec.Reason)
		assert.Equal(t, events.Actor{Type: "user", ID: "alice"}, rec.Actor)
		assert.Equal(t, corev1.ObjectReference{
			APIVersion: "core.openeverest.io/v1alpha1", Kind: "Instance",
			Namespace: "team-a", Name: "db1", UID: "uid-1", ResourceVersion: "100",
		}, rec.Regarding)
	})

	t.Run("the initial list only records objects created within the lookback", func(t *testing.T) {
		t.Parallel()
		r, emitter := newTestRecorder(t)
		handler := instanceHandler(t, r)
		handler.OnAdd(instance(1, "Ready", 2*DefaultLookback), true)
		assert.Empty(t, emitter.records, "objects older than the lookback were recorded already, or long ago")

		handler.OnAdd(instance(1, "Ready", 10*time.Minute), true)
		assert.Equal(t, []events.Reason{events.ReasonCreated}, emitter.reasons(),
			"an object created while no recorder ran is recorded")
	})

	t.Run("the initial list records deletions requested within the lookback", func(t *testing.T) {
		t.Parallel()
		r, emitter := newTestRecorder(t)
		obj := instance(1, "Terminating", 2*DefaultLookback)
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
		instanceHandler(t, r).OnUpdate(instance(1, "Provisioning", time.Hour), newObj)

		require.Len(t, emitter.records, 1)
		rec := emitter.records[0]
		assert.Equal(t, events.ReasonPhaseChanged, rec.Reason)
		assert.Equal(t, "Provisioning", rec.PhaseFrom)
		assert.Equal(t, "Ready", rec.PhaseTo)
		assert.Equal(t, "101", rec.Key)
		assert.Equal(t, events.ControllerActor, rec.Actor)
		assert.False(t, rec.Warning)
	})

	t.Run("a failure is recorded as a warning", func(t *testing.T) {
		t.Parallel()
		r, emitter := newTestRecorder(t)
		instanceHandler(t, r).OnUpdate(instance(1, "Ready", time.Hour), instance(1, "Failed", time.Hour))
		require.Len(t, emitter.records, 1)
		assert.True(t, emitter.records[0].Warning)
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
	handler.OnDelete(instance(1, "Terminating", time.Hour))
	// A delete missed by the watch is delivered as a tombstone.
	handler.OnDelete(toolscache.DeletedFinalStateUnknown{Key: "team-a/db1", Obj: instance(1, "Terminating", time.Hour)})
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
