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

package monitoring

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
)

// ClassController is the handle a monitoring extension uses to claim the
// MonitoringClass it serves and report on its bindings and destinations.
type ClassController struct {
	Client client.Client
	// FieldOwner is the server-side-apply field manager of every status write
	// this controller makes (class Accepted, destination Ready, binding
	// Configured).
	FieldOwner string
	// ClassName is the MonitoringClass this controller serves.
	ClassName string
}

// Claim gets the class named ClassName and claims it. A missing class is not
// an error: there is nothing to claim yet.
func (cc ClassController) Claim(ctx context.Context) error {
	class := &monitoringv1alpha1.MonitoringClass{}
	if err := cc.Client.Get(ctx, client.ObjectKey{Name: cc.ClassName}, class); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("get MonitoringClass %q: %w", cc.ClassName, err)
	}
	return cc.ClaimClass(ctx, class)
}

// ClaimClass sets Accepted=True, observed at the class generation, on class
// if it is the one named ClassName and leaves any other class untouched. Call
// it from a MonitoringClass reconciler so a failed claim is retried.
func (cc ClassController) ClaimClass(ctx context.Context, class *monitoringv1alpha1.MonitoringClass) error {
	if !cc.OwnsClass(class) {
		return nil
	}
	cond := NewCondition(class, monitoringv1alpha1.MonitoringClassConditionAccepted, metav1.ConditionTrue,
		monitoringv1alpha1.ReasonAccepted, "claimed by "+cc.FieldOwner,
	)
	return ApplyCondition(ctx, cc.Client, class, class.Status.Conditions, cc.FieldOwner, cond)
}

// OwnsClass reports whether class is the one this controller serves.
func (cc ClassController) OwnsClass(class *monitoringv1alpha1.MonitoringClass) bool {
	return class.Name == cc.ClassName
}

// BindingPredicate passes only ExtensionManaged bindings whose class label
// names a class in classes; use it on the MonitoringBinding watch so the
// extension reconciles only its own bindings. It filters events only: the
// informer still caches every binding in the cluster.
func BindingPredicate(classes ...string) predicate.Predicate { //nolint:ireturn // controller-runtime's builder takes the interface
	set := make(map[string]struct{}, len(classes))
	for _, c := range classes {
		set[c] = struct{}{}
	}
	return predicate.NewPredicateFuncs(func(obj client.Object) bool {
		b, ok := obj.(*monitoringv1alpha1.MonitoringBinding)
		if !ok || b.Spec.ExecutionMode != corev1alpha1.MonitoringExecutionModeExtensionManaged {
			return false
		}
		_, mine := set[obj.GetLabels()[monitoringv1alpha1.BindingClassLabel]]
		return mine
	})
}

// Resolve loads the Instance and MonitoringDestination a binding points at.
// The Instance's status.monitoring.sources is what an ExtensionManaged
// controller consumes.
func (cc ClassController) Resolve(ctx context.Context, b *monitoringv1alpha1.MonitoringBinding) (*corev1alpha1.Instance, *monitoringv1alpha1.MonitoringDestination, error) {
	in := &corev1alpha1.Instance{}
	if err := cc.Client.Get(ctx, client.ObjectKey{Namespace: b.Namespace, Name: b.Spec.InstanceRef.Name}, in); err != nil {
		return nil, nil, fmt.Errorf("get Instance %q: %w", b.Spec.InstanceRef.Name, err)
	}
	dst := &monitoringv1alpha1.MonitoringDestination{}
	if err := cc.Client.Get(ctx, client.ObjectKey{Namespace: b.Namespace, Name: b.Spec.DestinationRef.Name}, dst); err != nil {
		return in, nil, fmt.Errorf("get MonitoringDestination %q: %w", b.Spec.DestinationRef.Name, err)
	}
	return in, dst, nil
}

// SetConfigured writes the binding's Configured condition, the only binding
// condition an ExtensionManaged controller owns. The status is derived from
// the reason.
func (cc ClassController) SetConfigured(ctx context.Context, b *monitoringv1alpha1.MonitoringBinding, reason monitoringv1alpha1.ConfiguredReason, message string) error {
	cond := NewCondition(b, monitoringv1alpha1.MonitoringBindingConditionConfigured, reason.Status(), string(reason), message)
	return ApplyCondition(ctx, cc.Client, b, b.Status.Conditions, cc.FieldOwner, cond)
}

// SetDestinationReady writes the destination's Ready condition and
// serverVersion, the only destination status a class controller owns, in
// one apply. The status is derived from the reason. An empty serverVersion
// keeps the last reported one.
func (cc ClassController) SetDestinationReady(ctx context.Context, dst *monitoringv1alpha1.MonitoringDestination, reason monitoringv1alpha1.ReadyReason, message, serverVersion string) error {
	cond := NewCondition(dst, monitoringv1alpha1.MonitoringDestinationConditionReady, reason.Status(), string(reason), message)
	if serverVersion == "" {
		serverVersion = dst.Status.ServerVersion
	}
	extra := make(map[string]any)
	if serverVersion != "" {
		extra["serverVersion"] = serverVersion
	}
	return ApplyCondition(ctx, cc.Client, dst, dst.Status.Conditions, cc.FieldOwner, cond, extra)
}

// OwnedBy returns the controller owner reference an extension puts on the
// backend objects it creates for a binding, so removing the destination
// entry garbage-collects them even if the extension is uninstalled.
// BlockOwnerDeletion stays unset: it would need update permission on
// monitoringbindings/finalizers, which extensions are not granted.
func OwnedBy(b *monitoringv1alpha1.MonitoringBinding) metav1.OwnerReference {
	return metav1.OwnerReference{
		APIVersion: monitoringv1alpha1.GroupVersion.String(),
		Kind:       "MonitoringBinding",
		Name:       b.Name,
		UID:        b.UID,
		Controller: new(true),
	}
}
