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

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

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
	// Seq is a monotonic counter the Hub assigns to every event as it's
	// broadcast, regardless of source. Unlike ResourceVersion, it's always
	// set, so it's the one cursor value that covers direct-publish events
	// (UserLogin, SettingsUpdated, ...) too. See Epoch and Cursor.
	Seq uint64 `json:"seq"`
	// Epoch identifies the Hub instance that assigned Seq. It's generated
	// once when the Hub starts, so a Seq from a previous process is never
	// mistaken for one from this one after a restart.
	Epoch           string        `json:"epoch"`
	ResourceVersion string        `json:"resourceVersion"`
	Type            Type          `json:"type"`
	OccurredAt      time.Time     `json:"occurredAt"`
	Namespace       string        `json:"namespace"`
	Resource        ResourceRef   `json:"resource"`
	PrevState       StateSnapshot `json:"prevState,omitempty"`
	NewState        StateSnapshot `json:"newState,omitempty"`
	Actor           Actor         `json:"actor,omitempty"`
}

// Cursor is a subscriber's position in the event stream: the last Seq it
// has seen, scoped to the Epoch it came from. It travels over the wire as
// a single opaque string (see ParseCursor / String), so clients only ever
// need to persist one value instead of tracking ResourceVersion and Seq
// separately.
type Cursor struct {
	Epoch string
	Seq   uint64
}

// String encodes the cursor to its wire form: "<epoch>:<seq>".
func (c Cursor) String() string {
	return c.Epoch + ":" + strconv.FormatUint(c.Seq, 10)
}

// ParseCursor decodes the "<epoch>:<seq>" wire form used by the `since`
// query param. A parse error should be treated the same as an epoch
// mismatch by the caller: the cursor can't be trusted, so resync (410).
func ParseCursor(raw string) (Cursor, error) {
	epoch, seqStr, ok := strings.Cut(raw, ":")
	if !ok {
		return Cursor{}, fmt.Errorf("invalid cursor %q: missing ':' separator", raw)
	}
	seq, err := strconv.ParseUint(seqStr, 10, 64)
	if err != nil {
		return Cursor{}, fmt.Errorf("invalid cursor %q: %w", raw, err)
	}
	return Cursor{Epoch: epoch, Seq: seq}, nil
}
