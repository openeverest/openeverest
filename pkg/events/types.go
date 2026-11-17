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

// Package events defines OpenEverest lifecycle events.
//
// Lifecycle events are Kubernetes Events (events.k8s.io/v1) that the OpenEverest
// controllers write for the objects they manage, whether those objects were changed
// through the OpenEverest API, kubectl or a GitOps tool, and that the API server writes
// for authentication. They carry a small, kind-independent set of reasons; the kind is
// taken from the object the Event refers to. GET /v1/clusters/{cluster}/events serves
// them to clients such as audit plugins.
package events

// Reason is the kind-independent reason of a lifecycle event.
type Reason string

// Lifecycle event reasons.
const (
	// ReasonCreated is recorded when an object appears.
	ReasonCreated Reason = "Created"
	// ReasonUpdated is recorded when the spec of an object changes, that is when its
	// metadata.generation increases.
	ReasonUpdated Reason = "Updated"
	// ReasonPhaseChanged is recorded when the phase (or state) in the status of an
	// object changes.
	ReasonPhaseChanged Reason = "PhaseChanged"
	// ReasonDeletionRequested is recorded when the deletion of an object is requested.
	ReasonDeletionRequested Reason = "DeletionRequested"
	// ReasonDeleted is recorded when an object is gone.
	ReasonDeleted Reason = "Deleted"

	// ReasonLoginSucceeded is recorded when a user logs in.
	ReasonLoginSucceeded Reason = "LoginSucceeded"
	// ReasonLoginFailed is recorded when a login attempt fails.
	ReasonLoginFailed Reason = "LoginFailed"
	// ReasonLogout is recorded when a user logs out.
	ReasonLogout Reason = "Logout"
)

// AllReasons lists every lifecycle event reason.
//
//nolint:gochecknoglobals // immutable lookup table
var AllReasons = []Reason{
	ReasonCreated, ReasonUpdated, ReasonPhaseChanged, ReasonDeletionRequested, ReasonDeleted,
	ReasonLoginSucceeded, ReasonLoginFailed, ReasonLogout,
}

const (
	// LabelLifecycleEvent marks the Kubernetes Events written by OpenEverest, so they can
	// be listed and watched with a label selector.
	LabelLifecycleEvent = "openeverest.io/lifecycle-event"
	// LabelLifecycleEventValue is the value of LabelLifecycleEvent.
	LabelLifecycleEventValue = "true"

	// ReportingController is the reportingController of every lifecycle event.
	ReportingController = "openeverest.io/everest"

	// AnnotationEventActorType records Actor.Type on a lifecycle event.
	AnnotationEventActorType = "openeverest.io/actor-type"
	// AnnotationEventActorID records Actor.ID on a lifecycle event.
	AnnotationEventActorID = "openeverest.io/actor-id"
	// AnnotationPhaseFrom records the previous phase of a PhaseChanged event.
	AnnotationPhaseFrom = "openeverest.io/phase-from"
	// AnnotationPhaseTo records the new phase of a PhaseChanged event.
	AnnotationPhaseTo = "openeverest.io/phase-to"
	// AnnotationGeneration records the new metadata.generation of an Updated event.
	AnnotationGeneration = "openeverest.io/generation"
	// AnnotationSourceIP records the client IP of an authentication event.
	AnnotationSourceIP = "openeverest.io/source-ip"
)

// Actor types.
const (
	// ActorTypeUser is an OpenEverest user, identified by the subject of their token.
	ActorTypeUser = "user"
	// ActorTypeSystem is an OpenEverest component, such as a controller.
	ActorTypeSystem = "system"
	// ActorTypeKubernetesClient is a client that changed the object directly in
	// Kubernetes (kubectl, a GitOps tool), identified by its field manager.
	ActorTypeKubernetesClient = "kubernetes-client"
)
