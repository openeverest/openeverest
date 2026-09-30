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

// Package monitoring holds the plumbing shared by everything that touches
// MonitoringBindings: the core materialiser, provider-runtime (ProviderManaged
// executor) and out-of-tree class controllers (ExtensionManaged executor).
package monitoring

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
)

// Field managers. Each condition type on a binding has exactly one owner.
const (
	// MaterializerFieldOwner owns MonitoringBinding.spec and the Accepted
	// condition on bindings and destinations.
	MaterializerFieldOwner = "openeverest.io/binding-materializer"
)

// BindingName returns the deterministic MonitoringBinding name for an
// Instance entry: <instance>-<entry>-<first 8 hex chars of
// sha256("<instance>/<entry>")>, truncated so the hash suffix always fits.
func BindingName(instance, entry string) string {
	sum := sha256.Sum256([]byte(instance + "/" + entry))
	suffix := hex.EncodeToString(sum[:])[:8]
	base := instance + "-" + entry
	const maxBase = 253 - 1 - 8
	if len(base) > maxBase {
		base = base[:maxBase]
	}
	return base + "-" + suffix
}

// BindingLabels returns the labels core stamps on a binding.
func BindingLabels(instance, entry, class string) map[string]string {
	l := map[string]string{
		monitoringv1alpha1.BindingInstanceLabel: instance,
		monitoringv1alpha1.BindingEntryLabel:    entry,
	}
	if class != "" {
		l[monitoringv1alpha1.BindingClassLabel] = class
	}
	return l
}

// ListBindingsForInstance lists the bindings core created for an Instance.
func ListBindingsForInstance(ctx context.Context, c client.Client, namespace, instance string) ([]monitoringv1alpha1.MonitoringBinding, error) {
	list := &monitoringv1alpha1.MonitoringBindingList{}
	if err := c.List(ctx, list,
		client.InNamespace(namespace),
		client.MatchingLabels{monitoringv1alpha1.BindingInstanceLabel: instance},
	); err != nil {
		return nil, fmt.Errorf("list monitoring bindings for instance %q: %w", instance, err)
	}
	return list.Items, nil
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
		for k, v := range e {
			status[k] = v
		}
	}
	return ApplyStatus(ctx, c, obj, fieldOwner, status)
}

// ApplyStatus server-side-applies the given status fragment on obj's status
// subresource under fieldOwner.
func ApplyStatus(ctx context.Context, c client.Client, obj client.Object, fieldOwner string, status map[string]any) error {
	gvk, err := apiutil.GVKForObject(obj, c.Scheme())
	if err != nil {
		return fmt.Errorf("resolve GVK: %w", err)
	}
	u := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": gvk.GroupVersion().String(),
		"kind":       gvk.Kind,
		"metadata": map[string]any{
			"name":      obj.GetName(),
			"namespace": obj.GetNamespace(),
		},
		"status": status,
	}}
	if obj.GetNamespace() == "" {
		delete(u.Object["metadata"].(map[string]any), "namespace")
	}
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

// IsAccepted reports whether the binding's Accepted condition is True for
// the binding's current generation.
func IsAccepted(b *monitoringv1alpha1.MonitoringBinding) bool {
	cond := meta.FindStatusCondition(b.Status.Conditions, monitoringv1alpha1.MonitoringBindingConditionAccepted)
	return cond != nil && cond.Status == metav1.ConditionTrue && cond.ObservedGeneration == b.Generation
}

// ClassIsAccepted reports whether a class has been claimed by its controller.
func ClassIsAccepted(class *monitoringv1alpha1.MonitoringClass) bool {
	return meta.IsStatusConditionTrue(class.Status.Conditions, monitoringv1alpha1.MonitoringClassConditionAccepted)
}
