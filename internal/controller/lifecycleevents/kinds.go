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
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	backupState := func(o *backupv1alpha1.Backup) string { return string(o.Status.State) }
	restoreState := func(o *backupv1alpha1.Restore) string { return string(o.Status.State) }
	importState := func(o *backupv1alpha1.BackupImport) string { return string(o.Status.State) }
	extensionPhase := func(o *extensionsv1alpha1.InstalledExtension) string { return string(o.Status.Phase) }

	return []Kind{
		{
			New: func() client.Object { return &corev1alpha1.Instance{} },
			// The status word, the Ready condition's reason, is the state
			// kubectl and the UI show. The phase only says Pending,
			// Provisioned or Terminating.
			Phase:  phaseOf(instanceStatusWord),
			Failed: objectIs(instanceNeedsAttention),
			Owner: ownerOf(func(o *corev1alpha1.Instance) (string, string) {
				return o.Name, o.Spec.ProviderRef.Name
			}),
		},
		{
			New:    func() client.Object { return &backupv1alpha1.Backup{} },
			Phase:  phaseOf(backupState),
			Failed: stateIn(backupState, string(backupv1alpha1.BackupStateFailed), string(backupv1alpha1.BackupStateError)),
			Owner: ownerOf(func(o *backupv1alpha1.Backup) (string, string) {
				if o.Spec.Origin.InstanceRef == nil {
					return "", ""
				}
				return o.Spec.Origin.InstanceRef.Name, ""
			}),
		},
		{
			New:    func() client.Object { return &backupv1alpha1.Restore{} },
			Phase:  phaseOf(restoreState),
			Failed: stateIn(restoreState, string(backupv1alpha1.RestoreStateFailed), string(backupv1alpha1.RestoreStateError)),
			Owner: ownerOf(func(o *backupv1alpha1.Restore) (string, string) {
				return o.Spec.InstanceRef.Name, ""
			}),
		},
		{
			New:    func() client.Object { return &backupv1alpha1.BackupImport{} },
			Phase:  phaseOf(importState),
			Failed: stateIn(importState, string(backupv1alpha1.BackupImportStateFailed), string(backupv1alpha1.BackupImportStateError)),
		},
		{New: func() client.Object { return &backupv1alpha1.BackupStorage{} }},
		{New: func() client.Object { return &monitoringv1alpha1.MonitoringDestination{} }},
		{New: func() client.Object { return &extensionsv1alpha1.Plugin{} }},
		{
			New:    func() client.Object { return &extensionsv1alpha1.InstalledExtension{} },
			Phase:  phaseOf(extensionPhase),
			Failed: stateIn(extensionPhase, string(extensionsv1alpha1.InstalledExtensionPhaseFailed)),
		},
		{
			New: func() client.Object { return &corev1.Namespace{} },
			Include: func(o client.Object) bool {
				return o.GetLabels()[common.KubernetesManagedByLabel] == common.Everest
			},
		},
	}
}

// instanceStatusWord returns the reason of the Unattended condition while
// core says nobody attends to the Instance's current spec, and otherwise the
// reason of its Ready condition, or "" before a provider has reported on it.
func instanceStatusWord(o *corev1alpha1.Instance) string {
	if unattended := instanceUnattended(o); unattended != nil {
		return unattended.Reason
	}
	if ready := meta.FindStatusCondition(o.Status.Conditions, corev1alpha1.ConditionReady); ready != nil {
		return ready.Reason
	}
	return ""
}

// instanceNeedsAttention reports whether something is wrong with an Instance
// that a person may want to look at, stalled or not, including that nobody
// attends to it.
func instanceNeedsAttention(o *corev1alpha1.Instance) bool {
	return instanceUnattended(o) != nil ||
		meta.IsStatusConditionTrue(o.Status.Conditions, corev1alpha1.ConditionNeedsAttention)
}

// instanceUnattended returns the Unattended condition while it is to be
// honoured: core wrote it True or Unknown, and no provider has reported on
// the current spec since, which would have answered it.
func instanceUnattended(o *corev1alpha1.Instance) *metav1.Condition {
	if o.Status.ObservedGeneration >= o.Generation {
		return nil
	}
	c := meta.FindStatusCondition(o.Status.Conditions, corev1alpha1.ConditionUnattended)
	if c == nil || c.Status == metav1.ConditionFalse {
		return nil
	}
	return c
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

// ownerOf adapts a typed owner accessor to Kind.Owner.
func ownerOf[T client.Object](get func(T) (string, string)) func(client.Object) (string, string) {
	return func(o client.Object) (string, string) {
		obj, ok := o.(T)
		if !ok {
			return "", ""
		}
		return get(obj)
	}
}

// objectIs adapts a typed predicate to Kind.Failed.
func objectIs[T client.Object](pred func(T) bool) func(client.Object) bool {
	return func(o client.Object) bool {
		obj, ok := o.(T)
		return ok && pred(obj)
	}
}

// stateIn returns a Kind.Failed that reports whether the state get returns
// is one of states.
func stateIn[T client.Object](get func(T) string, states ...string) func(client.Object) bool {
	return objectIs(func(o T) bool { return slices.Contains(states, get(o)) })
}
