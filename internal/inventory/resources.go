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

package inventory

import (
	"sort"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

// ComponentResources is what one Instance component asks for in its spec,
// multiplied by its replicas. CPU is in cores; memory and storage in bytes.
// Only resources the spec sets are present.
type ComponentResources struct {
	Component string
	Requests  map[corev1.ResourceName]float64
	Limits    map[corev1.ResourceName]float64
}

// InstanceResources returns the resources of every component the Instance
// spec lists, sorted by component name.
//
// It is a lower bound of what the Instance runs: components that come only
// from the provider's defaults, sidecars and exporters are not in the spec.
func InstanceResources(inst *corev1alpha1.Instance) []ComponentResources {
	out := make([]ComponentResources, 0, len(inst.Spec.Components))
	for name, c := range inst.Spec.Components {
		replicas := float64(1)
		if c.Replicas != nil {
			replicas = float64(*c.Replicas)
		}
		r := ComponentResources{
			Component: name,
			Requests:  make(map[corev1.ResourceName]float64),
			Limits:    make(map[corev1.ResourceName]float64),
		}
		if c.Resources != nil {
			for _, res := range []corev1.ResourceName{corev1.ResourceCPU, corev1.ResourceMemory} {
				if q, ok := c.Resources.Requests[res]; ok {
					r.Requests[res] = q.AsApproximateFloat64() * replicas
				}
				if q, ok := c.Resources.Limits[res]; ok {
					r.Limits[res] = q.AsApproximateFloat64() * replicas
				}
			}
		}
		if c.Storage != nil && !c.Storage.Size.IsZero() {
			r.Requests[corev1.ResourceStorage] = c.Storage.Size.AsApproximateFloat64() * replicas
		}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Component < out[j].Component })
	return out
}

// ConditionIs reports whether the condition of the given type has the given
// status.
func ConditionIs(conditions []metav1.Condition, conditionType string, status metav1.ConditionStatus) bool {
	c := meta.FindStatusCondition(conditions, conditionType)
	return c != nil && c.Status == status
}
