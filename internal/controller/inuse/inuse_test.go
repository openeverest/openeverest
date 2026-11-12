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

package inuse

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const testFinalizer = "test.openeverest.io/in-use-protection"

func newGuard(objs ...client.Object) (Guard, *events.FakeRecorder) {
	rec := events.NewFakeRecorder(10)
	return Guard{
		Client:       fake.NewClientBuilder().WithObjects(objs...).Build(),
		Recorder:     rec,
		Finalizer:    testFinalizer,
		ReferrerKind: "Instance",
	}, rec
}

func referent(finalizers ...string) *corev1.ConfigMap {
	return &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "referent", Namespace: "team-a", Finalizers: finalizers}}
}

// deleting is a referent whose deletion has started.
func deleting(finalizers ...string) *corev1.ConfigMap {
	cm := referent(finalizers...)
	cm.DeletionTimestamp = &metav1.Time{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	return cm
}

// referrer is an object referring to the referent, being deleted or not.
func referrer(name string, beingDeleted bool) *corev1.ConfigMap {
	cm := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "team-a"}}
	if beingDeleted {
		cm.Finalizers = []string{"other.io/finalizer"}
		cm.DeletionTimestamp = &metav1.Time{Time: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	}
	return cm
}

func get(t *testing.T, c client.Client) (*corev1.ConfigMap, error) {
	t.Helper()
	cm := &corev1.ConfigMap{}
	err := c.Get(t.Context(), client.ObjectKeyFromObject(referent()), cm)
	return cm, err
}

func TestGuardProtect(t *testing.T) {
	t.Parallel()

	t.Run("adds the finalizer once", func(t *testing.T) {
		t.Parallel()
		g, _ := newGuard(referent("other.io/finalizer"))
		cm, err := get(t, g.Client)
		require.NoError(t, err)

		require.NoError(t, g.Protect(t.Context(), cm))
		version := cm.ResourceVersion
		require.NoError(t, g.Protect(t.Context(), cm))

		stored, err := get(t, g.Client)
		require.NoError(t, err)
		assert.Equal(t, []string{"other.io/finalizer", testFinalizer}, stored.Finalizers)
		assert.Equal(t, version, stored.ResourceVersion)
	})

	t.Run("leaves an object being deleted alone", func(t *testing.T) {
		t.Parallel()
		g, _ := newGuard(deleting("other.io/finalizer"))
		cm, err := get(t, g.Client)
		require.NoError(t, err)

		require.NoError(t, g.Protect(t.Context(), cm))

		stored, err := get(t, g.Client)
		require.NoError(t, err)
		assert.Equal(t, []string{"other.io/finalizer"}, stored.Finalizers)
	})
}

func TestGuardRelease(t *testing.T) {
	t.Parallel()

	t.Run("keeps the finalizer of an object not being deleted", func(t *testing.T) {
		t.Parallel()
		g, rec := newGuard(referent(testFinalizer))
		cm, err := get(t, g.Client)
		require.NoError(t, err)

		released, err := g.Release(t.Context(), cm, nil)

		require.NoError(t, err)
		assert.False(t, released)
		stored, err := get(t, g.Client)
		require.NoError(t, err)
		assert.Equal(t, []string{testFinalizer}, stored.Finalizers)
		assert.Empty(t, rec.Events)
	})

	t.Run("holds the deletion while referrers remain", func(t *testing.T) {
		t.Parallel()
		g, rec := newGuard(deleting(testFinalizer))
		cm, err := get(t, g.Client)
		require.NoError(t, err)

		released, err := g.Release(t.Context(), cm, []string{"payments", "orders"})

		require.NoError(t, err)
		assert.False(t, released)
		stored, err := get(t, g.Client)
		require.NoError(t, err)
		assert.Equal(t, []string{testFinalizer}, stored.Finalizers)
		require.Len(t, rec.Events, 1)
		assert.Equal(t, "Warning DeletionBlocked deletion waits for 2 Instance(s) to stop using it: orders, payments", <-rec.Events)
	})

	t.Run("removes the finalizer once no referrer is left", func(t *testing.T) {
		t.Parallel()
		g, rec := newGuard(deleting(testFinalizer))
		cm, err := get(t, g.Client)
		require.NoError(t, err)

		released, err := g.Release(t.Context(), cm, nil)

		require.NoError(t, err)
		assert.True(t, released)
		_, err = get(t, g.Client)
		assert.True(t, apierrors.IsNotFound(err), "referent still present: %v", err)
		assert.Empty(t, rec.Events)
	})

	t.Run("removes only its own finalizer", func(t *testing.T) {
		t.Parallel()
		g, _ := newGuard(deleting(testFinalizer, "other.io/finalizer"))
		cm, err := get(t, g.Client)
		require.NoError(t, err)

		released, err := g.Release(t.Context(), cm, nil)

		require.NoError(t, err)
		assert.True(t, released)
		stored, err := get(t, g.Client)
		require.NoError(t, err)
		assert.Equal(t, []string{"other.io/finalizer"}, stored.Finalizers)
	})
}

func TestSummarise(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "a, b", summarise([]string{"b", "a"}))
	assert.Equal(t, "a, b, c, d, e and 2 more", summarise([]string{"g", "f", "e", "d", "c", "b", "a"}))
}

func TestCount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		referrers        []*corev1.ConfigMap
		referentDeleting bool
		want             int
	}{
		{name: "no referrer", want: 0},
		{
			name:      "referrers not being deleted count",
			referrers: []*corev1.ConfigMap{referrer("orders", false), referrer("payments", false)},
			want:      2,
		},
		{
			name:      "a referrer being deleted does not count while the referent is not being deleted",
			referrers: []*corev1.ConfigMap{referrer("orders", false), referrer("payments", true)},
			want:      1,
		},
		{
			name:      "referrers all being deleted count zero while the referent is not being deleted",
			referrers: []*corev1.ConfigMap{referrer("orders", true), referrer("payments", true)},
			want:      0,
		},
		{
			name:             "a referrer being deleted counts once the referent is being deleted",
			referrers:        []*corev1.ConfigMap{referrer("orders", false), referrer("payments", true)},
			referentDeleting: true,
			want:             2,
		},
		{name: "no referrer for a referent being deleted", referentDeleting: true, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, Count(tt.referrers, tt.referentDeleting))
		})
	}
}

func TestMeasure(t *testing.T) {
	t.Parallel()

	// The API server has a referrer the cache has not seen yet.
	cached := []*corev1.ConfigMap{referrer("orders", false), referrer("payments", true)}
	live := []*corev1.ConfigMap{referrer("orders", false), referrer("payments", true), referrer("invoices", false)}
	boom := errors.New("boom")

	tests := []struct {
		name     string
		referent *corev1.ConfigMap
		cached   []*corev1.ConfigMap
		live     []*corev1.ConfigMap
		listErr  error
		want     Usage
	}{
		{
			name:     "a referent not being deleted counts the cached referrers not being deleted",
			referent: referent(testFinalizer),
			cached:   cached,
			live:     live,
			want:     Usage{Count: 1, Referrers: []string{"orders", "payments"}},
		},
		{
			name:     "a referent being deleted counts every referrer the API server has",
			referent: deleting(testFinalizer),
			cached:   cached,
			live:     live,
			want:     Usage{Count: 3, Referrers: []string{"orders", "payments", "invoices"}},
		},
		{
			name:     "no referrer",
			referent: referent(testFinalizer),
			want:     Usage{},
		},
		{
			name:     "a list error is returned",
			referent: deleting(testFinalizer),
			listErr:  boom,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			usage, err := Measure(t.Context(), tt.referent, func(_ context.Context, isLive bool) ([]*corev1.ConfigMap, error) {
				if isLive {
					return tt.live, tt.listErr
				}
				return tt.cached, tt.listErr
			})

			if tt.listErr != nil {
				require.ErrorIs(t, err, tt.listErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, usage)
		})
	}
}
