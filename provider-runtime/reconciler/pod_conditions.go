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
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

// unschedulableGrace keeps short scheduling waits, such as a volume still
// being provisioned, off the PodsScheduled condition.
const unschedulableGrace = time.Minute

const (
	// crashLoopRestarts is how often a container must have restarted before
	// its pod counts as crash-looping. Fewer restarts are part of starting:
	// a member that exits until its peers are up is not a problem.
	crashLoopRestarts = 3

	// crashLoopQuiet is how long a container that has restarted that often
	// must stay up before its pod no longer counts as crash-looping. The
	// kubelet waits at most this long before it restarts a container, so a
	// looping container is never quiet for longer, while one that recovered
	// stops being reported after this time.
	crashLoopQuiet = 5 * time.Minute

	// backOffGrace is how long a pod may be unable to pull an image or to
	// create a container before that is a problem. Nothing restarts in these
	// back-offs, so there is no restart count to wait for.
	backOffGrace = 3 * time.Minute
)

// notReadyRank orders the PodsReady reasons from least to most severe.
func notReadyRank(reason string) int {
	switch reason {
	case v1alpha1.ReasonNotReady:
		return 1
	case v1alpha1.ReasonCreateContainerConfigError:
		return 2
	case v1alpha1.ReasonImagePullBackOff:
		return 3
	case v1alpha1.ReasonCrashLoopBackOff:
		return 4
	}

	return 0
}

// podState is what decides whether a pod's Instance is reconciled again: the
// pod conditions and counts are computed from it.
type podState struct {
	counted, ready, unschedulable bool
	restarts                      int32
	waiting                       string
}

func observePod(pod *corev1.Pod) podState {
	state := podState{
		counted:       podCounted(pod),
		ready:         podReady(pod),
		unschedulable: unschedulableCondition(pod) != nil,
	}
	for _, statuses := range [][]corev1.ContainerStatus{pod.Status.InitContainerStatuses, pod.Status.ContainerStatuses} {
		for i := range statuses {
			state.restarts += statuses[i].RestartCount
			if waiting := statuses[i].State.Waiting; waiting != nil {
				state.waiting += statuses[i].Name + "=" + waiting.Reason + ";"
			}
		}
	}

	return state
}

// podsScheduled reports the components whose pods no node has accepted for
// longer than unschedulableGrace, and how long until the next pod still within
// the grace period would be reported.
func podsScheduled(components []string, pods map[string][]corev1.Pod, now time.Time) (metav1.Condition, time.Duration) {
	var problems []string
	var recheck time.Duration
	for _, component := range components {
		stuck, explanation := 0, ""
		for i := range pods[component] {
			cond := unschedulableCondition(&pods[component][i])
			if cond == nil {
				continue
			}
			if waited := now.Sub(cond.LastTransitionTime.Time); waited < unschedulableGrace {
				if remaining := unschedulableGrace - waited; recheck == 0 || remaining < recheck {
					recheck = remaining
				}
				continue
			}
			stuck++
			if explanation == "" {
				// Plugin notes, such as the preemption analysis, follow the scheduler's first sentence.
				explanation = cond.Message
				if first, _, found := strings.Cut(cond.Message, ". "); found {
					explanation = first + "."
				}
			}
		}
		if stuck > 0 {
			problems = append(problems, fmt.Sprintf("%s: %d of %d pods cannot be scheduled: %s", component, stuck, len(pods[component]), explanation))
		}
	}

	if len(problems) == 0 {
		return metav1.Condition{Status: metav1.ConditionTrue, Reason: v1alpha1.ReasonScheduled}, recheck
	}

	return metav1.Condition{
		Status:  metav1.ConditionFalse,
		Reason:  v1alpha1.ReasonUnschedulable,
		Message: strings.Join(problems, "\n"),
	}, recheck
}

// podsReady reports the components with pods that are not Ready, each with its
// most severe problem. It is False while a pod is in a back-off, with the most
// severe of the back-offs as its reason, and Unknown while pods are merely not
// Ready yet. It also returns how long until a pod's verdict changes with no
// change to the pod: a back-off that becomes a problem, or a crash loop that
// has been quiet for long enough.
func podsReady(components []string, pods map[string][]corev1.Pod, now time.Time) (metav1.Condition, time.Duration) {
	worst := ""
	var problems []string
	var recheck time.Duration
	for _, component := range components {
		reason, count, detail := "", 0, ""
		for i := range pods[component] {
			failure, failureDetail, again := podFailure(&pods[component][i], now)
			recheck = soonest(recheck, again)
			switch {
			case failure == "":
			case notReadyRank(failure) > notReadyRank(reason):
				reason, count, detail = failure, 1, failureDetail
			case failure == reason:
				count++
			}
		}
		if reason == "" {
			continue
		}
		if notReadyRank(reason) > notReadyRank(worst) {
			worst = reason
		}
		problems = append(problems, notReadyMessage(component, reason, count, len(pods[component]), detail))
	}

	status := metav1.ConditionFalse
	switch worst {
	case "":
		return metav1.Condition{Status: metav1.ConditionTrue, Reason: v1alpha1.ReasonReady}, recheck
	case v1alpha1.ReasonNotReady:
		status = metav1.ConditionUnknown
	}

	return metav1.Condition{
		Status:  status,
		Reason:  worst,
		Message: strings.Join(problems, "\n"),
	}, recheck
}

func notReadyMessage(component, reason string, count, total int, detail string) string {
	problem := "are not ready"
	switch reason {
	case v1alpha1.ReasonCrashLoopBackOff:
		problem = "keep crashing"
	case v1alpha1.ReasonImagePullBackOff:
		problem = "cannot pull their image"
	case v1alpha1.ReasonCreateContainerConfigError:
		problem = "cannot create their containers"
	}
	message := fmt.Sprintf("%s: %d of %d pods %s", component, count, total, problem)
	if detail != "" {
		message += ": " + detail
	}

	return message
}

// podFailure returns the PodsReady reason of a pod, with the detail that
// explains it, or "" for a Ready pod with no container in a back-off. It also
// returns how long until the answer changes with no change to the pod.
//
// The back-offs are looked for first: a container that keeps crashing makes
// its pod Ready for a moment after each restart, which must not read as
// recovered.
func podFailure(pod *corev1.Pod, now time.Time) (string, string, time.Duration) {
	reason, detail := "", ""
	var recheck time.Duration
	for _, statuses := range [][]corev1.ContainerStatus{pod.Status.InitContainerStatuses, pod.Status.ContainerStatuses} {
		for i := range statuses {
			failure, failureDetail, again := containerBackOff(pod, &statuses[i], now)
			recheck = soonest(recheck, again)
			if notReadyRank(failure) > notReadyRank(reason) {
				reason, detail = failure, failureDetail
			}
		}
	}
	if reason == "" && !podReady(pod) {
		reason = v1alpha1.ReasonNotReady
	}

	return reason, detail, recheck
}

// containerBackOff returns the back-off a container is in, if it is one to
// report yet, and how long until that changes with no change to the pod.
func containerBackOff(pod *corev1.Pod, status *corev1.ContainerStatus, now time.Time) (string, string, time.Duration) {
	if looping, clears := crashLooping(status, now); looping {
		return v1alpha1.ReasonCrashLoopBackOff, crashDetail(status), clears
	}

	waiting := status.State.Waiting
	if waiting == nil {
		return "", "", 0
	}
	var reason string
	switch waiting.Reason {
	case v1alpha1.ReasonImagePullBackOff, "ErrImagePull":
		reason = v1alpha1.ReasonImagePullBackOff
	case v1alpha1.ReasonCreateContainerConfigError:
		reason = v1alpha1.ReasonCreateContainerConfigError
	default:
		return "", "", 0
	}
	if remaining := backOffGrace - now.Sub(notReadySince(pod)); remaining > 0 {
		return "", "", remaining
	}

	return reason, waiting.Message, 0
}

// crashLooping reports whether the container is crash-looping, read from its
// status alone: it has restarted at least crashLoopRestarts times, and is
// either waiting to be restarted or was last started or stopped less than
// crashLoopQuiet ago. The second half is what keeps the answer from flipping
// each time the container comes up between two crashes. It also returns how
// long until a container that stays up stops being reported.
func crashLooping(status *corev1.ContainerStatus, now time.Time) (bool, time.Duration) {
	if status.RestartCount < crashLoopRestarts {
		return false, 0
	}
	if done := status.State.Terminated; done != nil && done.ExitCode == 0 {
		// An init container that finished, however many attempts it took.
		return false, 0
	}
	if waiting := status.State.Waiting; waiting != nil && waiting.Reason == v1alpha1.ReasonCrashLoopBackOff {
		return true, 0
	}
	last := lastStartOrExit(status)
	if last.IsZero() {
		return false, 0
	}
	if quiet := now.Sub(last); quiet < crashLoopQuiet {
		return true, crashLoopQuiet - quiet
	}

	return false, 0
}

// lastStartOrExit returns when a running container started, and when any
// other last stopped.
func lastStartOrExit(status *corev1.ContainerStatus) time.Time {
	switch {
	case status.State.Running != nil:
		return status.State.Running.StartedAt.Time
	case status.State.Terminated != nil:
		return status.State.Terminated.FinishedAt.Time
	case status.LastTerminationState.Terminated != nil:
		return status.LastTerminationState.Terminated.FinishedAt.Time
	}

	return time.Time{}
}

func crashDetail(status *corev1.ContainerStatus) string {
	last := status.LastTerminationState.Terminated
	if last == nil {
		last = status.State.Terminated
	}
	if last != nil {
		return fmt.Sprintf("container %s last exited with %s (exit code %d)", status.Name, last.Reason, last.ExitCode)
	}
	if waiting := status.State.Waiting; waiting != nil && waiting.Message != "" {
		return waiting.Message
	}

	return fmt.Sprintf("container %s restarted %d times", status.Name, status.RestartCount)
}

// notReadySince returns when the pod last stopped being Ready, or when it
// started if it has never been.
func notReadySince(pod *corev1.Pod) time.Time {
	for _, c := range pod.Status.Conditions {
		if c.Type == corev1.PodReady && !c.LastTransitionTime.IsZero() {
			return c.LastTransitionTime.Time
		}
	}
	if pod.Status.StartTime != nil {
		return pod.Status.StartTime.Time
	}

	return pod.CreationTimestamp.Time
}

// soonest returns the shorter of two waits, where zero means no wait.
func soonest(a, b time.Duration) time.Duration {
	if a <= 0 || (b > 0 && b < a) {
		return b
	}

	return a
}

func unschedulableCondition(pod *corev1.Pod) *corev1.PodCondition {
	if pod.Status.Phase != corev1.PodPending {
		return nil
	}
	for i := range pod.Status.Conditions {
		c := &pod.Status.Conditions[i]
		if c.Type == corev1.PodScheduled && c.Status == corev1.ConditionFalse && c.Reason == corev1.PodReasonUnschedulable {
			return c
		}
	}

	return nil
}
