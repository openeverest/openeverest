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
	"testing/synctest"
	"time"
)

func TestRingBuffer_SinceExpiresIdleEvents(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		b := NewRingBuffer(10, time.Minute)
		b.Append(&Event{Seq: 1})
		b.Append(&Event{Seq: 2})
		time.Sleep(2 * time.Minute)

		for _, seq := range []uint64{0, 1} {
			if got, ok := b.Since(seq); ok || len(got) != 0 {
				t.Errorf("Since(%d) after idle expiry = (%v, %t), want no replay and resync", seq, got, ok)
			}
		}
		if got, ok := b.Since(2); !ok || len(got) != 0 {
			t.Errorf("caught-up cursor after idle expiry = (%v, %t), want an empty successful replay", got, ok)
		}
		if _, ok := b.Since(3); ok {
			t.Error("future cursor after idle expiry must require resync")
		}

		b.Append(&Event{Seq: 3})
		if got, ok := b.Since(2); !ok || len(got) != 1 || got[0].Seq != 3 {
			t.Fatalf("replay after publishing resumes = (%v, %t), want only seq 3", got, ok)
		}
	})
}

func TestRingBuffer_SinceExpiresOnlyOldEvents(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) {
		b := NewRingBuffer(10, 3*time.Minute)
		b.Append(&Event{Seq: 1})
		time.Sleep(2 * time.Minute)
		b.Append(&Event{Seq: 2})
		time.Sleep(2 * time.Minute)

		if got, ok := b.Since(0); ok || len(got) != 0 {
			t.Errorf("cursor before expired event = (%v, %t), want no replay and resync", got, ok)
		}
		if got, ok := b.Since(1); !ok || len(got) != 1 || got[0].Seq != 2 {
			t.Fatalf("cursor at retained boundary = (%v, %t), want only seq 2", got, ok)
		}
	})
}
