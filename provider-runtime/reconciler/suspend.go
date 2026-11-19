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

	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

const (
	suspendingMessage = "Waiting for the instance to stop"
	resumingMessage   = "Waiting for the instance to start"
)

// suspendAwareStatus reconciles the provider's report with spec.suspend, so a
// provider only has to report Suspended once the operator has stopped the
// engine.
func suspendAwareStatus(in *v1alpha1.Instance, reported controller.Status) controller.Status {
	if in.SuspendRequested() {
		if reported.Reason == v1alpha1.ReasonSuspended {
			return reported
		}
		reported.Converged = false
		reported.Reason = v1alpha1.ReasonSuspending
		if reported.Message == "" {
			reported.Message = suspendingMessage
		}
		return reported
	}

	ready := meta.FindStatusCondition(in.Status.Conditions, v1alpha1.ConditionReady)
	wasSuspended := ready != nil && ready.Reason == v1alpha1.ReasonSuspended
	if wasSuspended && (reported.Reason == v1alpha1.ReasonSuspended || !reported.Converged) {
		reported.Converged = false
		reported.Reason = v1alpha1.ReasonResuming
		if reported.Message == "" || reported.Available == controller.Unavailable {
			reported.Message = resumingMessage
		}
		return reported
	}

	if reported.Reason == v1alpha1.ReasonSuspended {
		reported.Converged = false
		reported.Reason = ""
		reported.Problems = append(slices.Clip(reported.Problems), controller.Problem{
			Reason:  "StoppedOutsideOpenEverest",
			Message: "the engine was stopped outside OpenEverest",
		})
		return reported
	}

	return reported
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
