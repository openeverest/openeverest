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
	"time"

	everestv1alpha1 "github.com/percona/everest-operator/api/everest/v1alpha1"
	"k8s.io/apimachinery/pkg/watch"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

// NormalizeDatabaseCluster converts a kube watch event on a DatabaseCluster
// into zero or more plugin-facing events.
func NormalizeDatabaseCluster(we watch.Event, old *everestv1alpha1.DatabaseCluster) []Event {
	obj, ok := we.Object.(*everestv1alpha1.DatabaseCluster)
	if !ok {
		return nil
	}

	ref := ResourceRef{
		Kind: "DatabaseCluster",
		Name: obj.Name,
		UID:  string(obj.UID),
	}
	if obj.Spec.Engine.Type != "" {
		ref.Engine = string(obj.Spec.Engine.Type)
	}
	if obj.Spec.Engine.Version != "" {
		ref.Version = obj.Spec.Engine.Version
	}

	rv := obj.ResourceVersion
	ns := obj.Namespace
	now := time.Now().UTC()

	var out []Event

	switch we.Type {
	case watch.Added:
		out = append(out, Event{
			ResourceVersion: rv,
			Type:            DatabaseClusterCreated,
			OccurredAt:      now,
			Namespace:       ns,
			Resource:        ref,
			NewState:        StateSnapshot{Phase: string(obj.Status.Status)},
		})
	case watch.Modified:
		newPhase := string(obj.Status.Status)
		oldPhase := ""
		if old != nil {
			oldPhase = string(old.Status.Status)
		}

		// Emit a targeted event when the phase transitions.
		switch {
		case newPhase == "Ready" && oldPhase != "Ready":
			out = append(out, Event{
				ResourceVersion: rv,
				Type:            DatabaseClusterReady,
				OccurredAt:      now,
				Namespace:       ns,
				Resource:        ref,
				PrevState:       StateSnapshot{Phase: oldPhase},
				NewState:        StateSnapshot{Phase: newPhase},
			})
		case newPhase == "Error" && oldPhase != "Error":
			out = append(out, Event{
				ResourceVersion: rv,
				Type:            DatabaseClusterFailed,
				OccurredAt:      now,
				Namespace:       ns,
				Resource:        ref,
				PrevState:       StateSnapshot{Phase: oldPhase},
				NewState:        StateSnapshot{Phase: newPhase},
			})
		default:
			out = append(out, Event{
				ResourceVersion: rv,
				Type:            DatabaseClusterUpdated,
				OccurredAt:      now,
				Namespace:       ns,
				Resource:        ref,
				PrevState:       StateSnapshot{Phase: oldPhase},
				NewState:        StateSnapshot{Phase: newPhase},
			})
		}
	case watch.Deleted:
		prevPhase := ""
		if old != nil {
			prevPhase = string(old.Status.Status)
		}
		out = append(out, Event{
			ResourceVersion: rv,
			Type:            DatabaseClusterDeleted,
			OccurredAt:      now,
			Namespace:       ns,
			Resource:        ref,
			PrevState:       StateSnapshot{Phase: prevPhase},
			NewState:        StateSnapshot{Phase: "Deleting"},
		})
	}
	return out
}

// NormalizeBackup converts a kube watch event on a DatabaseClusterBackup.
func NormalizeBackup(we watch.Event, old *backupv1alpha1.Backup) []Event {
	obj, ok := we.Object.(*backupv1alpha1.Backup)
	if !ok {
		return nil
	}

	ref := ResourceRef{
		Kind: "DatabaseClusterBackup",
		Name: obj.Name,
		UID:  string(obj.UID),
	}

	rv := obj.ResourceVersion
	ns := obj.Namespace
	now := time.Now().UTC()

	var out []Event
	newState := string(obj.Status.State)
	oldState := ""
	if old != nil {
		oldState = string(old.Status.State)
	}

	switch we.Type {
	case watch.Added:
		out = append(out, Event{
			ResourceVersion: rv,
			Type:            BackupStarted,
			OccurredAt:      now,
			Namespace:       ns,
			Resource:        ref,
			NewState:        StateSnapshot{Phase: newState},
		})
	case watch.Modified:
		switch {
		case isBackupComplete(newState) && !isBackupComplete(oldState):
			out = append(out, Event{
				ResourceVersion: rv,
				Type:            BackupCompleted,
				OccurredAt:      now,
				Namespace:       ns,
				Resource:        ref,
				PrevState:       StateSnapshot{Phase: oldState},
				NewState:        StateSnapshot{Phase: newState},
			})
		case isBackupFailed(newState) && !isBackupFailed(oldState):
			out = append(out, Event{
				ResourceVersion: rv,
				Type:            BackupFailed,
				OccurredAt:      now,
				Namespace:       ns,
				Resource:        ref,
				PrevState:       StateSnapshot{Phase: oldState},
				NewState:        StateSnapshot{Phase: newState},
			})
		}
	case watch.Deleted:
		// Backup deletion is not a distinct event type in the taxonomy.
	}
	return out
}

// NormalizeRestore converts a kube watch event on a DatabaseClusterRestore.
func NormalizeRestore(we watch.Event, old *backupv1alpha1.Restore) []Event {
	obj, ok := we.Object.(*backupv1alpha1.Restore)
	if !ok {
		return nil
	}

	ref := ResourceRef{
		Kind: "DatabaseClusterRestore",
		Name: obj.Name,
		UID:  string(obj.UID),
	}

	rv := obj.ResourceVersion
	ns := obj.Namespace
	now := time.Now().UTC()

	var out []Event
	newState := string(obj.Status.State)
	oldState := ""
	if old != nil {
		oldState = string(old.Status.State)
	}

	switch we.Type {
	case watch.Added:
		out = append(out, Event{
			ResourceVersion: rv,
			Type:            RestoreStarted,
			OccurredAt:      now,
			Namespace:       ns,
			Resource:        ref,
			NewState:        StateSnapshot{Phase: newState},
		})
	case watch.Modified:
		switch {
		case isRestoreComplete(newState) && !isRestoreComplete(oldState):
			out = append(out, Event{
				ResourceVersion: rv,
				Type:            RestoreCompleted,
				OccurredAt:      now,
				Namespace:       ns,
				Resource:        ref,
				PrevState:       StateSnapshot{Phase: oldState},
				NewState:        StateSnapshot{Phase: newState},
			})
		case isRestoreFailed(newState) && !isRestoreFailed(oldState):
			out = append(out, Event{
				ResourceVersion: rv,
				Type:            RestoreFailed,
				OccurredAt:      now,
				Namespace:       ns,
				Resource:        ref,
				PrevState:       StateSnapshot{Phase: oldState},
				NewState:        StateSnapshot{Phase: newState},
			})
		}
	case watch.Deleted:
		// Restore deletion is not a distinct event type.
	}
	return out
}

// NormalizeInstance converts a kube watch event on an Instance.
func NormalizeInstance(we watch.Event) []Event {
	obj, ok := we.Object.(*corev1alpha1.Instance)
	if !ok {
		return nil
	}

	ref := ResourceRef{
		Kind: "Instance",
		Name: obj.Name,
		UID:  string(obj.UID),
	}

	rv := obj.ResourceVersion
	ns := obj.Namespace
	now := time.Now().UTC()

	var out []Event
	switch we.Type {
	case watch.Added:
		out = append(out, Event{
			ResourceVersion: rv,
			Type:            InstanceCreated,
			OccurredAt:      now,
			Namespace:       ns,
			Resource:        ref,
			NewState:        StateSnapshot{Phase: string(obj.Status.Phase)},
		})
	case watch.Deleted:
		out = append(out, Event{
			ResourceVersion: rv,
			Type:            InstanceDeleted,
			OccurredAt:      now,
			Namespace:       ns,
			Resource:        ref,
			PrevState:       StateSnapshot{Phase: string(obj.Status.Phase)},
		})
	}
	return out
}

func isBackupComplete(state string) bool {
	return state == "Succeeded" || state == "Completed"
}

func isBackupFailed(state string) bool {
	return state == "Failed" || state == "Error"
}

func isRestoreComplete(state string) bool {
	return state == "Succeeded" || state == "Completed"
}

func isRestoreFailed(state string) bool {
	return state == "Failed" || state == "Error"
}
