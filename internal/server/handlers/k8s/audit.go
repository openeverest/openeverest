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

package k8s

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openeverest/openeverest/v2/pkg/events"
	"github.com/openeverest/openeverest/v2/pkg/rbac"
)

// metaObject is the subset of metav1.Object the actor stamper needs.
// Both ObjectMeta (via *Instance, *Backup, *Restore embedded pointer access)
// and any controller-runtime client.Object satisfy it.
type metaObject interface {
	GetAnnotations() map[string]string
	SetAnnotations(map[string]string)
}

// stampActor records the calling user's identity onto the object's
// annotations so the event normalizer can attribute the resulting watch
// event back to them. Returns true when at least one annotation changed,
// which the caller may use to decide whether a follow-up Update is needed
// (e.g. before a Delete to ensure the tombstone carries the deleter).
//
// No-op when the context has no JWT user (system-internal calls, tests).
// Plugin-issued service-account JWTs also flow through here; we don't
// currently distinguish them from user JWTs because rbac.GetUser collapses
// both into a Subject string.
func stampActor(ctx context.Context, obj metaObject) bool {
	user, err := rbac.GetUser(ctx)
	if err != nil || user.Subject == "" {
		return false
	}
	ann := obj.GetAnnotations()
	if ann == nil {
		ann = map[string]string{}
	}
	changed := false
	if ann[events.AnnotationActorType] != "user" {
		ann[events.AnnotationActorType] = "user"
		changed = true
	}
	if ann[events.AnnotationActorID] != user.Subject {
		ann[events.AnnotationActorID] = user.Subject
		changed = true
	}
	if changed {
		obj.SetAnnotations(ann)
	}
	return changed
}

// Compile-time assertion that *metav1.ObjectMeta satisfies metaObject — the
// helper is only useful if the resources we mutate already expose it.
var _ metaObject = &metav1.ObjectMeta{}
