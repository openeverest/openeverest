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

package plugin

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

	pluginv1alpha1 "github.com/openeverest/openeverest/v2/api/plugin/v1alpha1"
)

const (
	pluginInstallationFinalizer = "plugininstallation.plugin.openeverest.io/finalizer"

	reasonPluginNotFound  = "PluginNotFound"
	reasonPluginDisabled  = "PluginDisabled"
	reasonInstallDisabled = "InstallationDisabled"
	reasonReady           = "Ready"
)

// PluginInstallationReconciler reconciles a PluginInstallation object
//
// Phase 1 responsibilities:
//   - Add/remove a finalizer.
//   - Validate that the referenced Plugin CR exists and is enabled.
//   - Surface errors as status conditions.
//
// Phase 2+: per-namespace config injection, token minting.
// Phase 3+: Deployment lifecycle management.
type PluginInstallationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=plugin.openeverest.io,resources=plugininstallations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=plugin.openeverest.io,resources=plugininstallations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=plugin.openeverest.io,resources=plugininstallations/finalizers,verbs=update
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *PluginInstallationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).
		WithName("PluginInstallationReconciler").
		WithValues("name", req.Name, "namespace", req.Namespace)
	logger.Info("Reconciling")

	pi := &pluginv1alpha1.PluginInstallation{}
	if err := r.Client.Get(ctx, req.NamespacedName, pi); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// --- Deletion path ---
	if !pi.GetDeletionTimestamp().IsZero() {
		if hasFinalizer(pi, pluginInstallationFinalizer) {
			// Clean up RBAC resources created for kubePermissions.
			if err := r.deletePluginRole(ctx, pi); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to clean up plugin Role on deletion: %w", err)
			}
			if err := r.deletePluginRoleBinding(ctx, pi); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to clean up plugin RoleBinding on deletion: %w", err)
			}
			removeFinalizer(pi, pluginInstallationFinalizer)
			if err := r.Client.Update(ctx, pi); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to remove finalizer: %w", err)
			}
		}
		return ctrl.Result{}, nil
	}

	// --- Ensure finalizer ---
	if !hasFinalizer(pi, pluginInstallationFinalizer) {
		addFinalizer(pi, pluginInstallationFinalizer)
		if err := r.Client.Update(ctx, pi); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to add finalizer: %w", err)
		}
		if err := r.Client.Get(ctx, req.NamespacedName, pi); err != nil {
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
	}

	// --- Reconcile status conditions ---
	patch := client.MergeFrom(pi.DeepCopy())

	plugin := &pluginv1alpha1.Plugin{}
	pluginErr := r.Client.Get(ctx, types.NamespacedName{Name: pi.Spec.PluginName}, plugin)

	r.reconcileConditions(pi, plugin, pluginErr)

	if err := r.Client.Status().Patch(ctx, pi, patch); err != nil && !apierrors.IsNotFound(err) {
		return ctrl.Result{}, fmt.Errorf("failed to patch plugininstallation status: %w", err)
	}

	// --- Ensure Role/RoleBinding for kubePermissions ---
	if pluginErr == nil && plugin.Spec.Enabled && pi.Spec.Enabled {
		if err := r.ensurePluginRole(ctx, pi, plugin); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to ensure plugin Role: %w", err)
		}
		if err := r.ensurePluginRoleBinding(ctx, pi, plugin); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to ensure plugin RoleBinding: %w", err)
		}
	} else if pluginErr == nil {
		// Plugin or installation disabled — clean up RBAC resources.
		if err := r.deletePluginRole(ctx, pi); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to delete plugin Role: %w", err)
		}
		if err := r.deletePluginRoleBinding(ctx, pi); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to delete plugin RoleBinding: %w", err)
		}
	}

	logger.Info("Reconciled", "ready", isConditionTrue(pi.Status.Conditions, ConditionTypeReady))
	return ctrl.Result{}, nil
}

func (r *PluginInstallationReconciler) reconcileConditions(
	pi *pluginv1alpha1.PluginInstallation,
	plugin *pluginv1alpha1.Plugin,
	pluginErr error,
) {
	now := metav1.Now()
	readyCond := metav1.Condition{
		Type:               ConditionTypeReady,
		ObservedGeneration: pi.Generation,
		LastTransitionTime: now,
	}

	switch {
	case pluginErr != nil && apierrors.IsNotFound(pluginErr):
		readyCond.Status = metav1.ConditionFalse
		readyCond.Reason = reasonPluginNotFound
		readyCond.Message = fmt.Sprintf("Plugin %q not found", pi.Spec.PluginName)

	case pluginErr != nil:
		readyCond.Status = metav1.ConditionFalse
		readyCond.Reason = reasonPluginNotFound
		readyCond.Message = fmt.Sprintf("Failed to look up Plugin %q: %v", pi.Spec.PluginName, pluginErr)

	case !plugin.Spec.Enabled:
		readyCond.Status = metav1.ConditionFalse
		readyCond.Reason = reasonPluginDisabled
		readyCond.Message = fmt.Sprintf("Plugin %q is disabled", pi.Spec.PluginName)

	case !pi.Spec.Enabled:
		readyCond.Status = metav1.ConditionFalse
		readyCond.Reason = reasonInstallDisabled
		readyCond.Message = "PluginInstallation is disabled (spec.enabled=false)"

	default:
		readyCond.Status = metav1.ConditionTrue
		readyCond.Reason = reasonReady
		readyCond.Message = fmt.Sprintf("Plugin %q is installed and ready", pi.Spec.PluginName)
	}

	meta.SetStatusCondition(&pi.Status.Conditions, readyCond)
}

// --- small finalizer helpers to avoid importing controllerutil twice ---

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

// SetupWithManager sets up the controller with the Manager.
func (r *PluginInstallationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&pluginv1alpha1.PluginInstallation{}).
		Watches(
			&pluginv1alpha1.Plugin{},
			handler.EnqueueRequestsFromMapFunc(r.pluginToInstallations),
		).
		Named("plugin-plugininstallation").
		Complete(r)
}

// pluginToInstallations maps a Plugin event to all PluginInstallations that reference it.
func (r *PluginInstallationReconciler) pluginToInstallations(
	ctx context.Context,
	obj client.Object,
) []reconcile.Request {
	pluginName := obj.GetName()
	list := &pluginv1alpha1.PluginInstallationList{}
	if err := r.Client.List(ctx, list); err != nil {
		return nil
	}
	var requests []reconcile.Request
	for _, pi := range list.Items {
		if pi.Spec.PluginName == pluginName {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: pi.Namespace,
					Name:      pi.Name,
				},
			})
		}
	}
	return requests
}
