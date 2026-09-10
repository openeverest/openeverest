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
	"testing"
	"time"
)

func TestRingBuffer_EpochIsStableAndUnique(t *testing.T) {
	t.Parallel()
	b1 := NewRingBuffer(10, time.Minute)
	b2 := NewRingBuffer(10, time.Minute)

	first := b1.Epoch()
	second := b1.Epoch()
	if first == "" {
		t.Fatal("epoch must not be empty")
	}
	if first != second {
		t.Fatal("epoch must be stable across calls")
	}
	if b1.Epoch() == b2.Epoch() {
		t.Fatal("two independently constructed buffers must not share an epoch")
	}
}

func TestRingBuffer_AppendAndSinceBasic(t *testing.T) {
	t.Parallel()
	b := NewRingBuffer(10, time.Minute)

	for i := uint64(1); i <= 5; i++ {
		b.Append(&Event{Seq: i, Type: InstanceCreated})
	}

	got, ok := b.Since(0)
	if !ok {
		t.Fatal("Since(0) should succeed when nothing has aged out")
	}
	if len(got) != 5 {
		t.Fatalf("expected 5 events since seq 0, got %d", len(got))
	}
	for i, evt := range got {
		if evt.Seq != uint64(i+1) {
			t.Fatalf("expected events in order starting at 1, got seq %d at index %d", evt.Seq, i)
		}
	}

	got, ok = b.Since(3)
	if !ok {
		t.Fatal("Since(3) should succeed")
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 events since seq 3, got %d", len(got))
	}
	if got[0].Seq != 4 || got[1].Seq != 5 {
		t.Fatalf("expected seq 4,5 - got %d,%d", got[0].Seq, got[1].Seq)
	}

	got, ok = b.Since(5)
	if !ok {
		t.Fatal("Since(5) (fully caught up) should succeed with an empty replay")
	}
	if len(got) != 0 {
		t.Fatalf("expected 0 events since the newest seq, got %d", len(got))
	}
}

func TestRingBuffer_SinceEmptyBuffer(t *testing.T) {
	t.Parallel()
	b := NewRingBuffer(10, time.Minute)

	if _, ok := b.Since(0); !ok {
		t.Fatal("Since(0) on an empty buffer is trivially satisfiable (nothing missed)")
	}
	if _, ok := b.Since(1); ok {
		t.Fatal("Since(1) on an empty buffer can't be verified - must signal resync")
	}
}

// TestRingBuffer_SinceBoundaryConditions pins down all four boundary cases
// against a single buffer state (seq 3,4,5 retained, 1,2 evicted by count) -
// this exact off-by-one is the easiest place for a bug to hide here, so
// every case lives in one test against one fixed buffer state rather than
// scattered across separately-configured tests.
func TestRingBuffer_SinceBoundaryConditions(t *testing.T) {
	t.Parallel()
	b := NewRingBuffer(3, time.Minute)

	for i := uint64(1); i <= 5; i++ {
		b.Append(&Event{Seq: i})
	}
	// maxLen=3 means only seq 3,4,5 are retained; 1,2 evicted by count.

	// seq 1: resuming from here would need seq 2 too, which is gone -
	// unrecoverable gap.
	if _, ok := b.Since(1); ok {
		t.Fatal("seq 1 needs seq 2 to resume safely, and seq 2 was evicted - expected ok=false")
	}

	// seq 2: the caller claims to have already seen exactly up through the
	// last *evicted* event. Everything retained (3,4,5) is > 2, so this is
	// a complete, gap-free resume - the buffer doesn't need to still hold
	// seq 2 itself to serve it correctly.
	got, ok := b.Since(2)
	if !ok {
		t.Fatal("seq 2 is exactly the retained boundary - expected ok=true")
	}
	if len(got) != 3 {
		t.Fatalf("expected seq 3,4,5 replayed, got %d events", len(got))
	}

	got, ok = b.Since(3)
	if !ok {
		t.Fatal("seq 3 is well within the retained range - expected ok=true")
	}
	if len(got) != 2 {
		t.Fatalf("expected seq 4,5 replayed, got %d events", len(got))
	}

	// seq 5: fully caught up already - valid, empty replay, straight to live.
	got, ok = b.Since(5)
	if !ok {
		t.Fatal("seq 5 is the newest retained seq (fully caught up) - expected ok=true")
	}
	if len(got) != 0 {
		t.Fatalf("expected an empty replay, got %d events", len(got))
	}

	// seq 99: a cursor from "the future" relative to anything this buffer
	// has ever held. Must 410, not silently succeed with an empty replay -
	// there's no way to distinguish that from a client that's confused
	// about which epoch it's talking to.
	if _, ok := b.Since(99); ok {
		t.Fatal("seq 99 is ahead of anything the buffer has seen - expected ok=false")
	}
}

func TestRingBuffer_SinceAheadOfBuffer(t *testing.T) {
	t.Parallel()
	b := NewRingBuffer(10, time.Minute)
	b.Append(&Event{Seq: 1})
	b.Append(&Event{Seq: 2})

	if _, ok := b.Since(99); ok {
		t.Fatal("a seq the buffer has never seen must not be treated as caught-up - expected ok=false")
	}
}

func TestRingBuffer_EvictByCount(t *testing.T) {
	t.Parallel()
	b := NewRingBuffer(3, time.Hour) // age bound effectively disabled for this test

	for i := uint64(1); i <= 10; i++ {
		b.Append(&Event{Seq: i})
	}
	// Only the 3 most recent (8,9,10) are retained; 1-7 evicted by count.

	// A cursor claiming to have seen nothing (0) needs everything back to
	// seq 1, which is long gone - unrecoverable, must be rejected.
	if _, ok := b.Since(0); ok {
		t.Fatal("Since(0) after evicting 1-7 by count must fail - those events are unrecoverable")
	}

	// The exact retained boundary (7 = oldest retained seq minus 1) is a
	// complete, gap-free resume point.
	got, ok := b.Since(7)
	if !ok {
		t.Fatal("Since(7) is exactly the retained boundary - expected ok=true")
	}
	wantSeqs := []uint64{8, 9, 10}
	if len(got) != len(wantSeqs) {
		t.Fatalf("expected %d events, got %d", len(wantSeqs), len(got))
	}
	for i, evt := range got {
		if evt.Seq != wantSeqs[i] {
			t.Fatalf("expected seq %d at index %d, got %d", wantSeqs[i], i, evt.Seq)
		}
	}
}

func TestRingBuffer_EvictByAge(t *testing.T) {
	t.Parallel()
	b := NewRingBuffer(100, 20*time.Millisecond) // count bound effectively disabled

	b.Append(&Event{Seq: 1})
	b.Append(&Event{Seq: 2})
	time.Sleep(40 * time.Millisecond)
	b.Append(&Event{Seq: 3}) // triggers eviction of 1,2 on this Append

	// A cursor claiming to have seen nothing (0) needs seq 1 back, which
	// aged out - unrecoverable.
	if _, ok := b.Since(0); ok {
		t.Fatal("Since(0) after 1,2 aged out must fail - those events are unrecoverable")
	}
	if _, ok := b.Since(1); ok {
		t.Fatal("seq 1 needs seq 2 to resume safely, and seq 2 aged out too - expected ok=false")
	}

	// seq 2 is exactly the retained boundary (oldest retained, 3, minus 1).
	got, ok := b.Since(2)
	if !ok {
		t.Fatal("seq 2 is exactly the retained boundary - expected ok=true")
	}
	if len(got) != 1 || got[0].Seq != 3 {
		t.Fatalf("expected only seq 3 replayed, got %+v", got)
	}
}

func TestRingBuffer_WrapAround(t *testing.T) {
	t.Parallel()
	// Exercises repeated evict-while-appending past the count bound many
	// times over, the way a long-running Hub would.
	b := NewRingBuffer(5, time.Hour)

	const total = 137
	for i := uint64(1); i <= total; i++ {
		b.Append(&Event{Seq: i})
	}

	got, ok := b.Since(total - 5)
	if !ok {
		t.Fatal("Since at the exact retained boundary should succeed")
	}
	if len(got) != 5 {
		t.Fatalf("expected 5 events, got %d", len(got))
	}
	for i, evt := range got {
		want := total - 5 + uint64(i) + 1
		if evt.Seq != want {
			t.Fatalf("expected seq %d at index %d, got %d", want, i, evt.Seq)
		}
	}

	if _, ok := b.Since(total - 6); ok {
		t.Fatal("one seq older than the retained boundary must fail")
	}
}

// TestRingBuffer_ConcurrentAppendAndSince checks that a single sequential
// writer (the only pattern the Hub actually produces - it serializes seq
// assignment and Append under its own lock before this buffer ever sees
// them) can safely race against many concurrent readers calling Since.
// Ordering-correctness under *out-of-order concurrent Append* isn't a
// contract this buffer makes - the Append doc requires ev.Seq to already be
// set, i.e. the caller owns ordering. This test only proves no data race
// and that whatever Since does return is internally consistent. Run with
// -race.
func TestRingBuffer_ConcurrentAppendAndSince(t *testing.T) {
	t.Parallel()
	b := NewRingBuffer(1000, time.Minute)

	const total = 500
	writerDone := make(chan struct{})
	go func() {
		defer close(writerDone)
		for i := uint64(1); i <= total; i++ {
			b.Append(&Event{Seq: i})
		}
	}()

	const readers = 8
	readerDone := make(chan struct{})
	for range readers {
		go func() {
			defer func() { readerDone <- struct{}{} }()
			for i := range 200 {
				got, ok := b.Since(uint64(i))
				if !ok {
					continue // aged/ahead - fine, just not what this test checks
				}
				for j := 1; j < len(got); j++ {
					if got[j].Seq <= got[j-1].Seq {
						t.Errorf("Since returned non-increasing seq: %d then %d", got[j-1].Seq, got[j].Seq)
					}
				}
			}
		}()
	}

	<-writerDone
	for range readers {
		<-readerDone
	}
}

// TestRingBuffer_NonPositiveMaxLenClampsToDefault confirms that a
// misconfigured (<= 0) maxLen doesn't leave the buffer unbounded - it must
// still evict, at the documented default.
func TestRingBuffer_NonPositiveMaxLenClampsToDefault(t *testing.T) {
	t.Parallel()
	for _, maxLen := range []int{0, -1, -100} {
		b := NewRingBuffer(maxLen, time.Hour)
		for i := uint64(1); i <= defaultMaxLen+50; i++ {
			b.Append(&Event{Seq: i})
		}
		if _, ok := b.Since(0); ok {
			t.Fatalf("maxLen=%d: expected Since(0) to fail once eviction has happened (not unbounded)", maxLen)
		}
		got, ok := b.Since(defaultMaxLen + 50 - defaultMaxLen) // oldest retained - 1
		if !ok {
			t.Fatalf("maxLen=%d: expected the clamped-default boundary to be servable", maxLen)
		}
		if len(got) != defaultMaxLen {
			t.Fatalf("maxLen=%d: expected exactly %d retained events (clamped to default), got %d", maxLen, defaultMaxLen, len(got))
		}
	}
}

// TestRingBuffer_NonPositiveMaxAgeClampsToDefault confirms that a
// misconfigured (<= 0) maxAge doesn't disable age-based eviction entirely.
func TestRingBuffer_NonPositiveMaxAgeClampsToDefault(t *testing.T) {
	t.Parallel()
	for _, maxAge := range []time.Duration{0, -time.Second, -time.Hour} {
		b := NewRingBuffer(1000000, maxAge) // count bound effectively disabled
		b.Append(&Event{Seq: 1})
		// The clamped default (5m) is long enough that this event must
		// still be retained immediately after appending it.
		got, ok := b.Since(0)
		if !ok || len(got) != 1 {
			t.Fatalf("maxAge=%s: expected the just-appended event to still be retained under the clamped default", maxAge)
		}
	}
}
