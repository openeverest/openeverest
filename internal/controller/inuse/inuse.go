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

// Package inuse holds the deletion safety of objects other objects refer
// to: the referent's usage count and its in-use protection finalizer.
//
// An admission policy refuses deleting a referent while its usage count is
// above zero. Count is the rule every count follows, for every kind of
// referent and referrer:
//
//   - while the referent is not being deleted, referrers being deleted
//     themselves are left out, so a referrer and its referent can be
//     deleted together;
//   - once the referent is being deleted, every referrer counts, read from
//     the API server, so the count states what its deletion waits for.
//
// Measure applies the rule to the referrers of one kind, reading them from
// the cache or the API server as it requires.
//
// The finalizer (Guard) is the backstop for a count that is not up to date
// yet and holds the referent while referrers being deleted remain. On core
// kinds it is added to every object, not only to those in use, so it is
// already there when a deletion slips through. On Secrets, which core does
// not own, it is held only while the Secret is in use (Guard.WhileInUse),
// so deleting an unused Secret is immediate. Either way it is removed only
// once a read of the API server finds no referrer, those being deleted
// included.
package inuse

import (
	"context"
	"fmt"
	"slices"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	// ReferrerKind names the referrers Release is given in events, e.g.
	// "Instance".
	ReferrerKind string
	// WhileInUse holds the finalizer only while the object is in use, as on
	// Secrets, instead of on every object: Unhold then also removes it from
	// an object that is not being deleted. The caller adds it when the
	// first referrer appears; Protect, which adds it unconditionally, is
	// not used.
	WhileInUse bool
}

// Referrers names the referrers of one kind an object has.
type Referrers struct {
	// Kind is the kind of the referrers, e.g. "MonitoringDestination".
	Kind string
	// Names are their names, those being deleted included.
	Names []string
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
// from the API server, not the cache, and include referrers being deleted
// themselves: the cache may not have seen a referrer created just before
// the deletion, and a referrer being deleted may still need obj to tear
// down. Measure's Referrers for a referent being deleted are such a list.
// While referrers remain, it keeps the finalizer and records a
// DeletionBlocked event naming them. It is Unhold, for referrers of
// ReferrerKind, followed by the write.
func (g Guard) Release(ctx context.Context, obj client.Object, referrers []string) (bool, error) {
	base, ok := obj.DeepCopyObject().(client.Object)
	if !ok {
		return false, fmt.Errorf("copy %T", obj)
	}
	if !g.Unhold(obj, Referrers{Kind: g.ReferrerKind, Names: referrers}) {
		return false, nil
	}
	if err := g.patch(ctx, base, obj); err != nil {
		return false, client.IgnoreNotFound(err)
	}
	return true, nil
}

// Unhold is Release without the write, for referrers of several kinds: it
// removes the finalizer from obj in memory when referrers name no object,
// and reports whether it did. The caller then writes obj, with its other
// changes, in one patch locked on obj's resourceVersion. referrers must be
// read from the API server, as for Release.
//
// It removes the finalizer only from an object being deleted, or from any
// object with WhileInUse. While referrers remain and obj is being deleted,
// it keeps the finalizer and records a DeletionBlocked event naming them
// by kind, e.g. "1 MonitoringDestination: a; 2 BackupStorages: b, c".
func (g Guard) Unhold(obj client.Object, referrers ...Referrers) bool {
	deleting := !obj.GetDeletionTimestamp().IsZero()
	if (!deleting && !g.WhileInUse) || !controllerutil.ContainsFinalizer(obj, g.Finalizer) {
		return false
	}
	if message, inUse := blockedMessage(referrers); inUse {
		if deleting {
			g.Recorder.Eventf(obj, nil, corev1.EventTypeWarning, ReasonDeletionBlocked, "Delete", "%s", message)
		}
		return false
	}
	return controllerutil.RemoveFinalizer(obj, g.Finalizer)
}

// blockedMessage describes the referrers a deletion waits for, and reports
// whether there is any.
func blockedMessage(referrers []Referrers) (string, bool) {
	var kinds []Referrers
	total := 0
	for _, r := range referrers {
		if len(r.Names) > 0 {
			kinds = append(kinds, r)
			total += len(r.Names)
		}
	}
	switch len(kinds) {
	case 0:
		return "", false
	case 1:
		return fmt.Sprintf("deletion waits for %d %s(s) to stop using it: %s",
			len(kinds[0].Names), kinds[0].Kind, summarise(kinds[0].Names),
		), true
	}
	parts := make([]string, 0, len(kinds))
	for _, r := range kinds {
		kind := r.Kind
		if len(r.Names) != 1 {
			kind += "s"
		}
		parts = append(parts, fmt.Sprintf("%d %s: %s", len(r.Names), kind, summarise(r.Names)))
	}
	return fmt.Sprintf("deletion waits for %d objects to stop using it: %s", total, strings.Join(parts, "; ")), true
}

// patchFinalizers applies mutate to obj's finalizers with an optimistic
// lock, so a concurrent change to the list is never overwritten.
func (g Guard) patchFinalizers(ctx context.Context, obj client.Object, mutate func(client.Object, string) bool) error {
	base, ok := obj.DeepCopyObject().(client.Object)
	if !ok {
		return fmt.Errorf("copy %T", obj)
	}
	mutate(obj, g.Finalizer)
	return g.patch(ctx, base, obj)
}

// patch writes the changes from base to obj with an optimistic lock.
func (g Guard) patch(ctx context.Context, base, obj client.Object) error {
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

// Count returns the usage count a referent publishes for referrers, the
// objects of one kind that refer to it. Every usage count follows this rule:
//
//   - While the referent is not being deleted, referrers being deleted
//     themselves do not count. The in-use policy then admits deleting the
//     referent once its referrers are being deleted, as when a GitOps prune
//     or the UI deletes an Instance and then its destination, and the
//     finalizer still holds the referent until those referrers are gone.
//   - Once the referent is being deleted, every referrer counts, so the
//     count states what the deletion waits for. referrers must then be read
//     from the API server, as for Guard.Release; Measure reads them so.
func Count[T metav1.Object](referrers []T, referentDeleting bool) int {
	if referentDeleting {
		return len(referrers)
	}
	n := 0
	for _, r := range referrers {
		if r.GetDeletionTimestamp().IsZero() {
			n++
		}
	}
	return n
}

// Lister lists the objects of one kind that refer to a referent, those
// being deleted included. With live set it reads the API server; otherwise
// it may read the cache and its field indexes, which can lag the API
// server.
type Lister[T metav1.Object] func(ctx context.Context, live bool) ([]T, error)

// Usage is how the objects of one kind use a referent.
type Usage struct {
	// Count is the usage count the referent publishes, by Count.
	Count int
	// Referrers names every referrer found, those being deleted included.
	// Once the referent is being deleted, they were read from the API
	// server, so they are the list Guard.Release takes.
	Referrers []string
}

// Measure lists the referrers of referent with list and counts them by
// Count: from the cache while referent is not being deleted, and from the
// API server once it is.
func Measure[T metav1.Object](ctx context.Context, referent metav1.Object, list Lister[T]) (Usage, error) {
	deleting := !referent.GetDeletionTimestamp().IsZero()
	referrers, err := list(ctx, deleting)
	if err != nil {
		return Usage{}, err
	}
	usage := Usage{Count: Count(referrers, deleting)}
	for _, r := range referrers {
		usage.Referrers = append(usage.Referrers, r.GetName())
	}
	return usage, nil
}
