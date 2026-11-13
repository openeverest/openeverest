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

// Annotation keys used to stamp the originating actor onto user-mutated
// resources. The API server writes these on every Create/Update/Delete so
// the lifecycle events the controller records for the change can be
// attributed back to the user who triggered it.
const (
	// AnnotationActorType is "user", "system", "plugin", or "".
	AnnotationActorType = "openeverest.io/last-actor-type"
	// AnnotationActorID is the JWT subject (for users), plugin name (for
	// plugins), or component identifier (for system actions).
	AnnotationActorID = "openeverest.io/last-actor-id"
)

// Actor describes who or what triggered an event. Both fields are omitted when empty,
// so an unattributed event has no actor rather than one that looks populated.
type Actor struct {
	Type string `json:"type,omitempty"`
	ID   string `json:"id,omitempty"`
}

// IsZero reports whether the actor is unknown.
func (a Actor) IsZero() bool {
	return a.Type == "" && a.ID == ""
}

// ControllerActor identifies changes made by the OpenEverest controllers, such as phase
// transitions.
//
//nolint:gochecknoglobals // immutable value
var ControllerActor = Actor{Type: ActorTypeSystem, ID: "controller"}

// ActorFromAnnotations reconstructs an Actor from the annotation map carried by a
// Kubernetes resource. Returns the zero Actor when the annotations are absent.
func ActorFromAnnotations(annotations map[string]string) Actor {
	if annotations == nil {
		return Actor{}
	}
	return Actor{
		Type: annotations[AnnotationActorType],
		ID:   annotations[AnnotationActorID],
	}
}
