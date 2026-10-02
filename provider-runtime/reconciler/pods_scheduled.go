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
	"context"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

const (
	// unschedulableGrace keeps short scheduling waits, such as a volume still
	// being provisioned, off the PodsScheduled condition.
	unschedulableGrace = time.Minute

	// unscheduledRecheck paces reconciles while a pod waits for a node: no
	// watched object changes in the meantime.
	unscheduledRecheck = 15 * time.Second
)

// setPodsScheduledCondition reports the pods the scheduler cannot place and
// returns how soon to look again, or zero once every pod has a node.
func (r *ProviderReconciler) setPodsScheduledCondition(ctx context.Context, c *controller.Context, in *v1alpha1.Instance) (time.Duration, error) {
	sp, ok := r.provider.(controller.PodSelectorProvider)
	if !ok {
		return 0, nil
	}

	podsByComponent := map[string][]corev1.Pod{}
	for component, selector := range sp.PodSelectors(c) {
		pods := &corev1.PodList{}
		// The API reader skips the cache, which would otherwise hold every pod in the cluster.
		if err := r.manager.GetAPIReader().List(ctx, pods,
			client.InNamespace(in.Namespace), client.MatchingLabelsSelector{Selector: selector}); err != nil {
			return 0, fmt.Errorf("list pods of component %q: %w", component, err)
		}
		podsByComponent[component] = pods.Items
	}

	now := time.Now()
	cond, waiting := podsScheduled(podsByComponent, now)
	setCondition(in, v1alpha1.ConditionPodsScheduled, cond.Status, cond.Reason, cond.Message, metav1.NewTime(now))
	if waiting {
		return unscheduledRecheck, nil
	}
	return 0, nil
}

// podsScheduled summarises the components whose pods no node has accepted
// for longer than unschedulableGrace. The returned bool is true while any pod
// lacks a node, reported yet or not.
func podsScheduled(podsByComponent map[string][]corev1.Pod, now time.Time) (metav1.Condition, bool) {
	waiting := false
	var problems []string
	for _, component := range slices.Sorted(maps.Keys(podsByComponent)) {
		pods := podsByComponent[component]
		stuck, explanation := 0, ""
		for i := range pods {
			cond := unschedulableCondition(&pods[i])
			if cond == nil {
				continue
			}
			waiting = true
			if now.Sub(cond.LastTransitionTime.Time) < unschedulableGrace {
				continue
			}
			stuck++
			if explanation == "" {
				// The preemption analysis only repeats the node counts.
				explanation, _, _ = strings.Cut(cond.Message, " preemption:")
			}
		}
		if stuck > 0 {
			problems = append(problems, fmt.Sprintf("%s: %d of %d pods cannot be scheduled: %s",
				component, stuck, len(pods), explanation))
		}
	}

	if len(problems) == 0 {
		return metav1.Condition{Status: metav1.ConditionTrue, Reason: v1alpha1.ReasonScheduled}, waiting
	}
	return metav1.Condition{
		Status:  metav1.ConditionFalse,
		Reason:  v1alpha1.ReasonUnschedulable,
		Message: strings.Join(problems, "; "),
	}, waiting
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
