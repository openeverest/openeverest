// everest
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

package instancestatus

import (
	"strings"
	"unicode"
)

// labels is the dictionary from a status word to what a person reads. It
// only improves the wording: a reason that is missing here is still shown,
// spelled out by Label, and nothing but the wording depends on a reason
// being listed.
//
//nolint:gochecknoglobals,gosec // immutable dictionary; "secret" and "credentials" are words in it, not values
var labels = map[string]string{
	"Ready":     "Ready",
	"Suspended": "Stopped",
	"Creating":  "Creating",
	"Deleting":  "Deleting",

	"ApplyingChanges": "Applying changes",
	"Restoring":       "Restoring",
	"Upgrading":       "Upgrading",
	"StorageResizing": "Resizing storage",
	"Scaling":         "Scaling",

	"ValidationFailed": "Invalid configuration",
	"AwaitingApproval": "Awaiting approval",
	"RetriesExhausted": "Maintenance failed",

	"ProviderNotFound":   "Provider not found",
	"UserSecretNotFound": "Secret not found",
	"InvalidReference":   "Invalid reference",

	"Unschedulable":              "Pods can't be scheduled",
	"CrashLoopBackOff":           "Crash loop",
	"ImagePullBackOff":           "Image can't be pulled",
	"CreateContainerConfigError": "Container can't be created",
	"NotReady":                   "Pods starting",

	"ReconcileError":           "Provider error",
	"EngineError":              "Engine error",
	"ProgressDeadlineExceeded": "Not progressing",

	"BackupStorageNotFound":            "Backup storage not found",
	"BackupClassNotFound":              "Backup class not found",
	"BackupStorageCredentialsNotFound": "Backup credentials not found",

	ReasonNotPickedUp: "Not picked up by provider",
}

// Label is how a person reads a reason. A reason the dictionary does not
// have, such as one a provider made up for its own operation, is split into
// words: "RebalancingShards" becomes "Rebalancing shards".
func Label(reason string) string {
	if label, ok := labels[reason]; ok {
		return label
	}
	return splitWords(reason)
}

// splitWords puts a space before every word of a CamelCase name and lowers
// the first letter of every word but the first. A run of capitals is one
// word and keeps its case: "PVCResizeFailed" becomes "PVC resize failed".
func splitWords(name string) string {
	runes := []rune(name)
	var b strings.Builder
	for i, r := range runes {
		if i == 0 || !startsWord(runes, i) {
			b.WriteRune(r)
			continue
		}
		b.WriteRune(' ')
		if i+1 < len(runes) && unicode.IsLower(runes[i+1]) {
			r = unicode.ToLower(r)
		}
		b.WriteRune(r)
	}
	return b.String()
}

// startsWord reports whether the capital at i begins a word: it follows a
// lowercase letter or a digit, or it ends a run of capitals.
func startsWord(runes []rune, i int) bool {
	if !unicode.IsUpper(runes[i]) {
		return false
	}
	if !unicode.IsUpper(runes[i-1]) {
		return unicode.IsLetter(runes[i-1]) || unicode.IsDigit(runes[i-1])
	}
	return i+1 < len(runes) && unicode.IsLower(runes[i+1])
}
