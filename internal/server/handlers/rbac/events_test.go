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
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	extensionsv1alpha1 "github.com/openeverest/openeverest/v2/api/extensions/v1alpha1"
	"github.com/openeverest/openeverest/v2/internal/controller/lifecycleevents"
	"github.com/openeverest/openeverest/v2/internal/server/handlers"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

func lifecycleEvent(namespace, object string) eventsv1.Event {
	return eventsv1.Event{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: object + ".created"},
		Regarding: corev1.ObjectReference{
			APIVersion: corev1alpha1.GroupVersion.String(), Kind: "Instance", Namespace: namespace, Name: object,
		},
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
		want:   []string{"db1.created", "db2.created", "my-provider.created", "my-plugin.created"},
	},
	{
		desc:   "namespace grant reads only the events of that namespace",
		policy: newPolicy("p, role:test, instances, read-events, prod/team-a/*", "g, bob, role:test"),
		want:   []string{"db1.created"},
	},
	{
		desc:   "object grant reads only the events about that object",
		policy: newPolicy("p, role:test, instances, read-events, prod/team-b/db2", "g, bob, role:test"),
		want:   []string{"db2.created"},
	},
	{
		desc:   "read on instances does not grant events",
		policy: newPolicy("p, role:test, instances, read, prod/*/*", "g, bob, role:test"),
		want:   []string{},
	},
	{
		desc:   "all actions on instances grant events",
		policy: newPolicy("p, role:test, instances, *, prod/team-a/*", "g, bob, role:test"),
		want:   []string{"db1.created"},
	},
	{
		desc:   "read-events on another kind reads nothing",
		policy: newPolicy("p, role:test, backups, read-events, prod/*/*", "g, bob, role:test"),
		want:   []string{},
	},
	{
		desc:   "grant on another cluster reads nothing",
		policy: newPolicy("p, role:test, instances, read-events, staging/*/*", "g, bob, role:test"),
		want:   []string{},
	},
}

func allLifecycleEvents() []eventsv1.Event {
	provider := lifecycleEvent("", "my-provider")
	provider.Namespace = "default"
	provider.Regarding.APIVersion = corev1alpha1.GroupVersion.String()
	provider.Regarding.Kind = "Provider"
	provider.Regarding.Namespace = ""
	return []eventsv1.Event{
		lifecycleEvent("team-a", "db1"),
		lifecycleEvent("team-b", "db2"),
		provider,
		lifecycleEvent("everest-system", "my-plugin"),
	}
}

func TestRBAC_ListEventsAuthorizesTheRegardingObject(t *testing.T) {
	t.Parallel()

	events := []eventsv1.Event{
		lifecycleEvent("team-a", "db1"),
		{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "prod-provider.created"},
			Regarding: corev1.ObjectReference{
				APIVersion: corev1alpha1.GroupVersion.String(), Kind: "Provider", Name: "prod-provider",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "prod-plugin.created"},
			Regarding: corev1.ObjectReference{
				APIVersion: extensionsv1alpha1.GroupVersion.String(), Kind: "Plugin", Name: "prod-plugin",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "team-a.created"},
			Regarding:  corev1.ObjectReference{APIVersion: "v1", Kind: "Namespace", Name: "team-a"},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Namespace: "everest-system", Name: "alice.login"},
			Regarding:  corev1.ObjectReference{Kind: "User", Namespace: "everest-system", Name: "alice"},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Namespace: "team-a", Name: "widget.created"},
			Regarding:  corev1.ObjectReference{APIVersion: "example.com/v1", Kind: "Widget", Namespace: "team-a", Name: "widget"},
		},
	}
	cases := []struct {
		name   string
		policy string
		want   []string
	}{
		{
			name:   "admin reads the events of every recorded kind",
			policy: newPolicy("g, bob, role:admin"),
			want:   []string{"db1.created", "prod-provider.created", "prod-plugin.created", "team-a.created", "alice.login"},
		},
		{
			name:   "namespaced grant reads no cluster-scoped or login events",
			policy: newPolicy("p, role:test, instances, read-events, prod/team-a/*", "g, bob, role:test"),
			want:   []string{"db1.created"},
		},
		{
			name:   "provider events use provider read-events permissions",
			policy: newPolicy("p, role:test, providers, read-events, prod/*", "g, bob, role:test"),
			want:   []string{"prod-provider.created"},
		},
		{
			name:   "cluster-scoped resource uses the cluster/name object",
			policy: newPolicy("p, role:test, plugins, read-events, prod/*", "g, bob, role:test"),
			want:   []string{"prod-plugin.created"},
		},
		{
			name:   "namespace events need read-events on the namespace",
			policy: newPolicy("p, role:test, namespaces, read-events, prod/team-a", "g, bob, role:test"),
			want:   []string{"team-a.created"},
		},
		{
			name: "resource wildcard reads every object's events but no logins",
			policy: newPolicy(
				"p, role:test, *, read-events, */*/*", "p, role:test, *, read-events, */*", "g, bob, role:test",
			),
			want: []string{"db1.created", "prod-provider.created", "prod-plugin.created", "team-a.created"},
		},
		{
			name:   "platform permission reads login events",
			policy: newPolicy("p, role:test, platform-events, read, *", "g, bob, role:test"),
			want:   []string{"alice.login"},
		},
	}
	ctx := testUserContext(rbac.User{Subject: "bob"})
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			enf, err := rbac.NewEnforcer(ctx, newConfigMapMock(tc.policy), zap.NewNop().Sugar())
			require.NoError(t, err)
			next := &handlers.MockHandler{}
			next.On("ListEvents", mock.Anything, "prod", mock.Anything, int64(100), "").
				Return(&eventsv1.EventList{Items: events}, nil)
			h := &rbacHandler{next: next, log: zap.NewNop().Sugar(), enforcer: enf, userGetter: testUserGetter}

			list, err := h.ListEvents(ctx, "prod", nil, 100, "")
			require.NoError(t, err)
			assert.Equal(t, tc.want, eventNames(list.Items))
		})
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
			next.On("ListEvents", mock.Anything, "prod", mock.Anything, int64(100), "").
				Return(&eventsv1.EventList{Items: allLifecycleEvents()}, nil)
			h := &rbacHandler{next: next, log: zap.NewNop().Sugar(), enforcer: enf, userGetter: testUserGetter}

			list, err := h.ListEvents(ctx, "prod", nil, 100, "")
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

// Events about a recorded kind without an RBAC resource would be hidden from everyone.
func TestEventResourcesCoverRecordedKinds(t *testing.T) {
	t.Parallel()

	scheme := kubernetes.CreateScheme()
	for _, kind := range lifecycleevents.DefaultKinds() {
		gvk, err := apiutil.GVKForObject(kind.New(), scheme)
		require.NoError(t, err)
		resource, ok := eventResources[gvk.GroupKind()]
		if assert.True(t, ok, "no RBAC resource for %s", gvk.GroupKind()) {
			assert.Contains(t, rbac.AllResources, resource)
		}
	}
}

func TestRBAC_ListEventsWithPluginServiceToken(t *testing.T) {
	t.Parallel()

	events := []eventsv1.Event{
		lifecycleEvent("team-a", "db1"),
		{
			ObjectMeta: metav1.ObjectMeta{Namespace: "default", Name: "prod-provider.created"},
			Regarding: corev1.ObjectReference{
				APIVersion: corev1alpha1.GroupVersion.String(), Kind: "Provider", Name: "prod-provider",
			},
		},
		{
			ObjectMeta: metav1.ObjectMeta{Namespace: "everest-system", Name: "alice.login"},
			Regarding:  corev1.ObjectReference{Kind: "User", Namespace: "everest-system", Name: "alice"},
		},
	}
	// The plugin is in no RBAC policy: only the permissions its token carries apply.
	ctx := testUserContext(rbac.User{
		Subject: "plugin:audit",
		Permissions: []extensionsv1alpha1.PluginPermission{
			{Verb: rbac.ActionReadEvents, Resource: rbac.ResourceInstances},
			{Verb: rbac.ActionRead, Resource: rbac.ResourcePlatformEvents},
		},
	})
	enf, err := rbac.NewEnforcer(ctx, newConfigMapMock(newPolicy()), zap.NewNop().Sugar())
	require.NoError(t, err)
	next := &handlers.MockHandler{}
	next.On("ListEvents", mock.Anything, "prod", mock.Anything, int64(100), "").
		Return(&eventsv1.EventList{Items: events}, nil)
	h := &rbacHandler{next: next, log: zap.NewNop().Sugar(), enforcer: enf, userGetter: testUserGetter}

	list, err := h.ListEvents(ctx, "prod", nil, 100, "")
	require.NoError(t, err)
	assert.Equal(t, []string{"db1.created", "alice.login"}, eventNames(list.Items))
}

func TestRBAC_PluginServiceTokenCannotWriteSpec001Resources(t *testing.T) {
	t.Parallel()

	ctx := testUserContext(rbac.User{
		Subject:     "plugin:pmm",
		Permissions: []extensionsv1alpha1.PluginPermission{{Verb: "*", Resource: "*"}},
	})
	enf, err := rbac.NewEnforcer(ctx, newConfigMapMock(newPolicy()), zap.NewNop().Sugar())
	require.NoError(t, err)
	h := &rbacHandler{log: zap.NewNop().Sugar(), enforcer: enf, userGetter: testUserGetter}

	require.NoError(t, h.enforce(ctx, rbac.ResourceMonitoringDestinations, rbac.ActionRead, "prod/ns/dest"))
	require.ErrorIs(t, h.enforce(ctx, rbac.ResourceMonitoringDestinations, rbac.ActionCreate, "prod/ns/dest"),
		ErrInsufficientPermissions)
}
