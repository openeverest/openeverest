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
	BackupStarted   Type = "backup.started"
	BackupCompleted Type = "backup.completed"
	BackupFailed    Type = "backup.failed"

	RestoreStarted   Type = "restore.started"
	RestoreCompleted Type = "restore.completed"
	RestoreFailed    Type = "restore.failed"

	InstanceCreated Type = "instance.created"
	InstanceDeleted Type = "instance.deleted"

	UserLogin       Type = "user.login"
	UserLoginFailed Type = "user.login-failed"
	UserLogout      Type = "user.logout"

	PluginInstalled   Type = "plugin.installed"
	PluginUninstalled Type = "plugin.uninstalled"
	PluginEnabled     Type = "plugin.enabled"
	PluginDisabled    Type = "plugin.disabled"

	NamespaceAdded   Type = "namespace.added"
	NamespaceRemoved Type = "namespace.removed"

	SettingsUpdated Type = "settings.updated"
)

// AllTypes is the complete set of recognised event types.
var AllTypes = []Type{
	BackupStarted, BackupCompleted, BackupFailed,
	RestoreStarted, RestoreCompleted, RestoreFailed,
	InstanceCreated, InstanceDeleted,
	UserLogin, UserLoginFailed, UserLogout,
	PluginInstalled, PluginUninstalled, PluginEnabled, PluginDisabled,
	NamespaceAdded, NamespaceRemoved,
	SettingsUpdated,
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

// Actor describes who or what triggered the event. Both fields use
// `omitempty` so an unattributed event simply omits the `actor` object
// rather than emitting `{"type":"","id":""}` and looking deceptively
// populated to audit consumers.
type Actor struct {
	Type string `json:"type,omitempty"` // "user", "system", "plugin"
	ID   string `json:"id,omitempty"`
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
