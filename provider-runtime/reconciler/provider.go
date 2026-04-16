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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
	"github.com/openeverest/openeverest/v2/provider-runtime/server"
)

const finalizerName = "everest.percona.com/provider-finalizer"

// =============================================================================
// PROVIDER RECONCILER - Works with both Interface and Builder providers
// =============================================================================

// ProviderReconciler reconciles Instance resources using a Provider.
type ProviderReconciler struct {
	provider     providerAdapter
	manager      ctrl.Manager
	serverConfig *server.ServerConfig
	server       *server.Server
	client.Client
}

// providerAdapter is the internal interface that both provider types satisfy.
type providerAdapter interface {
	Name() string
	Types() func(*runtime.Scheme) error
	Validate(c *controller.Context) error
	Sync(c *controller.Context) error
	Status(c *controller.Context) (controller.Status, error)
	Cleanup(c *controller.Context) error
}

// ServerConfig is re-exported from server package for convenience.
// See server.ServerConfig for documentation.
type ServerConfig = server.ServerConfig

// New creates a reconciler from a provider.
func New(ctx context.Context, p controller.ProviderInterface, opts ...ReconcilerOption) (*ProviderReconciler, error) {
	return newReconciler(ctx, p, opts...)
}

// ReconcilerOption configures the reconciler.
type ReconcilerOption func(*reconcilerOptions)

type reconcilerOptions struct {
	serverConfig       *server.ServerConfig
	metricsBindAddress string
}

// WithServer enables the integrated HTTP server for the validation webhook.
//
// The server provides:
// - Validation webhook: Accepts validation requests and runs the provider's Validate() method
// - Health/Ready endpoints: For Kubernetes probes
//
// Example:
//
//	r, err := reconciler.NewFromInterface(provider,
//	    reconciler.WithServer(reconciler.ServerConfig{
//	        Port:           8080,
//	        ValidationPath: "/validate",
//	    }),
//	)
//
// Validation is handled by the provider's Validate() method - the same validation
// used during reconciliation is exposed via the webhook.
func WithServer(config server.ServerConfig) ReconcilerOption {
	return func(o *reconcilerOptions) {
		o.serverConfig = &config
	}
}

// WithMetrics configures the metrics server bind address.
//
// The metrics server exposes Prometheus metrics for the controller.
// By default, it binds to ":8080". You can customize the address or disable
// it entirely by passing "0".
//
// Example:
//
//	// Custom port
//	r, err := reconciler.New(provider,
//	    reconciler.WithMetrics(":9090"),
//	)
//
//	// Disable metrics
//	r, err := reconciler.New(provider,
//	    reconciler.WithMetrics("0"),
//	)
func WithMetrics(bindAddress string) ReconcilerOption {
	return func(o *reconcilerOptions) {
		o.metricsBindAddress = bindAddress
	}
}

// newReconciler creates a reconciler from any provider that satisfies providerAdapter.
func newReconciler(ctx context.Context, p providerAdapter, opts ...ReconcilerOption) (*ProviderReconciler, error) {
	// Apply options
	options := &reconcilerOptions{}
	for _, opt := range opts {
		opt(options)
	}
	scheme := runtime.NewScheme()

	// Register core Kubernetes types
	if err := corev1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add corev1 scheme: %w", err)
	}

	// Register core types
	if err := v1alpha1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add v1alpha1 scheme: %w", err)
	}

	// Register provider-specific types
	if typesFunc := p.Types(); typesFunc != nil {
		if err := typesFunc(scheme); err != nil {
			return nil, fmt.Errorf("failed to add provider scheme: %w", err)
		}
	}

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&zap.Options{Development: true})))

	// Configure manager options
	mgrOpts := ctrl.Options{Scheme: scheme}
	if options.metricsBindAddress != "" {
		mgrOpts.Metrics = metricsserver.Options{
			BindAddress: options.metricsBindAddress,
		}
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), mgrOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create manager: %w", err)
	}

	// Setup field indexes if provider implements FieldIndexProvider
	if fip, ok := p.(controller.FieldIndexProvider); ok {
		for _, fi := range fip.FieldIndexes() {
			if err := mgr.GetFieldIndexer().IndexField(
				ctx,
				fi.Object,
				fi.FieldPath,
				fi.Extractor,
			); err != nil {
				return nil, fmt.Errorf("failed to create field index %s on %T: %w", fi.FieldPath, fi.Object, err)
			}
		}
	}

	r := &ProviderReconciler{
		provider:     p,
		manager:      mgr,
		serverConfig: options.serverConfig,
		Client:       mgr.GetClient(),
	}

	// Setup server if configured
	if options.serverConfig != nil {
		if err := r.setupServer(p); err != nil {
			return nil, fmt.Errorf("failed to setup server: %w", err)
		}
	}

	if err := r.setup(); err != nil {
		return nil, fmt.Errorf("failed to setup reconciler: %w", err)
	}

	return r, nil
}

// GetManager returns the controller manager.
func (r *ProviderReconciler) GetManager() ctrl.Manager {
	return r.manager
}

// setupServer initializes the HTTP server with a validation webhook.
func (r *ProviderReconciler) setupServer(p providerAdapter) error {
	// Create validator function that wraps the provider's Validate method
	validator := func(ctx context.Context, c client.Client, in *v1alpha1.Instance) error {
		if err := validateVersionBundle(ctx, c, in); err != nil {
			return err
		}
		inCtx := controller.NewContext(ctx, c, in, p.Name())
		return p.Validate(inCtx)
	}

	r.server = server.NewServer(*r.serverConfig, validator)
	return nil
}

// Start starts the reconciler and server (blocking).
func (r *ProviderReconciler) Start(ctx context.Context) error {
	// Start server if configured
	if r.server != nil {
		r.server.SetClient(r.Client)
		go func() {
			if err := r.server.Start(ctx); err != nil {
				log.FromContext(ctx).Error(err, "Server error")
			}
		}()
		// Mark server as ready once manager is ready
		r.server.SetReady(true)
	}

	return r.manager.Start(ctx)
}

// StartWithSignalHandler starts the reconciler and server with OS signal handling.
func (r *ProviderReconciler) StartWithSignalHandler() error {
	ctx := ctrl.SetupSignalHandler()

	// Start server if configured
	if r.server != nil {
		r.server.SetClient(r.Client)
		go func() {
			if err := r.server.Start(ctx); err != nil {
				log.FromContext(ctx).Error(err, "Server error")
			}
		}()
		// Mark server as ready once manager is ready
		r.server.SetReady(true)
	}

	return r.manager.Start(ctx)
}

func (r *ProviderReconciler) setup() error {
	// Filter to only handle Instance for this provider
	filter := predicate.NewPredicateFuncs(func(object client.Object) bool {
		in, ok := object.(*v1alpha1.Instance)
		if !ok {
			return false
		}
		return in.Spec.Provider == r.provider.Name()
	})

	b := ctrl.NewControllerManagedBy(r.manager).
		For(&v1alpha1.Instance{}, builder.WithPredicates(filter)).
		Named(r.provider.Name() + "-controller")

	// Configure watches if provider implements WatchProvider
	if wp, ok := r.provider.(controller.WatchProvider); ok {
		for _, wc := range wp.Watches() {
			if wc.Owned {
				// Owned resource: use Owns() for automatic owner-reference handling
				if len(wc.Predicates) > 0 {
					opts := []builder.OwnsOption{builder.WithPredicates(wc.Predicates...)}
					b.Owns(wc.Object, opts...)
				} else {
					b.Owns(wc.Object)
				}
			} else {
				// External resource: use Watches() with custom handler
				h := wc.Handler
				if h == nil {
					// Default to EnqueueRequestForObject if no handler specified
					// (though this is rarely useful for external resources)
					h = &handler.EnqueueRequestForObject{}
				}
				opts := wc.WatchOptions
				if len(wc.Predicates) > 0 {
					opts = append(opts, builder.WithPredicates(wc.Predicates...))
				}
				b.Watches(wc.Object, h, opts...)
			}
		}
	}

	return b.Complete(r)
}

// Reconcile implements the reconciliation loop.
func (r *ProviderReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger := log.FromContext(ctx).WithValues("provider", r.provider.Name())

	// Fetch the Instance
	in := &v1alpha1.Instance{}
	if err := r.Client.Get(ctx, req.NamespacedName, in); err != nil {
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	// Create the Context handle
	inCtx := controller.NewContext(ctx, r.Client, in, r.provider.Name())

	// Handle deletion
	if !in.GetDeletionTimestamp().IsZero() {
		return r.handleDeletion(ctx, inCtx, in, logger)
	}

	// Ensure finalizer is present
	if !controllerutil.ContainsFinalizer(in, finalizerName) {
		controllerutil.AddFinalizer(in, finalizerName)
		if err := r.Client.Update(ctx, in); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{Requeue: true}, nil
	}

	// Run validation
	if err := validateVersionBundle(ctx, r.Client, in); err != nil {
		logger.Error(err, "Version bundle validation failed")
		in.Status.Phase = v1alpha1.InstancePhaseFailed
		if updateErr := r.Client.Status().Update(ctx, in); updateErr != nil {
			logger.Error(updateErr, "Failed to update status after validation error")
		}
		return reconcile.Result{}, err
	}
	if err := r.provider.Validate(inCtx); err != nil {
		logger.Error(err, "Validation failed")
		// Update status to failed
		in.Status.Phase = v1alpha1.InstancePhaseFailed
		if updateErr := r.Client.Status().Update(ctx, in); updateErr != nil {
			logger.Error(updateErr, "Failed to update status after validation error")
		}
		return reconcile.Result{}, err
	}

	// Resolve version bundle into a deep-copied instance so the stored spec is
	// never mutated. The resolved copy is used for Sync() and Status() only.
	// effectiveBundleName is the bundle that was applied (may differ from
	// spec.version when the default was resolved on first reconcile).
	effectiveBundleName, resolvedIn, err := r.resolveVersionBundle(ctx, in)
	if err != nil {
		logger.Error(err, "Version bundle resolution failed")
		return reconcile.Result{}, err
	}
	syncCtx := controller.NewContext(ctx, r.Client, resolvedIn, r.provider.Name())

	// Run sync
	logger.Info("Running sync")
	if err := r.provider.Sync(syncCtx); err != nil {
		if controller.IsWaitError(err) {
			logger.Info("Sync waiting", "reason", err.Error())
			return reconcile.Result{RequeueAfter: controller.GetWaitDuration(err)}, nil
		}
		logger.Error(err, "Sync failed")
		return reconcile.Result{}, err
	}

	// Compute and update status
	logger.Info("Computing status")
	status, err := r.provider.Status(syncCtx)
	if err != nil {
		logger.Error(err, "Status computation failed")
		return reconcile.Result{}, err
	}

	in.Status = status.ToV2Alpha1()

	// Freeze the effective bundle name in status so it remains stable across
	// Provider upgrades. On subsequent reconciliations the reconciler reads this
	// value back (when spec.version is empty) instead of re-resolving the
	// Provider's current default — preventing silent upgrades on existing
	// Instances. GitOps tools exclude status from diff calculations by default
	// so this field never causes spurious out-of-sync alerts.
	if effectiveBundleName != "" {
		in.Status.Version = effectiveBundleName
	}

	// Write connection details Secret and set the ConnectionDetailsReady condition.
	if err := r.reconcileConnectionSecret(ctx, in, status); err != nil {
		logger.Error(err, "Failed to reconcile connection secret")
		return reconcile.Result{}, err
	}

	if err := r.Client.Status().Update(ctx, in); err != nil {
		logger.Error(err, "Failed to update status")
		return reconcile.Result{}, err
	}

	logger.Info("Reconciliation complete", "phase", in.Status.Phase)
	return reconcile.Result{}, nil
}

func (r *ProviderReconciler) handleDeletion(
	ctx context.Context,
	inCtx *controller.Context,
	in *v1alpha1.Instance,
	logger interface{ Info(string, ...interface{}) },
) (reconcile.Result, error) {
	if !controllerutil.ContainsFinalizer(in, finalizerName) {
		return reconcile.Result{}, nil
	}

	logger.Info("Running cleanup")

	// Update status to deleting
	if in.Status.Phase != v1alpha1.InstancePhaseTerminating {
		in.Status.Phase = v1alpha1.InstancePhaseTerminating
		if err := r.Client.Status().Update(ctx, in); err != nil {
			return reconcile.Result{}, err
		}
	}

	// Run cleanup
	if err := r.provider.Cleanup(inCtx); err != nil {
		if controller.IsWaitError(err) {
			logger.Info("Cleanup waiting", "reason", err.Error())
			return reconcile.Result{RequeueAfter: controller.GetWaitDuration(err)}, nil
		}
		return reconcile.Result{}, err
	}

	// Remove finalizer
	controllerutil.RemoveFinalizer(in, finalizerName)
	if err := r.Client.Update(ctx, in); err != nil {
		return reconcile.Result{}, err
	}

	logger.Info("Cleanup complete")
	return reconcile.Result{}, nil
}

// reconcileConnectionSecret creates or updates the connection details Secret
// and sets the ConnectionDetailsReady condition on the Instance status.
func (r *ProviderReconciler) reconcileConnectionSecret(
	ctx context.Context,
	in *v1alpha1.Instance,
	status controller.Status,
) error {
	now := metav1.Now()

	if status.ConnectionDetails.IsEmpty() {
		if status.Phase == v1alpha1.InstancePhaseReady {
			setCondition(in, v1alpha1.ConditionConnectionDetailsReady, metav1.ConditionFalse,
				"NotReported", "Provider did not report connection details", now)
		}
		return nil
	}

	secretName := in.Name + controller.ConnectionSecretSuffix

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: in.Namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "everest",
				"app.kubernetes.io/instance":   in.Name,
			},
		},
	}

	if err := controllerutil.SetControllerReference(in, secret, r.manager.GetScheme()); err != nil {
		return fmt.Errorf("failed to set owner reference on connection secret: %w", err)
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, secret, func() error {
		secret.Data = status.ConnectionDetails.ToSecretData()
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to create or update connection secret: %w", err)
	}

	in.Status.ConnectionSecretRef.Name = secretName
	setCondition(in, v1alpha1.ConditionConnectionDetailsReady, metav1.ConditionTrue,
		"Available", "Connection details are available in Secret "+secretName, now)

	return nil
}

// setCondition sets or updates a condition on the Instance status.
func setCondition(in *v1alpha1.Instance, condType string, status metav1.ConditionStatus, reason, message string, now metav1.Time) {
	for i, c := range in.Status.Conditions {
		if c.Type == condType {
			if c.Status != status {
				in.Status.Conditions[i].LastTransitionTime = now
			}
			in.Status.Conditions[i].Status = status
			in.Status.Conditions[i].Reason = reason
			in.Status.Conditions[i].Message = message
			in.Status.Conditions[i].ObservedGeneration = in.Generation
			return
		}
	}
	in.Status.Conditions = append(in.Status.Conditions, metav1.Condition{
		Type:               condType,
		Status:             status,
		LastTransitionTime: now,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: in.Generation,
	})
}

// resolveVersionBundle determines the effective version bundle, applies it to
// a deep copy of in, and returns both the effective bundle name and the
// resolved Instance. The original Instance stored in etcd is never mutated.
//
// Resolution order:
//  1. spec.version — explicitly set by the user (always honoured).
//  2. status.version — the bundle name frozen on the first reconciliation;
//     prevents a Provider upgrade from silently upgrading existing Instances.
//  3. Provider's default bundle — resolved once on the very first reconcile
//     of a new Instance; the name is then written to status.version so
//     subsequent reconciles use step 2 instead.
//  4. No bundle — returns ("" , in, nil); Sync() falls back to per-type
//     defaults from the componentTypes catalog.
//
// For each component the bundle version is applied only when the component's
// Version field is not already explicitly set by the user.
func (r *ProviderReconciler) resolveVersionBundle(ctx context.Context, in *v1alpha1.Instance) (effectiveBundleName string, resolved *v1alpha1.Instance, err error) {
	providerObj := &v1alpha1.Provider{}
	if err = r.Client.Get(ctx, client.ObjectKey{Name: in.Spec.Provider}, providerObj); err != nil {
		return "", nil, fmt.Errorf("fetching provider for version resolution: %w", err)
	}
	spec := &providerObj.Spec

	switch {
	case in.Spec.Version != "":
		// User explicitly chose a bundle.
		effectiveBundleName = in.Spec.Version
	case in.Status.Version != "":
		// Default was frozen on a previous reconcile; honour it regardless of
		// what the Provider's current default is.
		effectiveBundleName = in.Status.Version
	default:
		// First reconcile of a new Instance with no explicit version: resolve
		// the Provider's current default and freeze it in status.
		effectiveBundleName = controller.GetDefaultVersionBundleName(spec)
	}

	if effectiveBundleName == "" {
		return "", in, nil
	}

	bundle, err := controller.ResolveVersionBundle(spec, effectiveBundleName)
	if err != nil {
		return "", nil, err
	}

	resolved = in.DeepCopy()
	for compName, bundleVersion := range bundle.Components {
		compSpec, exists := resolved.Spec.Components[compName]
		if !exists {
			continue
		}
		if compSpec.Version == "" {
			compSpec.Version = bundleVersion
			resolved.Spec.Components[compName] = compSpec
		}
	}
	return effectiveBundleName, resolved, nil
}

// validateVersionBundle checks that spec.version (if set) exists in the
// Provider's Versions map. This runs for both the reconciler and the webhook
// so users get immediate feedback on an invalid version selection.
func validateVersionBundle(ctx context.Context, c client.Client, in *v1alpha1.Instance) error {
	if in.Spec.Version == "" {
		return nil
	}
	providerObj := &v1alpha1.Provider{}
	if err := c.Get(ctx, client.ObjectKey{Name: in.Spec.Provider}, providerObj); err != nil {
		return fmt.Errorf("fetching provider for version validation: %w", err)
	}
	found := false
	for _, b := range providerObj.Spec.Versions {
		if b.Name == in.Spec.Version {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("version %q is not defined by provider %q", in.Spec.Version, in.Spec.Provider)
	}
	return nil
}
