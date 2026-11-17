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
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apicommon "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
)

func TestTopologySpreadConstraints(t *testing.T) {
	t.Parallel()

	podLabels := map[string]string{"app.kubernetes.io/instance": "db", "app.kubernetes.io/component": "pxc"}
	ownPods := &metav1.LabelSelector{MatchLabels: podLabels}
	userSelector := &metav1.LabelSelector{MatchLabels: map[string]string{"app": "other"}}

	tests := []struct {
		name   string
		policy *apicommon.SchedulingPolicy
		want   []corev1.TopologySpreadConstraint
	}{
		{
			name: "no policy sets no constraints",
		},
		{
			name:   "policy without spread constraints sets none",
			policy: &apicommon.SchedulingPolicy{Affinity: &corev1.Affinity{}},
		},
		{
			name:   "empty list turns spreading off",
			policy: &apicommon.SchedulingPolicy{TopologySpreadConstraints: &[]corev1.TopologySpreadConstraint{}},
			want:   []corev1.TopologySpreadConstraint{},
		},
		{
			name: "user constraint without a selector counts the component's pods",
			policy: &apicommon.SchedulingPolicy{TopologySpreadConstraints: &[]corev1.TopologySpreadConstraint{
				{MaxSkew: 2, TopologyKey: corev1.LabelTopologyZone, WhenUnsatisfiable: corev1.DoNotSchedule},
			}},
			want: []corev1.TopologySpreadConstraint{
				{MaxSkew: 2, TopologyKey: corev1.LabelTopologyZone, WhenUnsatisfiable: corev1.DoNotSchedule, LabelSelector: ownPods},
			},
		},
		{
			name: "user selectors and matchLabelKeys are kept",
			policy: &apicommon.SchedulingPolicy{TopologySpreadConstraints: &[]corev1.TopologySpreadConstraint{
				{MaxSkew: 1, TopologyKey: corev1.LabelHostname, WhenUnsatisfiable: corev1.DoNotSchedule, LabelSelector: userSelector},
				{MaxSkew: 1, TopologyKey: corev1.LabelTopologyZone, WhenUnsatisfiable: corev1.DoNotSchedule, MatchLabelKeys: []string{"pod-template-hash"}},
			}},
			want: []corev1.TopologySpreadConstraint{
				{MaxSkew: 1, TopologyKey: corev1.LabelHostname, WhenUnsatisfiable: corev1.DoNotSchedule, LabelSelector: userSelector},
				{MaxSkew: 1, TopologyKey: corev1.LabelTopologyZone, WhenUnsatisfiable: corev1.DoNotSchedule, MatchLabelKeys: []string{"pod-template-hash"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, TopologySpreadConstraints(tt.policy, podLabels))
		})
	}
}

func TestTopologySpreadConstraintsLeavesThePolicyUntouched(t *testing.T) {
	t.Parallel()

	policy := &apicommon.SchedulingPolicy{TopologySpreadConstraints: &[]corev1.TopologySpreadConstraint{
		{MaxSkew: 1, TopologyKey: corev1.LabelHostname, WhenUnsatisfiable: corev1.ScheduleAnyway},
	}}

	TopologySpreadConstraints(policy, map[string]string{"app": "db"})

	assert.Nil(t, (*policy.TopologySpreadConstraints)[0].LabelSelector)
}
