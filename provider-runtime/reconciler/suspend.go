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

package reconciler

import (
	"slices"

	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

const (
	suspendingMessage = "Waiting for the instance to stop"
	resumingMessage   = "Waiting for the instance to start"
)

// suspendAwarePhase reconciles the phase the provider reports with
// spec.suspend, so a provider only has to report Suspended once the operator
// has stopped the engine. While a suspend is requested, any phase other than
// Suspended or Failed means the operator is still stopping it. After a
// resume, a provider still reporting Suspending or Suspended, or a start-up
// phase following a suspension, means the operator is still starting it.
func suspendAwarePhase(in *v1alpha1.Instance, reported v1alpha1.InstancePhase, message string) (v1alpha1.InstancePhase, string) {
	if in.SuspendRequested() {
		if reported == v1alpha1.InstancePhaseSuspended || reported == v1alpha1.InstancePhaseFailed {
			return reported, message
		}
		if reported != v1alpha1.InstancePhaseSuspending || message == "" {
			message = suspendingMessage
		}
		return v1alpha1.InstancePhaseSuspending, message
	}

	stillStopped := reported == v1alpha1.InstancePhaseSuspending || reported == v1alpha1.InstancePhaseSuspended
	startingAfterSuspend := isSuspendPhase(in.Status.Phase) && slices.Contains([]v1alpha1.InstancePhase{
		v1alpha1.InstancePhasePending,
		v1alpha1.InstancePhaseProvisioning,
		v1alpha1.InstancePhaseInitializing,
		v1alpha1.InstancePhaseUpdating,
	}, reported)
	if stillStopped || startingAfterSuspend {
		if stillStopped || message == "" {
			message = resumingMessage
		}
		return v1alpha1.InstancePhaseResuming, message
	}
	return reported, message
}

func isSuspendPhase(phase v1alpha1.InstancePhase) bool {
	return phase == v1alpha1.InstancePhaseSuspending ||
		phase == v1alpha1.InstancePhaseSuspended ||
		phase == v1alpha1.InstancePhaseResuming
}

// suspendChanged passes Instance updates that suspend or resume it, so the
// Backups and Restores held while it is suspended continue once it resumes.
func suspendChanged() predicate.Funcs {
	return predicate.Funcs{
		CreateFunc:  func(event.CreateEvent) bool { return false },
		DeleteFunc:  func(event.DeleteEvent) bool { return false },
		GenericFunc: func(event.GenericEvent) bool { return false },
		UpdateFunc: func(e event.UpdateEvent) bool {
			old, oldOK := e.ObjectOld.(*v1alpha1.Instance)
			updated, updatedOK := e.ObjectNew.(*v1alpha1.Instance)
			return oldOK && updatedOK && old.SuspendRequested() != updated.SuspendRequested()
		},
	}
}
