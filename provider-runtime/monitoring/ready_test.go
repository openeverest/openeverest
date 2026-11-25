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

package monitoring_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"

	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/monitoring"
)

func TestDestinationReadinessChanged(t *testing.T) {
	t.Parallel()
	destination := func(attached int32, generation int64, ready ...metav1.Condition) *monitoringv1alpha1.MonitoringDestination {
		return &monitoringv1alpha1.MonitoringDestination{
			ObjectMeta: metav1.ObjectMeta{Name: "pmm-prod", Namespace: "team-a", Generation: generation},
			Status:     monitoringv1alpha1.MonitoringDestinationStatus{AttachedInstances: attached, Conditions: ready},
		}
	}
	ready := func(status metav1.ConditionStatus, reason, message string, observedGeneration int64) metav1.Condition {
		return metav1.Condition{
			Type: monitoringv1alpha1.MonitoringDestinationConditionReady, Status: status, Reason: reason,
			Message: message, ObservedGeneration: observedGeneration,
		}
	}
	ok := ready(metav1.ConditionTrue, "Ready", "PMM 3.8.0", 2)

	tests := []struct {
		name     string
		old, new *monitoringv1alpha1.MonitoringDestination
		want     bool
	}{
		{name: "first probe", old: destination(0, 2), new: destination(0, 2, ok), want: true},
		{name: "the token is revoked", old: destination(1, 2, ok), new: destination(1, 2, ready(metav1.ConditionFalse, "Unauthorized", "401", 2)), want: true},
		{name: "a probe catches up with the generation", old: destination(1, 2, ready(metav1.ConditionTrue, "Ready", "", 1)), new: destination(1, 2, ok), want: true},
		{name: "only the message changes", old: destination(1, 2, ok), new: destination(1, 2, ready(metav1.ConditionTrue, "Ready", "PMM 3.9.0", 2)), want: false},
		{name: "only the attached count changes", old: destination(1, 2, ok), new: destination(2, 2, ok), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, monitoring.DestinationReadinessChanged().Update(event.UpdateEvent{ObjectOld: tt.old, ObjectNew: tt.new}))
		})
	}
}
