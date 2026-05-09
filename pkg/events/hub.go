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

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/openeverest/openeverest/v2/pkg/kubernetes"
)

const (
	// defaultBufferSize is the per-subscriber buffer. Slow consumers are dropped.
	defaultBufferSize = 256
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
	clusterCache map[string]*everestv1alpha1.DatabaseCluster
	backupCache  map[string]*everestv1alpha1.DatabaseClusterBackup
	restoreCache map[string]*everestv1alpha1.DatabaseClusterRestore
	cacheMu      sync.RWMutex
}

// NewHub creates a new event hub.
func NewHub(l *zap.SugaredLogger, kc kubernetes.KubernetesConnector) *Hub {
	return &Hub{
		subscribers:  make(map[*Subscriber]struct{}),
		l:            l.With("component", "event-hub"),
		kc:           kc,
		clusterCache: make(map[string]*everestv1alpha1.DatabaseCluster),
		backupCache:  make(map[string]*everestv1alpha1.DatabaseClusterBackup),
		restoreCache: make(map[string]*everestv1alpha1.DatabaseClusterRestore),
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
func (h *Hub) Start(ctx context.Context) error {
	errCh := make(chan error, 4) //nolint:mnd

	go func() { errCh <- h.watchDatabaseClusters(ctx) }()
	go func() { errCh <- h.watchBackups(ctx) }()
	go func() { errCh <- h.watchRestores(ctx) }()
	go func() { errCh <- h.watchInstances(ctx) }()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		return err
	}
}

func (h *Hub) watchDatabaseClusters(ctx context.Context) error {
	watcher, err := h.kc.WatchDatabaseClusters(ctx)
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
				return nil // channel closed, reconnect handled by caller
			}
			obj, ok := we.Object.(*everestv1alpha1.DatabaseCluster)
			if !ok {
				continue
			}
			key := cacheKey(obj.Namespace, obj.Name)

			h.cacheMu.RLock()
			old := h.clusterCache[key]
			h.cacheMu.RUnlock()

			events := NormalizeDatabaseCluster(we, old)
			for _, evt := range events {
				h.broadcast(evt)
			}

			// Update cache.
			h.cacheMu.Lock()
			if we.Type == watch.Deleted {
				delete(h.clusterCache, key)
			} else {
				h.clusterCache[key] = obj.DeepCopy()
			}
			h.cacheMu.Unlock()
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
			obj, ok := we.Object.(*everestv1alpha1.DatabaseClusterBackup)
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
			obj, ok := we.Object.(*everestv1alpha1.DatabaseClusterRestore)
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
			events := NormalizeInstance(we)
			for _, evt := range events {
				h.broadcast(evt)
			}
		}
	}
}
