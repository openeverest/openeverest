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
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	authorizationv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	toolscache "k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/events"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	commonv1alpha1 "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
	monitoringv1alpha1 "github.com/openeverest/openeverest/v2/api/monitoring/v1alpha1"
	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
	"github.com/openeverest/openeverest/v2/provider-runtime/internal/instanceprep"
	"github.com/openeverest/openeverest/v2/provider-runtime/server"
	"github.com/openeverest/openeverest/v2/provider-runtime/summary"
	"github.com/openeverest/openeverest/v2/provider-runtime/validation"
)

const finalizerName = "everest.percona.com/provider-finalizer"

// conflictRequeue gives the cache time to catch up with the write that won
// the optimistic lock.
const conflictRequeue = time.Second

// referenceRecheck is how often an Instance with an invalid reference is
// checked again: a referent's contents change without an event on the Instance.
const referenceRecheck = time.Minute

// childrenRecheck is how often a deletion that waits for the Instance's
// Backups and Restores to go looks again.
const childrenRecheck = 10 * time.Second

// =============================================================================
// PROVIDER RECONCILER - Works with both Interface and Builder providers
// =============================================================================

// ProviderReconciler reconciles Instance resources using a Provider.
type ProviderReconciler struct {
	provider      providerAdapter
	manager       ctrl.Manager
	server        *server.Server
	breaker       maintenanceBreaker
	pods          atomic.Value // client.Reader, set once the pod watch runs
	apiReader     client.Reader
	kubeSystemUID atomic.Value // string, the cluster identity of monitoring sources
	recorder      events.EventRecorder
	warned        sync.Map // Instance UID -> the generation its warnings were recorded for

	tracker          tracker
	problemDeadline  time.Duration
	progressDeadline time.Duration
	clock            func() time.Time         // time.Now unless a test sets it
	requeue          atomic.Pointer[requeuer] // set once the controller's queue exists
	client.Client
}

// requeuer queues an Instance to be reconciled again after a delay.
type requeuer func(req reconcile.Request, after time.Duration)

// providerAdapter is the internal interface that both provider types satisfy.
type providerAdapter interface {
	Name() string
	Types() func(*runtime.Scheme) error
	Validate(in controller.ValidationInput) controller.ValidationResult
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
	admissionCertDir   string // set by WithAdmission
	problemDeadline    time.Duration
	progressDeadline   time.Duration
}

// WithProblemDeadline sets how long a problem that may clear by itself, such
// as a missing Secret or a crash loop, is observed on an Instance before the
// Instance is reported as stalled. The default is 10 minutes.
func WithProblemDeadline(d time.Duration) ReconcilerOption {
	return func(o *reconcilerOptions) {
		o.problemDeadline = d
	}
}

// WithProgressDeadline sets how long an Instance may converge with no named
// operation running before it is reported as stalled. The default is 30
// minutes.
func WithProgressDeadline(d time.Duration) ReconcilerOption {
	return func(o *reconcilerOptions) {
		o.progressDeadline = d
	}
}

// WithServer enables the integrated HTTP server, which serves the health and
// readiness endpoints for Kubernetes probes.
//
// Example:
//
//	r, err := reconciler.New(ctx, provider,
//	    reconciler.WithServer(reconciler.ServerConfig{Port: 8082}),
//	)
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
	if err := authorizationv1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add authorizationv1 scheme: %w", err)
	}

	// Register core types
	if err := v1alpha1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add v1alpha1 scheme: %w", err)
	}

	// Register backup types so providers and the runtime can read/write
	// BackupClass/Backup/Restore/BackupStorage CRs without requiring each
	// provider to register them explicitly.
	if err := backupv1alpha1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add backup v1alpha1 scheme: %w", err)
	}

	if err := monitoringv1alpha1.AddToScheme(scheme); err != nil {
		return nil, fmt.Errorf("failed to add monitoring v1alpha1 scheme: %w", err)
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
	if options.admissionCertDir != "" {
		if err := checkAdmissionCert(options.admissionCertDir); err != nil {
			return nil, err
		}
		mgrOpts.WebhookServer = webhook.NewServer(webhook.Options{
			Port:    AdmissionPort,
			CertDir: options.admissionCertDir,
		})
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), mgrOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create manager: %w", err)
	}

	if options.admissionCertDir != "" {
		mgr.GetWebhookServer().Register(AdmissionPath, &webhook.Admission{Handler: &instanceAdmission{
			provider: p,
			client:   mgr.GetClient(),
			decoder:  admission.NewDecoder(scheme),
		}})
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

	// Setup field indexes required by BackupProvider helpers (Backups/Restores
	// for an Instance) when the provider opts in.
	if _, isBackupProvider := p.(controller.BackupProvider); isBackupProvider {
		if err := mgr.GetFieldIndexer().IndexField(ctx, &backupv1alpha1.Backup{}, controller.IndexBackupInstanceName, func(obj client.Object) []string {
			b, ok := obj.(*backupv1alpha1.Backup)
			if !ok || b.Spec.Origin.InstanceRef == nil {
				return nil
			}
			return []string{b.Spec.Origin.InstanceRef.Name}
		}); err != nil {
			return nil, fmt.Errorf("failed to register backup instanceName index: %w", err)
		}
		if err := mgr.GetFieldIndexer().IndexField(ctx, &backupv1alpha1.Restore{}, controller.IndexRestoreInstanceName, func(obj client.Object) []string {
			rs, ok := obj.(*backupv1alpha1.Restore)
			if !ok || rs.Spec.InstanceRef.Name == "" {
				return nil
			}
			return []string{rs.Spec.InstanceRef.Name}
		}); err != nil {
			return nil, fmt.Errorf("failed to register restore instanceName index: %w", err)
		}
	}

	if _, ok := p.(controller.MonitoringIntegrationsProvider); ok {
		if err := mgr.GetFieldIndexer().IndexField(ctx, &monitoringv1alpha1.MonitoringDestination{},
			indexDestinationSecrets, destinationSecrets,
		); err != nil {
			return nil, fmt.Errorf("failed to register monitoring destination credentials index: %w", err)
		}
	}

	r := &ProviderReconciler{
		provider:         p,
		manager:          mgr,
		apiReader:        mgr.GetAPIReader(),
		recorder:         mgr.GetEventRecorder(p.Name() + "-provider"),
		problemDeadline:  options.problemDeadline,
		progressDeadline: options.progressDeadline,
		Client:           mgr.GetClient(),
	}

	if options.serverConfig != nil {
		r.server = server.NewServer(*options.serverConfig)
	}

	if err := r.setup(); err != nil {
		return nil, fmt.Errorf("failed to setup reconciler: %w", err)
	}

	// Register the BackupProvider-aware ancillary reconcilers when the
	// provider implements the optional interface. These dispatch to
	// SyncBackup/SyncRestore for ProviderManaged BackupClasses.
	if bp, ok := p.(controller.BackupProvider); ok {
		if err := setupBackupReconciler(mgr, bp, p.Name()); err != nil {
			return nil, fmt.Errorf("failed to setup backup reconciler: %w", err)
		}
		if err := setupRestoreReconciler(mgr, bp, p.Name()); err != nil {
			return nil, fmt.Errorf("failed to setup restore reconciler: %w", err)
		}
	}

	if bm, ok := p.(controller.BackupMirror); ok {
		if err := setupBackupMirrorReconciler(mgr, bm, p.Name()); err != nil {
			return nil, fmt.Errorf("failed to setup backup mirror reconciler: %w", err)
		}
	}

	if bi, ok := p.(controller.BackupImporter); ok {
		if err := setupBackupImportReconciler(mgr, bi, p.Name()); err != nil {
			return nil, fmt.Errorf("failed to setup backup import reconciler: %w", err)
		}
	}

	return r, nil
}

// GetManager returns the controller manager.
func (r *ProviderReconciler) GetManager() ctrl.Manager {
	return r.manager
}

// Start starts the reconciler and server (blocking).
func (r *ProviderReconciler) Start(ctx context.Context) error {
	if err := r.startServer(ctx); err != nil {
		return err
	}
	return r.manager.Start(ctx)
}

// StartWithSignalHandler starts the reconciler and server with OS signal handling.
func (r *ProviderReconciler) StartWithSignalHandler() error {
	ctx := ctrl.SetupSignalHandler()
	if err := r.startServer(ctx); err != nil {
		return err
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
		return in.Spec.ProviderRef.Name == r.provider.Name()
	})

	b := ctrl.NewControllerManagedBy(r.manager).
		For(&v1alpha1.Instance{}, builder.WithPredicates(filter)).
		Watches(&v1alpha1.Provider{},
			handler.EnqueueRequestsFromMapFunc(r.instancesForProvider),
			builder.WithPredicates(
				predicate.NewPredicateFuncs(func(o client.Object) bool { return o.GetName() == r.provider.Name() }),
				predicate.GenerationChangedPredicate{},
			),
		).
		WatchesMetadata(&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.instancesForSecret),
			builder.WithPredicates(existenceChanged()),
		).
		// The controller hands its queue to its sources when it starts. This
		// one watches nothing: it keeps the queue, so that an Instance can be
		// queued for the time its status is next due to change.
		WatchesRawSource(source.Func(func(_ context.Context, queue workqueue.TypedRateLimitingInterface[reconcile.Request]) error {
			var again requeuer = queue.AddAfter
			r.requeue.Store(&again)

			return nil
		})).
		Named(r.provider.Name() + "-controller")

	if r.isBackupProvider() {
		b.Watches(&backupv1alpha1.BackupClass{},
			handler.EnqueueRequestsFromMapFunc(r.instancesForBackupClass),
			builder.WithPredicates(predicate.GenerationChangedPredicate{}),
		)
		b.Watches(&backupv1alpha1.BackupStorage{},
			handler.EnqueueRequestsFromMapFunc(r.instancesForBackupStorage),
			builder.WithPredicates(predicate.GenerationChangedPredicate{}),
		)
	}

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

	if monitoringEnabled(r.provider) {
		r.watchMonitoringBindings(b)
	}

	c, err := b.Build(r)
	if err != nil {
		return err
	}

	return r.manager.Add(manager.RunnableFunc(func(ctx context.Context) error {
		return r.watchPods(ctx, c)
	}))
}

// Reconcile implements the reconciliation loop.
func (r *ProviderReconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger := log.FromContext(ctx).WithValues("provider", r.provider.Name())
	ctx = log.IntoContext(ctx, logger)

	// Fetch the Instance
	in := &v1alpha1.Instance{}
	if err := r.Client.Get(ctx, req.NamespacedName, in); err != nil {
		if apierrors.IsNotFound(err) {
			r.tracker.forget(req.NamespacedName)
		}

		return reconcile.Result{}, client.IgnoreNotFound(err)
	}

	// Create the Context handle
	inCtx := controller.NewContext(ctx, r.Client, in, r.provider.Name())

	// Handle deletion
	if !in.GetDeletionTimestamp().IsZero() {
		return r.handleDeletion(ctx, inCtx, in, logger)
	}

	// Ensure provider label and finalizer are present. A merge patch, unlike
	// Update, cannot erase spec fields this binary's types do not know.
	var needsUpdate bool
	base := in.DeepCopy()
	if in.Labels == nil {
		in.Labels = make(map[string]string)
	}
	if in.Labels[controller.ProviderLabel] != r.provider.Name() {
		in.Labels[controller.ProviderLabel] = r.provider.Name()
		needsUpdate = true
	}
	if !controllerutil.ContainsFinalizer(in, finalizerName) {
		controllerutil.AddFinalizer(in, finalizerName)
		needsUpdate = true
	}
	if needsUpdate {
		if err := r.Patch(ctx, in, client.MergeFromWithOptions(base, client.MergeFromWithOptimisticLock{})); err != nil {
			if apierrors.IsConflict(err) {
				return reconcile.Result{RequeueAfter: conflictRequeue}, nil
			}
			return reconcile.Result{}, err
		}
		// Nothing is written to the status on this pass. Until the first
		// write the Instance reads as in progress, because its
		// status.observedGeneration defaults to -1.
		return reconcile.Result{Requeue: true}, nil
	}

	// Every way out from here goes through finish, which writes the status.
	p := r.begin(in)

	return r.finish(ctx, p, r.reconcileInstance(ctx, p, inCtx))
}

// reconcileInstance runs one pass over an Instance that is not being
// deleted, recording what it finds on p. It stops at the first step that
// cannot go on; the status then says why.
func (r *ProviderReconciler) reconcileInstance(ctx context.Context, p *pass, inCtx *controller.Context) error {
	if ok, err := r.checkReferents(ctx, p); !ok {
		return err
	}
	if ok, err := r.checkSpec(ctx, p, inCtx); !ok {
		return err
	}
	syncCtx, err := r.syncContext(ctx, p)
	if err != nil {
		return err
	}
	if ok, err := r.sync(ctx, p, syncCtx); !ok {
		return err
	}

	return r.observe(ctx, p, syncCtx)
}

// checkReferents reports whether the objects the Instance names exist and
// can be used. The provider is not called while one is missing; the watches
// re-enqueue the Instance once it appears.
func (r *ProviderReconciler) checkReferents(ctx context.Context, p *pass) (bool, error) {
	logger := log.FromContext(ctx)
	in := p.in

	b, err := missingReferent(ctx, r.Client, in)
	if err != nil {
		return false, err
	}
	if b.reason != "" {
		logger.Info("Waiting for a referent", "reason", b.reason, "message", b.message)
		p.check(v1alpha1.ConditionReferencesResolved, metav1.ConditionFalse, b.reason, b.message)

		return false, nil
	}
	p.referentsExist()

	backupMiss, err := r.missingBackupReferent(ctx, in)
	if err != nil {
		return false, err
	}
	if backupMiss.reason != "" {
		logger.Info("Waiting for a backup referent", "reason", backupMiss.reason, "message", backupMiss.message)
		p.check(v1alpha1.ConditionBackupConfigured, metav1.ConditionFalse, backupMiss.reason, backupMiss.message)

		return false, nil
	}

	return true, nil
}

// checkSpec reports whether the spec can be handed to the provider. A
// rejected spec waits for the Instance or the Provider to change, and both
// enqueue the Instance, so it is not requeued. Only a person can change
// them, which is why the Instance is reported as stalled at once.
func (r *ProviderReconciler) checkSpec(ctx context.Context, p *pass, inCtx *controller.Context) (bool, error) {
	logger := log.FromContext(ctx)
	in := p.in

	if err := validateInstanceBackupConfig(ctx, r.Client, in); err != nil {
		var reason string
		switch {
		case errors.Is(err, controller.ErrPITRConfigInvalid):
			reason = controller.PITRConfigInvalidReason
		case errors.Is(err, controller.ErrBackupClassLimitsExceeded):
			reason = controller.LimitsExceededReason
		default:
			return false, err
		}
		// The violation is also reported on BackupConfigured: the engine
		// itself may be healthy, and the user can fix the configuration
		// without a full redeploy.
		logger.Info("Backup configuration rejected", "reason", reason, "message", err.Error())
		p.check(v1alpha1.ConditionBackupConfigured, metav1.ConditionFalse, reason, err.Error())
		p.reject(reason, err.Error())

		return false, nil
	}
	p.backupUsable = true

	errs, err := r.validate(ctx, in)
	if err != nil {
		return false, err
	}
	if len(errs) > 0 {
		message := validation.Summary(errs)
		logger.Info("Validation failed", "errors", message)
		p.reject(v1alpha1.ReasonValidationFailed, message)

		return false, nil
	}
	if rv, ok := r.provider.(controller.ReferenceValidator); ok {
		errs, err := rv.ValidateReferences(inCtx)
		if err != nil {
			logger.Error(err, "Reference validation could not run")

			return false, err
		}
		if len(errs) > 0 {
			message := validation.Summary(errs)
			logger.Info("Reference validation failed", "errors", message)
			p.check(v1alpha1.ConditionReferencesResolved, metav1.ConditionFalse, v1alpha1.ReasonInvalidReference, message)
			p.lookAgainIn(referenceRecheck)

			return false, nil
		}
	}
	p.check(v1alpha1.ConditionReferencesResolved, metav1.ConditionTrue, v1alpha1.ReasonResolved, "")

	return true, nil
}

// syncContext returns the context the provider is called with. The provider
// is handed a prepared copy of the Instance, never the stored object.
func (r *ProviderReconciler) syncContext(ctx context.Context, p *pass) (*controller.Context, error) {
	in := p.in
	resolvedIn, effectiveBundleName, err := r.prepareInstance(ctx, in)
	if err != nil {
		log.FromContext(ctx).Error(err, "Preparing the instance for sync failed")

		return nil, err
	}
	p.version = effectiveBundleName

	syncCtx := controller.NewContext(ctx, r.Client, resolvedIn, r.provider.Name())
	syncCtx.BlockMaintenance(r.breaker.blockedTokens(client.ObjectKeyFromObject(in), approvedMaintenanceValue(in))...)
	if ip, ok := r.provider.(controller.MonitoringIntegrationsProvider); ok {
		syncCtx.SetMonitoringIntegrations(ip.MonitoringIntegrations()...)
	}

	return syncCtx, nil
}

// sync runs the provider's Sync and reports whether the provider can go on
// to say what it observes.
func (r *ProviderReconciler) sync(ctx context.Context, p *pass, syncCtx *controller.Context) (bool, error) {
	in := p.in

	// Passive warning for owners: flag effective component versions the
	// installed catalog marks as deprecated, before any upgrade is attempted.
	r.setDeprecationCondition(ctx, in)

	log.FromContext(ctx).Info("Running sync")
	err := r.provider.Sync(syncCtx)
	// What Sync found out about the seeding of a new Instance stands,
	// however Sync returned.
	p.flushDataSource(syncCtx)
	if err != nil {
		return false, r.syncStopped(ctx, p, syncCtx, err)
	}

	// A pass that merely held a tripped action succeeds trivially; resetting
	// then would un-trip the breaker and restart the failure bursts.
	if !syncCtx.MaintenanceBreakerHeld() {
		r.breaker.reset(client.ObjectKeyFromObject(in))
	}
	// Clear any stale BackupConfigured=False condition left from a previous failed Sync.
	if r.isBackupProvider() {
		p.check(v1alpha1.ConditionBackupConfigured, metav1.ConditionTrue, "Configured", "Backup configuration applied to engine")
	}

	// Flush the actions Context.RequestMaintenance held during Sync. The
	// pending list is rebuilt from scratch every pass so it can never go
	// stale, while the approval stays durable in spec.
	flushPendingMaintenance(syncCtx, in)

	return true, nil
}

// syncStopped handles a Sync that did not complete. A Sync that waits, or
// that could not configure backups or the seeding of a new Instance, ends
// the pass without failing it: the status says what it waits for or what is
// wrong, and the engine itself may be healthy.
func (r *ProviderReconciler) syncStopped(ctx context.Context, p *pass, syncCtx *controller.Context, err error) error {
	logger := log.FromContext(ctx)
	in := p.in

	if controller.IsWaitError(err) {
		logger.Info("Sync waiting", "reason", err.Error())
		p.wait(err)

		return nil
	}

	bce, dse := controller.AsBackupConfigError(err), controller.AsDataSourceConfigError(err)
	switch {
	case bce != nil:
		logger.Error(err, "Backup configuration failed")
		p.check(v1alpha1.ConditionBackupConfigured, metav1.ConditionFalse, bce.Reason, bce.Message)
	case dse != nil:
		logger.Error(err, "DataSource configuration failed")
		p.check(v1alpha1.ConditionDataSourceReady, metav1.ConditionFalse, dse.Reason, dse.Message)
	default:
		logger.Error(err, "Sync failed")
		r.breaker.recordFailure(client.ObjectKeyFromObject(in), approvedMaintenanceValue(in), syncCtx.GetApprovedMaintenance())

		return err
	}
	p.providerRan = true
	// The provider got as far as declaring maintenance before failing, so
	// the staged set is authoritative; without this the pending list and
	// condition go stale until a fully successful pass.
	if syncCtx.MaintenanceRequested() {
		flushPendingMaintenance(syncCtx, in)
	}

	return nil
}

// observe asks the provider what it observes about the engine and records
// what the runtime observes itself.
func (r *ProviderReconciler) observe(ctx context.Context, p *pass, syncCtx *controller.Context) error {
	logger := log.FromContext(ctx)
	in := p.in

	logger.Info("Computing status")
	status, err := r.provider.Status(syncCtx)
	if err != nil {
		logger.Error(err, "Status computation failed")

		return err
	}
	p.reported(ctx, status)
	p.lookAgainIn(r.setPodStatus(ctx, in, syncCtx.LabelledComponents(), p.now))
	p.podsObserved = true

	// Collect per-storage backup observability data (e.g. the latest
	// restorable time for PITR) when the provider opts into reporting it.
	// Failures here are logged but never fail the reconcile: this is
	// observability data and the engine itself is healthy.
	if reporter, ok := r.provider.(controller.InstanceBackupStatusReporter); ok {
		storageStatuses, repErr := reporter.BackupStorageStatuses(syncCtx)
		switch {
		case repErr != nil:
			logger.Error(repErr, "Failed to collect backup storage statuses")
		case len(storageStatuses) > 0:
			in.Status.Backup = &v1alpha1.InstanceBackupStatus{Storages: storageStatuses}
		default:
			in.Status.Backup = nil
		}
	}

	// Freeze the effective bundle name in status so it remains stable across
	// Provider upgrades. On subsequent reconciliations the reconciler reads this
	// value back (when spec.version is empty) instead of re-resolving the
	// Provider's current default — preventing silent upgrades on existing
	// Instances. GitOps tools exclude status from diff calculations by default
	// so this field never causes spurious out-of-sync alerts.
	if p.version != "" {
		in.Status.Version = p.version
	}

	r.reconcileMonitoring(ctx, syncCtx, in)

	// Write connection details Secret and set the ConnectionDetailsReady condition.
	if err := r.reconcileConnectionSecret(ctx, in, status); err != nil {
		logger.Error(err, "Failed to reconcile connection secret")

		return err
	}

	return nil
}

// leaderElectionFree marks a Runnable as one that must run on every replica,
// not only the elected leader: readiness is a per-pod property.
type leaderElectionFree struct{ manager.Runnable }

func (leaderElectionFree) NeedLeaderElection() bool { return false }

// startServer wires the health server into the manager lifecycle. It is a
// no-op when no server is configured.
//
// Readiness is gated on the manager's caches: the server is marked ready from a
// manager Runnable only after the Instance and Provider informers have synced,
// so /readyz reports ready only once the cache-backed client is usable. This
// keeps the Service from routing admission requests, which read the Provider
// from that cache, to a pod before it can serve them. Readiness is cleared
// again on shutdown.
//
// The Runnable is wrapped in leaderElectionFree so it runs on every replica, not
// only the elected leader. Must be called before manager.Start.
func (r *ProviderReconciler) startServer(ctx context.Context) error {
	if r.server == nil {
		return nil
	}

	go func() {
		if err := r.server.Start(ctx); err != nil {
			log.FromContext(ctx).Error(err, "Server error")
		}
	}()

	return r.manager.Add(leaderElectionFree{manager.RunnableFunc(func(runnableCtx context.Context) error {
		// Report ready only once the Instance and Provider informers have
		// synced. Stay not-ready if the context is cancelled first (shutdown).
		for _, obj := range []client.Object{&v1alpha1.Instance{}, &v1alpha1.Provider{}} {
			if !r.waitForCacheSync(runnableCtx, obj) {
				return nil
			}
		}
		r.server.SetReady(true)
		<-runnableCtx.Done()
		r.server.SetReady(false)
		return nil
	})})
}

// waitForCacheSync blocks until the informer for obj has been built and synced,
// returning true on success or false if ctx is cancelled first. GetInformer
// returns an error while the API server is unreachable, so it is retried with a
// short backoff to tolerate a transient blip at startup rather than latching
// not-ready until the process restarts.
func (r *ProviderReconciler) waitForCacheSync(ctx context.Context, obj client.Object) bool {
	for {
		informer, err := r.manager.GetCache().GetInformer(ctx, obj)
		if err == nil {
			return toolscache.WaitForCacheSync(ctx.Done(), informer.HasSynced)
		}
		// The API server can be briefly unreachable at startup; retry until it
		// is reachable (controller-runtime logs the underlying failure) or the
		// context is cancelled.
		select {
		case <-ctx.Done():
			return false
		case <-time.After(time.Second):
		}
	}
}

// handleDeletion cleans up after an Instance whose deletion was requested,
// and removes the finalizer once nothing is left. Until then every pass
// reports on the status what the deletion waits for.
func (r *ProviderReconciler) handleDeletion(
	ctx context.Context,
	inCtx *controller.Context,
	in *v1alpha1.Instance,
	logger interface{ Info(string, ...interface{}) },
) (reconcile.Result, error) {
	r.warned.Delete(in.UID)
	if !controllerutil.ContainsFinalizer(in, finalizerName) {
		return reconcile.Result{}, nil
	}

	p := r.begin(in)
	p.deleting = true

	// Say that the Instance is being deleted before cleanup starts, which can
	// take long. Later passes write only at their end, once they know what
	// cleanup waits for, so the status does not alternate between two
	// reasons.
	if in.Status.Phase != v1alpha1.InstancePhaseTerminating {
		if err := r.writeStatus(ctx, p); err != nil {
			return reconcile.Result{}, err
		}
	}

	logger.Info("Running cleanup")
	done, err := r.cleanup(ctx, p, inCtx, logger)
	if !done {
		return r.finish(ctx, p, err)
	}

	// Remove finalizer
	base := in.DeepCopy()
	controllerutil.RemoveFinalizer(in, finalizerName)
	if err := r.Patch(ctx, in, client.MergeFromWithOptions(base, client.MergeFromWithOptimisticLock{})); err != nil {
		if apierrors.IsConflict(err) {
			return reconcile.Result{RequeueAfter: conflictRequeue}, nil
		}
		return reconcile.Result{}, err
	}
	// Drop what is remembered about the Instance, so a recreated Instance
	// with the same name does not inherit this lifetime's failure counts.
	r.breaker.reset(client.ObjectKeyFromObject(in))
	r.tracker.forget(client.ObjectKeyFromObject(in))

	logger.Info("Cleanup complete")
	return reconcile.Result{}, nil
}

// cleanup deletes what the Instance leaves behind. It returns false while
// something is still being deleted, having recorded on p what it waits for.
func (r *ProviderReconciler) cleanup(
	ctx context.Context,
	p *pass,
	inCtx *controller.Context,
	logger interface {
		Info(msg string, keysAndValues ...any)
	},
) (bool, error) {
	// Cascade delete Backup/Restore CRs that reference this Instance before
	// tearing down the engine. We only attempt this when the provider opts
	// into BackupProvider (the case in which the Backup/Restore field
	// indexes are registered and the user is expected to be creating
	// Backup CRs against this Instance).
	if r.isBackupProvider() && p.in.Spec.DeletionPolicy == v1alpha1.InstanceDeletionPolicyCascade {
		backups, restores, err := r.cascadeDeleteChildren(ctx, inCtx, logger)
		if err != nil {
			return false, err
		}
		if backups+restores > 0 {
			logger.Info("Waiting for child Backup/Restore CRs to terminate", "remaining", backups+restores)
			p.waiting = waitingForChildren(backups, restores)
			p.lookAgainIn(childrenRecheck)

			return false, nil
		}
	}

	if err := r.provider.Cleanup(inCtx); err != nil {
		if controller.IsWaitError(err) {
			logger.Info("Cleanup waiting", "reason", err.Error())
			p.wait(err)

			return false, nil
		}

		return false, err
	}

	return true, nil
}

// waitingForChildren names the Backups and Restores a deletion waits for.
func waitingForChildren(backups, restores int) *summary.Cause {
	count := func(n int, noun string) string {
		if n == 1 {
			return "1 " + noun
		}

		return fmt.Sprintf("%d %ss", n, noun)
	}
	switch {
	case backups == 0:
		return &summary.Cause{
			Reason:  v1alpha1.ReasonWaitingForRestores,
			Message: fmt.Sprintf("Waiting for %s to be deleted", count(restores, "restore")),
		}
	case restores == 0:
		return &summary.Cause{
			Reason:  v1alpha1.ReasonWaitingForBackups,
			Message: fmt.Sprintf("Waiting for %s to be deleted", count(backups, "backup")),
		}
	default:
		return &summary.Cause{
			Reason:  v1alpha1.ReasonWaitingForBackups,
			Message: fmt.Sprintf("Waiting for %s and %s to be deleted", count(backups, "backup"), count(restores, "restore")),
		}
	}
}

// cascadeDeleteChildren issues Delete on every Backup and Restore CR in the
// Instance's namespace where Backup.spec.origin.instanceRef.name or
// Restore.spec.instanceRef.name matches this Instance, then
// returns the number of Backups and of Restores still present (counting both
// freshly-deleted ones that have not yet been reaped by their own controllers
// and any older ones still finalizing). The caller should requeue while any
// remain.
//
// Each child Backup's own .spec.deletionPolicy controls whether the
// underlying data in the BackupStorage is purged or retained — this function
// only triggers Delete on the CR itself and never overrides that decision.
func (r *ProviderReconciler) cascadeDeleteChildren(
	ctx context.Context,
	inCtx *controller.Context,
	logger interface{ Info(string, ...interface{}) },
) (int, int, error) {
	backups, err := inCtx.BackupsForInstance()
	if err != nil {
		return 0, 0, fmt.Errorf("list backups for cascade delete: %w", err)
	}
	for i := range backups {
		b := &backups[i]
		if !b.GetDeletionTimestamp().IsZero() {
			continue
		}
		if err := r.Client.Delete(ctx, b); err != nil && !apierrors.IsNotFound(err) {
			return 0, 0, fmt.Errorf("delete child Backup %q: %w", b.Name, err)
		}
		logger.Info("Cascade delete: issued Delete on Backup", "name", b.Name)
	}

	restores, err := inCtx.RestoresForInstance()
	if err != nil {
		return 0, 0, fmt.Errorf("list restores for cascade delete: %w", err)
	}
	for i := range restores {
		rs := &restores[i]
		if !rs.GetDeletionTimestamp().IsZero() {
			continue
		}
		if err := r.Client.Delete(ctx, rs); err != nil && !apierrors.IsNotFound(err) {
			return 0, 0, fmt.Errorf("delete child Restore %q: %w", rs.Name, err)
		}
		logger.Info("Cascade delete: issued Delete on Restore", "name", rs.Name)
	}

	return len(backups), len(restores), nil
}

// reconcileConnectionSecret creates or updates the connection details Secret
// and sets the ConnectionDetailsReady condition on the Instance status. The
// Secret's "provider" entry is always the Provider name, whatever the provider
// reported.
func (r *ProviderReconciler) reconcileConnectionSecret(
	ctx context.Context,
	in *v1alpha1.Instance,
	status controller.Status,
) error {
	now := metav1.Now()

	if status.ConnectionDetails.IsEmpty() {
		if status.Converged {
			setCondition(in, v1alpha1.ConditionConnectionDetailsReady, metav1.ConditionFalse,
				"NotReported", "Provider did not report connection details", now)
		}
		return nil
	}

	details := status.ConnectionDetails
	details.Provider = r.provider.Name()

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

	if err := controllerutil.SetControllerReference(in, secret, r.Scheme()); err != nil {
		return fmt.Errorf("failed to set owner reference on connection secret: %w", err)
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, secret, func() error {
		secret.Data = details.ToSecretData()
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to create or update connection secret: %w", err)
	}

	in.Status.ConnectionSecretRef = &commonv1alpha1.SecretRef{Name: secretName}
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

// setDeprecationCondition maintains the read-only ComponentVersionDeprecated
// condition: True while any of the Instance's effective component versions is
// flagged as deprecated — or no longer supported at all — in the installed
// Provider catalog. It reuses the upgrade preflight's catalog check, so this
// passive warning and the pre-upgrade hook's verdict never disagree. The
// condition is informational: lookup failures are skipped and never fail the
// reconcile, and it is only flipped to False (never removed) once every issue
// it reported clears.
func (r *ProviderReconciler) setDeprecationCondition(ctx context.Context, in *v1alpha1.Instance) {
	installed := &v1alpha1.Provider{}
	if err := r.Get(ctx, client.ObjectKey{Name: in.Spec.ProviderRef.Name}, installed); err != nil {
		log.FromContext(ctx).V(1).Info("skipping deprecation condition: fetching installed Provider failed", "error", err)
		return
	}

	var deprecations, unsupported []string
	for _, issue := range controller.PreflightUpgrade(&installed.Spec, []v1alpha1.Instance{*in}) {
		msg := issue.Message
		if issue.Component != "" {
			msg = issue.Component + ": " + msg
		}
		switch issue.Severity {
		case controller.UpgradeError:
			unsupported = append(unsupported, msg)
		case controller.UpgradeWarning:
			if issue.Reason == controller.UpgradeReasonVersionDeprecated {
				deprecations = append(deprecations, msg)
			}
		}
	}

	// A version the installed catalog no longer supports at all is strictly
	// worse than a deprecation; it must never read as "all supported".
	if len(unsupported) > 0 {
		setCondition(in, v1alpha1.ConditionComponentVersionDeprecated, metav1.ConditionTrue,
			v1alpha1.ReasonVersionsUnsupported, strings.Join(unsupported, "; "), metav1.Now())
		return
	}
	if len(deprecations) > 0 {
		setCondition(in, v1alpha1.ConditionComponentVersionDeprecated, metav1.ConditionTrue,
			v1alpha1.ReasonScheduledForRemoval, strings.Join(deprecations, "; "), metav1.Now())
		return
	}
	for _, c := range in.Status.Conditions {
		if c.Type == v1alpha1.ConditionComponentVersionDeprecated {
			setCondition(in, v1alpha1.ConditionComponentVersionDeprecated, metav1.ConditionFalse,
				v1alpha1.ReasonVersionsSupported,
				"All component versions are supported by the installed provider", metav1.Now())
			return
		}
	}
}

// flushPendingMaintenance writes the actions held by RequestMaintenance to
// status.pendingMaintenance and maintains the MaintenancePending condition:
// True while anything is held, flipped to False (never removed) once the
// last held action clears. When a hold is due to exhausted retries the
// reason says so, surfacing the breaker state.
func flushPendingMaintenance(syncCtx *controller.Context, in *v1alpha1.Instance) {
	pending := syncCtx.GetPendingMaintenance()
	in.Status.PendingMaintenance = pending

	if len(pending) > 0 {
		reason := v1alpha1.ReasonAwaitingApproval
		message := fmt.Sprintf("%d action(s) require approval to proceed", len(pending))
		if syncCtx.MaintenanceBreakerHeld() {
			reason = v1alpha1.ReasonRetriesExhausted
			message = fmt.Sprintf(
				"%d action(s) held; an approved action kept failing and is no longer retried — change spec.maintenance.approved (set it to the pending action's token, or clear and re-set it) to retry",
				len(pending))
		}
		setCondition(in, v1alpha1.ConditionMaintenancePending, metav1.ConditionTrue,
			reason, message, metav1.Now())
		return
	}
	for _, c := range in.Status.Conditions {
		if c.Type == v1alpha1.ConditionMaintenancePending {
			setCondition(in, v1alpha1.ConditionMaintenancePending, metav1.ConditionFalse,
				v1alpha1.ReasonNoActionsPending,
				"No disruptive actions are held", metav1.Now())
			return
		}
	}
}

// approvedMaintenanceValue returns spec.maintenance.approved, tolerating an
// unset maintenance block.
func approvedMaintenanceValue(in *v1alpha1.Instance) string {
	if in.Spec.Maintenance == nil {
		return ""
	}
	return in.Spec.Maintenance.Approved
}

// prepareInstance fetches the Provider and returns the Instance the provider
// should be handed, together with the version bundle that was applied. The
// stored Instance is never mutated: the copy is used for Sync and Status only.
//
// The applied bundle name is written to status.version by the caller, so that
// later reconciles keep the bundle an Instance started on.
func (r *ProviderReconciler) prepareInstance(
	ctx context.Context,
	in *v1alpha1.Instance,
) (*v1alpha1.Instance, string, error) {
	providerObj := &v1alpha1.Provider{}
	if err := r.Get(ctx, client.ObjectKey{Name: in.Spec.ProviderRef.Name}, providerObj); err != nil {
		return nil, "", fmt.Errorf("fetching provider for version resolution: %w", err)
	}
	return instanceprep.PrepareForSync(&providerObj.Spec, in)
}

// fetchBackupClassForInstance returns the BackupClass referenced by
// .spec.backup.classRef, or (nil, nil) when the Instance has no backup
// configuration. Returns (nil, nil) when the referenced BackupClass does not
// (yet) exist; the reconciler will requeue via Sync and surface the missing
// dependency as a BackupConfigError.
func fetchBackupClassForInstance(ctx context.Context, c client.Client, in *v1alpha1.Instance) (*backupv1alpha1.BackupClass, error) {
	if in.Spec.Backup == nil || in.Spec.Backup.ClassRef.Name == "" {
		return nil, nil
	}
	bc := &backupv1alpha1.BackupClass{}
	if err := c.Get(ctx, client.ObjectKey{Name: in.Spec.Backup.ClassRef.Name}, bc); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("fetching backup class for limits validation: %w", err)
	}
	return bc, nil
}

// validateInstanceBackupConfig enforces the generic backup configuration
// rules declared on a ProviderManaged BackupClass against an Instance's
// .spec.backup: the numeric limits (maxStorages, maxPITREnabledStorages,
// maxSchedulesPerStorage) and the per-storage PITR config schema. It is a
// no-op when no class is referenced or when the class is Job-mode. Returns
// the sentinels controller.ErrBackupClassLimitsExceeded and
// controller.ErrPITRConfigInvalid that providers see via the helpers.
func validateInstanceBackupConfig(ctx context.Context, c client.Client, in *v1alpha1.Instance) error {
	bc, err := fetchBackupClassForInstance(ctx, c, in)
	if err != nil {
		return err
	}
	if err := controller.ValidateInstanceBackupAgainstClass(in, bc); err != nil {
		return err
	}
	return controller.ValidateInstanceBackupPITRParameters(in, bc)
}
