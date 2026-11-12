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

	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/monitoring"
)

func TestBindingReady(t *testing.T) {
	t.Parallel()

	cond := func(status metav1.ConditionStatus, reason string) metav1.Condition {
		return metav1.Condition{Status: status, Reason: reason}
	}
	accepted := cond(metav1.ConditionTrue, monitoringv1alpha1.ReasonAccepted)
	configured := cond(metav1.ConditionTrue, string(monitoringv1alpha1.ReasonAgentTLSNotVerified))
	destination := func(status metav1.ConditionStatus, reason string, observedGeneration int64) *monitoringv1alpha1.MonitoringDestination {
		return &monitoringv1alpha1.MonitoringDestination{
			ObjectMeta: metav1.ObjectMeta{Name: "pmm-prod", Namespace: "team-a", Generation: 2},
			Status: monitoringv1alpha1.MonitoringDestinationStatus{Conditions: []metav1.Condition{{
				Type: monitoringv1alpha1.MonitoringDestinationConditionReady, Status: status, Reason: reason,
				Message: `GET https://pmm.example.com: 401, Secret "pmm-prod-credentials"`, ObservedGeneration: observedGeneration,
			}}},
		}
	}
	ready := destination(metav1.ConditionTrue, string(monitoringv1alpha1.ReasonReady), 2)
	unauthorized := destination(metav1.ConditionFalse, string(monitoringv1alpha1.ReasonUnauthorized), 2)

	type want struct {
		status  metav1.ConditionStatus
		reason  monitoringv1alpha1.BindingReadyReason
		message string
	}
	tests := []struct {
		name       string
		accepted   metav1.Condition
		configured metav1.Condition
		dst        *monitoringv1alpha1.MonitoringDestination
		want       want
	}{
		{
			name: "configured and the destination is ready", accepted: accepted, configured: configured, dst: ready,
			want: want{metav1.ConditionTrue, monitoringv1alpha1.ReasonBindingReady, `Configured, and MonitoringDestination "pmm-prod" is Ready`},
		},
		{
			name: "configured but the destination rejects the credentials", accepted: accepted, configured: configured, dst: unauthorized,
			want: want{metav1.ConditionFalse, monitoringv1alpha1.ReasonDestinationNotReady, `MonitoringDestination "pmm-prod" is not Ready (Unauthorized)`},
		},
		{
			name:     "not accepted outweighs everything else",
			accepted: cond(metav1.ConditionFalse, monitoringv1alpha1.ReasonClassNotAccepted), configured: cond(metav1.ConditionFalse, "NotAccepted"), dst: unauthorized,
			want: want{metav1.ConditionFalse, monitoringv1alpha1.ReasonBindingNotAccepted, "Binding is not Accepted (ClassNotAccepted)"},
		},
		{
			name:     "not configured outweighs a destination that is not ready",
			accepted: accepted, configured: cond(metav1.ConditionFalse, string(monitoringv1alpha1.ReasonRenderFailed)), dst: unauthorized,
			want: want{metav1.ConditionFalse, monitoringv1alpha1.ReasonBindingNotConfigured, "Binding is not Configured (RenderFailed)"},
		},
		{
			name: "a destination that is not ready outweighs a pending input", configured: configured, dst: unauthorized,
			want: want{metav1.ConditionFalse, monitoringv1alpha1.ReasonDestinationNotReady, `MonitoringDestination "pmm-prod" is not Ready (Unauthorized)`},
		},
		{
			name: "not accepted yet", configured: configured, dst: ready,
			want: want{metav1.ConditionUnknown, monitoringv1alpha1.ReasonBindingReadyPending, "Waiting for the binding to be Accepted"},
		},
		{
			name: "not configured yet", accepted: accepted, configured: cond(metav1.ConditionUnknown, "Pending"), dst: ready,
			want: want{metav1.ConditionUnknown, monitoringv1alpha1.ReasonBindingReadyPending, "Waiting for the binding to be Configured"},
		},
		{
			name: "the destination does not exist", accepted: accepted, configured: configured,
			want: want{metav1.ConditionUnknown, monitoringv1alpha1.ReasonBindingReadyPending, `MonitoringDestination "pmm-prod" not found`},
		},
		{
			name: "the destination is not probed yet", accepted: accepted, configured: configured,
			dst:  destination(metav1.ConditionUnknown, string(monitoringv1alpha1.ReasonReadyPending), 2),
			want: want{metav1.ConditionUnknown, monitoringv1alpha1.ReasonBindingReadyPending, `Waiting for MonitoringDestination "pmm-prod" to report Ready`},
		},
		{
			name: "the destination reported for an older generation", accepted: accepted, configured: configured,
			dst:  destination(metav1.ConditionFalse, string(monitoringv1alpha1.ReasonUnauthorized), 1),
			want: want{metav1.ConditionUnknown, monitoringv1alpha1.ReasonBindingReadyPending, `Waiting for MonitoringDestination "pmm-prod" to report Ready`},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			b := &monitoringv1alpha1.MonitoringBinding{
				ObjectMeta: metav1.ObjectMeta{Name: "orders-pmm", Namespace: "team-a", Generation: 3},
				Spec:       monitoringv1alpha1.MonitoringBindingSpec{DestinationRef: common.MonitoringDestinationRef{Name: "pmm-prod"}},
			}

			got := monitoring.BindingReady(b, tt.accepted, tt.configured, tt.dst)

			assert.Equal(t, monitoringv1alpha1.MonitoringBindingConditionReady, got.Type)
			assert.Equal(t, tt.want, want{got.Status, monitoringv1alpha1.BindingReadyReason(got.Reason), got.Message})
			assert.Equal(t, int64(3), got.ObservedGeneration)
		})
	}
}

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
