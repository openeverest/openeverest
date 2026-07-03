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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	extensionsv1alpha1 "github.com/openeverest/openeverest/v2/api/extensions/v1alpha1"
	"github.com/openeverest/openeverest/v2/pkg/common"
)

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
	userActor := ActorFromAnnotations(obj.GetAnnotations())

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
			Actor:           userActor,
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
				Actor:           systemActor,
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
				Actor:           systemActor,
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
	userActor := ActorFromAnnotations(obj.GetAnnotations())

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
			Actor:           userActor,
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
				Actor:           systemActor,
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
				Actor:           systemActor,
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
	// Both create and delete are user-triggered API calls, so the actor
	// recorded on the object by the API server applies to both branches.
	actor := ActorFromAnnotations(obj.GetAnnotations())

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
			Actor:           actor,
		})
	case watch.Deleted:
		out = append(out, Event{
			ResourceVersion: rv,
			Type:            InstanceDeleted,
			OccurredAt:      now,
			Namespace:       ns,
			Resource:        ref,
			PrevState:       StateSnapshot{Phase: string(obj.Status.Phase)},
			Actor:           actor,
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

// NormalizeNamespace converts a kube watch event on a managed Namespace
// into a namespace.added / namespace.removed event.
func NormalizeNamespace(we watch.Event) []Event {
	obj, ok := we.Object.(*corev1.Namespace)
	if !ok {
		return nil
	}
	ref := ResourceRef{
		Kind: "Namespace",
		Name: obj.Name,
		UID:  string(obj.UID),
	}
	now := time.Now().UTC()
	switch we.Type {
	case watch.Added:
		return []Event{{
			ResourceVersion: obj.ResourceVersion,
			Type:            NamespaceAdded,
			OccurredAt:      now,
			Namespace:       obj.Name,
			Resource:        ref,
		}}
	case watch.Deleted:
		return []Event{{
			ResourceVersion: obj.ResourceVersion,
			Type:            NamespaceRemoved,
			OccurredAt:      now,
			Namespace:       obj.Name,
			Resource:        ref,
		}}
	}
	return nil
}

// NormalizePlugin converts a kube watch event on a Plugin CR into a
// plugin.installed / plugin.uninstalled event. Plugin is cluster-scoped.
func NormalizePlugin(we watch.Event) []Event {
	obj, ok := we.Object.(*extensionsv1alpha1.Plugin)
	if !ok {
		return nil
	}
	ref := ResourceRef{
		Kind: "Plugin",
		Name: obj.Name,
		UID:  string(obj.UID),
	}
	now := time.Now().UTC()
	switch we.Type {
	case watch.Added:
		return []Event{{
			ResourceVersion: obj.ResourceVersion,
			Type:            PluginInstalled,
			OccurredAt:      now,
			Resource:        ref,
		}}
	case watch.Deleted:
		return []Event{{
			ResourceVersion: obj.ResourceVersion,
			Type:            PluginUninstalled,
			OccurredAt:      now,
			Resource:        ref,
		}}
	}
	return nil
}

// NormalizeInstalledExtension converts a kube watch event on an
// InstalledExtension CR into a plugin.enabled / plugin.disabled event when
// the rolled-up phase crosses the Installed boundary.
func NormalizeInstalledExtension(we watch.Event, old *extensionsv1alpha1.InstalledExtension) []Event {
	obj, ok := we.Object.(*extensionsv1alpha1.InstalledExtension)
	if !ok {
		return nil
	}
	pluginName := ""
	if obj.Spec.Plugin != nil {
		pluginName = obj.Spec.Plugin.PluginCRName
	}
	ref := ResourceRef{
		Kind: "InstalledExtension",
		Name: obj.Name,
		UID:  string(obj.UID),
	}
	now := time.Now().UTC()
	newPhase := string(obj.Status.Phase)
	oldPhase := ""
	if old != nil {
		oldPhase = string(old.Status.Phase)
	}
	installed := func(p string) bool {
		return p == string(extensionsv1alpha1.InstalledExtensionPhaseInstalled)
	}

	switch we.Type {
	case watch.Added:
		if installed(newPhase) {
			return []Event{{
				ResourceVersion: obj.ResourceVersion,
				Type:            PluginEnabled,
				OccurredAt:      now,
				Namespace:       obj.Namespace,
				Resource:        ref,
				NewState:        StateSnapshot{Phase: newPhase},
				Actor:           Actor{Type: "plugin", ID: pluginName},
			}}
		}
	case watch.Modified:
		switch {
		case installed(newPhase) && !installed(oldPhase):
			return []Event{{
				ResourceVersion: obj.ResourceVersion,
				Type:            PluginEnabled,
				OccurredAt:      now,
				Namespace:       obj.Namespace,
				Resource:        ref,
				PrevState:       StateSnapshot{Phase: oldPhase},
				NewState:        StateSnapshot{Phase: newPhase},
				Actor:           Actor{Type: "plugin", ID: pluginName},
			}}
		case !installed(newPhase) && installed(oldPhase):
			return []Event{{
				ResourceVersion: obj.ResourceVersion,
				Type:            PluginDisabled,
				OccurredAt:      now,
				Namespace:       obj.Namespace,
				Resource:        ref,
				PrevState:       StateSnapshot{Phase: oldPhase},
				NewState:        StateSnapshot{Phase: newPhase},
				Actor:           Actor{Type: "plugin", ID: pluginName},
			}}
		}
	case watch.Deleted:
		if installed(oldPhase) || installed(newPhase) {
			return []Event{{
				ResourceVersion: obj.ResourceVersion,
				Type:            PluginDisabled,
				OccurredAt:      now,
				Namespace:       obj.Namespace,
				Resource:        ref,
				PrevState:       StateSnapshot{Phase: oldPhase},
				Actor:           Actor{Type: "plugin", ID: pluginName},
			}}
		}
	}
	return nil
}

// NormalizeEverestSettings converts a kube watch event on the Everest
// settings ConfigMap into a settings.updated event. Non-settings ConfigMaps
// in the watched namespace are filtered out.
func NormalizeEverestSettings(we watch.Event) []Event {
	obj, ok := we.Object.(*corev1.ConfigMap)
	if !ok {
		return nil
	}
	if obj.Name != common.EverestSettingsConfigMapName {
		return nil
	}
	// Only Modified is meaningful: Added fires on every controller restart for
	// the bootstrap configmap, and Deleted is effectively unrecoverable here.
	if we.Type != watch.Modified {
		return nil
	}
	return []Event{{
		ResourceVersion: obj.ResourceVersion,
		Type:            SettingsUpdated,
		OccurredAt:      time.Now().UTC(),
		Namespace:       obj.Namespace,
		Resource: ResourceRef{
			Kind: "ConfigMap",
			Name: obj.Name,
			UID:  string(obj.UID),
		},
	}}
}
