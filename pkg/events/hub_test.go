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
	"testing"
	"time"

	"go.uber.org/zap"
)

func newTestHub(maxLen int) *Hub {
	return NewHub(zap.NewNop().Sugar(), nil, NewRingBuffer(maxLen, time.Minute))
}

func TestHub_SubscribeNilCursorIsLiveOnly(t *testing.T) {
	t.Parallel()
	h := newTestHub(100)

	h.Publish(Event{Type: UserLogin}) // published before Subscribe - must not appear

	sub, ok := h.Subscribe(nil, nil, nil)
	if !ok {
		t.Fatal("Subscribe with nil cursor must always succeed")
	}
	defer sub.Cancel()

	if len(sub.Replay) != 0 {
		t.Fatalf("nil cursor must produce an empty replay, got %d events", len(sub.Replay))
	}

	h.Publish(Event{Type: UserLogout})

	select {
	case evt := <-sub.Ch:
		if evt.Type != UserLogout {
			t.Fatalf("expected the post-subscribe event, got %s", evt.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for live event")
	}
}

func TestHub_SubscribeCursorReplaysGap(t *testing.T) {
	t.Parallel()
	h := newTestHub(100)

	h.Publish(Event{Type: InstanceCreated, Resource: ResourceRef{Name: "a"}})
	h.Publish(Event{Type: InstanceCreated, Resource: ResourceRef{Name: "b"}})
	h.Publish(Event{Type: InstanceCreated, Resource: ResourceRef{Name: "c"}})

	// Simulate: a plugin saw event "a" (seq 1), then disconnected before b/c
	// were published, and now reconnects with that cursor.
	cursor := &Cursor{Epoch: h.Epoch(), Seq: 1}

	sub, ok := h.Subscribe(nil, nil, cursor)
	if !ok {
		t.Fatal("expected a valid cursor to succeed")
	}
	defer sub.Cancel()

	if len(sub.Replay) != 2 {
		t.Fatalf("expected 2 replayed events (b, c), got %d", len(sub.Replay))
	}
	if sub.Replay[0].Resource.Name != "b" || sub.Replay[1].Resource.Name != "c" {
		t.Fatalf("expected replay order b,c - got %s,%s", sub.Replay[0].Resource.Name, sub.Replay[1].Resource.Name)
	}

	h.Publish(Event{Type: InstanceCreated, Resource: ResourceRef{Name: "d"}})
	select {
	case evt := <-sub.Ch:
		if evt.Resource.Name != "d" {
			t.Fatalf("expected live event 'd', got %q", evt.Resource.Name)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for live event after replay")
	}
}

func TestHub_SubscribeEpochMismatchIs410(t *testing.T) {
	t.Parallel()
	h := newTestHub(100)
	h.Publish(Event{Type: UserLogin})

	_, ok := h.Subscribe(nil, nil, &Cursor{Epoch: "some-other-process-entirely", Seq: 1})
	if ok {
		t.Fatal("a cursor from a different epoch must be rejected")
	}
}

func TestHub_SubscribeSeqAgedOutIs410(t *testing.T) {
	t.Parallel()
	h := newTestHub(2) // tiny buffer, easy to age out by count

	for range 5 {
		h.Publish(Event{Type: UserLogin})
	}
	// Only the last 2 seqs (4, 5) are retained now.

	_, ok := h.Subscribe(nil, nil, &Cursor{Epoch: h.Epoch(), Seq: 1})
	if ok {
		t.Fatal("a seq older than everything retained must be rejected (410)")
	}
}

func TestHub_SubscribeFiltersApplyToReplayAndLiveIdentically(t *testing.T) {
	t.Parallel()
	h := newTestHub(100)

	h.Publish(Event{Type: UserLogin, Namespace: "team-a"})
	h.Publish(Event{Type: UserLogout, Namespace: "team-a"})
	h.Publish(Event{Type: UserLogin, Namespace: "team-b"})

	cursor := &Cursor{Epoch: h.Epoch(), Seq: 0}
	sub, ok := h.Subscribe([]Type{UserLogin}, []string{"team-a"}, cursor)
	if !ok {
		t.Fatal("expected valid subscribe")
	}
	defer sub.Cancel()

	if len(sub.Replay) != 1 {
		t.Fatalf("expected exactly 1 replayed event matching type+namespace filter, got %d", len(sub.Replay))
	}
	if sub.Replay[0].Namespace != "team-a" || sub.Replay[0].Type != UserLogin {
		t.Fatalf("unexpected replayed event: %+v", sub.Replay[0])
	}

	// A live event that doesn't match the filter must not arrive.
	h.Publish(Event{Type: UserLogout, Namespace: "team-a"})
	h.Publish(Event{Type: UserLogin, Namespace: "team-a"}) // matches - should arrive

	select {
	case evt := <-sub.Ch:
		if evt.Namespace != "team-a" || evt.Type != UserLogin {
			t.Fatalf("expected the matching live event, got %+v", evt)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for the matching live event")
	}

	select {
	case evt := <-sub.Ch:
		t.Fatalf("unexpected extra event delivered (filter should have excluded it): %+v", evt)
	case <-time.After(50 * time.Millisecond):
		// expected: nothing else should arrive
	}
}

func TestHub_SlowSubscriberIsDroppedThenResyncs(t *testing.T) {
	t.Parallel()
	h := newTestHub(1000)

	sub, ok := h.Subscribe(nil, nil, nil)
	if !ok {
		t.Fatal("expected valid subscribe")
	}

	// Fill the subscriber's own channel buffer (defaultBufferSize) without
	// draining it, then publish one more to trigger the drop.
	for range defaultBufferSize + 1 {
		h.Publish(Event{Type: UserLogin})
	}

	select {
	case <-sub.Dropped:
		// expected
	case <-time.After(time.Second):
		t.Fatal("expected the slow subscriber to be dropped")
	}

	// The dropped subscriber's own cursor (wherever it last got to) can
	// still resync from the buffer - that's the whole point of #2582.
	_, ok = h.Subscribe(nil, nil, &Cursor{Epoch: h.Epoch(), Seq: 1})
	if !ok {
		t.Fatal("expected the post-drop resync to succeed since the buffer still retains recent history")
	}
}

// TestHub_ConcurrentPublishReconnectNoGapsNoDuplicates is the core
// correctness property this PR exists for: with multiple publishers
// running concurrently and a consumer that repeatedly disconnects and
// reconnects with its last-seen cursor, every event must be seen exactly
// once. Run with -race.
func TestHub_ConcurrentPublishReconnectNoGapsNoDuplicates(t *testing.T) {
	t.Parallel()
	h := newTestHub(10000)

	const publishers = 4
	const perPublisher = 250
	const total = publishers * perPublisher

	var wg sync.WaitGroup
	for range publishers {
		wg.Go(func() {
			for range perPublisher {
				h.Publish(Event{Type: SettingsUpdated})
			}
		})
	}

	seen := make(map[uint64]bool, total)
	recordSeq := func(seq uint64) {
		if seen[seq] {
			t.Fatalf("duplicate delivery of seq %d", seq)
		}
		seen[seq] = true
	}

	// A handful of genuine reconnect cycles while publishers are still
	// actively running, each blocking on live events for a bounded window.
	// This is deliberately not a tight reconnect spin-loop: hammering
	// Subscribe/Cancel as fast as possible starves the publisher goroutines
	// of scheduler time (worse under -race, which makes each lock
	// acquisition heavier) without adding anything to what's being tested -
	// the race this test exists to catch is in the boundary between one
	// subscription's Replay and its Ch, not in reconnect frequency.
	// Start from an explicit "seen nothing yet" cursor rather than nil -
	// nil means live-only by design (see Subscribe's doc comment), which
	// would legitimately, correctly miss anything published in the race
	// between the goroutines above starting and this loop's first
	// Subscribe call. An explicit Seq:0 cursor asks for full replay from
	// the beginning instead, which is what actually proves no-loss here.
	cursor := &Cursor{Epoch: h.Epoch(), Seq: 0}
	const reconnects = 5
	for range reconnects {
		sub, ok := h.Subscribe(nil, nil, cursor)
		if !ok {
			t.Fatalf("unexpected resync-required with cursor %+v", cursor)
		}
		for _, evt := range sub.Replay {
			recordSeq(evt.Seq)
			c := Cursor{Epoch: evt.Epoch, Seq: evt.Seq}
			cursor = &c
		}

		readWindow := time.After(50 * time.Millisecond)
	drain:
		for {
			select {
			case evt := <-sub.Ch:
				recordSeq(evt.Seq)
				c := Cursor{Epoch: evt.Epoch, Seq: evt.Seq}
				cursor = &c
			case <-readWindow:
				break drain
			}
		}
		sub.Cancel()
	}

	wg.Wait()

	// Final subscribe picks up everything published after the last
	// reconnect window closed - the buffer (maxLen 10000) comfortably holds
	// all of it, so nothing can be structurally missed here regardless of
	// timing above.
	sub, ok := h.Subscribe(nil, nil, cursor)
	if !ok {
		t.Fatalf("unexpected resync-required on final subscribe, cursor %+v", cursor)
	}
	for _, evt := range sub.Replay {
		recordSeq(evt.Seq)
	}
	sub.Cancel()

	if len(seen) != total {
		var missing []uint64
		for i := uint64(1); i <= uint64(total); i++ {
			if !seen[i] {
				missing = append(missing, i)
			}
		}
		t.Fatalf("expected exactly %d distinct events seen, got %d - missing: %v", total, len(seen), missing)
	}
}
