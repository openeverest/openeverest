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

package reconciler

import (
	"sync"

	"k8s.io/apimachinery/pkg/types"
)

// maintenanceBreakerThreshold is the number of consecutive failed Syncs after
// which an approved disruptive action stops being retried.
const maintenanceBreakerThreshold = 5

// maintenanceBreaker counts consecutive Sync failures per approved disruptive
// action so a crash-looping provider cannot repeatedly disrupt a database:
// once the threshold is reached the action is held despite its approval.
// Exponential backoff between the attempts themselves comes from
// controller-runtime's rate-limited requeue.
//
// State is in-memory: a controller restart re-arms the retries. The durable
// clear-stuck signal is the user changing spec.maintenance.approved, which
// resets the counts for the Instance.
type maintenanceBreaker struct {
	mu        sync.Mutex
	instances map[types.NamespacedName]*breakerEntry
}

type breakerEntry struct {
	// approved is spec.maintenance.approved as last observed; when the user
	// changes it the counts reset (the documented clear-stuck signal).
	approved string
	// failures counts consecutive failed Syncs per approved token.
	failures map[string]int
}

// blockedTokens returns the tokens whose retries are exhausted. approved is
// the Instance's current spec.maintenance.approved; a change since the last
// observation resets the counts.
func (b *maintenanceBreaker) blockedTokens(nn types.NamespacedName, approved string) []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	entry, ok := b.instances[nn]
	if !ok {
		return nil
	}
	if entry.approved != approved {
		delete(b.instances, nn)
		return nil
	}
	var blocked []string
	for token, count := range entry.failures {
		if count >= maintenanceBreakerThreshold {
			blocked = append(blocked, token)
		}
	}
	return blocked
}

// recordFailure counts a failed Sync against every disruptive action it had
// approved.
func (b *maintenanceBreaker) recordFailure(nn types.NamespacedName, approved string, tokens []string) {
	if len(tokens) == 0 {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.instances == nil {
		b.instances = make(map[types.NamespacedName]*breakerEntry)
	}
	entry, ok := b.instances[nn]
	if !ok || entry.approved != approved {
		entry = &breakerEntry{approved: approved, failures: make(map[string]int)}
		b.instances[nn] = entry
	}
	for _, token := range tokens {
		entry.failures[token]++
	}
}

// reset clears all counts for the Instance after a successful Sync.
func (b *maintenanceBreaker) reset(nn types.NamespacedName) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.instances, nn)
}
