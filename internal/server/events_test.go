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

package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	eventsv1 "k8s.io/api/events/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	api "github.com/openeverest/openeverest/v2/internal/server/api"
	"github.com/openeverest/openeverest/v2/internal/server/handlers"
	"github.com/openeverest/openeverest/v2/pkg/events"
)

func kubeEvent(name, rv, kind, reason string, at time.Time) eventsv1.Event {
	return eventsv1.Event{
		ObjectMeta: metav1.ObjectMeta{Namespace: "team-a", Name: name, ResourceVersion: rv},
		EventTime:  metav1.NewMicroTime(at),
		Reason:     reason,
		Type:       corev1.EventTypeNormal,
		Regarding:  corev1.ObjectReference{APIVersion: "core.openeverest.io/v1alpha1", Kind: kind, Name: "db1", UID: "uid-1"},
	}
}

//nolint:ireturn // echo.Context is an interface by design; this is a test helper.
func newEventsRequest(ctx context.Context, t *testing.T) (echo.Context, *httptest.ResponseRecorder) {
	t.Helper()
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/v1/clusters/main/events", nil)
	rec := httptest.NewRecorder()
	return echo.New().NewContext(req, rec), rec
}

func TestToLifecycleEvent(t *testing.T) {
	t.Parallel()

	at := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)
	ev := kubeEvent("db1.uid-1.phasechanged-7", "7", "Instance", "PhaseChanged", at)
	ev.Type = corev1.EventTypeWarning
	ev.Note = "Instance team-a/db1 phase changed"
	ev.Annotations = map[string]string{
		events.AnnotationEventActorType: "system", events.AnnotationEventActorID: "controller",
		events.AnnotationPhaseFrom: "Ready", events.AnnotationPhaseTo: "Failed",
	}

	systemActor := api.LifecycleEventActorTypeSystem
	want := api.LifecycleEvent{
		Name:       "db1.uid-1.phasechanged-7",
		Cursor:     "7",
		OccurredAt: at,
		Type:       "instance.phase-changed",
		Reason:     api.LifecycleEventReasonPhaseChanged,
		Severity:   api.LifecycleEventSeverityWarning,
		Cluster:    "main",
		Namespace:  new("team-a"),
		Object: api.LifecycleEventObject{
			ApiVersion: new("core.openeverest.io/v1alpha1"), Kind: "Instance", Name: "db1", Uid: new("uid-1"),
		},
		Actor:   &api.LifecycleEventActor{Type: &systemActor, Id: new("controller")},
		Phase:   &api.LifecycleEventPhase{From: new("Ready"), To: new("Failed")},
		Message: new("Instance team-a/db1 phase changed"),
	}
	assert.Equal(t, want, toLifecycleEvent("main", &ev))

	assert.Equal(t, "user.login-succeeded", eventType("User", "LoginSucceeded"))
}

func TestListEvents(t *testing.T) {
	t.Parallel()

	at := time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)
	next := &handlers.MockHandler{}
	list := &eventsv1.EventList{
		ListMeta: metav1.ListMeta{ResourceVersion: "50"},
		Items: []eventsv1.Event{
			kubeEvent("later", "12", "Instance", "PhaseChanged", at.Add(time.Minute)),
			kubeEvent("earlier", "11", "Instance", "Created", at),
			kubeEvent("other-kind", "13", "Backup", "Created", at),
		},
	}
	next.On("ListEvents", mock.Anything, "main", []string{"team-a"}).Return(list, nil)
	srv := &EverestServer{handler: next}

	t.Run("lists the matching events, oldest first, with the resourceVersion to stream from", func(t *testing.T) {
		t.Parallel()
		c, rec := newEventsRequest(context.Background(), t)
		require.NoError(t, srv.ListEvents(c, "main", api.ListEventsParams{
			Namespaces: new("team-a"), Kinds: new("instance"),
		}))
		require.Equal(t, http.StatusOK, rec.Code)
		var list api.LifecycleEventList
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &list))
		assert.Equal(t, "50", list.ResourceVersion)
		require.Len(t, list.Items, 2)
		assert.Equal(t, "earlier", list.Items[0].Name)
		assert.Equal(t, "later", list.Items[1].Name)
	})

	for name, params := range map[string]api.ListEventsParams{
		"unknown reason":      {Reasons: new("Exploded")},
		"since without watch": {Since: new("42")},
	} {
		t.Run(name+" is a bad request", func(t *testing.T) {
			t.Parallel()
			c, rec := newEventsRequest(context.Background(), t)
			require.NoError(t, srv.ListEvents(c, "main", params))
			assert.Equal(t, http.StatusBadRequest, rec.Code)
		})
	}
}

func TestStreamEvents(t *testing.T) {
	t.Parallel()

	source := watch.NewFake()
	next := &handlers.MockHandler{}
	next.On("WatchEvents", mock.Anything, "main", []string(nil), "40").Return(source, nil)
	srv := &EverestServer{handler: next}

	go func() {
		created := kubeEvent("db1.created", "41", "Instance", "Created", time.Now())
		source.Add(&created)
		filtered := kubeEvent("db1.deleted", "42", "Instance", "Deleted", time.Now())
		source.Add(&filtered)
		// Updates and expiry of the Events are not lifecycle events.
		source.Modify(&created)
		source.Delete(&created)
		source.Action(watch.Bookmark, &eventsv1.Event{ObjectMeta: metav1.ObjectMeta{ResourceVersion: "45"}})
		source.Error(&metav1.Status{Code: http.StatusGone, Reason: metav1.StatusReasonExpired})
	}()

	c, rec := newEventsRequest(context.Background(), t)
	require.NoError(t, srv.ListEvents(c, "main", api.ListEventsParams{
		Watch: new(true), Since: new("40"), Reasons: new("Created"),
	}))

	assert.Equal(t, "text/event-stream", rec.Header().Get(echo.HeaderContentType))
	frames := strings.Split(strings.TrimSpace(rec.Body.String()), "\n\n")
	require.Len(t, frames, 4, rec.Body.String())
	assert.Equal(t, ": connected", frames[0])
	assert.True(t, strings.HasPrefix(frames[1], "id: 41\nevent: instance.created\ndata: {"), frames[1])
	assert.Equal(t, "id: 45\nevent: bookmark\ndata: {}", frames[2])
	assert.True(t, strings.HasPrefix(frames[3], "event: error\ndata: {\"code\":410,"), frames[3])
}
