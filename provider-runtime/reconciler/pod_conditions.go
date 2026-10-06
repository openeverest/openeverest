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

// podState is what the pod conditions and counts are computed from.
type podState struct {
	counted, ready, unschedulable bool
	failure                       string
}

func observePod(pod *corev1.Pod) podState {
	failure, _ := podFailure(pod)

	return podState{
		counted:       podCounted(pod),
		ready:         podReady(pod),
		unschedulable: unschedulableCondition(pod) != nil,
		failure:       failure,
	}
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
				// The preemption analysis only repeats the node counts.
				explanation, _, _ = strings.Cut(cond.Message, " preemption:")
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
// most severe problem. The condition's reason is the most severe of them all.
func podsReady(components []string, pods map[string][]corev1.Pod) metav1.Condition {
	worst := ""
	var problems []string
	for _, component := range components {
		reason, count, detail := "", 0, ""
		for i := range pods[component] {
			failure, failureDetail := podFailure(&pods[component][i])
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

	if len(problems) == 0 {
		return metav1.Condition{Status: metav1.ConditionTrue, Reason: v1alpha1.ReasonReady}
	}

	return metav1.Condition{
		Status:  metav1.ConditionFalse,
		Reason:  worst,
		Message: strings.Join(problems, "\n"),
	}
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

// podFailure returns the PodsReady reason of a pod that is not Ready, with the
// detail that explains it, or "" for a Ready pod.
func podFailure(pod *corev1.Pod) (string, string) {
	if podReady(pod) {
		return "", ""
	}
	reason, detail := v1alpha1.ReasonNotReady, ""
	for _, statuses := range [][]corev1.ContainerStatus{pod.Status.InitContainerStatuses, pod.Status.ContainerStatuses} {
		for i := range statuses {
			failure, failureDetail := containerFailure(&statuses[i])
			if notReadyRank(failure) > notReadyRank(reason) {
				reason, detail = failure, failureDetail
			}
		}
	}

	return reason, detail
}

func containerFailure(status *corev1.ContainerStatus) (string, string) {
	waiting := status.State.Waiting
	if waiting == nil {
		return "", ""
	}
	switch waiting.Reason {
	case v1alpha1.ReasonCrashLoopBackOff:
		if last := status.LastTerminationState.Terminated; last != nil {
			return v1alpha1.ReasonCrashLoopBackOff, fmt.Sprintf("container %s last exited with %s (exit code %d)", status.Name, last.Reason, last.ExitCode)
		}
		return v1alpha1.ReasonCrashLoopBackOff, waiting.Message
	case v1alpha1.ReasonImagePullBackOff, "ErrImagePull":
		return v1alpha1.ReasonImagePullBackOff, waiting.Message
	case v1alpha1.ReasonCreateContainerConfigError:
		return v1alpha1.ReasonCreateContainerConfigError, waiting.Message
	}

	return "", ""
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
