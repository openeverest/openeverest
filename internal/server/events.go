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
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"

	api "github.com/openeverest/openeverest/v2/internal/server/api"
	"github.com/openeverest/openeverest/v2/pkg/events"
)

// eventsHandler streams lifecycle events to the client as SSE.
//
//	GET /v1/events?types=<csv>&namespaces=<csv>&since=<epoch>:<seq>
//	Accept: text/event-stream
//
// since is optional. When present it must be a cursor previously read off
// an event's seq/epoch fields (openeverest#2582); an unparseable cursor, an
// epoch that doesn't match this process, or a seq that's aged out of (or is
// ahead of) the replay buffer all get a 410 — the client must resync (list
// current state, then reconnect without since) rather than treat it as a
// retryable error.
func (e *EverestServer) eventsHandler(c echo.Context) error {
	// Parse optional filters.
	var typeFilter []events.Type
	if raw := c.QueryParam("types"); raw != "" {
		for _, t := range strings.Split(raw, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				typeFilter = append(typeFilter, events.Type(t))
			}
		}
	}

	var nsFilter []string
	if raw := c.QueryParam("namespaces"); raw != "" {
		for _, ns := range strings.Split(raw, ",") {
			ns = strings.TrimSpace(ns)
			if ns != "" {
				nsFilter = append(nsFilter, ns)
			}
		}
	}

	var cursor *events.Cursor
	if raw := c.QueryParam("since"); raw != "" {
		parsed, err := events.ParseCursor(raw)
		if err != nil {
			return c.JSON(http.StatusGone, api.Error{
				Message: new(fmt.Sprintf("invalid cursor: %s", err)),
			})
		}
		cursor = &parsed
	}

	sub, ok := e.eventHub.Subscribe(typeFilter, nsFilter, cursor)
	if !ok {
		return c.JSON(http.StatusGone, api.Error{
			Message: new("cursor is no longer valid; relist current state and reconnect without since"),
		})
	}
	defer sub.Cancel()

	// Set SSE headers.
	w := c.Response()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable nginx/proxy buffering
	w.WriteHeader(http.StatusOK)

	// Send an initial SSE comment so the browser sees the response immediately.
	if _, err := fmt.Fprint(w, ": connected\n\n"); err != nil {
		return nil
	}
	w.Flush()

	for _, evt := range sub.Replay {
		if err := writeEventFrame(w, evt); err != nil {
			return nil // client disconnected
		}
		w.Flush()
	}

	ctx := c.Request().Context()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-sub.Dropped:
			// Slow consumer - broadcast gave up on us. Ch won't receive
			// anything further.
			return nil
		case evt := <-sub.Ch:
			if err := writeEventFrame(w, evt); err != nil {
				return nil // client disconnected
			}
			w.Flush()
		}
	}
}

// writeEventFrame writes evt as a single SSE frame. id carries the
// "<epoch>:<seq>" cursor rather than the legacy resourceVersion, since
// seq/epoch is the cursor that's actually honored on reconnect and it's the
// only one that's always populated (direct-publish events have no
// resourceVersion). Native EventSource clients that use Last-Event-ID get a
// usable value here; this server's own since handling still only reads the
// since query param, not that header.
func writeEventFrame(w http.ResponseWriter, evt events.Event) error {
	data, err := json.Marshal(evt)
	if err != nil {
		return nil //nolint:nilerr // malformed envelope is skipped, not a disconnect
	}
	cursor := events.Cursor{Epoch: evt.Epoch, Seq: evt.Seq}
	_, err = fmt.Fprintf(w, "id: %s\nevent: %s\ndata: %s\n\n", cursor.String(), evt.Type, data)
	return err
}
