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
	"encoding/json"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// deletionAttributionWindow is how close to the deletion request the last write that
// stamped the actor annotations must be for the deletion to be attributed to that actor.
// The API server stamps the deleting user right before deleting.
const deletionAttributionWindow = 30 * time.Second

// The OpenEverest API stamps the acting user in the actor annotations. Objects changed
// directly in Kubernetes (kubectl, GitOps tools) carry no such annotations, or stale ones
// left by an earlier API call, so the managed fields decide which writer to trust: the
// annotations are only used when the manager that made the change also owns them.

// CreatorActor returns who created obj.
func CreatorActor(obj metav1.Object) Actor {
	if actor := ActorFromAnnotations(obj.GetAnnotations()); !actor.IsZero() {
		return actor
	}
	// The creator wrote first, so its managed fields entry is the oldest one.
	var creator *metav1.ManagedFieldsEntry
	for _, entry := range mainResourceEntries(obj) {
		if creator == nil || entryTime(entry).Before(entryTime(*creator)) {
			creator = &entry
		}
	}
	return clientActor(creator)
}

// UpdaterActor returns who made the latest change to the spec of obj.
func UpdaterActor(obj metav1.Object) Actor {
	var updater *metav1.ManagedFieldsEntry
	for _, entry := range mainResourceEntries(obj) {
		if !ownsField(entry, "f:spec") {
			continue
		}
		if updater == nil || !entryTime(entry).Before(entryTime(*updater)) {
			updater = &entry
		}
	}
	if updater != nil && ownsActorAnnotation(*updater) {
		return ActorFromAnnotations(obj.GetAnnotations())
	}
	return clientActor(updater)
}

// DeleterActor returns who requested the deletion of obj, or the zero Actor when it is
// unknown: Kubernetes does not record who deleted an object, so only deletions through
// the OpenEverest API, which stamps the deleting user first, are attributed.
func DeleterActor(obj metav1.Object) Actor {
	deletion := obj.GetDeletionTimestamp()
	if deletion == nil {
		return Actor{}
	}
	var latest *metav1.ManagedFieldsEntry
	for _, entry := range mainResourceEntries(obj) {
		if latest == nil || !entryTime(entry).Before(entryTime(*latest)) {
			latest = &entry
		}
	}
	if latest == nil || !ownsActorAnnotation(*latest) {
		return Actor{}
	}
	if deletion.Sub(entryTime(*latest)) > deletionAttributionWindow {
		return Actor{}
	}
	return ActorFromAnnotations(obj.GetAnnotations())
}

// mainResourceEntries returns the managed fields entries of writes to the object itself,
// skipping status and other subresources that controllers write.
func mainResourceEntries(obj metav1.Object) []metav1.ManagedFieldsEntry {
	entries := make([]metav1.ManagedFieldsEntry, 0, len(obj.GetManagedFields()))
	for _, entry := range obj.GetManagedFields() {
		if entry.Subresource == "" {
			entries = append(entries, entry)
		}
	}
	return entries
}

func entryTime(entry metav1.ManagedFieldsEntry) time.Time {
	if entry.Time == nil {
		return time.Time{}
	}
	return entry.Time.Time
}

func clientActor(entry *metav1.ManagedFieldsEntry) Actor {
	if entry == nil || entry.Manager == "" {
		return Actor{}
	}
	return Actor{Type: ActorTypeKubernetesClient, ID: entry.Manager}
}

func ownsActorAnnotation(entry metav1.ManagedFieldsEntry) bool {
	return ownsField(entry, "f:metadata", "f:annotations", "f:"+AnnotationActorID)
}

// ownsField reports whether entry owns the field at path, given in managed fields
// notation (e.g. "f:metadata", "f:annotations", "f:example.com/key").
func ownsField(entry metav1.ManagedFieldsEntry, path ...string) bool {
	if entry.FieldsV1 == nil {
		return false
	}
	var node map[string]json.RawMessage
	if err := json.Unmarshal(entry.FieldsV1.GetRawBytes(), &node); err != nil {
		return false
	}
	for i, key := range path {
		raw, ok := node[key]
		if !ok {
			return false
		}
		if i == len(path)-1 {
			return true
		}
		node = nil
		if err := json.Unmarshal(raw, &node); err != nil {
			return false
		}
	}
	return false
}
