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

// Package monitoring contains the MonitoringConfigReconciler which manages MonitoringConfig resources.
package monitoring

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/AlekSi/pointer"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
	"github.com/openeverest/openeverest/v2/pkg/pmm"
)

const (
	// inUseFinalizer prevents deletion of a MonitoringConfig that is referenced by Instances.
	inUseFinalizer = "monitoring.openeverest.io/in-use-protection"

	// instanceMonitoringConfigField is the field path used for indexing Instances
	// by their monitoring config name.
	instanceMonitoringConfigField = ".spec.components.monitoring.customSpec.monitoringConfigName"
)

// MonitoringConfigReconciler reconciles a MonitoringConfig object.
type MonitoringConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (r *MonitoringConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := r.initIndexers(context.Background(), mgr); err != nil {
		return fmt.Errorf("init field indexers: %w", err)
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named("MonitoringConfig").
		For(&monitoringv1alpha1.MonitoringConfig{}).
		Watches(&corev1.Namespace{},
			enqueueObjectsInNamespace(r.Client, &monitoringv1alpha1.MonitoringConfigList{})).
		Complete(r)
}

// +kubebuilder:rbac:groups=monitoring.openeverest.io,resources=monitoringconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=monitoring.openeverest.io,resources=monitoringconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=monitoring.openeverest.io,resources=monitoringconfigs/finalizers,verbs=update
// +kubebuilder:rbac:groups=core.openeverest.io,resources=instances,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// Modify the Reconcile function to compare the state specified by
// the MonitoringConfig object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
// The reconciliation flow:
//  1. Fetch the MonitoringConfig; bail out on deletion or not-found.
//  2. Determine whether any Instance references this config (in-use check).
//  3. Defer a status update so it always runs after the main logic.
//  4. Add or remove the in-use protection finalizer.
//  5. Ensure the credentials Secret is owned by this MonitoringConfig.
func (r *MonitoringConfigReconciler) Reconcile( //nolint:nonamedreturns
	ctx context.Context,
	req ctrl.Request,
) (rr ctrl.Result, rerr error) {
	logger := log.FromContext(ctx).
		WithName("MonitoringConfigReconciler").
		WithValues("name", req.Name, "namespace", req.Namespace)

	logger.Info("Reconciling")
	defer func() { logger.Info("Reconciled") }()

	mc := &monitoringv1alpha1.MonitoringConfig{}
	if err := r.Get(ctx, req.NamespacedName, mc); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Nothing to reconcile on a resource that is being deleted.
	if !mc.GetDeletionTimestamp().IsZero() {
		return ctrl.Result{}, nil
	}

	// Determine whether any Instance references this MonitoringConfig.
	inUse, err := r.isInUse(ctx, mc)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to check in-use status: %w", err)
	}

	// Sync status after reconciliation completes.
	defer func() {
		if updErr := r.updateStatus(ctx, mc, inUse); updErr != nil {
			logger.Error(updErr, "Failed to update status")
			rr = ctrl.Result{}
			rerr = errors.Join(rerr, updErr)
		}
	}()

	// Manage the in-use protection finalizer.
	if err := r.ensureInUseFinalizer(ctx, mc, inUse); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to ensure in-use finalizer: %w", err)
	}

	// Ensure the credentials Secret is owned by this MonitoringConfig.
	if err := r.ensureSecretOwnership(ctx, mc); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to ensure secret ownership: %w", err)
	}

	return ctrl.Result{}, nil
}

// isInUse returns true if any Instance in the same namespace references this MonitoringConfig.
func (r *MonitoringConfigReconciler) isInUse(ctx context.Context, mc *monitoringv1alpha1.MonitoringConfig) (bool, error) {
	instances := &corev1alpha1.InstanceList{}
	if err := r.List(ctx, instances, &client.ListOptions{
		FieldSelector: fields.OneTermEqualSelector(instanceMonitoringConfigField, mc.GetName()),
		Namespace:     mc.GetNamespace(),
	}); err != nil {
		return false, fmt.Errorf("failed to list instances: %w", err)
	}

	return len(instances.Items) > 0, nil
}

// ensureInUseFinalizer adds or removes the in-use protection finalizer based on
// whether the MonitoringConfig is referenced by any Instance.
func (r *MonitoringConfigReconciler) ensureInUseFinalizer(
	ctx context.Context,
	mc *monitoringv1alpha1.MonitoringConfig,
	inUse bool,
) error {
	var updated bool
	if inUse {
		updated = controllerutil.AddFinalizer(mc, inUseFinalizer)
	} else {
		updated = controllerutil.RemoveFinalizer(mc, inUseFinalizer)
	}

	if updated {
		if err := r.Update(ctx, mc); err != nil {
			return fmt.Errorf("failed toupdate finalizer: %w", err)
		}
	}

	return nil
}

// ensureSecretOwnership sets this MonitoringConfig as the controller owner of the
// referenced credentials Secret, if not already owned.
func (r *MonitoringConfigReconciler) ensureSecretOwnership(
	ctx context.Context,
	mc *monitoringv1alpha1.MonitoringConfig,
) error {
	secret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{
		Name:      mc.Spec.CredentialsSecretName,
		Namespace: mc.GetNamespace(),
	}, secret); err != nil {
		return fmt.Errorf("failed to get credentials secret %q: %w", mc.Spec.CredentialsSecretName, err)
	}

	// Skip if the Secret already has a controller owner.
	if metav1.GetControllerOf(secret) != nil {
		return nil
	}

	if err := controllerutil.SetControllerReference(mc, secret, r.Scheme); err != nil {
		return fmt.Errorf("failed to set controller reference: %w", err)
	}

	if err := r.Update(ctx, secret); err != nil {
		return fmt.Errorf("failed to update secret: %w", err)
	}

	return nil
}

// updateStatus rebuilds and persists the MonitoringConfig status subresource.
func (r *MonitoringConfigReconciler) updateStatus(
	ctx context.Context,
	mc *monitoringv1alpha1.MonitoringConfig,
	inUse bool,
) error {
	mc.Status.InUse = inUse
	mc.Status.LastObservedGeneration = mc.GetGeneration()

	v, pmmErr := r.fetchPMMServerVersion(ctx, mc)
	if pmmErr == nil {
		mc.Status.PMMServerVersion = v
	}

	updErr := r.Client.Status().Update(ctx, mc)

	return errors.Join(pmmErr, updErr)
}

// fetchPMMServerVersion calls the PMM API to retrieve the server version
// using the API key stored in the credentials Secret.
func (r *MonitoringConfigReconciler) fetchPMMServerVersion(
	ctx context.Context,
	mc *monitoringv1alpha1.MonitoringConfig,
) (monitoringv1alpha1.PMMServerVersion, error) {
	secret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{
		Name:      mc.Spec.CredentialsSecretName,
		Namespace: mc.GetNamespace(),
	}, secret); err != nil {
		return "", fmt.Errorf("failed to get credentials secret %q: %w", mc.Spec.CredentialsSecretName, err)
	}

	apiKey, ok := secret.Data["apiKey"]
	if !ok {
		return "", fmt.Errorf("apiKey not found in secret %q", mc.Spec.CredentialsSecretName)
	}

	var skipVerifyTLS bool
	if mc.Spec.VerifyTLS != nil {
		skipVerifyTLS = !pointer.Get(mc.Spec.VerifyTLS)
	}

	v, err := pmm.GetPMMServerVersion(ctx, mc.Spec.PMM.URL, string(apiKey), skipVerifyTLS)
	if err != nil {
		return "", fmt.Errorf("failed to get PMM server version: %w", err)
	}

	return monitoringv1alpha1.PMMServerVersion(v), nil
}

// initIndexers registers the field indexers required by this controller.
func (r *MonitoringConfigReconciler) initIndexers(ctx context.Context, mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(
		ctx,
		&monitoringv1alpha1.MonitoringConfig{},
		".spec.credentialsSecretName",
		func(obj client.Object) []string {
			mc, ok := obj.(*monitoringv1alpha1.MonitoringConfig)
			if !ok {
				return nil
			}
			return []string{mc.Spec.CredentialsSecretName}
		},
	); err != nil {
		return fmt.Errorf("indexing monitoringconfig by credentialsSecretName: %w", err)
	}

	if err := mgr.GetFieldIndexer().IndexField(
		ctx,
		&corev1alpha1.Instance{},
		instanceMonitoringConfigField,
		func(obj client.Object) []string {
			instance, ok := obj.(*corev1alpha1.Instance)
			if !ok {
				return nil
			}

			monitoringSpec, ok := instance.Spec.Components["monitoring"]
			if !ok {
				return nil
			}

			if monitoringSpec.CustomSpec == nil || monitoringSpec.CustomSpec.Raw == nil {
				return nil
			}

			m := map[string]any{}
			if err := json.Unmarshal(monitoringSpec.CustomSpec.Raw, &m); err != nil {
				return nil
			}

			_, ok = m["monitoringConfigName"]
			if !ok {
				return nil
			}

			return []string{instanceMonitoringConfigField}
		},
	); err != nil {
		return fmt.Errorf("indexing instance by monitoring config name: %w", err)
	}

	return nil
}

// enqueueObjectsInNamespace returns an event handler that, when a Namespace event
// fires, enqueues reconcile requests for all objects of the given list type in
// that namespace. This ensures MonitoringConfigs are re-evaluated when a namespace changes.
func enqueueObjectsInNamespace(c client.Client, list client.ObjectList) handler.EventHandler { //nolint:ireturn
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
		if _, ok := obj.(*corev1.Namespace); !ok {
			return nil
		}

		if err := c.List(ctx, list, client.InNamespace(obj.GetName())); err != nil {
			return nil
		}

		var requests []reconcile.Request
		_ = meta.EachListItem(list, func(item runtime.Object) error {
			o, ok := item.(client.Object)
			if !ok {
				return nil
			}

			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: o.GetNamespace(),
					Name:      o.GetName(),
				},
			})

			return nil
		})

		return requests
	})
}
