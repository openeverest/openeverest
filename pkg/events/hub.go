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
}

// Hub fans-out normalised lifecycle events to connected subscribers.
// It watches Kubernetes resources through the provided KubernetesConnector.
type Hub struct {
	mu          sync.RWMutex
	subscribers map[*Subscriber]struct{}
	l           *zap.SugaredLogger
	kc          kubernetes.KubernetesConnector

	// prevState caches for normalisation — keyed by namespace/name.
	instanceCache map[string]*corev1alpha1.Instance
	backupCache   map[string]*backupv1alpha1.Backup
	restoreCache  map[string]*backupv1alpha1.Restore
	ieCache       map[string]*extensionsv1alpha1.InstalledExtension
	cacheMu       sync.RWMutex
}

// NewHub creates a new event hub.
func NewHub(l *zap.SugaredLogger, kc kubernetes.KubernetesConnector) *Hub {
	return &Hub{
		subscribers:   make(map[*Subscriber]struct{}),
		l:             l.With("component", "event-hub"),
		kc:            kc,
		instanceCache: make(map[string]*corev1alpha1.Instance),
		backupCache:   make(map[string]*backupv1alpha1.Backup),
		restoreCache:  make(map[string]*backupv1alpha1.Restore),
		ieCache:       make(map[string]*extensionsv1alpha1.InstalledExtension),
	}
}

// Subscribe creates a subscriber with optional type and namespace filters.
// Returns the subscriber's event channel and a cancel function.
func (h *Hub) Subscribe(types []Type, namespaces []string) (<-chan Event, func()) {
	sub := &Subscriber{
		ch:         make(chan Event, defaultBufferSize),
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
	h.subscribers[sub] = struct{}{}
	h.mu.Unlock()

	cancel := func() {
		h.mu.Lock()
		delete(h.subscribers, sub)
		h.mu.Unlock()
		close(sub.ch)
	}
	return sub.ch, cancel
}

// broadcast sends an event to all matching subscribers.
// Slow subscribers whose buffers are full are dropped.
func (h *Hub) broadcast(evt Event) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for sub := range h.subscribers {
		if !sub.matches(evt) {
			continue
		}
		select {
		case sub.ch <- evt:
		default:
			// Buffer full — drop this subscriber (design doc §8.5).
			h.l.Warnf("Dropping slow subscriber (buffer full), event %s", evt.Type)
			go func(s *Subscriber) {
				h.mu.Lock()
				delete(h.subscribers, s)
				h.mu.Unlock()
				close(s.ch)
			}(sub)
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
