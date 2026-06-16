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

package extension

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	coreapi "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	corev1alpha1 "github.com/openeverest/openeverest/v2/api/extensions/v1alpha1"
)

const (
	installedExtensionFinalizer = "installedextension.core.openeverest.io/finalizer"
)

// InstalledExtensionReconciler reconciles an InstalledExtension object.
//
// Responsibilities:
//   - Add/remove a finalizer for clean teardown of provisioned RBAC.
//   - For type=plugin: validate the referenced Plugin CR exists and is enabled;
//     gate cluster-scope RBAC behind spec.plugin.allowClusterScope; provision
//     ClusterRole/ClusterRoleBinding or per-namespace Role/RoleBinding from
//     the Plugin's kubePermissions.
//   - For type=provider: validate the referenced Provider CR exists.
//   - Set status.phase + per-aspect Conditions.
type InstalledExtensionReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=extensions.openeverest.io,resources=installedextensions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=extensions.openeverest.io,resources=installedextensions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=extensions.openeverest.io,resources=installedextensions/finalizers,verbs=update
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles;rolebindings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles;clusterrolebindings,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop.
func (r *InstalledExtensionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).
		WithName("InstalledExtensionReconciler").
		WithValues("name", req.Name)
	logger.Info("Reconciling")

	ie := &corev1alpha1.InstalledExtension{}
	if err := r.Client.Get(ctx, req.NamespacedName, ie); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// --- Deletion path ---
	if !ie.GetDeletionTimestamp().IsZero() {
		if hasFinalizer(ie, installedExtensionFinalizer) {
			if err := r.cleanupRBAC(ctx, ie); err != nil {
				return ctrl.Result{}, fmt.Errorf("cleanup RBAC: %w", err)
			}
			removeFinalizer(ie, installedExtensionFinalizer)
			if err := r.Client.Update(ctx, ie); err != nil {
				return ctrl.Result{}, fmt.Errorf("remove finalizer: %w", err)
			}
		}
		return ctrl.Result{}, nil
	}

	// --- Ensure finalizer ---
	if !hasFinalizer(ie, installedExtensionFinalizer) {
		addFinalizer(ie, installedExtensionFinalizer)
		if err := r.Client.Update(ctx, ie); err != nil {
			return ctrl.Result{}, fmt.Errorf("add finalizer: %w", err)
		}
		if err := r.Client.Get(ctx, req.NamespacedName, ie); err != nil {
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
	}

	// --- Reconcile status + RBAC ---
	patch := client.MergeFrom(ie.DeepCopy())

	switch ie.Spec.Type {
	case corev1alpha1.InstalledExtensionTypePlugin:
		if err := r.reconcilePlugin(ctx, ie); err != nil {
			return ctrl.Result{}, err
		}
	case corev1alpha1.InstalledExtensionTypeProvider:
		r.reconcileProvider(ctx, ie)
	default:
		setCondition(ie, corev1alpha1.ConditionReady, metav1.ConditionFalse,
			corev1alpha1.ReasonInvalidSpec, fmt.Sprintf("unsupported spec.type: %q", ie.Spec.Type))
		ie.Status.Phase = corev1alpha1.InstalledExtensionPhaseFailed
	}

	if err := r.Client.Status().Patch(ctx, ie, patch); err != nil && !apierrors.IsNotFound(err) {
		return ctrl.Result{}, fmt.Errorf("patch InstalledExtension status: %w", err)
	}

	logger.Info("Reconciled", "phase", ie.Status.Phase)
	return ctrl.Result{}, nil
}

// reconcilePlugin handles type=plugin: looks up the Plugin CR, validates
// scope/opt-in, then provisions or refuses to provision RBAC.
func (r *InstalledExtensionReconciler) reconcilePlugin(ctx context.Context, ie *corev1alpha1.InstalledExtension) error {
	if ie.Spec.Plugin == nil {
		setCondition(ie, corev1alpha1.ConditionReady, metav1.ConditionFalse,
			corev1alpha1.ReasonInvalidSpec, "spec.plugin is required when spec.type=plugin")
		ie.Status.Phase = corev1alpha1.InstalledExtensionPhaseFailed
		return nil
	}

	plugin := &corev1alpha1.Plugin{}
	pluginErr := r.Client.Get(ctx, types.NamespacedName{Name: ie.Spec.Plugin.PluginCRName}, plugin)

	switch {
	case apierrors.IsNotFound(pluginErr):
		setCondition(ie, corev1alpha1.ConditionReady, metav1.ConditionFalse,
			corev1alpha1.ReasonPluginNotFound,
			fmt.Sprintf("Plugin %q not found", ie.Spec.Plugin.PluginCRName))
		ie.Status.Phase = corev1alpha1.InstalledExtensionPhaseFailed
		return nil
	case pluginErr != nil:
		setCondition(ie, corev1alpha1.ConditionReady, metav1.ConditionFalse,
			corev1alpha1.ReasonPluginNotFound,
			fmt.Sprintf("Failed to look up Plugin %q: %v", ie.Spec.Plugin.PluginCRName, pluginErr))
		return pluginErr
	case !plugin.Spec.Enabled:
		setCondition(ie, corev1alpha1.ConditionReady, metav1.ConditionFalse,
			corev1alpha1.ReasonPluginDisabled,
			fmt.Sprintf("Plugin %q is disabled", plugin.Name))
		ie.Status.Phase = corev1alpha1.InstalledExtensionPhaseFailed
		// Still clean up RBAC for a disabled plugin.
		return r.cleanupRBAC(ctx, ie)
	}

	// Cluster scope gating.
	if ie.Spec.Plugin.Scope == corev1alpha1.PluginInstallScopeCluster && !ie.Spec.Plugin.AllowClusterScope {
		setCondition(ie, corev1alpha1.ConditionRoleSynced, metav1.ConditionFalse,
			corev1alpha1.ReasonClusterScopeNotAllowed,
			"spec.plugin.scope=Cluster requires spec.plugin.allowClusterScope=true")
		setCondition(ie, corev1alpha1.ConditionReady, metav1.ConditionFalse,
			corev1alpha1.ReasonClusterScopeNotAllowed,
			"cluster-scope RBAC refused; set spec.plugin.allowClusterScope=true to opt in")
		ie.Status.Phase = corev1alpha1.InstalledExtensionPhaseFailed
		return nil
	}

	// Provision RBAC.
	if err := r.ensureRBAC(ctx, ie, plugin); err != nil {
		setCondition(ie, corev1alpha1.ConditionRoleSynced, metav1.ConditionFalse,
			corev1alpha1.ReasonReconciling, err.Error())
		return err
	}
	setCondition(ie, corev1alpha1.ConditionRoleSynced, metav1.ConditionTrue,
		corev1alpha1.ReasonReady, "RBAC provisioned")
	setCondition(ie, corev1alpha1.ConditionReady, metav1.ConditionTrue,
		corev1alpha1.ReasonReady,
		fmt.Sprintf("InstalledExtension %q is installed", ie.Name))
	ie.Status.Phase = corev1alpha1.InstalledExtensionPhaseInstalled
	return nil
}

// reconcileProvider handles type=provider: validates the Provider CR exists.
func (r *InstalledExtensionReconciler) reconcileProvider(ctx context.Context, ie *corev1alpha1.InstalledExtension) {
	if ie.Spec.Provider == nil {
		setCondition(ie, corev1alpha1.ConditionReady, metav1.ConditionFalse,
			corev1alpha1.ReasonInvalidSpec, "spec.provider is required when spec.type=provider")
		ie.Status.Phase = corev1alpha1.InstalledExtensionPhaseFailed
		return
	}
	provider := &coreapi.Provider{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: ie.Spec.Provider.ProviderName}, provider)
	switch {
	case apierrors.IsNotFound(err):
		setCondition(ie, corev1alpha1.ConditionProviderRegistered, metav1.ConditionFalse,
			corev1alpha1.ReasonProviderNotFound,
			fmt.Sprintf("Provider %q not found", ie.Spec.Provider.ProviderName))
		setCondition(ie, corev1alpha1.ConditionReady, metav1.ConditionFalse,
			corev1alpha1.ReasonProviderNotFound, "provider missing")
		ie.Status.Phase = corev1alpha1.InstalledExtensionPhaseFailed
	case err != nil:
		setCondition(ie, corev1alpha1.ConditionProviderRegistered, metav1.ConditionFalse,
			corev1alpha1.ReasonReconciling, err.Error())
	default:
		setCondition(ie, corev1alpha1.ConditionProviderRegistered, metav1.ConditionTrue,
			corev1alpha1.ReasonReady, "Provider registered")
		setCondition(ie, corev1alpha1.ConditionReady, metav1.ConditionTrue,
			corev1alpha1.ReasonReady,
			fmt.Sprintf("Provider %q is installed", provider.Name))
		ie.Status.Phase = corev1alpha1.InstalledExtensionPhaseInstalled
	}
}

func setCondition(ie *corev1alpha1.InstalledExtension, condType string, status metav1.ConditionStatus, reason, message string) {
	meta.SetStatusCondition(&ie.Status.Conditions, metav1.Condition{
		Type:               condType,
		Status:             status,
		ObservedGeneration: ie.Generation,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            message,
	})
}

// --- small finalizer helpers ---

func hasFinalizer(obj metav1.Object, finalizer string) bool {
	for _, f := range obj.GetFinalizers() {
		if f == finalizer {
			return true
		}
	}
	return false
}

func addFinalizer(obj metav1.Object, finalizer string) {
	obj.SetFinalizers(append(obj.GetFinalizers(), finalizer))
}

func removeFinalizer(obj metav1.Object, finalizer string) {
	finalizers := obj.GetFinalizers()
	updated := finalizers[:0]
	for _, f := range finalizers {
		if f != finalizer {
			updated = append(updated, f)
		}
	}
	obj.SetFinalizers(updated)
}

// SetupWithManager wires the controller with the Manager.
func (r *InstalledExtensionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1alpha1.InstalledExtension{}).
		Watches(
			&corev1alpha1.Plugin{},
			handler.EnqueueRequestsFromMapFunc(r.pluginToInstalls),
		).
		Named("core-installedextension").
		Complete(r)
}

// pluginToInstalls maps a Plugin event to all InstalledExtension records that
// reference it.
func (r *InstalledExtensionReconciler) pluginToInstalls(ctx context.Context, obj client.Object) []reconcile.Request {
	pluginName := obj.GetName()
	list := &corev1alpha1.InstalledExtensionList{}
	if err := r.Client.List(ctx, list); err != nil {
		return nil
	}
	var requests []reconcile.Request
	for _, ie := range list.Items {
		if ie.Spec.Type != corev1alpha1.InstalledExtensionTypePlugin {
			continue
		}
		if ie.Spec.Plugin == nil || ie.Spec.Plugin.PluginCRName != pluginName {
			continue
		}
		requests = append(requests, reconcile.Request{
			NamespacedName: types.NamespacedName{Name: ie.Name},
		})
	}
	return requests
}
