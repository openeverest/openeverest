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
	"sync"
	"time"

	"github.com/google/uuid"
)

// Buffer is a bounded, replayable log of recently published events, keyed
// by the Hub-assigned monotonic Seq. It backs the Hub's replay path
// (openeverest#2582) and is defined as an interface so the backing store
// can move off-process later (e.g. valkey/redis, shared across replicas)
// without changing the Hub or the wire contract. Only an in-memory
// implementation exists today; see NewRingBuffer.
type Buffer interface {
	// Append records ev. ev.Seq must already be set. Older entries are
	// evicted per the buffer's own count/age bounds, oldest first.
	Append(ev *Event)

	// Since returns every buffered event with Seq greater than seq, oldest
	// first. ok is false when seq can't be safely served: it's older than
	// everything currently retained (aged out), or newer than anything the
	// buffer has seen (can't be verified). Either way the caller must
	// signal the client to resync rather than guess.
	Since(seq uint64) (events []Event, ok bool)

	// Epoch identifies this buffer instance. It's generated once, at
	// construction, so a cursor from a previous process — or a previous
	// buffer instance in this one — is never mistaken for a valid one.
	Epoch() string
}

// ringBuffer is the in-memory, single-process default Buffer. Bounded by
// both a maximum entry count and a maximum age; whichever limit is hit
// first evicts the oldest entries. Safe for concurrent use.
type ringBuffer struct {
	mu     sync.Mutex
	epoch  string
	maxLen int
	maxAge time.Duration

	// entries is ordered oldest-first by Seq. Ages are tracked separately
	// from Event.OccurredAt (which is domain data - a resource's own
	// transition time - not necessarily close to wall-clock "now") so
	// buffer retention is driven purely by how long an event has actually
	// sat in the buffer.
	entries []bufferedEvent
}

type bufferedEvent struct {
	event      Event
	bufferedAt time.Time
}

const (
	// defaultMaxLen and defaultMaxAge are used when NewRingBuffer is given a
	// non-positive value for either bound. Treating <= 0 as "unbounded"
	// would let the buffer grow without limit on a simple misconfiguration
	// (e.g. an empty/zero env var) - a memory-growth risk, not a graceful
	// degradation - so it falls back to a sane bound instead. These match
	// EverestConfig's own defaults for EventBufferMaxEvents/MaxAge.
	defaultMaxLen = 4096
	defaultMaxAge = 5 * time.Minute
)

// NewRingBuffer creates an in-memory Buffer. maxLen and maxAge both bound
// retention; whichever is reached first evicts the oldest entry. A
// non-positive maxLen or maxAge is replaced with a default rather than
// treated as unbounded - see defaultMaxLen/defaultMaxAge.
//
//nolint:ireturn // the whole point of Buffer is a swappable backing store (see the interface doc) - concrete would defeat it.
func NewRingBuffer(maxLen int, maxAge time.Duration) Buffer {
	if maxLen <= 0 {
		maxLen = defaultMaxLen
	}
	if maxAge <= 0 {
		maxAge = defaultMaxAge
	}
	return &ringBuffer{
		epoch:  uuid.NewString(),
		maxLen: maxLen,
		maxAge: maxAge,
	}
}

func (b *ringBuffer) Epoch() string { return b.epoch }

func (b *ringBuffer) Append(ev *Event) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.entries = append(b.entries, bufferedEvent{event: *ev, bufferedAt: time.Now()})
	b.evictLocked()
}

func (b *ringBuffer) Since(seq uint64) ([]Event, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.entries) == 0 {
		// Nothing retained. seq==0 ("I've never seen an event") is the only
		// value that's trivially satisfiable against an empty buffer - there's
		// an empty replay and no gap. Anything else can't be verified: either
		// it aged out, or it's ahead of what this buffer instance has ever
		// held (shouldn't happen if the epoch already matched, but treated
		// the same way regardless: signal resync, don't guess).
		return nil, seq == 0
	}

	// entries[0].Seq is always >= 1 (Hub assigns Seq by pre-incrementing
	// from 0), so this never underflows.
	oldestExclusive := b.entries[0].event.Seq - 1
	newest := b.entries[len(b.entries)-1].event.Seq

	if seq < oldestExclusive || seq > newest {
		return nil, false
	}

	out := make([]Event, 0, len(b.entries))
	for _, be := range b.entries {
		if be.event.Seq > seq {
			out = append(out, be.event)
		}
	}
	return out, true
}

// evictLocked drops entries from the front while either bound is exceeded.
// Must be called with mu held. maxLen and maxAge are always positive by the
// time a ringBuffer exists - NewRingBuffer clamps non-positive input to a
// default rather than leaving either dimension unbounded.
func (b *ringBuffer) evictLocked() {
	cutoff := time.Now().Add(-b.maxAge)
	i := 0
	for i < len(b.entries) && (len(b.entries)-i > b.maxLen || b.entries[i].bufferedAt.Before(cutoff)) {
		i++
	}
	if i > 0 {
		b.entries = b.entries[i:]
	}
}
