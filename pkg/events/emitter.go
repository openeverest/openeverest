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
	"context"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	eventsv1 "k8s.io/api/events/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// maxNameLength is the maximum length of a Kubernetes object name.
	maxNameLength = 253
	// maxNoteLength is the maximum length of an events.k8s.io/v1 Event note.
	maxNoteLength = 1024
)

//nolint:gochecknoglobals // immutable value
var createBackoff = wait.Backoff{Steps: 5, Duration: 200 * time.Millisecond, Factor: 2, Jitter: 0.1}

// Record is a lifecycle event to write.
type Record struct {
	// Regarding is the object the event is about.
	Regarding corev1.ObjectReference
	Reason    Reason
	// Note is a human-readable description of the event.
	Note    string
	Warning bool
	Actor   Actor
	// PhaseFrom and PhaseTo are the previous and new phase of a PhaseChanged event.
	PhaseFrom string
	PhaseTo   string
	// Generation is the new metadata.generation of an Updated event.
	Generation int64
	SourceIP   string
	// Key tells apart events with the same reason about the same object, such as two
	// phase changes. For an object with a UID, an event with the same reason and key is
	// written only once, so recording it again after a restart or a retry is safe.
	Key string
}

// Emitter writes lifecycle events as Kubernetes Events.
//
// An event is stored in the namespace of the object it is about. Kubernetes requires the
// events about cluster-scoped objects, such as plugins and namespaces, to be in the
// default namespace.
type Emitter struct {
	client            client.Client
	reportingInstance string
	now               func() time.Time
}

// NewEmitter returns an Emitter that writes with c on behalf of reportingInstance,
// typically the pod name.
func NewEmitter(c client.Client, reportingInstance string) *Emitter {
	return &Emitter{
		client:            c,
		reportingInstance: reportingInstance,
		now:               time.Now,
	}
}

// Emit writes r. Transient API errors are retried; an event that was already written
// is not an error.
func (e *Emitter) Emit(ctx context.Context, r Record) error {
	err := retry.OnError(createBackoff, isRetriable, func() error {
		return e.client.Create(ctx, e.newEvent(r))
	})
	if apierrors.IsAlreadyExists(err) {
		return nil
	}
	return err
}

func isRetriable(err error) bool {
	return apierrors.IsServerTimeout(err) || apierrors.IsTimeout(err) || apierrors.IsTooManyRequests(err) ||
		apierrors.IsInternalError(err) || apierrors.IsServiceUnavailable(err) || apierrors.IsUnexpectedServerError(err)
}

func (e *Emitter) newEvent(r Record) *eventsv1.Event {
	namespace := r.Regarding.Namespace
	if namespace == "" {
		namespace = metav1.NamespaceDefault
	}
	eventType := corev1.EventTypeNormal
	if r.Warning {
		eventType = corev1.EventTypeWarning
	}

	annotations := make(map[string]string)
	setIfNotEmpty(annotations, AnnotationEventActorType, r.Actor.Type)
	setIfNotEmpty(annotations, AnnotationEventActorID, r.Actor.ID)
	setIfNotEmpty(annotations, AnnotationPhaseFrom, r.PhaseFrom)
	setIfNotEmpty(annotations, AnnotationPhaseTo, r.PhaseTo)
	setIfNotEmpty(annotations, AnnotationSourceIP, r.SourceIP)
	if r.Generation > 0 {
		annotations[AnnotationGeneration] = strconv.FormatInt(r.Generation, 10)
	}

	event := &eventsv1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   namespace,
			Labels:      map[string]string{LabelLifecycleEvent: LabelLifecycleEventValue},
			Annotations: annotations,
		},
		EventTime:           metav1.NewMicroTime(e.now()),
		ReportingController: ReportingController,
		ReportingInstance:   e.reportingInstance,
		Action:              string(r.Reason),
		Reason:              string(r.Reason),
		Regarding:           r.Regarding,
		Note:                truncate(r.Note, maxNoteLength),
		Type:                eventType,
	}
	if name := eventName(r); name != "" {
		event.Name = name
	} else {
		event.GenerateName = strings.ToLower(string(r.Reason)) + "-"
	}
	return event
}

// eventName returns the deterministic name of the event for r, or "" when the object
// has no UID to make it unique.
func eventName(r Record) string {
	if r.Regarding.UID == "" {
		return ""
	}
	suffix := "." + string(r.Regarding.UID) + "." + strings.ToLower(string(r.Reason))
	if r.Key != "" {
		suffix += "-" + strings.ToLower(r.Key)
	}
	prefix := truncate(r.Regarding.Name, maxNameLength-len(suffix))
	return strings.TrimRight(prefix, ".-") + suffix
}

func setIfNotEmpty(m map[string]string, key, value string) {
	if value != "" {
		m[key] = value
	}
}

func truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}
