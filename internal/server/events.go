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
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	eventsv1 "k8s.io/api/events/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/watch"

	api "github.com/openeverest/openeverest/v2/internal/server/api"
	"github.com/openeverest/openeverest/v2/pkg/events"
)

// eventStreamKeepAlive is how often an idle event stream sends a comment, so proxies
// don't close it.
const eventStreamKeepAlive = 30 * time.Second

// ListEvents lists the lifecycle events of a cluster, or streams them as Server-Sent
// Events with watch=true.
func (e *EverestServer) ListEvents(c echo.Context, cluster string, params api.ListEventsParams) error {
	filter, err := newEventFilter(params)
	if err != nil {
		return c.JSON(http.StatusBadRequest, api.Error{Message: new(err.Error())})
	}
	namespaces := splitCSV(params.Namespaces)
	since := ""
	if params.Since != nil {
		since = *params.Since
	}

	if params.Watch != nil && *params.Watch {
		return e.streamEvents(c, cluster, namespaces, since, filter)
	}
	if since != "" {
		return c.JSON(http.StatusBadRequest, api.Error{Message: new("since is only supported with watch=true")})
	}

	list, err := e.handler.ListEvents(c.Request().Context(), cluster, namespaces)
	if err != nil {
		return err
	}
	items := make([]api.LifecycleEvent, 0, len(list.Items))
	for i := range list.Items {
		if filter.matches(&list.Items[i]) {
			items = append(items, toLifecycleEvent(cluster, &list.Items[i]))
		}
	}
	slices.SortStableFunc(items, func(a, b api.LifecycleEvent) int {
		return a.OccurredAt.Compare(b.OccurredAt)
	})
	return c.JSON(http.StatusOK, api.LifecycleEventList{Items: items, ResourceVersion: list.ResourceVersion})
}

func (e *EverestServer) streamEvents(c echo.Context, cluster string, namespaces []string, since string, filter eventFilter) error {
	ctx := c.Request().Context()
	w, err := e.handler.WatchEvents(ctx, cluster, namespaces, since)
	if err != nil {
		return err
	}
	defer w.Stop()

	res := c.Response()
	res.Header().Set(echo.HeaderContentType, "text/event-stream")
	res.Header().Set(echo.HeaderCacheControl, "no-cache")
	res.Header().Set(echo.HeaderConnection, "keep-alive")
	res.Header().Set("X-Accel-Buffering", "no") // disable nginx/proxy buffering
	res.WriteHeader(http.StatusOK)
	if _, err := io.WriteString(res, ": connected\n\n"); err != nil {
		return nil // the client is gone
	}
	res.Flush()

	keepAlive := time.NewTicker(eventStreamKeepAlive)
	defer keepAlive.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-keepAlive.C:
			if _, err := io.WriteString(res, ": keepalive\n\n"); err != nil {
				return nil // the client is gone
			}
		case ev, ok := <-w.ResultChan():
			if !ok {
				return nil
			}
			done, err := writeWatchEvent(res, cluster, ev, filter)
			if err != nil || done {
				return nil // the client is gone or the stream ended
			}
		}
		res.Flush()
	}
}

// writeWatchEvent writes ev as a Server-Sent Event. It returns done when the stream
// must end.
func writeWatchEvent(w io.Writer, cluster string, ev watch.Event, filter eventFilter) (bool, error) {
	switch ev.Type {
	case watch.Added:
		kubeEvent, ok := ev.Object.(*eventsv1.Event)
		if !ok || !filter.matches(kubeEvent) {
			return false, nil
		}
		event := toLifecycleEvent(cluster, kubeEvent)
		return false, writeSSE(w, event.Cursor, event.Type, event)
	case watch.Bookmark:
		obj, err := meta.Accessor(ev.Object)
		if err != nil {
			return false, nil //nolint:nilerr // a malformed bookmark is skipped
		}
		return false, writeSSE(w, obj.GetResourceVersion(), "bookmark", struct{}{})
	case watch.Error:
		code, message := http.StatusInternalServerError, "the event stream failed"
		if statusErr, ok := errors.AsType[*apierrors.StatusError](apierrors.FromObject(ev.Object)); ok {
			code, message = int(statusErr.Status().Code), statusErr.Error()
		}
		if code == http.StatusGone {
			message = "the cursor is too old: list the events again and stream from the new resourceVersion"
		}
		return true, writeSSE(w, "", "error", eventStreamError{Code: code, Message: message})
	case watch.Modified, watch.Deleted:
		// Lifecycle events are only created; updates and their expiry are not new events.
		return false, nil
	}
	return false, nil
}

type eventStreamError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func writeSSE(w io.Writer, id, eventType string, data any) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return nil //nolint:nilerr // an event that can't be encoded is skipped
	}
	var frame strings.Builder
	if id != "" {
		fmt.Fprintf(&frame, "id: %s\n", id)
	}
	fmt.Fprintf(&frame, "event: %s\ndata: %s\n\n", eventType, payload)
	_, err = io.WriteString(w, frame.String())
	return err
}

// eventFilter keeps the events with one of the requested kinds and reasons.
type eventFilter struct {
	kinds   []string
	reasons []string
}

func newEventFilter(params api.ListEventsParams) (eventFilter, error) {
	filter := eventFilter{reasons: splitCSV(params.Reasons)}
	for _, kind := range splitCSV(params.Kinds) {
		filter.kinds = append(filter.kinds, strings.ToLower(kind))
	}
	for _, reason := range filter.reasons {
		if !slices.Contains(events.AllReasons, events.Reason(reason)) {
			return eventFilter{}, fmt.Errorf("unknown reason %q", reason)
		}
	}
	return filter, nil
}

func (f eventFilter) matches(ev *eventsv1.Event) bool {
	if len(f.kinds) > 0 && !slices.Contains(f.kinds, strings.ToLower(ev.Regarding.Kind)) {
		return false
	}
	return len(f.reasons) == 0 || slices.Contains(f.reasons, ev.Reason)
}

func splitCSV(raw *string) []string {
	if raw == nil {
		return nil
	}
	var values []string
	for v := range strings.SplitSeq(*raw, ",") {
		if v = strings.TrimSpace(v); v != "" {
			values = append(values, v)
		}
	}
	return values
}

func toLifecycleEvent(cluster string, ev *eventsv1.Event) api.LifecycleEvent {
	occurredAt := ev.EventTime.Time
	if occurredAt.IsZero() {
		occurredAt = ev.CreationTimestamp.Time
	}
	out := api.LifecycleEvent{
		Name:       ev.Name,
		Cursor:     ev.ResourceVersion,
		OccurredAt: occurredAt.UTC(),
		Type:       eventType(ev.Regarding.Kind, ev.Reason),
		Reason:     api.LifecycleEventReason(ev.Reason),
		Severity:   api.LifecycleEventSeverity(ev.Type),
		Cluster:    cluster,
		Object: api.LifecycleEventObject{
			ApiVersion: nonEmpty(ev.Regarding.APIVersion),
			Kind:       ev.Regarding.Kind,
			Name:       ev.Regarding.Name,
			Uid:        nonEmpty(string(ev.Regarding.UID)),
		},
		Namespace: nonEmpty(ev.Namespace),
		Message:   nonEmpty(ev.Note),
		SourceIP:  nonEmpty(ev.Annotations[events.AnnotationSourceIP]),
	}
	if actorType, actorID := ev.Annotations[events.AnnotationEventActorType], ev.Annotations[events.AnnotationEventActorID]; actorType != "" || actorID != "" {
		out.Actor = &api.LifecycleEventActor{Id: nonEmpty(actorID)}
		if actorType != "" {
			out.Actor.Type = new(api.LifecycleEventActorType(actorType))
		}
	}
	if from, to := ev.Annotations[events.AnnotationPhaseFrom], ev.Annotations[events.AnnotationPhaseTo]; from != "" || to != "" {
		out.Phase = &api.LifecycleEventPhase{From: nonEmpty(from), To: nonEmpty(to)}
	}
	if generation, err := strconv.ParseInt(ev.Annotations[events.AnnotationGeneration], 10, 64); err == nil {
		out.Generation = &generation
	}
	return out
}

// eventType returns the kind and reason of an event in the form "instance.phase-changed".
func eventType(kind, reason string) string {
	var kebab strings.Builder
	for i, r := range reason {
		if i > 0 && r >= 'A' && r <= 'Z' {
			kebab.WriteByte('-')
		}
		kebab.WriteRune(r)
	}
	return strings.ToLower(kind) + "." + strings.ToLower(kebab.String())
}

func nonEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
