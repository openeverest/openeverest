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

// Package inuse holds the in-use protection finalizer of objects other
// objects refer to. An admission policy refuses deleting such an object
// while its usage count is above zero; the finalizer is the backstop for a
// count that is not up to date yet. It is added to every object, not only
// to those in use, so it is already there when a deletion slips through,
// and it is removed only once a read of the API server finds no referrer.
package inuse

import (
	"context"
	"fmt"
	"slices"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// ReasonDeletionBlocked is the reason of the event recorded while the
// finalizer holds a deletion.
const ReasonDeletionBlocked = "DeletionBlocked"

// maxNamedReferrers caps the referrers an event names.
const maxNamedReferrers = 5

// Guard holds one in-use protection finalizer on the objects of one kind.
type Guard struct {
	Client   client.Client
	Recorder events.EventRecorder
	// Finalizer is the finalizer the guard holds.
	Finalizer string
	// ReferrerKind names the referrers in events, e.g. "Instance".
	ReferrerKind string
}

// Protect adds the finalizer to obj unless obj is already being deleted,
// when no finalizer can be added any more.
func (g Guard) Protect(ctx context.Context, obj client.Object) error {
	if !obj.GetDeletionTimestamp().IsZero() || controllerutil.ContainsFinalizer(obj, g.Finalizer) {
		return nil
	}
	return g.patchFinalizers(ctx, obj, controllerutil.AddFinalizer)
}

// Release removes the finalizer from obj, which is being deleted, when
// referrers is empty, and reports whether it did. referrers must be read
// from the API server, not the cache: the cache may not have seen a
// referrer created just before the deletion. While referrers remain, it
// keeps the finalizer and records a DeletionBlocked event naming them.
func (g Guard) Release(ctx context.Context, obj client.Object, referrers []string) (bool, error) {
	if obj.GetDeletionTimestamp().IsZero() || !controllerutil.ContainsFinalizer(obj, g.Finalizer) {
		return false, nil
	}
	if len(referrers) > 0 {
		g.Recorder.Eventf(obj, nil, corev1.EventTypeWarning, ReasonDeletionBlocked, "Delete",
			"deletion waits for %d %s(s) to stop using it: %s", len(referrers), g.ReferrerKind, summarise(referrers),
		)
		return false, nil
	}
	if err := g.patchFinalizers(ctx, obj, controllerutil.RemoveFinalizer); err != nil {
		return false, client.IgnoreNotFound(err)
	}
	return true, nil
}

// patchFinalizers applies mutate to obj's finalizers with an optimistic
// lock, so a concurrent change to the list is never overwritten.
func (g Guard) patchFinalizers(ctx context.Context, obj client.Object, mutate func(client.Object, string) bool) error {
	base, ok := obj.DeepCopyObject().(client.Object)
	if !ok {
		return fmt.Errorf("copy %T", obj)
	}
	mutate(obj, g.Finalizer)
	if err := g.Client.Patch(ctx, obj, client.MergeFromWithOptions(base, client.MergeFromWithOptimisticLock{})); err != nil {
		return fmt.Errorf("patch finalizers of %s: %w", client.ObjectKeyFromObject(obj), err)
	}
	return nil
}

// summarise lists names sorted, at most maxNamedReferrers of them.
func summarise(names []string) string {
	sorted := slices.Sorted(slices.Values(names))
	if len(sorted) <= maxNamedReferrers {
		return strings.Join(sorted, ", ")
	}
	return fmt.Sprintf("%s and %d more", strings.Join(sorted[:maxNamedReferrers], ", "), len(sorted)-maxNamedReferrers)
}
