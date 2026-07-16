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

package kubernetes

import (
	"context"
	"slices"

	corev1 "k8s.io/api/core/v1"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ListWorkerNodes returns list of cluster workers nodes.
// This method returns a list of full objects (meta and spec).
// It filters out nodes with the following taints:
// - node.cloudprovider.kubernetes.io/uninitialized=NoSchedule
// - node.kubernetes.io/unschedulable=NoSchedule
// - node-role.kubernetes.io/master=NoSchedule
// - node-role.kubernetes.io/control-plane=NoSchedule.
func (k *Kubernetes) ListWorkerNodes(ctx context.Context, opts ...ctrlclient.ListOption) (*corev1.NodeList, error) {
	result := &corev1.NodeList{}
	if err := k.k8sClient.List(ctx, result, opts...); err != nil {
		return nil, err
	}

	forbidenTaints := map[string]corev1.TaintEffect{
		"node.cloudprovider.kubernetes.io/uninitialized": corev1.TaintEffectNoSchedule,
		"node.kubernetes.io/unschedulable":               corev1.TaintEffectNoSchedule,
		"node-role.kubernetes.io/master":                 corev1.TaintEffectNoSchedule,
		"node-role.kubernetes.io/control-plane":          corev1.TaintEffectNoSchedule,
	}
	result.Items = slices.DeleteFunc(result.Items, func(node corev1.Node) bool {
		for _, taint := range node.Spec.Taints {
			effect, ok := forbidenTaints[taint.Key]
			if ok && effect == taint.Effect {
				return true
			}
		}
		return false
	})

	return result, nil
}

// IsNodeInCondition returns true if node's condition given as an argument has
// status "True". Otherwise, it returns false.
func IsNodeInCondition(node corev1.Node, conditionType corev1.NodeConditionType) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Status == corev1.ConditionTrue && condition.Type == conditionType {
			return true
		}
	}
	return false
}
