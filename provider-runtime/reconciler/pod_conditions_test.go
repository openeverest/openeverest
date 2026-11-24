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

	now := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	ready := corev1.Pod{Status: corev1.PodStatus{
		Phase:      corev1.PodRunning,
		Conditions: []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionTrue}},
	}}
	starting := corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodRunning}}
	// waiting is a pod that has not been Ready for notReadyFor, with one
	// container waiting for the reason.
	waiting := func(reason, message string, notReadyFor time.Duration) corev1.Pod {
		return corev1.Pod{Status: corev1.PodStatus{
			Phase: corev1.PodPending,
			Conditions: []corev1.PodCondition{{
				Type: corev1.PodReady, Status: corev1.ConditionFalse, LastTransitionTime: metav1.NewTime(now.Add(-notReadyFor)),
			}},
			ContainerStatuses: []corev1.ContainerStatus{
				{Name: "pbm-agent", Ready: true, State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}},
				{Name: "mongod", State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: reason, Message: message}}},
			},
		}}
	}
	crashing := func(restarts int32) corev1.Pod {
		pod := waiting("CrashLoopBackOff", "back-off 5m0s restarting failed container=mongod", time.Hour)
		pod.Status.ContainerStatuses[1].RestartCount = restarts
		pod.Status.ContainerStatuses[1].LastTerminationState.Terminated = &corev1.ContainerStateTerminated{Reason: "OOMKilled", ExitCode: 137}

		return pod
	}
	initConfigError := corev1.Pod{Status: corev1.PodStatus{
		Phase: corev1.PodPending,
		Conditions: []corev1.PodCondition{{
			Type: corev1.PodReady, Status: corev1.ConditionFalse, LastTransitionTime: metav1.NewTime(now.Add(-time.Hour)),
		}},
		InitContainerStatuses: []corev1.ContainerStatus{{
			Name:  "init",
			State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CreateContainerConfigError", Message: `secret "db-users" not found`}},
		}},
	}}

	tests := []struct {
		name        string
		pods        map[string][]corev1.Pod
		want        metav1.Condition
		wantRecheck time.Duration
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
			name: "a starting pod is in progress, not a problem",
			pods: map[string][]corev1.Pod{"engine": {ready, ready, starting}},
			want: metav1.Condition{
				Status:  metav1.ConditionUnknown,
				Reason:  v1alpha1.ReasonNotReady,
				Message: "engine: 1 of 3 pods are not ready",
			},
		},
		{
			name: "a container that crashed twice is still starting",
			pods: map[string][]corev1.Pod{"engine": {ready, crashing(2)}},
			want: metav1.Condition{
				Status:  metav1.ConditionUnknown,
				Reason:  v1alpha1.ReasonNotReady,
				Message: "engine: 1 of 2 pods are not ready",
			},
		},
		{
			name: "a crash loop quotes how the container last exited",
			pods: map[string][]corev1.Pod{"engine": {ready, crashing(3), crashing(40)}},
			want: metav1.Condition{
				Status:  metav1.ConditionFalse,
				Reason:  v1alpha1.ReasonCrashLoopBackOff,
				Message: "engine: 2 of 3 pods keep crashing: container mongod last exited with OOMKilled (exit code 137)",
			},
		},
		{
			name: "a pull that has failed for a short while is looked at again when the grace period ends",
			pods: map[string][]corev1.Pod{"engine": {waiting("ErrImagePull", "pull access denied", time.Minute)}},
			want: metav1.Condition{
				Status:  metav1.ConditionUnknown,
				Reason:  v1alpha1.ReasonNotReady,
				Message: "engine: 1 of 1 pods are not ready",
			},
			wantRecheck: 2 * time.Minute,
		},
		{
			name: "a failed pull is reported as ImagePullBackOff with the kubelet's message",
			pods: map[string][]corev1.Pod{"engine": {waiting("ErrImagePull", `pull access denied for percona/mongod, repository does not exist`, backOffGrace)}},
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
				"engine": {starting, waiting("ImagePullBackOff", "Back-off pulling image", time.Hour), starting},
				"proxy":  {crashing(7), ready},
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

			got, recheck := podsReady([]string{"engine", "proxy"}, tt.pods, now)

			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantRecheck, recheck)
		})
	}
}

// A crash loop is read from the pod's status as it is now. Every case below
// is a single pod with no history kept anywhere, so the verdict needs no
// stored state, and the cases pin the rest of what the test must guarantee:
// it does not flap between two crashes, it clears once the container has
// stayed up, and it does not fire while a pod is merely starting.
func TestPodFailure_CrashLoop(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	ago := func(d time.Duration) metav1.Time { return metav1.NewTime(now.Add(-d)) }
	exited := &corev1.ContainerStateTerminated{Reason: "Error", ExitCode: 14, FinishedAt: ago(20 * time.Second)}
	backingOff := corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"}}
	runningFor := func(d time.Duration) corev1.ContainerState {
		return corev1.ContainerState{Running: &corev1.ContainerStateRunning{StartedAt: ago(d)}}
	}
	pod := func(podReady bool, restarts int32, state corev1.ContainerState) *corev1.Pod {
		status := corev1.ConditionFalse
		if podReady {
			status = corev1.ConditionTrue
		}

		return &corev1.Pod{Status: corev1.PodStatus{
			Phase:      corev1.PodRunning,
			Conditions: []corev1.PodCondition{{Type: corev1.PodReady, Status: status, LastTransitionTime: ago(10 * time.Second)}},
			ContainerStatuses: []corev1.ContainerStatus{{
				Name:                 "mongod",
				RestartCount:         restarts,
				State:                state,
				LastTerminationState: corev1.ContainerState{Terminated: exited},
			}},
		}}
	}
	const crashLoop = v1alpha1.ReasonCrashLoopBackOff

	tests := []struct {
		name        string
		pod         *corev1.Pod
		want        string
		wantRecheck time.Duration
	}{
		// It does not flap: the same loop, seen at each point between two crashes.
		{name: "looping: waiting to be restarted", pod: pod(false, 6, backingOff), want: crashLoop},
		{
			name: "looping: just restarted and not ready yet",
			pod:  pod(false, 6, runningFor(5*time.Second)), want: crashLoop, wantRecheck: crashLoopQuiet - 5*time.Second,
		},
		{
			name: "looping: up and momentarily Ready",
			pod:  pod(true, 6, runningFor(40*time.Second)), want: crashLoop, wantRecheck: crashLoopQuiet - 40*time.Second,
		},
		{
			name: "looping: just exited, not restarted yet",
			pod: pod(false, 6, corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{
				Reason: "Error", ExitCode: 14, FinishedAt: ago(time.Second),
			}}),
			want: crashLoop, wantRecheck: crashLoopQuiet - time.Second,
		},
		{
			name: "looping: being started again",
			pod:  pod(false, 6, corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "ContainerCreating"}}),
			want: crashLoop, wantRecheck: crashLoopQuiet - 20*time.Second,
		},

		// It clears once the container has stopped crashing.
		{
			name: "recovering: up for almost long enough",
			pod:  pod(true, 6, runningFor(crashLoopQuiet-time.Second)), want: crashLoop, wantRecheck: time.Second,
		},
		{name: "recovered: up for long enough and Ready", pod: pod(true, 6, runningFor(crashLoopQuiet))},
		{
			name: "recovered: up for long enough, and not Ready for another reason",
			pod:  pod(false, 6, runningFor(20*time.Minute)), want: v1alpha1.ReasonNotReady,
		},
		{
			name: "a pod with many old restarts that restarted once is reported for a while",
			pod:  pod(true, 31, runningFor(time.Minute)), want: crashLoop, wantRecheck: crashLoopQuiet - time.Minute,
		},

		// It does not fire while a pod is merely starting.
		{name: "starting: backing off after two restarts", pod: pod(false, 2, backingOff), want: v1alpha1.ReasonNotReady},
		{name: "starting: up after two restarts", pod: pod(true, 2, runningFor(time.Second))},
		{
			name: "an init container that finished after several attempts is not looping",
			pod: pod(true, 4, corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{
				Reason: "Completed", ExitCode: 0, FinishedAt: ago(time.Second),
			}}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, _, recheck := podFailure(tt.pod, now)

			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantRecheck, recheck)
		})
	}
}

// A pull or a container creation that fails never restarts anything, so
// there is no restart count to wait for: such a back-off is a problem once
// the pod has not been Ready for the grace period.
func TestPodFailure_BackOffWithoutRestarts(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	pod := func(reason string, notReadyFor time.Duration) *corev1.Pod {
		return &corev1.Pod{Status: corev1.PodStatus{
			Phase: corev1.PodPending,
			Conditions: []corev1.PodCondition{{
				Type: corev1.PodReady, Status: corev1.ConditionFalse, LastTransitionTime: metav1.NewTime(now.Add(-notReadyFor)),
			}},
			ContainerStatuses: []corev1.ContainerStatus{{
				Name:  "mongod",
				State: corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: reason}},
			}},
		}}
	}

	tests := []struct {
		name        string
		pod         *corev1.Pod
		want        string
		wantRecheck time.Duration
	}{
		{
			name: "an image that cannot be pulled yet", pod: pod("ImagePullBackOff", backOffGrace-time.Second),
			want: v1alpha1.ReasonNotReady, wantRecheck: time.Second,
		},
		{name: "an image that still cannot be pulled", pod: pod("ImagePullBackOff", backOffGrace), want: v1alpha1.ReasonImagePullBackOff},
		{
			name: "a container that cannot be created yet", pod: pod("CreateContainerConfigError", time.Minute),
			want: v1alpha1.ReasonNotReady, wantRecheck: backOffGrace - time.Minute,
		},
		{
			name: "a container that still cannot be created", pod: pod("CreateContainerConfigError", time.Hour),
			want: v1alpha1.ReasonCreateContainerConfigError,
		},
		{name: "a container that is being created", pod: pod("ContainerCreating", time.Hour), want: v1alpha1.ReasonNotReady},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, _, recheck := podFailure(tt.pod, now)

			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantRecheck, recheck)
		})
	}
}

// The pod watch reconciles an Instance only when what is reported about a
// pod may change. A crash changes the restart count before it changes
// anything else.
func TestObservePod_ChangesWithContainerState(t *testing.T) {
	t.Parallel()

	pod := func(restarts int32, state corev1.ContainerState) *corev1.Pod {
		return &corev1.Pod{Status: corev1.PodStatus{
			Phase:             corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{{Name: "mongod", RestartCount: restarts, State: state}},
		}}
	}
	running := corev1.ContainerState{Running: &corev1.ContainerStateRunning{}}
	backingOff := corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"}}

	before, unchanged := observePod(pod(1, running)), observePod(pod(1, running))
	assert.Equal(t, before, unchanged)
	assert.NotEqual(t, observePod(pod(1, running)), observePod(pod(2, running)), "a restart")
	assert.NotEqual(t, observePod(pod(2, running)), observePod(pod(2, backingOff)), "a container that starts to wait")
}
