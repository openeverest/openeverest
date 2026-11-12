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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	eventsv1 "k8s.io/api/events/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/openeverest/openeverest/v2/internal/server/handlers"
	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

func lifecycleEvent(namespace, object string) eventsv1.Event {
	return eventsv1.Event{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: object + ".created"},
		Regarding:  corev1.ObjectReference{Kind: "Instance", Namespace: namespace, Name: object},
	}
}

func eventNames(evs []eventsv1.Event) []string {
	names := make([]string, 0, len(evs))
	for _, ev := range evs {
		names = append(names, ev.Name)
	}
	return names
}

//nolint:gochecknoglobals // shared test table
var eventPolicyCases = []struct {
	desc   string
	policy string
	want   []string
}{
	{
		desc:   "admin reads every event",
		policy: newPolicy("g, bob, role:admin"),
		want:   []string{"db1.created", "db2.created", "my-plugin.created"},
	},
	{
		desc:   "namespace grant reads only the events of that namespace",
		policy: newPolicy("p, role:test, events, read, prod/team-a/*", "g, bob, role:test"),
		want:   []string{"db1.created"},
	},
	{
		desc:   "object grant reads only the events about that object",
		policy: newPolicy("p, role:test, events, read, prod/team-b/db2", "g, bob, role:test"),
		want:   []string{"db2.created"},
	},
	{
		desc:   "read on instances does not grant events",
		policy: newPolicy("p, role:test, instances, read, prod/*/*", "g, bob, role:test"),
		want:   []string{},
	},
	{
		desc:   "grant on another cluster reads nothing",
		policy: newPolicy("p, role:test, events, read, staging/*/*", "g, bob, role:test"),
		want:   []string{},
	},
}

func allLifecycleEvents() []eventsv1.Event {
	return []eventsv1.Event{
		lifecycleEvent("team-a", "db1"),
		lifecycleEvent("team-b", "db2"),
		lifecycleEvent("everest-system", "my-plugin"),
	}
}

func TestRBAC_ListEvents(t *testing.T) {
	t.Parallel()

	ctx := testUserContext(rbac.User{Subject: "bob"})
	for _, tc := range eventPolicyCases {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()
			enf, err := rbac.NewEnforcer(ctx, newConfigMapMock(tc.policy), zap.NewNop().Sugar())
			require.NoError(t, err)
			next := &handlers.MockHandler{}
			next.On("ListEvents", mock.Anything, "prod", mock.Anything).
				Return(&eventsv1.EventList{Items: allLifecycleEvents()}, nil)
			h := &rbacHandler{next: next, log: zap.NewNop().Sugar(), enforcer: enf, userGetter: testUserGetter}

			list, err := h.ListEvents(ctx, "prod", nil)
			require.NoError(t, err)
			assert.Equal(t, tc.want, eventNames(list.Items))
		})
	}
}

func TestRBAC_WatchEvents(t *testing.T) {
	t.Parallel()

	ctx := testUserContext(rbac.User{Subject: "bob"})
	for _, tc := range eventPolicyCases {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()
			enf, err := rbac.NewEnforcer(ctx, newConfigMapMock(tc.policy), zap.NewNop().Sugar())
			require.NoError(t, err)
			source := watch.NewFake()
			next := &handlers.MockHandler{}
			next.On("WatchEvents", mock.Anything, "prod", mock.Anything, "42").Return(source, nil)
			h := &rbacHandler{next: next, log: zap.NewNop().Sugar(), enforcer: enf, userGetter: testUserGetter}

			w, err := h.WatchEvents(ctx, "prod", nil, "42")
			require.NoError(t, err)
			defer w.Stop()

			go func() {
				for _, ev := range allLifecycleEvents() {
					source.Add(&ev)
				}
				// Bookmarks and errors carry no event data and always pass.
				source.Action(watch.Bookmark, &eventsv1.Event{ObjectMeta: metav1.ObjectMeta{ResourceVersion: "100"}})
				source.Error(&metav1.Status{Code: 410})
			}()

			var got []string
			bookmarkSeen := false
			for !bookmarkSeen {
				select {
				case ev := <-w.ResultChan():
					if ev.Type == watch.Bookmark {
						bookmarkSeen = true
						continue
					}
					got = append(got, ev.Object.(*eventsv1.Event).Name) //nolint:forcetypeassert // only events are added
				case <-time.After(5 * time.Second):
					t.Fatal("timed out waiting for the bookmark")
				}
			}
			assert.Equal(t, watch.Error, (<-w.ResultChan()).Type)
			assert.ElementsMatch(t, tc.want, got)
		})
	}
}
