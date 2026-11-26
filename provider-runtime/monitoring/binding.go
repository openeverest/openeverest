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

// Package monitoring holds the plumbing the controllers that act on the
// monitoring kinds use: provider-runtime (the ProviderManaged executor) and
// out-of-tree class controllers (the ExtensionManaged executor). It has no
// engine operator dependencies. The rules these controllers share with core,
// such as the names of bindings and the field managers of their status, are
// in contract/monitoring.
package monitoring

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
)

// ListBindingsForInstance lists the bindings controlled by the Instance.
// Bindings of a deleted Instance with the same name, still awaiting garbage
// collection, are skipped.
func ListBindingsForInstance(ctx context.Context, c client.Client, in *corev1alpha1.Instance) ([]monitoringv1alpha1.MonitoringBinding, error) {
	list := &monitoringv1alpha1.MonitoringBindingList{}
	if err := c.List(ctx, list,
		client.InNamespace(in.Namespace),
		client.MatchingLabels{monitoringv1alpha1.BindingInstanceLabel: in.Name},
	); err != nil {
		return nil, fmt.Errorf("list monitoring bindings for instance %q: %w", in.Name, err)
	}
	return slices.DeleteFunc(list.Items, func(b monitoringv1alpha1.MonitoringBinding) bool {
		return !metav1.IsControlledBy(&b, in)
	}), nil
}

// ApplyCondition server-side-applies exactly one condition on obj's status
// subresource under fieldOwner, leaving every other condition to its own
// writer. LastTransitionTime is preserved from existing when the status did
// not change. extra holds further status fields the same owner writes; they
// MUST be sent in the same apply, since a later apply under the same owner
// prunes whatever it omits.
func ApplyCondition(ctx context.Context, c client.Client, obj client.Object, existing []metav1.Condition, fieldOwner string, cond metav1.Condition, extra ...map[string]any) error {
	if cond.LastTransitionTime.IsZero() {
		cond.LastTransitionTime = metav1.Now()
	}
	if prev := meta.FindStatusCondition(existing, cond.Type); prev != nil && prev.Status == cond.Status {
		cond.LastTransitionTime = prev.LastTransitionTime
	}
	raw, err := runtime.DefaultUnstructuredConverter.ToUnstructured(&cond)
	if err != nil {
		return fmt.Errorf("convert condition: %w", err)
	}
	status := map[string]any{"conditions": []any{raw}}
	for _, e := range extra {
		maps.Copy(status, e)
	}
	return ApplyStatus(ctx, c, obj, fieldOwner, status)
}

// ApplyStatus server-side-applies the given status fragment on obj's status
// subresource under fieldOwner. The fragment must carry every status field
// the owner writes on obj.
func ApplyStatus(ctx context.Context, c client.Client, obj client.Object, fieldOwner string, status map[string]any) error {
	gvk, err := apiutil.GVKForObject(obj, c.Scheme())
	if err != nil {
		return fmt.Errorf("resolve GVK: %w", err)
	}
	metadata := map[string]any{"name": obj.GetName()}
	if ns := obj.GetNamespace(); ns != "" {
		metadata["namespace"] = ns
	}
	u := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": gvk.GroupVersion().String(),
		"kind":       gvk.Kind,
		"metadata":   metadata,
		"status":     status,
	}}
	if err := c.Status().Apply(ctx,
		client.ApplyConfigurationFromUnstructured(u),
		client.FieldOwner(fieldOwner),
		client.ForceOwnership,
	); err != nil {
		return fmt.Errorf("apply status of %s %s/%s: %w", gvk.Kind, obj.GetNamespace(), obj.GetName(), err)
	}
	return nil
}

// NewCondition builds a condition stamped with the object's generation.
func NewCondition(obj client.Object, condType string, status metav1.ConditionStatus, reason, message string) metav1.Condition {
	return metav1.Condition{
		Type:               condType,
		Status:             status,
		ObservedGeneration: obj.GetGeneration(),
		Reason:             reason,
		Message:            message,
		LastTransitionTime: metav1.Now(),
	}
}
