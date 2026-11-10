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
