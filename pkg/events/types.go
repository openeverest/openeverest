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

// Package events defines the plugin-facing lifecycle event model.
package events

import "time"

// Type identifies the kind of lifecycle event.
type Type string

// Lifecycle event types.
const (
	DatabaseClusterCreated Type = "database-cluster.created"
	DatabaseClusterReady   Type = "database-cluster.ready"
	DatabaseClusterUpdated Type = "database-cluster.updated"
	DatabaseClusterDeleted Type = "database-cluster.deleted"
	DatabaseClusterFailed  Type = "database-cluster.failed"

	BackupStarted   Type = "backup.started"
	BackupCompleted Type = "backup.completed"
	BackupFailed    Type = "backup.failed"

	RestoreStarted   Type = "restore.started"
	RestoreCompleted Type = "restore.completed"
	RestoreFailed    Type = "restore.failed"

	InstanceCreated Type = "instance.created"
	InstanceDeleted Type = "instance.deleted"
)

// AllTypes is the complete set of recognised event types.
var AllTypes = []Type{
	DatabaseClusterCreated, DatabaseClusterReady,
	DatabaseClusterUpdated, DatabaseClusterDeleted,
	DatabaseClusterFailed,
	BackupStarted, BackupCompleted, BackupFailed,
	RestoreStarted, RestoreCompleted, RestoreFailed,
	InstanceCreated, InstanceDeleted,
}

// ResourceRef identifies the Kubernetes resource that triggered the event.
type ResourceRef struct {
	Kind    string `json:"kind"`
	Name    string `json:"name"`
	UID     string `json:"uid"`
	Engine  string `json:"engine,omitempty"`
	Version string `json:"version,omitempty"`
}

// StateSnapshot captures the phase of a resource at a point in time.
type StateSnapshot struct {
	Phase string `json:"phase,omitempty"`
}

// Actor describes who or what triggered the event.
type Actor struct {
	Type string `json:"type"` // "user" or "system"
	ID   string `json:"id"`
}

// Event is the normalised envelope streamed to plugins.
type Event struct {
	ResourceVersion string        `json:"resourceVersion"`
	Type            Type          `json:"type"`
	OccurredAt      time.Time     `json:"occurredAt"`
	Namespace       string        `json:"namespace"`
	Resource        ResourceRef   `json:"resource"`
	PrevState       StateSnapshot `json:"prevState,omitempty"`
	NewState        StateSnapshot `json:"newState,omitempty"`
	Actor           Actor         `json:"actor,omitempty"`
}
