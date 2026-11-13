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

// Package lifecycleevents records lifecycle events (see pkg/events) for the objects
// OpenEverest manages, however they were changed: through the OpenEverest API, kubectl or
// a GitOps tool.
package lifecycleevents

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	toolscache "k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	"github.com/openeverest/openeverest/v2/pkg/events"
)

// DefaultLookback is how far back the Recorder looks, when it starts, for objects created
// or deleted while no Recorder was running. It matches the default Kubernetes Event TTL:
// events older than that would already be gone, so recording them again is pointless.
const DefaultLookback = time.Hour

// Kind registers a kind of object to record lifecycle events for.
type Kind struct {
	// New returns an empty object of the kind.
	New func() client.Object
	// Phase returns the phase (or state) of an object. Nil for kinds without one, which
	// get no PhaseChanged events.
	Phase func(client.Object) string
	// Failed reports whether a phase is a failure. Changes to it are recorded as
	// Warning events.
	Failed func(phase string) bool
	// Include, when set, limits the events to the objects it accepts.
	Include func(client.Object) bool
}

type emitter interface {
	Emit(ctx context.Context, r events.Record) error
}

// Recorder records lifecycle events for the registered kinds. It only runs on the
// elected leader, so every change is recorded once.
type Recorder struct {
	informers cache.Informers
	scheme    *runtime.Scheme
	emitter   emitter
	kinds     []Kind
	lookback  time.Duration
	log       logr.Logger
	now       func() time.Time
}

// NewRecorder returns a Recorder for kinds. Add it to the manager with mgr.Add.
func NewRecorder(informers cache.Informers, scheme *runtime.Scheme, e emitter, kinds []Kind, log logr.Logger) *Recorder {
	return &Recorder{
		informers: informers,
		scheme:    scheme,
		emitter:   e,
		kinds:     kinds,
		lookback:  DefaultLookback,
		log:       log,
		now:       time.Now,
	}
}

// NeedLeaderElection implements manager.LeaderElectionRunnable.
func (r *Recorder) NeedLeaderElection() bool {
	return true
}

// Start registers the event handlers and blocks until ctx is done.
func (r *Recorder) Start(ctx context.Context) error {
	for _, kind := range r.kinds {
		informer, err := r.informers.GetInformer(ctx, kind.New())
		if err != nil {
			return fmt.Errorf("get informer for %T: %w", kind.New(), err)
		}
		if _, err := informer.AddEventHandler(r.handlerFor(ctx, kind)); err != nil {
			return fmt.Errorf("add event handler for %T: %w", kind.New(), err)
		}
	}
	<-ctx.Done()
	return nil
}

//nolint:ireturn // informers take client-go event handlers
func (r *Recorder) handlerFor(ctx context.Context, kind Kind) toolscache.ResourceEventHandler {
	return toolscache.ResourceEventHandlerDetailedFuncs{
		AddFunc: func(obj any, isInInitialList bool) {
			if o, ok := included(kind, obj); ok {
				r.onAdd(ctx, o, isInInitialList)
			}
		},
		UpdateFunc: func(oldObj, newObj any) {
			oldO, wasIncluded := included(kind, oldObj)
			newO, isIncluded := included(kind, newObj)
			switch {
			case wasIncluded && isIncluded:
				r.onUpdate(ctx, kind, oldO, newO)
			case isIncluded:
				// The object now matches Include, e.g. a namespace OpenEverest started
				// managing: it appears to OpenEverest.
				r.onAdd(ctx, newO, false)
			case wasIncluded:
				if o, ok := newObj.(client.Object); ok {
					r.recordDeleted(ctx, o)
				}
			}
		},
		DeleteFunc: func(obj any) {
			if tombstone, ok := obj.(toolscache.DeletedFinalStateUnknown); ok {
				obj = tombstone.Obj
			}
			if o, ok := included(kind, obj); ok {
				r.recordDeleted(ctx, o)
			}
		},
	}
}

//nolint:ireturn // informers deliver objects of any registered kind
func included(kind Kind, obj any) (client.Object, bool) {
	o, ok := obj.(client.Object)
	if !ok {
		return nil, false
	}
	if kind.Include != nil && !kind.Include(o) {
		return nil, false
	}
	return o, true
}

// onAdd records the creation of obj. The initial list of an informer replays every
// existing object as an addition; only objects created (or deleted) within the lookback
// window are recorded then, and the deterministic event names drop the ones already
// recorded.
func (r *Recorder) onAdd(ctx context.Context, obj client.Object, isInInitialList bool) {
	if !isInInitialList || r.recent(obj.GetCreationTimestamp().Time) {
		r.record(ctx, obj, events.Record{
			Reason: events.ReasonCreated,
			Note:   fmt.Sprintf("%s created", r.describe(obj)),
			Actor:  events.CreatorActor(obj),
		})
	}
	if deletion := obj.GetDeletionTimestamp(); isInInitialList && deletion != nil && r.recent(deletion.Time) {
		r.recordDeletionRequested(ctx, obj)
	}
}

func (r *Recorder) onUpdate(ctx context.Context, kind Kind, oldObj, newObj client.Object) {
	// Kubernetes also increments the generation when the deletion of an object starts,
	// which is not a change of its spec.
	if newObj.GetDeletionTimestamp() == nil &&
		oldObj.GetGeneration() > 0 && newObj.GetGeneration() > oldObj.GetGeneration() {
		r.record(ctx, newObj, events.Record{
			Reason:     events.ReasonUpdated,
			Note:       fmt.Sprintf("%s updated", r.describe(newObj)),
			Actor:      events.UpdaterActor(newObj),
			Generation: newObj.GetGeneration(),
			Key:        strconv.FormatInt(newObj.GetGeneration(), 10),
		})
	}
	if kind.Phase != nil {
		from, to := kind.Phase(oldObj), kind.Phase(newObj)
		if from != to && to != "" {
			r.record(ctx, newObj, events.Record{
				Reason:    events.ReasonPhaseChanged,
				Note:      fmt.Sprintf("%s phase changed from %q to %q", r.describe(newObj), from, to),
				Warning:   kind.Failed != nil && kind.Failed(to),
				Actor:     events.ControllerActor,
				PhaseFrom: from,
				PhaseTo:   to,
				Key:       newObj.GetResourceVersion(),
			})
		}
	}
	if oldObj.GetDeletionTimestamp() == nil && newObj.GetDeletionTimestamp() != nil {
		r.recordDeletionRequested(ctx, newObj)
	}
}

func (r *Recorder) recordDeleted(ctx context.Context, obj client.Object) {
	r.record(ctx, obj, events.Record{
		Reason: events.ReasonDeleted,
		Note:   fmt.Sprintf("%s deleted", r.describe(obj)),
		Actor:  events.DeleterActor(obj),
	})
}

func (r *Recorder) recordDeletionRequested(ctx context.Context, obj client.Object) {
	r.record(ctx, obj, events.Record{
		Reason: events.ReasonDeletionRequested,
		Note:   fmt.Sprintf("%s deletion requested", r.describe(obj)),
		Actor:  events.DeleterActor(obj),
	})
}

func (r *Recorder) record(ctx context.Context, obj client.Object, rec events.Record) {
	gvk, err := apiutil.GVKForObject(obj, r.scheme)
	if err != nil {
		r.log.Error(err, "Failed to determine the kind of an object", "object", client.ObjectKeyFromObject(obj))
		return
	}
	rec.Regarding = corev1.ObjectReference{
		APIVersion:      gvk.GroupVersion().String(),
		Kind:            gvk.Kind,
		Namespace:       obj.GetNamespace(),
		Name:            obj.GetName(),
		UID:             obj.GetUID(),
		ResourceVersion: obj.GetResourceVersion(),
	}
	if err := r.emitter.Emit(ctx, rec); err != nil {
		r.log.Error(err, "Failed to record a lifecycle event",
			"reason", rec.Reason, "kind", gvk.Kind, "object", client.ObjectKeyFromObject(obj),
		)
	}
}

func (r *Recorder) recent(t time.Time) bool {
	return r.now().Sub(t) <= r.lookback
}

func (r *Recorder) describe(obj client.Object) string {
	kind := fmt.Sprintf("%T", obj)
	if gvk, err := apiutil.GVKForObject(obj, r.scheme); err == nil {
		kind = gvk.Kind
	}
	if obj.GetNamespace() == "" {
		return kind + " " + obj.GetName()
	}
	return kind + " " + obj.GetNamespace() + "/" + obj.GetName()
}
