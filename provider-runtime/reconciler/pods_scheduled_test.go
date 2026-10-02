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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

func TestPodsScheduled(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	antiAffinity := "0/4 nodes are available: 1 node(s) had untolerated taint {node-role.kubernetes.io/control-plane: }, " +
		"3 node(s) didn't match pod anti-affinity rules. preemption: 0/4 nodes are available: " +
		"1 Preemption is not helpful for scheduling, 3 No preemption victims found for incoming pod."
	running := corev1.Pod{Status: corev1.PodStatus{
		Phase:      corev1.PodRunning,
		Conditions: []corev1.PodCondition{{Type: corev1.PodScheduled, Status: corev1.ConditionTrue}},
	}}
	pendingFor := func(d time.Duration, message string) corev1.Pod {
		return corev1.Pod{Status: corev1.PodStatus{
			Phase: corev1.PodPending,
			Conditions: []corev1.PodCondition{{
				Type:               corev1.PodScheduled,
				Status:             corev1.ConditionFalse,
				Reason:             corev1.PodReasonUnschedulable,
				Message:            message,
				LastTransitionTime: metav1.NewTime(now.Add(-d)),
			}},
		}}
	}

	tests := []struct {
		name        string
		pods        map[string][]corev1.Pod
		want        metav1.Condition
		wantWaiting bool
	}{
		{
			name: "no pods yet",
			pods: map[string][]corev1.Pod{"engine": nil},
			want: metav1.Condition{Status: metav1.ConditionTrue, Reason: v1alpha1.ReasonScheduled},
		},
		{
			name: "every pod has a node",
			pods: map[string][]corev1.Pod{"engine": {running, running, running}},
			want: metav1.Condition{Status: metav1.ConditionTrue, Reason: v1alpha1.ReasonScheduled},
		},
		{
			name:        "a short wait is not reported yet",
			pods:        map[string][]corev1.Pod{"engine": {running, running, pendingFor(30*time.Second, antiAffinity)}},
			want:        metav1.Condition{Status: metav1.ConditionTrue, Reason: v1alpha1.ReasonScheduled},
			wantWaiting: true,
		},
		{
			name: "a pod waiting past the grace period is reported without the preemption analysis",
			pods: map[string][]corev1.Pod{"engine": {running, running, pendingFor(2*time.Minute, antiAffinity)}},
			want: metav1.Condition{
				Status:  metav1.ConditionFalse,
				Reason:  v1alpha1.ReasonUnschedulable,
				Message: "engine: 1 of 3 pods cannot be scheduled: 0/4 nodes are available: 1 node(s) had untolerated taint {node-role.kubernetes.io/control-plane: }, 3 node(s) didn't match pod anti-affinity rules.",
			},
			wantWaiting: true,
		},
		{
			name: "components are listed in name order",
			pods: map[string][]corev1.Pod{
				"engine":       {running, pendingFor(5*time.Minute, "0/1 nodes are available: 1 Insufficient memory.")},
				"configServer": {pendingFor(time.Hour, antiAffinity), pendingFor(time.Hour, antiAffinity), running},
				"proxy":        {running},
			},
			want: metav1.Condition{
				Status: metav1.ConditionFalse,
				Reason: v1alpha1.ReasonUnschedulable,
				Message: "configServer: 2 of 3 pods cannot be scheduled: 0/4 nodes are available: 1 node(s) had untolerated taint {node-role.kubernetes.io/control-plane: }, 3 node(s) didn't match pod anti-affinity rules.; " +
					"engine: 1 of 2 pods cannot be scheduled: 0/1 nodes are available: 1 Insufficient memory.",
			},
			wantWaiting: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, waiting := podsScheduled(tt.pods, now)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantWaiting, waiting)
		})
	}
}
