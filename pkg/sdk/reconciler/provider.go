package reconciler

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/openeverest/openeverest/v2/pkg/apis/v2alpha1"
	"github.com/openeverest/openeverest/v2/pkg/sdk/controller"
	"github.com/openeverest/openeverest/v2/pkg/sdk/server"
)

const finalizerName = "everest.percona.com/provider-finalizer"

// =============================================================================
// PROVIDER RECONCILER - Works with both Interface and Builder providers
// =============================================================================

// ProviderReconciler reconciles Workload resources using a Provider.
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
	OwnedTypes() []client.Object
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
func New(p controller.ProviderInterface, opts ...ReconcilerOption) (*ProviderReconciler, error) {
	return newReconciler(p, opts...)
}

// ReconcilerOption configures the reconciler.
type ReconcilerOption func(*reconcilerOptions)

type reconcilerOptions struct {
	serverConfig *server.ServerConfig
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

// newReconciler creates a reconciler from any provider that satisfies providerAdapter.
func newReconciler(p providerAdapter, opts ...ReconcilerOption) (*ProviderReconciler, error) {
	// Apply options
	options := &reconcilerOptions{}
	for _, opt := range opts {
		opt(options)
	}
	scheme := runtime.NewScheme()

	// Register core types
	if err := v2alpha1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add v2alpha1 scheme: %w", err)
	}

	// Register provider-specific types
	if typesFunc := p.Types(); typesFunc != nil {
		if err := typesFunc(scheme); err != nil {
			return nil, fmt.Errorf("failed to add provider scheme: %w", err)
		}
	}

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&zap.Options{Development: true})))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{Scheme: scheme})
	if err != nil {
		return nil, fmt.Errorf("failed to create manager: %w", err)
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
	validator := func(ctx context.Context, c client.Client, wl *v2alpha1.Workload) error {
		// Create context handle with metadata if available
		var wlCtx *controller.Context
		if mp, ok := p.(controller.MetadataProvider); ok {
			metadata := mp.GetMetadata()
			wlCtx = controller.NewContextWithMetadata(ctx, c, wl, metadata)
		} else {
			wlCtx = controller.NewContext(ctx, c, wl)
		}
		return p.Validate(wlCtx)
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
	// Filter to only handle Workloawl for this provider
	filter := predicate.NewPredicateFuncs(func(object client.Object) bool {
		wl, ok := object.(*v2alpha1.Workload)
		if !ok {
			return false
		}
		return wl.Spec.Provider == r.provider.Name()
	})

	b := ctrl.NewControllerManagedBy(r.manager).
		For(&v2alpha1.Workload{}, builder.WithPredicates(filter)).
		Named(r.provider.Name() + "-controller")

	// Watch owned types
	for _, obj := range r.provider.OwnedTypes() {
		b.Owns(obj)
	}

	return b.Complete(r)
}

// Reconcile implements the reconciliation loop.
func (r *ProviderReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger := log.FromContext(ctx).WithValues("provider", r.provider.Name())

	// Fetch the Workload
	wl := &v2alpha1.Workload{}
	if err := r.Client.Get(ctx, req.NamespacedName, wl); err != nil {
		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	// Create the Context handle with metadata if available
	var wlCtx *controller.Context
	if mp, ok := r.provider.(controller.MetadataProvider); ok {
		metadata := mp.GetMetadata()
		wlCtx = controller.NewContextWithMetadata(ctx, r.Client, wl, metadata)
	} else {
		wlCtx = controller.NewContext(ctx, r.Client, wl)
	}

	// Handle deletion
	if !wl.GetDeletionTimestamp().IsZero() {
		return r.handleDeletion(ctx, wlCtx, wl, logger)
	}

	// Ensure finalizer is present
	if !controllerutil.ContainsFinalizer(wl, finalizerName) {
		controllerutil.AddFinalizer(wl, finalizerName)
		if err := r.Client.Update(ctx, wl); err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{Requeue: true}, nil
	}

	// Run validation
	if err := r.provider.Validate(wlCtx); err != nil {
		logger.Error(err, "Validation failed")
		// Update status to failed
		wl.Status.Phase = v2alpha1.WorkloadPhaseFailed
		if updateErr := r.Client.Status().Update(ctx, wl); updateErr != nil {
			logger.Error(updateErr, "Failed to update status after validation error")
		}
		return reconcile.Result{}, err
	}

	// Run sync
	logger.Info("Running sync")
	if err := r.provider.Sync(wlCtx); err != nil {
		if controller.IsWaitError(err) {
			logger.Info("Sync waiting", "reason", err.Error())
			return reconcile.Result{RequeueAfter: controller.GetWaitDuration(err)}, nil
		}
		logger.Error(err, "Sync failed")
		return reconcile.Result{}, err
	}

	// Compute and update status
	logger.Info("Computing status")
	status, err := r.provider.Status(wlCtx)
	if err != nil {
		logger.Error(err, "Status computation failed")
		return reconcile.Result{}, err
	}

	wl.Status = status.ToV2Alpha1()
	if err := r.Client.Status().Update(ctx, wl); err != nil {
		logger.Error(err, "Failed to update status")
		return reconcile.Result{}, err
	}

	logger.Info("Reconciliation complete", "phase", wl.Status.Phase)
	return reconcile.Result{}, nil
}

func (r *ProviderReconciler) handleDeletion(
	ctx context.Context,
	wlCtx *controller.Context,
	wl *v2alpha1.Workload,
	logger interface{ Info(string, ...interface{}) },
) (reconcile.Result, error) {
	if !controllerutil.ContainsFinalizer(wl, finalizerName) {
		return reconcile.Result{}, nil
	}

	logger.Info("Running cleanup")

	// Update status to deleting
	if wl.Status.Phase != v2alpha1.WorkloadPhaseDeleting {
		wl.Status.Phase = v2alpha1.WorkloadPhaseDeleting
		if err := r.Client.Status().Update(ctx, wl); err != nil {
			return reconcile.Result{}, err
		}
	}

	// Run cleanup
	if err := r.provider.Cleanup(wlCtx); err != nil {
		if controller.IsWaitError(err) {
			logger.Info("Cleanup waiting", "reason", err.Error())
			return reconcile.Result{RequeueAfter: controller.GetWaitDuration(err)}, nil
		}
		return reconcile.Result{}, err
	}

	// Remove finalizer
	controllerutil.RemoveFinalizer(wl, finalizerName)
	if err := r.Client.Update(ctx, wl); err != nil {
		return reconcile.Result{}, err
	}

	logger.Info("Cleanup complete")
	return reconcile.Result{}, nil
}
