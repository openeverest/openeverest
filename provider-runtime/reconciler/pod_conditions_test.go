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
	withDRANote := "0/1 nodes are available: 1 Insufficient memory. no new claims to deallocate, preemption: 0/1 nodes are available: 1 No preemption victims found for incoming pod."
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
		wantRecheck time.Duration
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
			name:        "short waits are not reported, the first to pass the grace period sets the recheck",
			pods:        map[string][]corev1.Pod{"engine": {running, pendingFor(20*time.Second, antiAffinity), pendingFor(50*time.Second, antiAffinity)}},
			want:        metav1.Condition{Status: metav1.ConditionTrue, Reason: v1alpha1.ReasonScheduled},
			wantRecheck: 10 * time.Second,
		},
		{
			name: "a pod waiting past the grace period is reported without the preemption analysis",
			pods: map[string][]corev1.Pod{"engine": {running, running, pendingFor(2*time.Minute, antiAffinity)}},
			want: metav1.Condition{
				Status:  metav1.ConditionFalse,
				Reason:  v1alpha1.ReasonUnschedulable,
				Message: "engine: 1 of 3 pods cannot be scheduled: 0/4 nodes are available: 1 node(s) had untolerated taint {node-role.kubernetes.io/control-plane: }, 3 node(s) didn't match pod anti-affinity rules.",
			},
		},
		{
			name: "other plugin notes before the preemption analysis are dropped too",
			pods: map[string][]corev1.Pod{"engine": {pendingFor(2*time.Minute, withDRANote)}},
			want: metav1.Condition{
				Status:  metav1.ConditionFalse,
				Reason:  v1alpha1.ReasonUnschedulable,
				Message: "engine: 1 of 1 pods cannot be scheduled: 0/1 nodes are available: 1 Insufficient memory.",
			},
		},
		{
			name: "each component with stuck pods is reported",
			pods: map[string][]corev1.Pod{
				"configServer": {pendingFor(time.Hour, antiAffinity), pendingFor(time.Hour, antiAffinity), running},
				"engine":       {running, pendingFor(5*time.Minute, "0/1 nodes are available: 1 Insufficient memory.")},
				"proxy":        {running},
			},
			want: metav1.Condition{
				Status: metav1.ConditionFalse,
				Reason: v1alpha1.ReasonUnschedulable,
				Message: "configServer: 2 of 3 pods cannot be scheduled: 0/4 nodes are available: 1 node(s) had untolerated taint {node-role.kubernetes.io/control-plane: }, 3 node(s) didn't match pod anti-affinity rules.\n" +
					"engine: 1 of 2 pods cannot be scheduled: 0/1 nodes are available: 1 Insufficient memory.",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, recheck := podsScheduled([]string{"configServer", "engine", "proxy"}, tt.pods, now)

			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantRecheck, recheck)
		})
	}
}

func TestPodsReady(t *testing.T) {
	t.Parallel()

	ready := corev1.Pod{Status: corev1.PodStatus{
		Phase:      corev1.PodRunning,
		Conditions: []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionTrue}},
	}}
	starting := corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodRunning}}
	waiting := func(reason, message string) corev1.Pod {
		return corev1.Pod{Status: corev1.PodStatus{
			Phase: corev1.PodPending,
			ContainerStatuses: []corev1.ContainerStatus{
				{Name: "pbm-agent", Ready: true, State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}},
				{Name: "mongod", State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: reason, Message: message}}},
			},
		}}
	}
	crashing := waiting("CrashLoopBackOff", "back-off 5m0s restarting failed container=mongod")
	crashing.Status.ContainerStatuses[1].LastTerminationState.Terminated = &corev1.ContainerStateTerminated{Reason: "OOMKilled", ExitCode: 137}
	initConfigError := corev1.Pod{Status: corev1.PodStatus{
		Phase: corev1.PodPending,
		InitContainerStatuses: []corev1.ContainerStatus{{
			Name:  "init",
			State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CreateContainerConfigError", Message: `secret "db-users" not found`}},
		}},
	}}

	tests := []struct {
		name string
		pods map[string][]corev1.Pod
		want metav1.Condition
	}{
		{
			name: "no pods yet",
			pods: map[string][]corev1.Pod{"engine": nil},
			want: metav1.Condition{Status: metav1.ConditionTrue, Reason: v1alpha1.ReasonReady},
		},
		{
			name: "every pod is ready",
			pods: map[string][]corev1.Pod{"engine": {ready, ready}, "proxy": {ready}},
			want: metav1.Condition{Status: metav1.ConditionTrue, Reason: v1alpha1.ReasonReady},
		},
		{
			name: "a starting pod is not ready",
			pods: map[string][]corev1.Pod{"engine": {ready, ready, starting}},
			want: metav1.Condition{
				Status:  metav1.ConditionFalse,
				Reason:  v1alpha1.ReasonNotReady,
				Message: "engine: 1 of 3 pods are not ready",
			},
		},
		{
			name: "a crash loop quotes how the container last exited",
			pods: map[string][]corev1.Pod{"engine": {ready, crashing, crashing}},
			want: metav1.Condition{
				Status:  metav1.ConditionFalse,
				Reason:  v1alpha1.ReasonCrashLoopBackOff,
				Message: "engine: 2 of 3 pods keep crashing: container mongod last exited with OOMKilled (exit code 137)",
			},
		},
		{
			name: "a failed pull is reported as ImagePullBackOff with the kubelet's message",
			pods: map[string][]corev1.Pod{"engine": {waiting("ErrImagePull", `pull access denied for percona/mongod, repository does not exist`)}},
			want: metav1.Condition{
				Status:  metav1.ConditionFalse,
				Reason:  v1alpha1.ReasonImagePullBackOff,
				Message: "engine: 1 of 1 pods cannot pull their image: pull access denied for percona/mongod, repository does not exist",
			},
		},
		{
			name: "init containers are checked too",
			pods: map[string][]corev1.Pod{"engine": {initConfigError}},
			want: metav1.Condition{
				Status:  metav1.ConditionFalse,
				Reason:  v1alpha1.ReasonCreateContainerConfigError,
				Message: `engine: 1 of 1 pods cannot create their containers: secret "db-users" not found`,
			},
		},
		{
			name: "each component reports its most severe problem, the reason is the most severe of all",
			pods: map[string][]corev1.Pod{
				"engine": {starting, waiting("ImagePullBackOff", "Back-off pulling image"), starting},
				"proxy":  {crashing, ready},
			},
			want: metav1.Condition{
				Status: metav1.ConditionFalse,
				Reason: v1alpha1.ReasonCrashLoopBackOff,
				Message: "engine: 1 of 3 pods cannot pull their image: Back-off pulling image\n" +
					"proxy: 1 of 2 pods keep crashing: container mongod last exited with OOMKilled (exit code 137)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, podsReady([]string{"engine", "proxy"}, tt.pods))
		})
	}
}
