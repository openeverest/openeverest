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
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
	contractmonitoring "github.com/openeverest/openeverest/v2/contract/monitoring"
)

// DestinationReadinessChanged passes MonitoringDestination updates that
// change what contractmonitoring.BindingReady quotes: the status and reason
// of the destination's Ready condition at its current generation.
func DestinationReadinessChanged() predicate.Funcs {
	return predicate.Funcs{UpdateFunc: func(e event.UpdateEvent) bool {
		before, okOld := e.ObjectOld.(*monitoringv1alpha1.MonitoringDestination)
		after, okNew := e.ObjectNew.(*monitoringv1alpha1.MonitoringDestination)
		if !okOld || !okNew {
			return false
		}
		beforeStatus, beforeReason := contractmonitoring.DestinationReadiness(before)
		afterStatus, afterReason := contractmonitoring.DestinationReadiness(after)
		return beforeStatus != afterStatus || beforeReason != afterReason
	}}
}
