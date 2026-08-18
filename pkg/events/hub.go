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
	"sync"
	"time"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	extensionsv1alpha1 "github.com/openeverest/openeverest/v2/api/extensions/v1alpha1"
	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
)

const (
	// defaultBufferSize is the per-subscriber buffer. Slow consumers are dropped.
	defaultBufferSize = 256

	// watchRetryDelay is the delay before reconnecting a closed/failed watch.
	watchRetryDelay = 2 * time.Second
)

// Subscriber receives events matching its filter criteria.
type Subscriber struct {
	ch         chan Event
	types      map[Type]struct{}
	namespaces map[string]struct{}

	// done signals subscriber termination — either the caller cancelled, or
	// broadcast dropped it for being too slow. It's closed exactly once via
	// closeOnce.
	//
	// ch itself is never closed. broadcast takes a point-in-time snapshot
	// of subscribers, releases h.mu, then sends — so by the time a send is
	// attempted, this subscriber may already be mid-teardown on another
	// goroutine (Cancel, or a *different* broadcast call dropping it for a
	// full buffer). Closing ch directly from that teardown could race with
	// a send already in flight and panic ("send on closed channel"). done
	// sidesteps that: closing it is always safe from any goroutine, any
	// number of times guarded by closeOnce, and a send that loses the race
	// to a close either succeeds harmlessly into an abandoned channel or is
	// skipped via the <-done case below.
	done      chan struct{}
	closeOnce sync.Once
}

func (s *Subscriber) close() {
	s.closeOnce.Do(func() { close(s.done) })
}

// Subscription is the result of a successful Hub.Subscribe call.
type Subscription struct {
	// Replay holds buffered events with Seq greater than the requested
	// cursor, oldest first. Empty when no cursor was given.
	Replay []Event
	// Ch streams events published after Subscribe returned. The boundary
	// with Replay is exact — see Subscribe's comment — so the caller can
	// simply deliver Replay first, then drain Ch, with no gap or overlap.
	// Never closed - select on Dropped (or your own context) to know when
	// to stop reading.
	Ch <-chan Event
	// Dropped is closed when the subscriber is torn down - either Cancel
	// was called, or broadcast dropped it for falling too far behind. Once
	// closed, no further events will arrive on Ch.
	Dropped <-chan struct{}
	// Cancel unregisters the subscriber and closes Dropped. Always call it,
	// typically via defer.
	Cancel func()
}

// Hub fans-out normalised lifecycle events to connected subscribers.
// It watches Kubernetes resources through the provided KubernetesConnector.
type Hub struct {
	mu          sync.Mutex
	subscribers map[*Subscriber]struct{}
	seq         uint64
	buffer      Buffer
	l           *zap.SugaredLogger
	kc          kubernetes.KubernetesConnector

	// prevState caches for normalisation — keyed by namespace/name.
	instanceCache map[string]*corev1alpha1.Instance
	backupCache   map[string]*backupv1alpha1.Backup
	restoreCache  map[string]*backupv1alpha1.Restore
	ieCache       map[string]*extensionsv1alpha1.InstalledExtension
	cacheMu       sync.RWMutex
}

// NewHub creates a new event hub backed by buf for replay (openeverest#2582).
// Pass NewRingBuffer(...) for the in-memory default.
func NewHub(l *zap.SugaredLogger, kc kubernetes.KubernetesConnector, buf Buffer) *Hub {
	return &Hub{
		subscribers:   make(map[*Subscriber]struct{}),
		buffer:        buf,
		l:             l.With("component", "event-hub"),
		kc:            kc,
		instanceCache: make(map[string]*corev1alpha1.Instance),
		backupCache:   make(map[string]*backupv1alpha1.Backup),
		restoreCache:  make(map[string]*backupv1alpha1.Restore),
		ieCache:       make(map[string]*extensionsv1alpha1.InstalledExtension),
	}
}

// Epoch identifies this Hub's buffer instance — see Buffer.Epoch.
func (h *Hub) Epoch() string { return h.buffer.Epoch() }

// Subscribe registers a subscriber with optional type/namespace filters and
// an optional replay cursor.
//
// If cursor is nil, behaviour matches the pre-replay Hub exactly: only
// events published after this call are delivered, via Subscription.Ch.
//
// If cursor is non-nil, ok is false when the request can't be safely
// resumed — its Epoch doesn't match this Hub's (the process restarted), or
// its Seq has aged out of (or is ahead of) the buffer. The caller must
// treat that as "resync required": relist current state and reconnect
// without a cursor (spec 003 §10.6), same as any other 410.
//
// Race safety: the buffer snapshot (Subscription.Replay) and the
// subscriber's registration happen under one acquisition of h.mu, which is
// the same lock broadcast holds while it appends to the buffer and decides
// who receives an event live. That makes the two mutually exclusive with
// broadcast, not just individually atomic:
//   - Any event broadcast strictly before this call's critical section was
//     already appended to the buffer when Replay is read, so it's in Replay
//     — and this subscriber wasn't registered yet when that broadcast chose
//     its recipients, so it won't also arrive on Ch. In Replay, not on Ch.
//   - Any event broadcast strictly after this call's critical section finds
//     the subscriber already registered (broadcast can't run concurrently
//     with this method — same lock), so it's delivered on Ch — and it
//     wasn't in the buffer yet when Replay was taken, so it's not
//     duplicated there. On Ch, not in Replay.
//
// No interleaving is possible between those two cases, because broadcast's
// append+recipient-selection and this method's snapshot+registration both
// hold h.mu for their entire critical section. The boundary is exact by
// construction, not by chance — there is no seq for which both, or
// neither, path could deliver it.
func (h *Hub) Subscribe(types []Type, namespaces []string, cursor *Cursor) (Subscription, bool) {
	sub := &Subscriber{
		ch:         make(chan Event, defaultBufferSize),
		done:       make(chan struct{}),
		types:      make(map[Type]struct{}, len(types)),
		namespaces: make(map[string]struct{}, len(namespaces)),
	}
	for _, t := range types {
		sub.types[t] = struct{}{}
	}
	for _, ns := range namespaces {
		sub.namespaces[ns] = struct{}{}
	}

	h.mu.Lock()

	var replay []Event
	if cursor != nil {
		if cursor.Epoch != h.buffer.Epoch() {
			h.mu.Unlock()
			return Subscription{}, false
		}
		evs, ok := h.buffer.Since(cursor.Seq)
		if !ok {
			h.mu.Unlock()
			return Subscription{}, false
		}
		for _, evt := range evs {
			if sub.matches(evt) {
				replay = append(replay, evt)
			}
		}
	}

	h.subscribers[sub] = struct{}{}
	h.mu.Unlock()

	cancel := func() {
		h.mu.Lock()
		delete(h.subscribers, sub)
		h.mu.Unlock()
		sub.close()
	}
	return Subscription{Replay: replay, Ch: sub.ch, Dropped: sub.done, Cancel: cancel}, true
}

// broadcast assigns evt its Seq/Epoch, appends it to the buffer, and
// fans it out to matching subscribers. Slow subscribers whose channel is
// full are dropped — safe now that a dropped subscriber can reconnect with
// a cursor and replay the gap instead of losing it permanently.
func (h *Hub) broadcast(evt Event) {
	h.mu.Lock()
	h.seq++
	evt.Seq = h.seq
	evt.Epoch = h.buffer.Epoch()
	h.buffer.Append(&evt)

	// Snapshot the current subscriber set under the same critical section
	// that just appended evt — this is the other half of the race-safety
	// argument in Subscribe's comment. Then release before sending: a send
	// can block a goroutine on a full channel's non-blocking select for a
	// moment, but must never hold h.mu while doing it, or a slow subscriber
	// would stall seq assignment and registration for everyone else.
	subs := make([]*Subscriber, 0, len(h.subscribers))
	for sub := range h.subscribers {
		subs = append(subs, sub)
	}
	h.mu.Unlock()

	var toDrop []*Subscriber
	for _, sub := range subs {
		if !sub.matches(evt) {
			continue
		}
		select {
		case sub.ch <- evt:
		case <-sub.done:
			// Already torn down by Cancel or a concurrent drop - nothing to
			// deliver to.
		default:
			h.l.Warnf("dropping slow subscriber (buffer full), event %s", evt.Type)
			toDrop = append(toDrop, sub)
		}
	}

	if len(toDrop) > 0 {
		h.mu.Lock()
		for _, sub := range toDrop {
			delete(h.subscribers, sub)
		}
		h.mu.Unlock()
		for _, sub := range toDrop {
			sub.close()
		}
	}
}

func (s *Subscriber) matches(evt Event) bool {
	if len(s.types) > 0 {
		if _, ok := s.types[evt.Type]; !ok {
			return false
		}
	}
	if len(s.namespaces) > 0 {
		if _, ok := s.namespaces[evt.Namespace]; !ok {
			return false
		}
	}
	return true
}

// cacheKey returns a unique key for a namespaced resource.
func cacheKey(namespace, name string) string {
	return namespace + "/" + name
}

// Start begins watching Kubernetes resources and broadcasting events.
// It blocks until ctx is cancelled. Must be called in a goroutine.
// Individual watch failures are logged and retried automatically.
func (h *Hub) Start(ctx context.Context) error {
	watchers := []struct {
		name string
		fn   func(context.Context) error
	}{
		{"Backups", h.watchBackups},
		{"Restores", h.watchRestores},
		{"Instances", h.watchInstances},
		{"Plugins", h.watchPlugins},
		{"InstalledExtensions", h.watchInstalledExtensions},
		{"Namespaces", h.watchNamespaces},
		{"EverestSettings", h.watchEverestSettings},
	}

	for _, w := range watchers {
		go h.watchWithRetry(ctx, w.name, w.fn)
	}

	<-ctx.Done()
	return ctx.Err()
}

// Publish broadcasts an event from a non-watch source (e.g. an API handler
// for session create/delete) into the same fan-out pipeline. Spec §10.5
// "Direct publish".
func (h *Hub) Publish(evt Event) {
	if evt.OccurredAt.IsZero() {
		evt.OccurredAt = time.Now().UTC()
	}
	h.broadcast(evt)
}

// watchWithRetry runs a watch function in a loop, reconnecting on close/error.
func (h *Hub) watchWithRetry(ctx context.Context, name string, fn func(context.Context) error) {
	for {
		h.l.Infof("starting watch: %s", name)
		err := fn(ctx)
		if ctx.Err() != nil {
			return // context cancelled, shutting down
		}
		if err != nil {
			h.l.Warnf("watch %s failed: %v — retrying in %s", name, err, watchRetryDelay)
		} else {
			h.l.Infof("watch %s closed — reconnecting in %s", name, watchRetryDelay)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(watchRetryDelay):
		}
	}
}

func (h *Hub) watchBackups(ctx context.Context) error {
	watcher, err := h.kc.WatchBackups(ctx)
	if err != nil {
		return err
	}
	defer watcher.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case we, ok := <-watcher.ResultChan():
			if !ok {
				return nil
			}
			obj, ok := we.Object.(*backupv1alpha1.Backup)
			if !ok {
				continue
			}
			key := cacheKey(obj.Namespace, obj.Name)

			h.cacheMu.RLock()
			old := h.backupCache[key]
			h.cacheMu.RUnlock()

			events := NormalizeBackup(we, old)
			for _, evt := range events {
				h.broadcast(evt)
			}

			h.cacheMu.Lock()
			if we.Type == watch.Deleted {
				delete(h.backupCache, key)
			} else {
				h.backupCache[key] = obj.DeepCopy()
			}
			h.cacheMu.Unlock()
		}
	}
}

func (h *Hub) watchRestores(ctx context.Context) error {
	watcher, err := h.kc.WatchRestores(ctx)
	if err != nil {
		return err
	}
	defer watcher.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case we, ok := <-watcher.ResultChan():
			if !ok {
				return nil
			}
			obj, ok := we.Object.(*backupv1alpha1.Restore)
			if !ok {
				continue
			}
			key := cacheKey(obj.Namespace, obj.Name)

			h.cacheMu.RLock()
			old := h.restoreCache[key]
			h.cacheMu.RUnlock()

			events := NormalizeRestore(we, old)
			for _, evt := range events {
				h.broadcast(evt)
			}

			h.cacheMu.Lock()
			if we.Type == watch.Deleted {
				delete(h.restoreCache, key)
			} else {
				h.restoreCache[key] = obj.DeepCopy()
			}
			h.cacheMu.Unlock()
		}
	}
}

func (h *Hub) watchInstances(ctx context.Context) error {
	watcher, err := h.kc.WatchInstances(ctx)
	if err != nil {
		return err
	}
	defer watcher.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case we, ok := <-watcher.ResultChan():
			if !ok {
				return nil
			}
			h.l.Debugf("instance watch event: type=%s", we.Type)
			events := NormalizeInstance(we)
			for _, evt := range events {
				h.l.Infof("broadcasting event: %s %s/%s", evt.Type, evt.Namespace, evt.Resource.Name)
				h.broadcast(evt)
			}
		}
	}
}

func (h *Hub) watchPlugins(ctx context.Context) error {
	watcher, err := h.kc.WatchPlugins(ctx)
	if err != nil {
		return err
	}
	defer watcher.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case we, ok := <-watcher.ResultChan():
			if !ok {
				return nil
			}
			for _, evt := range NormalizePlugin(we) {
				h.broadcast(evt)
			}
		}
	}
}

func (h *Hub) watchInstalledExtensions(ctx context.Context) error {
	watcher, err := h.kc.WatchInstalledExtensions(ctx)
	if err != nil {
		return err
	}
	defer watcher.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case we, ok := <-watcher.ResultChan():
			if !ok {
				return nil
			}
			obj, ok := we.Object.(*extensionsv1alpha1.InstalledExtension)
			if !ok {
				continue
			}
			key := cacheKey(obj.Namespace, obj.Name)

			h.cacheMu.RLock()
			old := h.ieCache[key]
			h.cacheMu.RUnlock()

			for _, evt := range NormalizeInstalledExtension(we, old) {
				h.broadcast(evt)
			}

			h.cacheMu.Lock()
			if we.Type == watch.Deleted {
				delete(h.ieCache, key)
			} else {
				h.ieCache[key] = obj.DeepCopy()
			}
			h.cacheMu.Unlock()
		}
	}
}

func (h *Hub) watchNamespaces(ctx context.Context) error {
	watcher, err := h.kc.WatchEverestManagedNamespaces(ctx)
	if err != nil {
		return err
	}
	defer watcher.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case we, ok := <-watcher.ResultChan():
			if !ok {
				return nil
			}
			if _, ok := we.Object.(*corev1.Namespace); !ok {
				continue
			}
			for _, evt := range NormalizeNamespace(we) {
				h.broadcast(evt)
			}
		}
	}
}

func (h *Hub) watchEverestSettings(ctx context.Context) error {
	watcher, err := h.kc.WatchEverestSettings(ctx)
	if err != nil {
		return err
	}
	defer watcher.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case we, ok := <-watcher.ResultChan():
			if !ok {
				return nil
			}
			for _, evt := range NormalizeEverestSettings(we) {
				h.broadcast(evt)
			}
		}
	}
}
