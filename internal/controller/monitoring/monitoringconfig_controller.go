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
	"encoding/json"
	"errors"
	"fmt"

	"github.com/AlekSi/pointer"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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
	// inUseFinalizer marks MonitoringConfig as "in-use" and prevents deletion of the resource.
	inUseFinalizer = "monitoring.openeverest.io/in-use-protection"
)

// MonitoringConfigReconciler reconciles a MonitoringConfig object.
type MonitoringConfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (r *MonitoringConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := r.initIndexers(context.Background(), mgr); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&monitoringv1alpha1.MonitoringConfig{}).
		Named("MonitoringConfig").
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
func (r *MonitoringConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (rr ctrl.Result, rerr error) { //nolint:nonamedreturns
	l := log.FromContext(ctx).
		WithName("MonitoringConfigReconciler").
		WithValues(
			"name", req.Name,
			"namespace", req.Namespace,
		)

	l.Info("Reconciling")

	defer func() {
		l.Info("Reconciled")
	}()

	mc := &monitoringv1alpha1.MonitoringConfig{}
	if rerr = r.Get(ctx, req.NamespacedName, mc); rerr != nil {
		return ctrl.Result{}, client.IgnoreNotFound(rerr)
	}

	mcName := mc.GetName()

	instances := new(corev1alpha1.InstanceList)

	// The selector assumes it knows the field path of the monitoring config name,
	// not great as it's specific for each provider.
	mcNameSelector := fields.OneTermEqualSelector(".spec.components.monitoring.customSpec.monitoringConfigName", mcName)
	if rerr = r.Client.List(ctx, instances, &client.ListOptions{
		FieldSelector: mcNameSelector,
		Namespace:     req.Namespace,
	}); rerr != nil {
		return ctrl.Result{}, fmt.Errorf("failed to fetch instances using monitoring config: %w", rerr)
	}

	secret := &corev1.Secret{}

	// update the status and finalizers of the MonitoringConfig object after the reconciliation
	defer func() {
		// nothing to process on delete events
		if !mc.GetDeletionTimestamp().IsZero() {
			return
		}

		mc.Status.InUse = len(instances.Items) > 0
		mc.Status.LastObservedGeneration = mc.GetGeneration()

		v, err := getPMMServerVersion(ctx, secret, mc)
		if err != nil {
			l.Error(err, "failed to get PMM server version")
			rerr = errors.Join(rerr, err)
		}

		mc.Status.PMMServerVersion = v

		if err := r.Client.Status().Update(ctx, mc); err != nil {
			l.Error(err, "failed to update status", "monitoringConfig", mcName)

			rr = ctrl.Result{}
			rerr = errors.Join(rerr, err)
		}
	}()

	if rerr = ensureInUseFinalizer(ctx, r.Client, len(instances.Items) > 0, mc); rerr != nil {
		l.Error(rerr, "failed to update finalizers", "monitoringConfig", mcName)

		return ctrl.Result{}, rerr
	}

	if rerr = r.Get(ctx, types.NamespacedName{
		Name:      mc.Spec.CredentialsSecretName,
		Namespace: mc.GetNamespace(),
	}, secret); rerr != nil {
		l.Error(rerr, "unable to fetch Secret")

		return ctrl.Result{}, rerr
	}

	if metav1.GetControllerOf(secret) != nil {
		return ctrl.Result{}, nil
	}

	l.Info("setting controller references for the secret")
	if rerr = controllerutil.SetControllerReference(mc, secret, r.Client.Scheme()); rerr != nil {
		return ctrl.Result{}, rerr
	}

	if rerr = r.Update(ctx, secret); rerr != nil {
		return ctrl.Result{}, rerr
	}

	return ctrl.Result{}, nil
}

// initIndexers initializes the field indexers for the controller.
func (r *MonitoringConfigReconciler) initIndexers(ctx context.Context, mgr ctrl.Manager) error {
	// Index the credentialsSecretName field in MonitoringConfig.
	err := mgr.GetFieldIndexer().IndexField(ctx, &monitoringv1alpha1.MonitoringConfig{}, ".spec.credentialsSecretName",
		func(c client.Object) []string {
			mc, ok := c.(*monitoringv1alpha1.MonitoringConfig)
			if !ok {
				return []string{}
			}

			return []string{mc.Spec.CredentialsSecretName}
		},
	)

	return err
}

// ensureInUseFinalizer adds or removes the InUseResourceFinalizer
// on the given object based on the used parameter.
func ensureInUseFinalizer(ctx context.Context, c client.Client, used bool, obj client.Object) error {
	var updated bool
	if used {
		updated = controllerutil.AddFinalizer(obj, inUseFinalizer)
	} else {
		updated = controllerutil.RemoveFinalizer(obj, inUseFinalizer)
	}

	if updated {
		return c.Update(ctx, obj)
	}
	return nil
}

// enqueueObjectsInNamespace returns an event handler used for Namespace watchers.
// It enqueues all objects specified by the type of list in the triggered namespace.
func enqueueObjectsInNamespace(c client.Client, list client.ObjectList) handler.EventHandler { //nolint:ireturn
	return handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, o client.Object) []reconcile.Request {
		if _, ok := o.(*corev1.Namespace); !ok {
			panic("enqueueObjectsInNamespace should be called on a Namespace")
		}

		if err := c.List(ctx, list, client.InNamespace(o.GetName())); err != nil {
			return nil
		}

		items, err := meta.ExtractList(list)
		if err != nil {
			return nil
		}

		requests := make([]reconcile.Request, 0, len(items))
		for _, item := range items {
			blob, err := json.Marshal(item)
			if err != nil {
				panic(err.Error())
			}

			uObj := &unstructured.Unstructured{}
			if err := json.Unmarshal(blob, uObj); err != nil {
				panic(err.Error())
			}

			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: uObj.GetNamespace(),
					Name:      uObj.GetName(),
				},
			})
		}

		return requests
	})
}

// getPMMServerVersion returns PMM server version by calling PMM API
// using the API token stored in the secret.
func getPMMServerVersion(
	ctx context.Context,
	secret *corev1.Secret,
	mc *monitoringv1alpha1.MonitoringConfig,
) (monitoringv1alpha1.PMMServerVersion, error) {
	if secret == nil || len(secret.Data) == 0 {
		return "", fmt.Errorf("empty secrets")
	}

	val, ok := secret.Data["apiKey"]
	if !ok {
		return "", fmt.Errorf("PMM token not found in the secret")
	}

	apiToken := string(val)

	var skipVerifyTLS bool
	if mc.Spec.VerifyTLS != nil {
		skipVerifyTLS = !pointer.Get(mc.Spec.VerifyTLS)
	}

	v, err := pmm.GetPMMServerVersion(ctx, mc.Spec.PMM.URL, apiToken, skipVerifyTLS)
	if err != nil {
		return "", fmt.Errorf("failed to get PMM server version: %w", err)
	}

	return monitoringv1alpha1.PMMServerVersion(v), nil
}
