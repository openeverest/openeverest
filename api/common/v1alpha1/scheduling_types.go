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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
)

// SchedulingPolicy groups the pod-scheduling knobs that OpenEverest applies to
// the pods of a single component.
//
// It lives in the common package rather than next to ComponentSpec because
// placement is not specific to an Instance: anything that runs pods on a user's
// behalf, such as a Job-mode backup, eventually needs the same set of fields.
//
// Deliberately omitted: nodeName. Pinning every replica of a component to one
// named node is meaningless for a replicated database; nodeSelector on a unique
// label covers the legitimate case.
type SchedulingPolicy struct {
	// SchedulerName selects the scheduler that dispatches the pods.
	// When omitted the cluster's default scheduler is used.
	// +optional
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	SchedulerName string `json:"schedulerName,omitempty"`

	// NodeSelector must match a node's labels for the pods to be schedulable
	// onto that node.
	// +optional
	// +mapType=atomic
	// +kubebuilder:validation:MaxProperties=32
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Affinity constrains node selection, pod co-location and pod
	// anti-affinity (spreading pods across nodes, zones or other topology
	// domains for high availability).
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// Tolerations allow the pods to schedule onto nodes carrying matching
	// taints, typically nodes reserved for database workloads.
	// +optional
	// +kubebuilder:validation:MaxItems=32
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// TopologySpreadConstraints describe how the pods spread across topology
	// domains. All constraints are ANDed.
	// +optional
	// +kubebuilder:validation:MaxItems=16
	TopologySpreadConstraints []corev1.TopologySpreadConstraint `json:"topologySpreadConstraints,omitempty"`
}
