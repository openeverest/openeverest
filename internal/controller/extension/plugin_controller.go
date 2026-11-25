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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	pluginv1alpha1 "github.com/openeverest/openeverest/v2/api/extensions/v1alpha1"
	"github.com/openeverest/openeverest/v2/internal/controller/ssa"
)

const (
	pluginFinalizer = "plugin.core.openeverest.io/finalizer"

	// pluginFieldOwner is the field manager of the status core writes on
	// Plugins: the Available and Ready conditions. Other writers apply their
	// own conditions to the same list under their own field managers.
	pluginFieldOwner = "openeverest-plugin"

	// ConditionTypeAvailable is True when the Plugin CR is valid and the
	// backend (if declared) was reachable at last check.
	ConditionTypeAvailable = "Available"

	// ConditionTypeReady is True when the plugin is enabled and operational.
	ConditionTypeReady = "Ready"

	reasonReconciling  = "Reconciling"
	reasonDisabled     = "Disabled"
	reasonNoBackend    = "NoBackend"
	reasonBackendReady = "BackendReady"
	reasonInvalidSpec  = "InvalidSpec"
)

// PluginReconciler reconciles a Plugin object
//
// Phase 1 responsibilities:
//   - Add/remove a finalizer so we can react to deletion.
//   - Validate required fields (displayName) and surface errors as status conditions.
//   - Set Available/Ready status conditions based on spec.enabled and spec.backend.
//
// Phase 2+ will add: serviceRef DNS resolution, health checks, RBAC auto-provisioning.
// Phase 3+ will add: daemon token minting, NetworkPolicy generation.
type PluginReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=extensions.openeverest.io,resources=plugins,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=extensions.openeverest.io,resources=plugins/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=extensions.openeverest.io,resources=plugins/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *PluginReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).
		WithName("PluginReconciler").
		WithValues("name", req.Name)
	logger.Info("Reconciling")

	plugin := &pluginv1alpha1.Plugin{}
	if err := r.Client.Get(ctx, req.NamespacedName, plugin); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// --- Deletion path ---
	if !plugin.GetDeletionTimestamp().IsZero() {
		if controllerutil.ContainsFinalizer(plugin, pluginFinalizer) {
			// Phase 2+: revoke tokens, remove auto-provisioned RBAC/NetworkPolicy here.
			controllerutil.RemoveFinalizer(plugin, pluginFinalizer)
			if err := r.Client.Update(ctx, plugin); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to remove finalizer: %w", err)
			}
		}
		return ctrl.Result{}, nil
	}

	// --- Ensure finalizer ---
	if !controllerutil.ContainsFinalizer(plugin, pluginFinalizer) {
		controllerutil.AddFinalizer(plugin, pluginFinalizer)
		if err := r.Client.Update(ctx, plugin); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to add finalizer: %w", err)
		}
		// Re-fetch after update so we work with the latest resourceVersion.
		if err := r.Client.Get(ctx, req.NamespacedName, plugin); err != nil {
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
	}

	// --- Reconcile status conditions ---
	// Apply only the conditions this controller owns, under its own field
	// manager: a merge patch replaces the whole list, so it would erase the
	// conditions other writers set on the Plugin, even ones set after our
	// read.
	conditions := pluginConditions(plugin)
	if err := ssa.ApplyConditions(ctx, r.Client, plugin, plugin.Status.Conditions, pluginFieldOwner, conditions); err != nil && !apierrors.IsNotFound(err) {
		return ctrl.Result{}, err
	}

	logger.Info("Reconciled", "ready", isConditionTrue(conditions, ConditionTypeReady))
	return ctrl.Result{}, nil
}

// pluginConditions returns the Available and Ready conditions, based on spec
// state.
func pluginConditions(plugin *pluginv1alpha1.Plugin) []metav1.Condition {
	// Available: plugin spec is valid.
	availCond := metav1.Condition{
		Type:               ConditionTypeAvailable,
		ObservedGeneration: plugin.Generation,
	}
	if plugin.Spec.DisplayName == "" {
		availCond.Status = metav1.ConditionFalse
		availCond.Reason = reasonInvalidSpec
		availCond.Message = "spec.displayName is required"
	} else {
		availCond.Status = metav1.ConditionTrue
		availCond.Reason = reasonReconciling
		availCond.Message = "Plugin spec is valid"
	}

	// Ready: plugin is enabled and has a backend (or is frontend-only).
	readyCond := metav1.Condition{
		Type:               ConditionTypeReady,
		ObservedGeneration: plugin.Generation,
	}
	switch {
	case !plugin.Spec.Enabled:
		readyCond.Status = metav1.ConditionFalse
		readyCond.Reason = reasonDisabled
		readyCond.Message = "Plugin is disabled (spec.enabled=false)"
	case plugin.Spec.Frontend == nil && plugin.Spec.Backend == nil:
		readyCond.Status = metav1.ConditionFalse
		readyCond.Reason = reasonNoBackend
		readyCond.Message = "Plugin has neither frontend nor backend configured"
	default:
		readyCond.Status = metav1.ConditionTrue
		readyCond.Reason = reasonBackendReady
		readyCond.Message = "Plugin is enabled and configured"
	}

	return []metav1.Condition{availCond, readyCond}
}

func isConditionTrue(conditions []metav1.Condition, condType string) bool {
	for _, c := range conditions {
		if c.Type == condType {
			return c.Status == metav1.ConditionTrue
		}
	}
	return false
}

// SetupWithManager sets up the controller with the Manager.
func (r *PluginReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&pluginv1alpha1.Plugin{}).
		Named("plugin-plugin").
		Complete(r)
}
