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

package controller

import (
	"maps"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apicommon "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
)

// TopologySpreadConstraints returns the spread constraints for one
// component's pods. When the policy leaves the field unset, the pods spread
// softly across nodes and zones (maxSkew 1, ScheduleAnyway); otherwise the
// user's list is used as is. Constraints that select no pods of their own get
// podLabels, the labels of the component's pods.
func TopologySpreadConstraints(policy *apicommon.SchedulingPolicy, podLabels map[string]string) []corev1.TopologySpreadConstraint {
	if policy == nil || policy.TopologySpreadConstraints == nil {
		return []corev1.TopologySpreadConstraint{
			softSpread(corev1.LabelHostname, podLabels),
			softSpread(corev1.LabelTopologyZone, podLabels),
		}
	}

	constraints := make([]corev1.TopologySpreadConstraint, 0, len(*policy.TopologySpreadConstraints))
	for _, c := range *policy.TopologySpreadConstraints {
		c = *c.DeepCopy()
		if c.LabelSelector == nil && len(c.MatchLabelKeys) == 0 {
			c.LabelSelector = &metav1.LabelSelector{MatchLabels: maps.Clone(podLabels)}
		}
		constraints = append(constraints, c)
	}
	return constraints
}

func softSpread(topologyKey string, podLabels map[string]string) corev1.TopologySpreadConstraint {
	return corev1.TopologySpreadConstraint{
		MaxSkew:           1,
		TopologyKey:       topologyKey,
		WhenUnsatisfiable: corev1.ScheduleAnyway,
		LabelSelector:     &metav1.LabelSelector{MatchLabels: maps.Clone(podLabels)},
	}
}
