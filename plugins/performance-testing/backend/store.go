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

package main

import (
	"fmt"
	"sort"
	"sync"
)

// RunStore persists benchmark runs for the comparison-over-time outcome.
// This PoC ships only the in-memory implementation: results live for the
// life of the backend pod, ephemeral by default, matching the maintainers'
// explicit steer against ConfigMap/PVC-by-default storage in the #2464
// issue thread. A durable backend (SQLite, or an Everest-managed Postgres
// instance) is a separate, opt-in implementation of this same interface —
// not built here.
type RunStore interface {
	Save(r Run) error
	Get(id string) (Run, bool)
	List() []Run
}

// memoryStore is a RunStore backed by a map, guarded by a mutex since the
// HTTP handlers and the background Job-status poller both write to it.
type memoryStore struct {
	mu   sync.RWMutex
	runs map[string]Run
}

func newMemoryStore() *memoryStore {
	return &memoryStore{runs: make(map[string]Run)}
}

func (s *memoryStore) Save(r Run) error {
	if r.ID == "" {
		return fmt.Errorf("run ID must not be empty")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runs[r.ID] = r
	return nil
}

func (s *memoryStore) Get(id string) (Run, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.runs[id]
	return r, ok
}

// List returns all runs, most recently started first.
func (s *memoryStore) List() []Run {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Run, 0, len(s.runs))
	for _, r := range s.runs {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].StartedAt.After(out[j].StartedAt)
	})
	return out
}
