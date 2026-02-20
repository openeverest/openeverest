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

// SchemaProvider is re-exported from controller package for convenience.
// See controller.SchemaProvider for documentation.
type SchemaProvider = controller.SchemaProvider

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

// WithServer enables the integrated HTTP server for schema exposure and validation webhook.
//
// The server provides:
// - Schema endpoint: Returns OpenAPI schemas for components, topologies, and global config
// - Validation webhook: Accepts validation requests and runs the provider's Validate() method
// - Health/Ready endpoints: For Kubernetes probes
//
// Example:
//
//	r, err := reconciler.NewFromInterface(provider,
//	    reconciler.WithServer(reconciler.ServerConfig{
//	        Port:           8080,
//	        SchemaPath:     "/schema",
//	        ValidationPath: "/validate",
//	    }),
//	)
//
// The provider must implement SchemaProvider interface to register component schemas.
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

// setupServer initializes the HTTP server with schemas from the provider.
func (r *ProviderReconciler) setupServer(p providerAdapter) error {
	registry := server.NewSchemaRegistry()

	// Check if provider implements SchemaProvider
	if sp, ok := p.(SchemaProvider); ok {
		// Register component schemas
		for name, schemaType := range sp.ComponentSchemas() {
			if err := registry.RegisterComponent(name, schemaType); err != nil {
				return err
			}
		}

		// Register topologies (schema + components)
		for name, def := range sp.Topologies() {
			if err := registry.RegisterTopology(name, def.Schema); err != nil {
				return err
			}
			// Extract component names from the definition
			components := make([]string, 0, len(def.Components))
			for compName := range def.Components {
				components = append(components, compName)
			}
			registry.RegisterTopologyComponents(name, components)
		}

		// Register global schema
		if globalSchema := sp.GlobalSchema(); globalSchema != nil {
			if err := registry.RegisterGlobal(globalSchema); err != nil {
				return err
			}
		}
	}

	// Create validator function that wraps the provider's Validate method
	validator := func(ctx context.Context, c client.Client, in *v1alpha1.Instance) error {
		// Create context handle with metadata if available
		var inCtx *controller.Context
		if mp, ok := p.(controller.MetadataProvider); ok {
			metadata := mp.GetMetadata()
			inCtx = controller.NewContextWithMetadata(ctx, c, in, metadata)
		} else {
			inCtx = controller.NewContext(ctx, c, in)
		}
		return p.Validate(inCtx)
	}

	r.server = server.NewServer(*r.serverConfig, registry, validator)
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

	// Create the Context handle with metadata if available
	var inCtx *controller.Context
	if mp, ok := r.provider.(controller.MetadataProvider); ok {
		metadata := mp.GetMetadata()
		inCtx = controller.NewContextWithMetadata(ctx, r.Client, in, metadata)
	} else {
		inCtx = controller.NewContext(ctx, r.Client, in)
	}

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
	if err := r.provider.Validate(inCtx); err != nil {
		logger.Error(err, "Validation failed")
		// Update status to failed
		in.Status.Phase = v1alpha1.InstancePhaseFailed
		if updateErr := r.Client.Status().Update(ctx, in); updateErr != nil {
			logger.Error(updateErr, "Failed to update status after validation error")
		}
		return reconcile.Result{}, err
	}

	// Run sync
	logger.Info("Running sync")
	if err := r.provider.Sync(inCtx); err != nil {
		if controller.IsWaitError(err) {
			logger.Info("Sync waiting", "reason", err.Error())
			return reconcile.Result{RequeueAfter: controller.GetWaitDuration(err)}, nil
		}
		logger.Error(err, "Sync failed")
		return reconcile.Result{}, err
	}

	// Compute and update status
	logger.Info("Computing status")
	status, err := r.provider.Status(inCtx)
	if err != nil {
		logger.Error(err, "Status computation failed")
		return reconcile.Result{}, err
	}

	in.Status = status.ToV2Alpha1()
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
	if in.Status.Phase != v1alpha1.InstancePhaseDeleting {
		in.Status.Phase = v1alpha1.InstancePhaseDeleting
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
