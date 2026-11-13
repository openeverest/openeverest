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

package events

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	eventsv1 "k8s.io/api/events/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

//nolint:ireturn // the fake client is a client.Client
func newTestEmitter(t *testing.T) (*Emitter, client.Client) {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, eventsv1.AddToScheme(scheme))
	c := fake.NewClientBuilder().WithScheme(scheme).Build()
	e := NewEmitter(c, "controller-0")
	e.now = func() time.Time { return time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC) }
	return e, c
}

func listEvents(t *testing.T, c client.Client) []eventsv1.Event {
	t.Helper()
	list := &eventsv1.EventList{}
	require.NoError(t, c.List(context.Background(), list))
	return list.Items
}

func TestEmitter(t *testing.T) {
	t.Parallel()

	instance := corev1.ObjectReference{
		APIVersion: "core.openeverest.io/v1alpha1", Kind: "Instance",
		Namespace: "team-a", Name: "db1", UID: "0b5e8f9c-1111-2222-3333-444455556666",
	}

	t.Run("records the event with its details", func(t *testing.T) {
		t.Parallel()
		e, c := newTestEmitter(t)
		require.NoError(t, e.Emit(context.Background(), Record{
			Regarding: instance, Reason: ReasonPhaseChanged, Note: "phase changed", Warning: true,
			Actor: ControllerActor, PhaseFrom: "Provisioning", PhaseTo: "Failed", Key: "1234",
		}))

		items := listEvents(t, c)
		require.Len(t, items, 1)
		ev := items[0]
		assert.Equal(t, "db1.0b5e8f9c-1111-2222-3333-444455556666.phasechanged-1234", ev.Name)
		assert.Equal(t, "team-a", ev.Namespace)
		assert.Equal(t, LabelLifecycleEventValue, ev.Labels[LabelLifecycleEvent])
		assert.Equal(t, "PhaseChanged", ev.Reason)
		assert.Equal(t, corev1.EventTypeWarning, ev.Type)
		assert.Equal(t, ReportingController, ev.ReportingController)
		assert.Equal(t, "controller-0", ev.ReportingInstance)
		assert.Equal(t, instance, ev.Regarding)
		wantAnnotations := map[string]string{
			AnnotationEventActorType: "system", AnnotationEventActorID: "controller",
			AnnotationPhaseFrom: "Provisioning", AnnotationPhaseTo: "Failed",
		}
		assert.Equal(t, wantAnnotations, ev.Annotations)
	})

	t.Run("recording the same event again is a no-op", func(t *testing.T) {
		t.Parallel()
		e, c := newTestEmitter(t)
		record := Record{Regarding: instance, Reason: ReasonCreated}
		require.NoError(t, e.Emit(context.Background(), record))
		require.NoError(t, e.Emit(context.Background(), record))
		assert.Len(t, listEvents(t, c), 1)
	})

	t.Run("events with different keys are distinct", func(t *testing.T) {
		t.Parallel()
		e, c := newTestEmitter(t)
		require.NoError(t, e.Emit(context.Background(), Record{Regarding: instance, Reason: ReasonUpdated, Key: "2"}))
		require.NoError(t, e.Emit(context.Background(), Record{Regarding: instance, Reason: ReasonUpdated, Key: "3"}))
		assert.Len(t, listEvents(t, c), 2)
	})

	t.Run("events about cluster-scoped objects go to the default namespace", func(t *testing.T) {
		t.Parallel()
		e, c := newTestEmitter(t)
		plugin := corev1.ObjectReference{Kind: "Plugin", Name: "audit", UID: "uid-1"}
		require.NoError(t, e.Emit(context.Background(), Record{Regarding: plugin, Reason: ReasonCreated}))
		assert.Equal(t, "default", listEvents(t, c)[0].Namespace)
	})

	t.Run("long object names are truncated to a valid event name", func(t *testing.T) {
		t.Parallel()
		ref := instance
		ref.Name = strings.Repeat("a", 253)
		assert.LessOrEqual(t, len(eventName(Record{Regarding: ref, Reason: ReasonDeletionRequested})), maxNameLength)
	})

	t.Run("events without a UID get a generated name", func(t *testing.T) {
		t.Parallel()
		e, _ := newTestEmitter(t)
		ev := e.newEvent(Record{Regarding: corev1.ObjectReference{Kind: "User", Name: "alice"}, Reason: ReasonLoginFailed})
		assert.Empty(t, ev.Name)
		assert.Equal(t, "loginfailed-", ev.GenerateName)
	})
}
