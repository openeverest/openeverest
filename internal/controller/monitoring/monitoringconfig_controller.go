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
	"net/url"
	"slices"

	"github.com/AlekSi/pointer"
	vmv1beta1 "github.com/VictoriaMetrics/operator/api/operator/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	corev1alpha1 "github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	monitoringv1alpha2 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha2"
	"github.com/openeverest/openeverest/v2/pkg/pmm"
)

const (
	// inUseFinalizer prevents deletion of a MonitoringConfig that is referenced by Instances.
	inUseFinalizer = "monitoring.openeverest.io/in-use-protection"

	// vmagentFinalizer prevents deletion of a MonitoringConfig if VMAgent is using it.
	vmagentFinalizer = "openeverest.io/vmagent"

	// cleanupSecretsFinalizer prevents deletion of a MonitoringConfig if a copied Secret
	// in the monitoring namespace exists.
	cleanupSecretsFinalizer = "openeverest.io/cleanup-secrets"

	// monitoringConfigRefNameLabel is used to reference a MonitoringConfig's name
	// on copied Secrets since owner references cannot be used across namespaces.
	monitoringConfigRefNameLabel = "openeverest.io/monitoring-config-ref-name"

	// monitoringConfigRefNamespaceLabel is used to reference a MonitoringConfig's namespace
	// on copied Secrets since owner references cannot be used across namespaces.
	monitoringConfigRefNamespaceLabel = "openeverest.io/monitoring-config-ref-namespace"

	// instanceMonitoringConfigField is the field path used for indexing Instances
	// by their monitoring config name.
	instanceMonitoringConfigField = ".spec.components.monitoring.customSpec.monitoringConfigName"
)

// MonitoringConfigReconciler reconciles a MonitoringConfig object.
type MonitoringConfigReconciler struct {
	client.Client
	Scheme              *runtime.Scheme
	MonitoringNamespace string
}

// SetupWithManager sets up the controller with the Manager.
func (r *MonitoringConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := r.initIndexers(context.Background(), mgr); err != nil {
		return fmt.Errorf("init field indexers: %w", err)
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named("MonitoringConfig").
		For(&monitoringv1alpha2.MonitoringConfig{}).
		Watches(&corev1.Namespace{},
			enqueueObjectsInNamespace(r.Client, &monitoringv1alpha2.MonitoringConfigList{})).
		Watches(&corev1alpha1.Instance{},
			handler.EnqueueRequestsFromMapFunc(r.enqueueInstances),
			builder.WithPredicates(predicate.GenerationChangedPredicate{}, instancePredicate())).
		Watches(&vmv1beta1.VMAgent{},
			handler.EnqueueRequestsFromMapFunc(r.enqueueMonitoringConfigs),
			builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Complete(r)
}

// +kubebuilder:rbac:groups=monitoring.openeverest.io,resources=monitoringconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=monitoring.openeverest.io,resources=monitoringconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=monitoring.openeverest.io,resources=monitoringconfigs/finalizers,verbs=update
// +kubebuilder:rbac:groups=core.openeverest.io,resources=instances,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=operator.victoriametrics.com,resources=vmagents,verbs=get;list;watch;create;update;delete

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

	mc := &monitoringv1alpha2.MonitoringConfig{}
	if err := r.Get(ctx, req.NamespacedName, mc); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Determine whether any Instance references this MonitoringConfig.
	inUse, err := r.isInUse(ctx, mc)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to check in-use status: %w", err)
	}

	// Sync status after reconciliation completes.
	defer func() {
		// Do not reconcile on a resource that is being deleted.
		if !mc.GetDeletionTimestamp().IsZero() {
			return
		}

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

	// Ensure the credentials Secret is owned by this MonitoringConfig (non-deletion only).
	if mc.GetDeletionTimestamp().IsZero() {
		if err := r.ensureSecretOwnership(ctx, mc); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to ensure secret ownership: %w", err)
		}
	}

	// Reconcil VMAgent
	logger.Info("Reconciling VMAgent")
	defer func() {
		logger.Info("Reconciled VMAgent")
	}()

	if err := r.reconcileVMAgent(ctx); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// isInUse returns true if any Instance in the same namespace references this MonitoringConfig.
func (r *MonitoringConfigReconciler) isInUse(ctx context.Context, mc *monitoringv1alpha2.MonitoringConfig) (bool, error) {
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
	mc *monitoringv1alpha2.MonitoringConfig,
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
			return fmt.Errorf("failed to update finalizer: %w", err)
		}
	}

	return nil
}

// ensureSecretOwnership sets this MonitoringConfig as the controller owner of the
// referenced credentials Secret, if not already owned.
func (r *MonitoringConfigReconciler) ensureSecretOwnership(
	ctx context.Context,
	mc *monitoringv1alpha2.MonitoringConfig,
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
	mc *monitoringv1alpha2.MonitoringConfig,
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
	mc *monitoringv1alpha2.MonitoringConfig,
) (monitoringv1alpha2.PMMServerVersion, error) {
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

	v, err := pmm.GetPMMServerVersion(ctx, mc.Spec.URL, string(apiKey), skipVerifyTLS)
	if err != nil {
		return "", fmt.Errorf("failed to get PMM server version: %w", err)
	}

	return monitoringv1alpha2.PMMServerVersion(v), nil
}

// reconcileVMAgent ensures a VMAgent exists with remote-write entries for all PMM-type MonitoringConfigs, and is removed when no longer needed.
func (r *MonitoringConfigReconciler) reconcileVMAgent(ctx context.Context) error {
	list := &monitoringv1alpha2.MonitoringConfigList{}
	if err := r.List(ctx, list, &client.ListOptions{}); err != nil {
		return fmt.Errorf("could not list monitoringconfigs: %w", err)
	}

	// Ensure each MonitoringConfig has the vmagent finalizer and its mirrored secret
	// in the monitoring namespace before building the VMAgent spec.
	for _, mc := range list.Items {
		if err := r.ensureVMAgentResources(ctx, &mc); err != nil {
			return fmt.Errorf("could not ensure vmagent resources: %w", err)
		}
	}

	kubeSystemNamespace := &corev1.Namespace{}
	if err := r.Get(ctx, types.NamespacedName{Name: "kube-system"}, kubeSystemNamespace); err != nil {
		return fmt.Errorf("could not get kube-system namespace: %w", err)
	}

	spec, err := r.genVMAgentSpec(list, string(kubeSystemNamespace.UID))
	if err != nil {
		return fmt.Errorf("could not generate VMAgent spec: %w", err)
	}

	vmAgent := &vmv1beta1.VMAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "everest-monitoring",
			Namespace: r.MonitoringNamespace,
		},
	}

	// No remote writes, delete the VMAgent.
	if len(spec.RemoteWrite) == 0 {
		if err := r.Delete(ctx, vmAgent); client.IgnoreNotFound(err) != nil {
			return fmt.Errorf("could not delete vmagent: %w", err)
		}

		return nil
	}

	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, vmAgent, func() error {
		vmAgent.SetLabels(map[string]string{
			"app.kubernetes.io/managed-by": "openeverest",
			"openeverest.io/type":          "monitoring",
		})
		vmAgent.Spec = *spec
		return nil
	})

	return err
}

// ensureVMAgentResources ensures the vmagent finalizer, and copied secret
// is in the monitoring namespace.
// - on deletion: removes the copied secret and the vmagent finalizer.
// - otherwise: adds the vmagent finalizer and copies the credentials secret.
func (r *MonitoringConfigReconciler) ensureVMAgentResources(ctx context.Context, mc *monitoringv1alpha2.MonitoringConfig) error {
	if mc.Spec.Type != monitoringv1alpha2.PMMMonitoringType {
		return nil
	}

	if !mc.GetDeletionTimestamp().IsZero() {
		if err := r.cleanupSecrets(ctx, mc); err != nil {
			return fmt.Errorf("could not clean up secrets: %w", err)
		}

		if removed := controllerutil.RemoveFinalizer(mc, vmagentFinalizer); removed {
			if err := r.Update(ctx, mc); err != nil {
				return fmt.Errorf("could not remove vmagent finalizer: %w", err)
			}
		}

		return nil
	}

	// Add the vmagent finalizer so the VMAgent is updated
	// before the MonitoringConfig is removed.
	if updated := controllerutil.AddFinalizer(mc, vmagentFinalizer); updated {
		if err := r.Update(ctx, mc); err != nil {
			return fmt.Errorf("could not add vmagent finalizer: %w", err)
		}
	}

	if _, err := r.reconcileSecret(ctx, mc); err != nil {
		return fmt.Errorf("could not reconcile destination secret: %w", err)
	}

	return nil
}

// genVMAgentSpec builds a VMAgentSpec from the current state of all MonitoringConfigs.
func (r *MonitoringConfigReconciler) genVMAgentSpec(mcList *monitoringv1alpha2.MonitoringConfigList, k8sClusterID string) (*vmv1beta1.VMAgentSpec, error) {
	remoteWrites := make([]vmv1beta1.VMAgentRemoteWriteSpec, 0, len(mcList.Items))
	for _, mc := range mcList.Items {
		if mc.Spec.Type != monitoringv1alpha2.PMMMonitoringType {
			continue
		}

		// Skip configs that are being deleted — they have no remote-write entry.
		if !mc.GetDeletionTimestamp().IsZero() {
			continue
		}

		u, err := url.Parse(mc.Spec.URL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse PMM URL for %q: %w", mc.GetName(), err)
		}

		remoteWriteURL := u.JoinPath("victoriametrics/api/v1/write").String()

		// Skip duplicate PMM endpoints.
		if slices.IndexFunc(remoteWrites, func(rw vmv1beta1.VMAgentRemoteWriteSpec) bool {
			return rw.URL == remoteWriteURL
		}) >= 0 {
			continue
		}

		// The secret name mirrors reconcileSecret's naming convention.
		secretName := r.monitoringSecretName(&mc)

		skipTLS := false
		if mc.Spec.VerifyTLS != nil {
			skipTLS = !*mc.Spec.VerifyTLS
		}

		remoteWrites = append(remoteWrites, vmv1beta1.VMAgentRemoteWriteSpec{
			BasicAuth: &vmv1beta1.BasicAuth{
				Password: corev1.SecretKeySelector{
					Key:                  "apiKey",
					LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
				},
				Username: corev1.SecretKeySelector{
					Key:                  "username",
					LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
				},
			},
			TLSConfig: &vmv1beta1.TLSConfig{InsecureSkipVerify: skipTLS},
			URL:       remoteWriteURL,
		})
	}

	return &vmv1beta1.VMAgentSpec{
		SelectAllByDefault: true,
		CommonApplicationDeploymentParams: vmv1beta1.CommonApplicationDeploymentParams{
			ExtraArgs: map[string]string{
				"memory.allowedPercent": "40",
			},
		},
		CommonDefaultableParams: vmv1beta1.CommonDefaultableParams{
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("250m"),
					corev1.ResourceMemory: resource.MustParse("350Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("500m"),
					corev1.ResourceMemory: resource.MustParse("850Mi"),
				},
			},
		},
		ExternalLabels: map[string]string{
			"k8s_cluster_id": k8sClusterID,
		},
		RemoteWrite: remoteWrites,
	}, nil
}

// reconcileSecret copies the source MonitoringConfig secret onto the monitoring namespace.
// Returns the name of the newly created/updated secret.
func (r *MonitoringConfigReconciler) reconcileSecret(ctx context.Context, mc *monitoringv1alpha2.MonitoringConfig) (string, error) {
	secretName := r.monitoringSecretName(mc)

	// If the MonitoringConfig is already in the monitoring namespace, use it.
	if mc.GetNamespace() == r.MonitoringNamespace {
		return secretName, nil
	}

	// Get the secret in the MonitoringConfig namespace.
	src := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{
		Name:      mc.Spec.CredentialsSecretName,
		Namespace: mc.GetNamespace(),
	}, src); err != nil {
		return "", fmt.Errorf("failed to get credentials secret %q: %w", mc.Spec.CredentialsSecretName, err)
	}

	// Create a copy in the monitoring namespace.
	dst := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: r.MonitoringNamespace,
		},
	}

	if _, err := controllerutil.CreateOrUpdate(ctx, r.Client, dst, func() error {
		dst.Data = src.DeepCopy().Data
		// Labels are used for cleanup, since we cannot have cross-namespace OwnerRefs.
		labels := dst.GetLabels()
		if labels == nil {
			labels = make(map[string]string)
		}

		labels[monitoringConfigRefNameLabel] = mc.GetName()
		labels[monitoringConfigRefNamespaceLabel] = mc.GetNamespace()
		dst.SetLabels(labels)
		return nil
	}); err != nil {
		return "", err
	}

	// Add a clean-up finalizer in the parent MonitoringConfig.
	if controllerutil.AddFinalizer(mc, cleanupSecretsFinalizer) {
		return secretName, r.Update(ctx, mc)
	}

	return secretName, nil
}

// monitoringSecretName returns the original or copied secret name depending
// on whether the MonitoringConfig is in the monitoring namespace or not.
func (r *MonitoringConfigReconciler) monitoringSecretName(mc *monitoringv1alpha2.MonitoringConfig) string {
	if mc.GetNamespace() == r.MonitoringNamespace {
		return mc.Spec.CredentialsSecretName
	}

	return mc.Spec.CredentialsSecretName + "-" + mc.GetNamespace()
}

// cleanupSecrets deletes all secrets in the monitoring namespace that belong to the given MonitoringConfig.
func (r *MonitoringConfigReconciler) cleanupSecrets(ctx context.Context, mc *monitoringv1alpha2.MonitoringConfig) error {
	// List secrets in the monitoring namespace that belong to this MonitoringConfig.
	secrets := &corev1.SecretList{}
	err := r.List(ctx, secrets, &client.ListOptions{
		Namespace: r.MonitoringNamespace,
		LabelSelector: labels.SelectorFromSet(map[string]string{
			monitoringConfigRefNameLabel:      mc.GetName(),
			monitoringConfigRefNamespaceLabel: mc.GetNamespace(),
		}),
	})
	if err != nil {
		return err
	}

	for _, secret := range secrets.Items {
		if err := r.Delete(ctx, &secret); err != nil {
			return err
		}
	}

	// Remove the finalizer from the MonitoringConfig.
	if controllerutil.RemoveFinalizer(mc, cleanupSecretsFinalizer) {
		return r.Update(ctx, mc)
	}

	return nil
}

// initIndexers registers the field indexers required by this controller.
func (r *MonitoringConfigReconciler) initIndexers(ctx context.Context, mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(
		ctx,
		&monitoringv1alpha2.MonitoringConfig{},
		".spec.credentialsSecretName",
		func(obj client.Object) []string {
			mc, ok := obj.(*monitoringv1alpha2.MonitoringConfig)
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
			monitoringConfigName := instanceMonitoringConfigName(obj)
			if monitoringConfigName == "" {
				return nil
			}

			return []string{monitoringConfigName}
		},
	); err != nil {
		return fmt.Errorf("indexing instance by monitoring config name: %w", err)
	}

	return nil
}

// instanceMonitoringConfigName extracts the monitoringConfigName value from an Instance's
// .spec.components.monitoring.customSpec.monitoringConfigName.
// Returns "" if not set.
func instanceMonitoringConfigName(obj client.Object) string {
	instance, ok := obj.(*corev1alpha1.Instance)
	if !ok {
		return ""
	}

	monitoringSpec, ok := instance.Spec.Components["monitoring"]
	if !ok {
		return ""
	}

	if monitoringSpec.CustomSpec == nil || monitoringSpec.CustomSpec.Raw == nil {
		return ""
	}

	m := map[string]any{}
	if err := json.Unmarshal(monitoringSpec.CustomSpec.Raw, &m); err != nil {
		return ""
	}

	name, _ := m["monitoringConfigName"].(string)
	return name
}

// enqueueInstances maps an Instance to a reconcile.Request for the MonitoringConfig
// referenced in .spec.components.monitoring.customSpec.monitoringConfigName.
func (r *MonitoringConfigReconciler) enqueueInstances(_ context.Context, obj client.Object) []reconcile.Request {
	name := instanceMonitoringConfigName(obj)
	if name == "" {
		return nil
	}

	return []reconcile.Request{
		{NamespacedName: types.NamespacedName{Name: name, Namespace: obj.GetNamespace()}},
	}
}

// instancePredicate returns a Predicate that passes only when the Instance's
// .spec.components.monitoring.customSpec.monitoringConfigName field is relevant:
//   - Create: the field is set on the new Instance.
//   - Update: the field is set on either the old or the new Instance, covering
//     the cases where the value is added, changed, or removed.
//   - Delete: the field is set on the deleted Instance.
func instancePredicate() predicate.Predicate { //nolint:ireturn
	return predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return instanceMonitoringConfigName(e.Object) != ""
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return instanceMonitoringConfigName(e.ObjectOld) != instanceMonitoringConfigName(e.ObjectNew)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return instanceMonitoringConfigName(e.Object) != ""
		},
		GenericFunc: func(event.GenericEvent) bool {
			return false
		},
	}
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

// enqueueMonitoringConfigs enqueues MonitoringConfig objects for reconciliation when a VMAgent is created/updated/deleted.
func (r *MonitoringConfigReconciler) enqueueMonitoringConfigs(ctx context.Context, o client.Object) []reconcile.Request {
	vmAgent, ok := o.(*vmv1beta1.VMAgent)
	if !ok {
		return nil
	}

	if vmAgent.GetNamespace() != r.MonitoringNamespace {
		return nil
	}

	list := &monitoringv1alpha2.MonitoringConfigList{}
	err := r.List(ctx, list)
	if err != nil {
		return nil
	}

	requests := make([]reconcile.Request, 0, len(list.Items))
	for _, mc := range list.Items {
		requests = append(requests, reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      mc.GetName(),
				Namespace: mc.GetNamespace(),
			},
		})
	}
	return requests
}
