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

package server

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"

	"github.com/openeverest/openeverest/v2/pkg/events"
)

func newTestEventsServer(maxLen int) *EverestServer {
	h := events.NewHub(zap.NewNop().Sugar(), nil, events.NewRingBuffer(maxLen, time.Minute))
	return &EverestServer{eventHub: h}
}

// syncRecorder wraps httptest.ResponseRecorder with a mutex-guarded body.
// Plain ResponseRecorder's Body is a bare *bytes.Buffer - fine for handlers
// that run synchronously in the test goroutine, but an SSE handler under
// test runs in its own goroutine and has to be observed mid-stream, so its
// writes and the test's reads are genuinely concurrent. Overriding Write
// here (Go's method promotion lets ours shadow the embedded one) keeps
// header/status-code handling on the real ResponseRecorder, which is never
// touched concurrently in these tests, while giving the body its own lock.
type syncRecorder struct {
	*httptest.ResponseRecorder

	mu  sync.Mutex
	buf bytes.Buffer
}

func newSyncRecorder() *syncRecorder {
	return &syncRecorder{ResponseRecorder: httptest.NewRecorder()}
}

func (r *syncRecorder) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.buf.Write(p)
}

func (r *syncRecorder) String() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.buf.String()
}

// runEventsHandler starts eventsHandler against a cancellable request context
// and returns the recorder, a cancel func, and a channel closed once the
// handler returns. Callers must cancel and wait on done before using rec's
// underlying *httptest.ResponseRecorder fields (Code, etc.) directly - those
// aren't guarded the way rec.String() is.
func runEventsHandler(t *testing.T, e *EverestServer, target string) (*syncRecorder, context.CancelFunc, chan struct{}) {
	t.Helper()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, target, nil)
	ctx, cancelFn := context.WithCancel(req.Context())
	req = req.WithContext(ctx)
	rec := newSyncRecorder()
	c := echo.New().NewContext(req, rec)

	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := e.eventsHandler(c); err != nil {
			t.Errorf("eventsHandler returned an error: %v", err)
		}
	}()
	return rec, cancelFn, done
}

// waitForBodyContains polls rec's body until it contains want or 5 seconds
// pass. Necessary because the handler streams asynchronously; 5s (rather
// than ~1s) gives a loaded CI runner enough room that this doesn't flake.
func waitForBodyContains(t *testing.T, rec *syncRecorder, want string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(rec.String(), want) {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for body to contain %q; got:\n%s", want, rec.String())
}

func TestEventsHandler_NoSinceIsLiveOnly(t *testing.T) {
	t.Parallel()
	e := newTestEventsServer(100)

	e.eventHub.Publish(events.Event{Type: events.UserLogin, Resource: events.ResourceRef{Name: "before"}})

	rec, cancel, done := runEventsHandler(t, e, "/v1/events")
	waitForBodyContains(t, rec, ": connected")

	e.eventHub.Publish(events.Event{Type: events.UserLogin, Resource: events.ResourceRef{Name: "after"}})
	waitForBodyContains(t, rec, `"name":"after"`)

	cancel()
	<-done

	if strings.Contains(rec.String(), `"name":"before"`) {
		t.Fatal("event published before the handler subscribed must not appear without a since param")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestEventsHandler_ValidSinceReplaysThenStreamsLive(t *testing.T) {
	t.Parallel()
	e := newTestEventsServer(100)

	e.eventHub.Publish(events.Event{Type: events.InstanceCreated, Resource: events.ResourceRef{Name: "a"}}) // seq 1
	e.eventHub.Publish(events.Event{Type: events.InstanceCreated, Resource: events.ResourceRef{Name: "b"}}) // seq 2

	cursor := (events.Cursor{Epoch: e.eventHub.Epoch(), Seq: 1}).String()
	rec, cancel, done := runEventsHandler(t, e, "/v1/events?since="+cursor)

	waitForBodyContains(t, rec, `"name":"b"`)

	e.eventHub.Publish(events.Event{Type: events.InstanceCreated, Resource: events.ResourceRef{Name: "c"}})
	waitForBodyContains(t, rec, `"name":"c"`)

	cancel()
	<-done

	if strings.Contains(rec.String(), `"name":"a"`) {
		t.Fatal("event at/before the cursor must not be replayed")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestEventsHandler_UnparseableSinceIs410(t *testing.T) {
	t.Parallel()
	e := newTestEventsServer(100)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/v1/events?since=garbage-no-colon", nil)
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(req, rec)

	if err := e.eventsHandler(c); err != nil {
		t.Fatalf("eventsHandler returned an error: %v", err)
	}
	if rec.Code != http.StatusGone {
		t.Fatalf("expected 410 for an unparseable cursor, got %d", rec.Code)
	}
}

// TestEventsHandler_LegacyBareResourceVersionCursorIs410 covers backward
// compatibility with the plugin-audit client running today, which sends a
// bare resourceVersion (e.g. "455") as since - the pre-#2582 cursor format,
// with no ":" in it. This must 410 exactly like any other unparseable
// cursor, so the existing client's current behaviour on 410 (reset cursor,
// reconnect live-only) kicks in unchanged. If this ever silently "succeeded"
// instead - parsing "455" as some malformed-but-accepted epoch, say - an
// old client would get stuck rather than degrade cleanly.
func TestEventsHandler_LegacyBareResourceVersionCursorIs410(t *testing.T) {
	t.Parallel()
	e := newTestEventsServer(100)
	e.eventHub.Publish(events.Event{Type: events.InstanceCreated, ResourceVersion: "455"})

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/v1/events?since=455", nil)
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(req, rec)

	if err := e.eventsHandler(c); err != nil {
		t.Fatalf("eventsHandler returned an error: %v", err)
	}
	if rec.Code != http.StatusGone {
		t.Fatalf("expected 410 for a legacy bare-resourceVersion cursor, got %d", rec.Code)
	}
}

func TestEventsHandler_EpochMismatchIs410(t *testing.T) {
	t.Parallel()
	e := newTestEventsServer(100)
	e.eventHub.Publish(events.Event{Type: events.UserLogin})

	cursor := (events.Cursor{Epoch: "not-this-hubs-epoch", Seq: 1}).String()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/v1/events?since="+cursor, nil)
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(req, rec)

	if err := e.eventsHandler(c); err != nil {
		t.Fatalf("eventsHandler returned an error: %v", err)
	}
	if rec.Code != http.StatusGone {
		t.Fatalf("expected 410 for an epoch mismatch, got %d", rec.Code)
	}
}

func TestEventsHandler_SeqAgedOutIs410(t *testing.T) {
	t.Parallel()
	e := newTestEventsServer(2) // tiny buffer
	for range 5 {
		e.eventHub.Publish(events.Event{Type: events.UserLogin})
	}

	cursor := (events.Cursor{Epoch: e.eventHub.Epoch(), Seq: 1}).String()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/v1/events?since="+cursor, nil)
	rec := httptest.NewRecorder()
	c := echo.New().NewContext(req, rec)

	if err := e.eventsHandler(c); err != nil {
		t.Fatalf("eventsHandler returned an error: %v", err)
	}
	if rec.Code != http.StatusGone {
		t.Fatalf("expected 410 for an aged-out seq, got %d", rec.Code)
	}
}
