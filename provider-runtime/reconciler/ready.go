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
	"slices"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

// blocker is a verdict that keeps an Instance from being Ready whatever its
// phase. The zero value blocks nothing.
type blocker struct {
	reason  string
	message string
}

// setReadyCondition derives the Ready condition from the blocker, the
// reference conditions and the phase, in that order. BackupConfigured and
// DataSourceReady keep an Instance in the Ready phase from being Ready while
// False.
func setReadyCondition(in *v1alpha1.Instance, b *blocker, now metav1.Time) {
	if b != nil {
		setCondition(in, v1alpha1.ConditionReady, metav1.ConditionFalse, b.reason, b.message, now)
		return
	}
	for _, t := range []string{v1alpha1.ConditionBackupConfigured, v1alpha1.ConditionDataSourceReady} {
		c := meta.FindStatusCondition(in.Status.Conditions, t)
		if c != nil && c.Status == metav1.ConditionFalse {
			setCondition(in, v1alpha1.ConditionReady, metav1.ConditionFalse, c.Reason, c.Message, now)
			return
		}
	}
	phase := in.Status.Phase
	if phase == v1alpha1.InstancePhaseReady {
		setCondition(in, v1alpha1.ConditionReady, metav1.ConditionTrue, v1alpha1.ReasonReady, "The instance is ready", now)
		return
	}
	if phase == "" {
		phase = v1alpha1.InstancePhasePending
	}
	message := in.Status.Message
	if message == "" {
		message = fmt.Sprintf("The instance is in phase %s", phase)
	}
	setCondition(in, v1alpha1.ConditionReady, metav1.ConditionFalse, string(phase), message, now)
}

// updateStatus records the observed generation and the Ready condition, then
// writes the status.
func (r *ProviderReconciler) updateStatus(ctx context.Context, in *v1alpha1.Instance, b *blocker) error {
	in.Status.ObservedGeneration = in.Generation
	setReadyCondition(in, b, metav1.Now())
	return r.Client.Status().Update(ctx, in)
}

// block reports b on the Instance without calling the provider. A new
// Instance is left Pending.
func (r *ProviderReconciler) block(ctx context.Context, in *v1alpha1.Instance, b *blocker) error {
	if in.Status.Phase == "" {
		in.Status.Phase = v1alpha1.InstancePhasePending
	}
	return r.updateStatus(ctx, in, b)
}

// missingReferent returns a blocker when the Provider or the user Secret the
// Instance names does not exist.
func (r *ProviderReconciler) missingReferent(ctx context.Context, in *v1alpha1.Instance) (blocker, error) {
	providerName := in.Spec.ProviderRef.Name
	if err := r.Get(ctx, client.ObjectKey{Name: providerName}, &v1alpha1.Provider{}); err != nil {
		if !apierrors.IsNotFound(err) {
			return blocker{}, fmt.Errorf("get provider %s: %w", providerName, err)
		}
		return blocker{
			reason:  v1alpha1.ReasonProviderNotFound,
			message: fmt.Sprintf("provider %s not found", providerName),
		}, nil
	}

	if ref := in.Spec.UserSecretRef; ref != nil {
		secret := &metav1.PartialObjectMetadata{}
		secret.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Secret"))
		if err := r.Get(ctx, client.ObjectKey{Namespace: in.Namespace, Name: ref.Name}, secret); err != nil {
			if !apierrors.IsNotFound(err) {
				return blocker{}, fmt.Errorf("get secret %s: %w", ref.Name, err)
			}
			return blocker{
				reason:  v1alpha1.ReasonUserSecretNotFound,
				message: fmt.Sprintf("secret %s named by spec.userSecretRef not found in %s", ref.Name, in.Namespace),
			}, nil
		}
	}
	return blocker{}, nil
}

// existenceChanged passes only creations and deletions: the referent checks
// read whether an object exists, not what it holds.
func existenceChanged() predicate.Funcs {
	return predicate.Funcs{
		UpdateFunc:  func(event.UpdateEvent) bool { return false },
		GenericFunc: func(event.GenericEvent) bool { return false },
	}
}

// instancesForProvider enqueues every Instance of this provider when its
// Provider is created, deleted or changes generation.
func (r *ProviderReconciler) instancesForProvider(ctx context.Context, _ client.Object) []reconcile.Request {
	return r.instancesMatching(ctx, "", func(*v1alpha1.Instance) bool { return true })
}

// instancesForSecret enqueues the Instances of this provider whose
// spec.userSecretRef names the Secret, or whose spec.backup.storages names a
// BackupStorage that keeps its credentials in it.
func (r *ProviderReconciler) instancesForSecret(ctx context.Context, obj client.Object) []reconcile.Request {
	var storages []string
	if r.isBackupProvider() {
		storages = storagesUsingSecret(ctx, r.Client, obj.GetNamespace(), obj.GetName())
	}
	return r.instancesMatching(ctx, obj.GetNamespace(), func(in *v1alpha1.Instance) bool {
		if in.Spec.UserSecretRef != nil && in.Spec.UserSecretRef.Name == obj.GetName() {
			return true
		}
		for _, name := range backupStorageNames(in) {
			if slices.Contains(storages, name) {
				return true
			}
		}
		return false
	})
}

// instancesMatching lists this provider's Instances in namespace (all
// namespaces when empty) that match keep.
func (r *ProviderReconciler) instancesMatching(
	ctx context.Context,
	namespace string,
	keep func(*v1alpha1.Instance) bool,
) []reconcile.Request {
	list := &v1alpha1.InstanceList{}
	if err := r.List(ctx, list, client.InNamespace(namespace)); err != nil {
		log.FromContext(ctx).Error(err, "Listing instances to enqueue failed")
		return nil
	}
	var reqs []reconcile.Request
	for i := range list.Items {
		in := &list.Items[i]
		if in.Spec.ProviderRef.Name == r.provider.Name() && keep(in) {
			reqs = append(reqs, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(in)})
		}
	}
	return reqs
}
