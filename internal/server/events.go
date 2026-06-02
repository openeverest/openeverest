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

	"github.com/openeverest/openeverest/v2/pkg/events"
)

// eventsHandler streams lifecycle events to the client as SSE.
//
//	GET /v1/events?types=<csv>&namespaces=<csv>
//	Accept: text/event-stream
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

	ch, cancel := e.eventHub.Subscribe(typeFilter, nsFilter)
	defer cancel()

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

	ctx := c.Request().Context()
	for {
		select {
		case <-ctx.Done():
			return nil
		case evt, ok := <-ch:
			if !ok {
				// Subscriber was dropped (slow consumer).
				return nil
			}
			data, err := json.Marshal(evt)
			if err != nil {
				continue
			}
			// Write SSE frame: id is resourceVersion, data is the JSON envelope.
			if _, err := fmt.Fprintf(w, "id: %s\nevent: %s\ndata: %s\n\n", evt.ResourceVersion, evt.Type, data); err != nil {
				return nil // client disconnected
			}
			w.Flush()
		}
	}
}
