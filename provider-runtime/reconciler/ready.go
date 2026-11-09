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
	"strings"
	"unicode"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/csaupgrade"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
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

// runtimeCondition reports whether the runtime writes Instance conditions
// of type t. It applies only these: a condition of any other type has
// another writer, such as core, and is left to it.
func runtimeCondition(t string) bool {
	switch t {
	case v1alpha1.ConditionReady,
		v1alpha1.ConditionBackupConfigured,
		v1alpha1.ConditionDataSourceReady,
		v1alpha1.ConditionConnectionDetailsReady,
		v1alpha1.ConditionComponentVersionDeprecated,
		v1alpha1.ConditionMaintenancePending,
		v1alpha1.ConditionMonitoringConfigured,
		v1alpha1.ConditionPodsScheduled,
		v1alpha1.ConditionPodsReady:
		return true
	}
	return false
}

// updateStatus records the observed generation and the Ready condition, then
// applies the status.
func (r *ProviderReconciler) updateStatus(ctx context.Context, in *v1alpha1.Instance, b *blocker) error {
	in.Status.ObservedGeneration = in.Generation
	setReadyCondition(in, b, metav1.Now())
	return r.applyStatus(ctx, in)
}

// applyStatus server-side-applies the status fields the runtime owns, every
// field of in.Status but the conditions runtimeCondition does not list,
// under the provider's status field owner, with in's resourceVersion as a
// precondition. A field the runtime stops setting is removed, while the
// fields other writers apply, and the fields this binary's types do not
// know, are left alone (spec 019 §4.8 invariant 7). in's resourceVersion is
// advanced to the written one.
func (r *ProviderReconciler) applyStatus(ctx context.Context, in *v1alpha1.Instance) error {
	owner := controller.StatusFieldOwner(r.provider.Name())
	if err := r.adoptLegacyStatusFields(ctx, in, owner); err != nil {
		return err
	}
	st := in.Status.DeepCopy()
	st.Conditions = slices.DeleteFunc(st.Conditions, func(c metav1.Condition) bool { return !runtimeCondition(c.Type) })
	status, err := runtime.DefaultUnstructuredConverter.ToUnstructured(st)
	if err != nil {
		return fmt.Errorf("convert the status of instance %s: %w", in.Name, err)
	}
	metadata := map[string]any{"name": in.Name, "namespace": in.Namespace}
	if in.ResourceVersion != "" {
		metadata["resourceVersion"] = in.ResourceVersion
	}
	u := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": v1alpha1.GroupVersion.String(),
		"kind":       instanceKind,
		"metadata":   metadata,
		"status":     status,
	}}
	if err := r.Status().Apply(ctx, client.ApplyConfigurationFromUnstructured(u),
		client.FieldOwner(owner), client.ForceOwnership,
	); err != nil {
		return err
	}
	if rv := u.GetResourceVersion(); rv != "" {
		in.ResourceVersion = rv
	}
	return nil
}

// instanceKind is the kind of Instance objects.
const instanceKind = "Instance"

// adoptLegacyStatusFields hands the status fields that an earlier runtime
// wrote with Update, under the client's default field manager, to owner, so
// the applies under owner own them alone and remove them once the runtime
// stops setting them. It patches the managed fields once per Instance and
// advances in's resourceVersion.
func (r *ProviderReconciler) adoptLegacyStatusFields(ctx context.Context, in *v1alpha1.Instance, owner string) error {
	legacy := r.legacyFieldManager
	if legacy == "" || !slices.ContainsFunc(in.ManagedFields, func(e metav1.ManagedFieldsEntry) bool {
		return e.Manager == legacy && e.Operation == metav1.ManagedFieldsOperationUpdate && e.Subresource == statusSubresource
	}) {
		return nil
	}
	patch, err := csaupgrade.UpgradeManagedFieldsPatch(in, sets.New(legacy), owner, csaupgrade.Subresource(statusSubresource))
	if err != nil || patch == nil {
		return err
	}
	adopted := in.DeepCopy()
	if err := r.Patch(ctx, adopted, client.RawPatch(types.JSONPatchType, patch)); err != nil {
		return fmt.Errorf("adopt the status fields of field manager %s: %w", legacy, err)
	}
	in.ResourceVersion, in.ManagedFields = adopted.ResourceVersion, adopted.ManagedFields
	return nil
}

// statusSubresource is the subresource managed fields record for status
// writes.
const statusSubresource = "status"

// defaultFieldManager returns the field manager the API server records for
// the writes that name none, made by a client with userAgent: its first
// segment, up to 128 printable characters.
func defaultFieldManager(userAgent string) string {
	if userAgent == "" {
		userAgent = rest.DefaultKubernetesUserAgent()
	}
	prefix, _, _ := strings.Cut(userAgent, "/")
	var b strings.Builder
	for _, c := range prefix {
		if b.Len() >= maxFieldManagerLength {
			break
		}
		if unicode.IsPrint(c) {
			b.WriteRune(c)
		}
	}
	return b.String()
}

// maxFieldManagerLength is the length the API server truncates field
// managers to.
const maxFieldManagerLength = 128

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
func missingReferent(ctx context.Context, c client.Reader, in *v1alpha1.Instance) (blocker, error) {
	providerName := in.Spec.ProviderRef.Name
	if err := c.Get(ctx, client.ObjectKey{Name: providerName}, &v1alpha1.Provider{}); err != nil {
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
		if err := c.Get(ctx, client.ObjectKey{Namespace: in.Namespace, Name: ref.Name}, secret); err != nil {
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
// spec.userSecretRef names the Secret.
func (r *ProviderReconciler) instancesForSecret(ctx context.Context, obj client.Object) []reconcile.Request {
	return r.instancesMatching(ctx, obj.GetNamespace(), func(in *v1alpha1.Instance) bool {
		return in.Spec.UserSecretRef != nil && in.Spec.UserSecretRef.Name == obj.GetName()
	})
}

// instancesForBackupStorageSecret enqueues the Instances of this provider
// whose spec.backup.storages names a BackupStorage that references the
// Secret, for credentials or a CA bundle.
func (r *ProviderReconciler) instancesForBackupStorageSecret(ctx context.Context, obj client.Object) []reconcile.Request {
	storages := storagesUsingSecret(ctx, r.Client, obj.GetNamespace(), obj.GetName())
	if len(storages) == 0 {
		return nil
	}
	return r.instancesMatching(ctx, obj.GetNamespace(), func(in *v1alpha1.Instance) bool {
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
