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

package lifecycleevents

import (
	"slices"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	extensionsv1alpha1 "github.com/openeverest/openeverest/v2/api/extensions/v1alpha1"
	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
	"github.com/openeverest/openeverest/v2/pkg/common"
)

// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch
// +kubebuilder:rbac:groups=core.openeverest.io,resources=instances,verbs=get;list;watch
// +kubebuilder:rbac:groups=backup.openeverest.io,resources=backups;restores;backupimports;backupstorages,verbs=get;list;watch
// +kubebuilder:rbac:groups=extensions.openeverest.io,resources=plugins;installedextensions,verbs=get;list;watch
// +kubebuilder:rbac:groups=monitoring.openeverest.io,resources=monitoringdestinations,verbs=get;list;watch
// +kubebuilder:rbac:groups=events.k8s.io,resources=events,verbs=create

// DefaultKinds are the kinds OpenEverest records lifecycle events for.
func DefaultKinds() []Kind {
	return []Kind{
		{
			New:    func() client.Object { return &corev1alpha1.Instance{} },
			Phase:  phaseOf(func(o *corev1alpha1.Instance) string { return string(o.Status.Phase) }),
			Failed: is(string(corev1alpha1.InstancePhaseFailed)),
		},
		{
			New:    func() client.Object { return &backupv1alpha1.Backup{} },
			Phase:  phaseOf(func(o *backupv1alpha1.Backup) string { return string(o.Status.State) }),
			Failed: is(string(backupv1alpha1.BackupStateFailed), string(backupv1alpha1.BackupStateError)),
		},
		{
			New:    func() client.Object { return &backupv1alpha1.Restore{} },
			Phase:  phaseOf(func(o *backupv1alpha1.Restore) string { return string(o.Status.State) }),
			Failed: is(string(backupv1alpha1.RestoreStateFailed), string(backupv1alpha1.RestoreStateError)),
		},
		{
			New:    func() client.Object { return &backupv1alpha1.BackupImport{} },
			Phase:  phaseOf(func(o *backupv1alpha1.BackupImport) string { return string(o.Status.State) }),
			Failed: is(string(backupv1alpha1.BackupImportStateFailed), string(backupv1alpha1.BackupImportStateError)),
		},
		{New: func() client.Object { return &backupv1alpha1.BackupStorage{} }},
		{New: func() client.Object { return &monitoringv1alpha1.MonitoringDestination{} }},
		{New: func() client.Object { return &extensionsv1alpha1.Plugin{} }},
		{
			New:    func() client.Object { return &extensionsv1alpha1.InstalledExtension{} },
			Phase:  phaseOf(func(o *extensionsv1alpha1.InstalledExtension) string { return string(o.Status.Phase) }),
			Failed: is(string(extensionsv1alpha1.InstalledExtensionPhaseFailed)),
		},
		{
			New: func() client.Object { return &corev1.Namespace{} },
			Include: func(o client.Object) bool {
				return o.GetLabels()[common.KubernetesManagedByLabel] == common.Everest
			},
		},
	}
}

// phaseOf adapts a typed phase accessor to Kind.Phase.
func phaseOf[T client.Object](get func(T) string) func(client.Object) string {
	return func(o client.Object) string {
		obj, ok := o.(T)
		if !ok {
			return ""
		}
		return get(obj)
	}
}

func is(phases ...string) func(string) bool {
	return func(phase string) bool {
		return slices.Contains(phases, phase)
	}
}
