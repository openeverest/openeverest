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

package monitoring

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
)

// BindingReady derives the Ready condition of binding b from its Accepted
// and Configured conditions and its destination dst, nil when it does not
// exist. Pass a zero Condition for one that is missing or was reported for
// an older generation of b. The first False input, in the order Accepted,
// Configured, destination, makes it False; otherwise any input not yet True
// makes it Pending; otherwise it is True. Only the destination's Ready reason
// is quoted, never its message, which can name Secrets or carry backend
// errors that readers of the binding and the Instance may not see otherwise.
func BindingReady(
	b *monitoringv1alpha1.MonitoringBinding,
	accepted, configured metav1.Condition,
	dst *monitoringv1alpha1.MonitoringDestination,
) metav1.Condition {
	name := b.Spec.DestinationRef.Name
	dstStatus, dstReason := DestinationReadiness(dst)
	var (
		reason monitoringv1alpha1.BindingReadyReason
		msg    string
	)
	switch {
	case accepted.Status == metav1.ConditionFalse:
		reason, msg = monitoringv1alpha1.ReasonBindingNotAccepted, fmt.Sprintf("Binding is not Accepted (%s)", accepted.Reason)
	case configured.Status == metav1.ConditionFalse:
		reason, msg = monitoringv1alpha1.ReasonBindingNotConfigured, fmt.Sprintf("Binding is not Configured (%s)", configured.Reason)
	case dstStatus == metav1.ConditionFalse:
		reason, msg = monitoringv1alpha1.ReasonDestinationNotReady, fmt.Sprintf("MonitoringDestination %q is not Ready (%s)", name, dstReason)
	case accepted.Status != metav1.ConditionTrue:
		reason, msg = monitoringv1alpha1.ReasonBindingReadyPending, "Waiting for the binding to be Accepted"
	case configured.Status != metav1.ConditionTrue:
		reason, msg = monitoringv1alpha1.ReasonBindingReadyPending, "Waiting for the binding to be Configured"
	case dst == nil:
		reason, msg = monitoringv1alpha1.ReasonBindingReadyPending, fmt.Sprintf("MonitoringDestination %q not found", name)
	case dstStatus != metav1.ConditionTrue:
		reason, msg = monitoringv1alpha1.ReasonBindingReadyPending, fmt.Sprintf("Waiting for MonitoringDestination %q to report Ready", name)
	default:
		reason, msg = monitoringv1alpha1.ReasonBindingReady, fmt.Sprintf("Configured, and MonitoringDestination %q is Ready", name)
	}
	return NewCondition(b, monitoringv1alpha1.MonitoringBindingConditionReady, reason.Status(), string(reason), msg)
}

// DestinationReadinessChanged passes MonitoringDestination updates that
// change what BindingReady quotes: the status and reason of the
// destination's Ready condition at its current generation.
func DestinationReadinessChanged() predicate.Funcs {
	return predicate.Funcs{UpdateFunc: func(e event.UpdateEvent) bool {
		before, okOld := e.ObjectOld.(*monitoringv1alpha1.MonitoringDestination)
		after, okNew := e.ObjectNew.(*monitoringv1alpha1.MonitoringDestination)
		if !okOld || !okNew {
			return false
		}
		beforeStatus, beforeReason := DestinationReadiness(before)
		afterStatus, afterReason := DestinationReadiness(after)
		return beforeStatus != afterStatus || beforeReason != afterReason
	}}
}

// DestinationReadiness returns the status and reason of the destination's
// Ready condition: Unknown and no reason when dst is nil, or the condition
// is missing or was reported for an older generation.
func DestinationReadiness(dst *monitoringv1alpha1.MonitoringDestination) (metav1.ConditionStatus, string) {
	if dst == nil {
		return metav1.ConditionUnknown, ""
	}
	cond := meta.FindStatusCondition(dst.Status.Conditions, monitoringv1alpha1.MonitoringDestinationConditionReady)
	if cond == nil || cond.ObservedGeneration != dst.Generation {
		return metav1.ConditionUnknown, ""
	}
	return cond.Status, cond.Reason
}
