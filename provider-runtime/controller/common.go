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

package controller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	backupv1alpha1 "github.com/openeverest/openeverest/v2/api/backup/v1alpha1"
	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

// =============================================================================
// CORE ABSTRACTION: The Context handle
// =============================================================================

// Context is the main handle for working with an Instance.
// It provides a simplified interface that hides Kubernetes complexity.
type Context struct {
	ctx          context.Context
	client       client.Client
	in           *v1alpha1.Instance
	providerName string

	// dataSourceStatus is staged by ReconcileDataSource so the reconciler can
	// flush the corresponding condition onto the Instance after Sync. It is
	// nil when the provider has not invoked the helper this reconcile pass.
	dataSourceStatus *DataSourceStatus
}

// NewContext creates a new Context handle (used internally by the reconciler).
func NewContext(ctx context.Context, c client.Client, in *v1alpha1.Instance, providerName string) *Context {
	return &Context{ctx: ctx, client: c, in: in, providerName: providerName}
}

// Context returns the underlying context.Context.
func (c *Context) Context() context.Context {
	return c.ctx
}

// Client returns the underlying Kubernetes client.
func (c *Context) Client() client.Client {
	return c.client
}

// Spec returns the instance specification.
func (c *Context) Spec() *v1alpha1.InstanceSpec {
	return &c.in.Spec
}

// Name returns the instance name.
func (c *Context) Name() string {
	return c.in.Name
}

// Namespace returns the instance namespace.
func (c *Context) Namespace() string {
	return c.in.Namespace
}

// Labels returns the instance labels.
func (c *Context) Labels() map[string]string {
	return c.in.Labels
}

// Annotations returns the instance annotations.
func (c *Context) Annotations() map[string]string {
	return c.in.Annotations
}

// ComponentsOfType returns all components of a given type.
func (c *Context) ComponentsOfType(componentType string) []v1alpha1.ComponentSpec {
	return c.in.GetComponentsOfType(componentType)
}

// Instance returns the underlying Instance for direct access.
func (c *Context) Instance() *v1alpha1.Instance {
	return c.in
}

// ProviderSpec fetches the Provider CR spec from the controller-runtime cache.
// This returns an always up-to-date version of the spec without hitting the
// Kubernetes API server, as reads go through the controller-runtime informer cache.
func (c *Context) ProviderSpec() (*v1alpha1.ProviderSpec, error) {
	provider := &v1alpha1.Provider{}
	if err := c.client.Get(c.ctx, client.ObjectKey{Name: c.providerName}, provider); err != nil {
		return nil, fmt.Errorf("failed to get provider spec: %w", err)
	}
	return &provider.Spec, nil
}

// =============================================================================
// RESOURCE OPERATIONS
// =============================================================================

// Apply creates or updates a resource, setting ownership automatically.
// This is the primary way to manage resources - just describe what you want.
func (c *Context) Apply(obj client.Object) error {
	// Set the owner reference automatically
	if err := controllerutil.SetControllerReference(c.in, obj, c.client.Scheme()); err != nil {
		return fmt.Errorf("failed to set owner: %w", err)
	}

	// Use create-or-update semantics
	existing := obj.DeepCopyObject().(client.Object)
	err := c.client.Get(c.ctx, client.ObjectKeyFromObject(obj), existing)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			return err
		}
		// Doesn't exist, create it
		return c.client.Create(c.ctx, obj)
	}
	// Exists, update it
	obj.SetResourceVersion(existing.GetResourceVersion())
	return c.client.Update(c.ctx, obj)
}

// Get retrieves a resource by name (in the instance's namespace).
func (c *Context) Get(obj client.Object, name string) error {
	return c.client.Get(c.ctx, client.ObjectKey{
		Namespace: c.in.Namespace,
		Name:      name,
	}, obj)
}

// Exists checks if a resource exists.
func (c *Context) Exists(obj client.Object, name string) (bool, error) {
	err := c.Get(obj, name)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			return false, err
		}
		return false, nil
	}
	return true, nil
}

// Delete removes a resource.
func (c *Context) Delete(obj client.Object) error {
	err := c.client.Delete(c.ctx, obj)
	return client.IgnoreNotFound(err)
}

// List retrieves resources matching optional filters.
func (c *Context) List(list client.ObjectList, opts ...client.ListOption) error {
	allOpts := append([]client.ListOption{client.InNamespace(c.in.Namespace)}, opts...)
	return c.client.List(c.ctx, list, allOpts...)
}

// =============================================================================
// HELPER METHODS
// =============================================================================

// ObjectMeta returns a pre-configured ObjectMeta for creating resources.
func (c *Context) ObjectMeta(name string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:      name,
		Namespace: c.Namespace(),
		Labels: map[string]string{
			"app.kubernetes.io/managed-by": "everest",
			"app.kubernetes.io/instance":   c.Name(),
		},
	}
}

// DecodeTopologyConfig unmarshals the topology configuration into the provided struct.
// The target should be a pointer to the expected config type.
// Returns an error if the config is nil, empty, or unmarshaling fails.
//
// Example:
//
//	var config psmdbspec.ShardedTopologyConfig
//	if err := c.DecodeTopologyConfig(&config); err != nil {
//	    // handle error or use defaults
//	}
func (c *Context) DecodeTopologyConfig(target interface{}) error {
	topologyConfig := c.in.GetTopologyConfig()
	if topologyConfig == nil || topologyConfig.Raw == nil {
		return fmt.Errorf("topology config not set")
	}
	return json.Unmarshal(topologyConfig.Raw, target)
}

// DecodeGlobalConfig unmarshals the global configuration into the provided struct.
// The target should be a pointer to the expected config type.
// Returns an error if the config is nil, empty, or unmarshaling fails.
//
// Example:
//
//	var config psmdbspec.GlobalConfig
//	if err := c.DecodeGlobalConfig(&config); err != nil {
//	    // handle error or use defaults
//	}
func (c *Context) DecodeGlobalConfig(target interface{}) error {
	globalConfig := c.in.Spec.Global
	if globalConfig == nil || globalConfig.Raw == nil {
		return fmt.Errorf("global config not set")
	}
	return json.Unmarshal(globalConfig.Raw, target)
}

// DecodeComponentCustomSpec unmarshals a component's custom spec into the provided struct.
// The target should be a pointer to the expected custom spec type.
// Returns an error if the custom spec is nil, empty, or unmarshaling fails.
//
// Example:
//
//	engine := c.wl.Spec.Components["engine"]
//	var customSpec psmdbspec.MongodCustomSpec
//	if err := c.DecodeComponentCustomSpec(engine, &customSpec); err != nil {
//	    // handle error or use defaults
//	}
func (c *Context) DecodeComponentCustomSpec(component v1alpha1.ComponentSpec, target interface{}) error {
	if component.CustomSpec == nil || component.CustomSpec.Raw == nil {
		return fmt.Errorf("component custom spec not set")
	}
	return json.Unmarshal(component.CustomSpec.Raw, target)
}

// TryDecodeTopologyConfig attempts to decode topology config, returning false if not set.
// This is a convenience method that doesn't return an error for missing configs.
//
// Example:
//
//	var config psmdbspec.ShardedTopologyConfig
//	if c.TryDecodeTopologyConfig(&config) {
//	    numShards = config.NumShards
//	} else {
//	    numShards = 2 // default
//	}
func (c *Context) TryDecodeTopologyConfig(target interface{}) bool {
	err := c.DecodeTopologyConfig(target)
	return err == nil
}

// TryDecodeGlobalConfig attempts to decode global config, returning false if not set.
func (c *Context) TryDecodeGlobalConfig(target interface{}) bool {
	err := c.DecodeGlobalConfig(target)
	return err == nil
}

// TryDecodeComponentCustomSpec attempts to decode component custom spec, returning false if not set.
func (c *Context) TryDecodeComponentCustomSpec(component v1alpha1.ComponentSpec, target interface{}) bool {
	err := c.DecodeComponentCustomSpec(component, target)
	return err == nil
}

// ComponentConfig fetches the configuration string for a component from its
// referenced Secret or ConfigMap (via component.Config).
// Returns ("", nil) when Config is nil or neither ref is set.
// The Config.Key field must be non-empty whenever a ref is provided.
//
// Example:
//
//	engine := c.Instance().Spec.Components["engine"]
//	config, err := c.ComponentConfig(engine)
//	if err != nil {
//	    return err
//	}
func (c *Context) ComponentConfig(component v1alpha1.ComponentSpec) (string, error) {
	cfg := component.Config
	if cfg == nil {
		return "", nil
	}

	if cfg.ConfigMapRef.Name != "" {
		cm := &corev1.ConfigMap{}
		if err := c.Get(cm, cfg.ConfigMapRef.Name); err != nil {
			return "", fmt.Errorf("get config ConfigMap %q: %w", cfg.ConfigMapRef.Name, err)
		}
		if cfg.Key == "" {
			return "", fmt.Errorf("config.key must be set when configMapRef is used")
		}
		value, ok := cm.Data[cfg.Key]
		if !ok {
			return "", fmt.Errorf("key %q not found in ConfigMap %q", cfg.Key, cfg.ConfigMapRef.Name)
		}
		return value, nil
	}

	if cfg.SecretRef.Name != "" {
		secret := &corev1.Secret{}
		if err := c.Get(secret, cfg.SecretRef.Name); err != nil {
			return "", fmt.Errorf("get config Secret %q: %w", cfg.SecretRef.Name, err)
		}
		if cfg.Key == "" {
			return "", fmt.Errorf("config.key must be set when secretRef is used")
		}
		data, ok := secret.Data[cfg.Key]
		if !ok {
			return "", fmt.Errorf("key %q not found in Secret %q", cfg.Key, cfg.SecretRef.Name)
		}
		return string(data), nil
	}

	return "", nil
}

// =============================================================================
// CONNECTION DETAILS
// =============================================================================

// ConnectionSecretSuffix is appended to the Instance name to form the
// auto-generated connection Secret name.
const ConnectionSecretSuffix = "-conn"

// +openapi:export=InstanceConnectionDetails
// ConnectionDetails holds the typed connection details for a database instance.
// These are written by the provider-runtime reconciler to a Kubernetes Secret
// and later read back by the API server to serve the connection endpoint.
// They follow the Service Binding well-known keys where applicable.
type ConnectionDetails struct {
	// Type is the type of database (e.g., mongodb, postgresql, mysql)
	// +optional
	Type string `json:"type,omitempty"`
	// Provider is the provider that manages this instance
	// +optional
	Provider string `json:"provider,omitempty"`
	// Host is the hostname or IP address to connect to
	// +optional
	Host string `json:"host,omitempty"`
	// Port is the port number to connect to
	// +optional
	Port string `json:"port,omitempty"`
	// Username is the username for authentication
	// +optional
	Username string `json:"username,omitempty"`
	// Password is the password for authentication
	// +optional
	Password string `json:"password,omitempty"`
	// URI is a pre-built connection URI
	// +optional
	URI string `json:"uri,omitempty"`
	// AdditionalProperties holds additional provider-specific connection details
	// +optional
	AdditionalProperties map[string]string `json:"-"`
}

// IsEmpty reports whether no connection details have been set.
func (cd ConnectionDetails) IsEmpty() bool {
	return cd.Type == "" && cd.Provider == "" && cd.Host == "" && cd.Port == "" &&
		cd.Username == "" && cd.Password == "" && cd.URI == "" &&
		len(cd.AdditionalProperties) == 0
}

// ToSecretData converts the typed struct to the map[string][]byte format
// required by corev1.Secret.Data. Named fields and AdditionalProperties are merged;
// named fields take precedence over any matching AdditionalProperties key.
func (cd ConnectionDetails) ToSecretData() map[string][]byte {
	data := make(map[string][]byte, 7+len(cd.AdditionalProperties))
	for k, v := range cd.AdditionalProperties {
		data[k] = []byte(v)
	}
	if cd.Type != "" {
		data["type"] = []byte(cd.Type)
	}
	if cd.Provider != "" {
		data["provider"] = []byte(cd.Provider)
	}
	if cd.Host != "" {
		data["host"] = []byte(cd.Host)
	}
	if cd.Port != "" {
		data["port"] = []byte(cd.Port)
	}
	if cd.Username != "" {
		data["username"] = []byte(cd.Username)
	}
	if cd.Password != "" {
		data["password"] = []byte(cd.Password)
	}
	if cd.URI != "" {
		data["uri"] = []byte(cd.URI)
	}
	return data
}

// =============================================================================
// STATUS TYPES
// =============================================================================

// Status represents the current state of the database cluster.
type Status struct {
	Phase             v1alpha1.InstancePhase
	Message           string
	ConnectionDetails ConnectionDetails
	Components        []ComponentStatus
}

// ComponentStatus represents the status of a single component.
type ComponentStatus struct {
	Name  string
	Ready int32
	Total int32
	State string // "Ready", "InProgress", "Error"
}

// ToV2Alpha1 converts Status to the API type.
func (s Status) ToV2Alpha1() v1alpha1.InstanceStatus {
	return v1alpha1.InstanceStatus{
		Phase: s.Phase,
	}
}

// =============================================================================
// STATUS HELPER FUNCTIONS
// =============================================================================
//
// These functions are the primary way for providers to report instance state
// from their Status() method. Each function corresponds to a phase in the
// Instance lifecycle (v1alpha1.InstancePhase).
//
// The Pending and Terminating phases are managed automatically by the
// reconciler and do not have corresponding helper functions — providers
// should not return those phases directly.

// Pending returns a status indicating the instance has been accepted but
// provisioning has not yet started (e.g., waiting on prerequisites).
func Pending(message string) Status {
	return Status{Phase: v1alpha1.InstancePhasePending, Message: message}
}

// Provisioning returns a status indicating the operator is actively creating
// the underlying infrastructure (StatefulSets, PVCs, Services, etc.).
func Provisioning(message string) Status {
	return Status{Phase: v1alpha1.InstancePhaseProvisioning, Message: message}
}

// Initializing returns a status indicating the infrastructure exists and the
// instance engine is booting (bootstrap scripts, initial quorum, etc.).
func Initializing(message string) Status {
	return Status{Phase: v1alpha1.InstancePhaseInitializing, Message: message}
}

// Ready returns a status indicating the instance is fully operational and
// accepting client connections. Use ReadyWithConnectionDetails when
// connection information is available.
func Ready() Status {
	return Status{Phase: v1alpha1.InstancePhaseReady}
}

// ReadyWithConnectionDetails returns a ready status with connection details.
// The reconciler writes these details to an auto-generated Secret.
//
// Providers should populate the well-known fields so the API server can expose
// them generically without any provider-specific logic.
//
// Example:
//
//	return controller.ReadyWithConnectionDetails(controller.ConnectionDetails{
//		Type:     "mongodb",
//		Provider: "percona-server-mongodb",
//		Host:     host,
//		Port:     "27017",
//		Username: user,
//		Password: pass,
//		URI:      uri,
//	})
func ReadyWithConnectionDetails(details ConnectionDetails) Status {
	return Status{
		Phase:             v1alpha1.InstancePhaseReady,
		ConnectionDetails: details,
	}
}

// Updating returns a status indicating a mutation is being rolled out
// (scaling, config change, version upgrade, etc.).
func Updating(message string) Status {
	return Status{Phase: v1alpha1.InstancePhaseUpdating, Message: message}
}

// Failed returns a status indicating the instance has encountered a terminal
// or semi-terminal error requiring human intervention.
func Failed(message string) Status {
	return Status{Phase: v1alpha1.InstancePhaseFailed, Message: message}
}

// =============================================================================
// STATUS HELPER FUNCTIONS — Data Recovery
// =============================================================================

// Restoring returns a status indicating the instance is downloading and
// unpacking data from an external backup source.
func Restoring(message string) Status {
	return Status{Phase: v1alpha1.InstancePhaseRestoring, Message: message}
}

// =============================================================================
// STATUS HELPER FUNCTIONS — Cost-Saving (Compute-to-Zero)
// =============================================================================

// Suspending returns a status indicating the instance engine is gracefully
// shutting down and preparing to scale compute to zero.
func Suspending(message string) Status {
	return Status{Phase: v1alpha1.InstancePhaseSuspending, Message: message}
}

// Suspended returns a status indicating the instance compute is scaled to zero.
// Storage (PVCs) remains intact.
func Suspended() Status {
	return Status{Phase: v1alpha1.InstancePhaseSuspended}
}

// Resuming returns a status indicating the instance is scaling compute back up
// and reattaching existing storage.
func Resuming(message string) Status {
	return Status{Phase: v1alpha1.InstancePhaseResuming, Message: message}
}

// =============================================================================
// WAIT HELPERS
// =============================================================================

// WaitError signals that a step is waiting for something.
type WaitError struct {
	Reason   string
	Duration time.Duration
}

// =============================================================================
// BACKUP CONFIG ERROR
// =============================================================================

// BackupConfigError is returned from Sync when backup configuration (storage
// resolution, schedule translation, PITR wiring) fails but the engine itself
// is otherwise healthy. The runtime surfaces it on the BackupConfigured
// condition instead of marking the Instance as Failed, so operators can see
// that the database is running but backups need attention.
//
// Usage inside Sync:
//
//	if err := buildBackupSpec(c); err != nil {
//	    return &controller.BackupConfigError{Reason: "StorageNotFound", Message: err.Error()}
//	}
type BackupConfigError struct {
	// Reason is a short CamelCase identifier (used as the condition reason).
	Reason string
	// Message is a human-readable description of the problem.
	Message string
}

func (e *BackupConfigError) Error() string {
	return e.Message
}

// AsBackupConfigError extracts a *BackupConfigError from err (including wrapped
// errors), returning nil if err is not a BackupConfigError.
func AsBackupConfigError(err error) *BackupConfigError {
	var bce *BackupConfigError
	if errors.As(err, &bce) {
		return bce
	}
	return nil
}

// DataSourceConfigError is returned from Sync when the initial-seeding
// preconditions cannot be met (e.g., the source Instance's credentials secret
// has been deleted). The runtime surfaces it on the DataSourceReady condition
// without marking the Instance as Failed, so operators can see the database
// is still running but seeding requires manual intervention.
//
// Usage inside Sync:
//
//	if err := ensureDataSourceCredentials(c, secretName); err != nil {
//	    return &controller.DataSourceConfigError{Reason: "SourceSecretNotFound", Message: err.Error()}
//	}
type DataSourceConfigError struct {
	// Reason is a short CamelCase identifier (used as the condition reason).
	Reason string
	// Message is a human-readable description of the problem.
	Message string
}

func (e *DataSourceConfigError) Error() string {
	return e.Message
}

// AsDataSourceConfigError extracts a *DataSourceConfigError from err
// (including wrapped errors), returning nil if err is not one.
func AsDataSourceConfigError(err error) *DataSourceConfigError {
	var dse *DataSourceConfigError
	if errors.As(err, &dse) {
		return dse
	}
	return nil
}

func (e *WaitError) Error() string {
	return fmt.Sprintf("waiting: %s", e.Reason)
}

// IsWaitError checks if an error is a WaitError.
func IsWaitError(err error) bool {
	_, ok := err.(*WaitError)
	return ok
}

// GetWaitDuration returns the wait duration from a WaitError.
func GetWaitDuration(err error) time.Duration {
	if we, ok := err.(*WaitError); ok {
		return we.Duration
	}
	return 10 * time.Second
}

// WaitFor returns an error indicating the step should be retried.
func WaitFor(reason string) error {
	return &WaitError{Reason: reason, Duration: 10 * time.Second}
}

// WaitForDuration returns an error indicating retry after a specific duration.
func WaitForDuration(reason string, d time.Duration) error {
	return &WaitError{Reason: reason, Duration: d}
}

// =============================================================================
// BACKUP / RESTORE HELPERS
// =============================================================================

// BackupClass fetches a BackupClass by name (cluster-scoped).
func (c *Context) BackupClass(name string) (*backupv1alpha1.BackupClass, error) {
	bc := &backupv1alpha1.BackupClass{}
	if err := c.client.Get(c.ctx, client.ObjectKey{Name: name}, bc); err != nil {
		return nil, fmt.Errorf("failed to get BackupClass %q: %w", name, err)
	}
	return bc, nil
}

// BackupClassForInstance fetches the BackupClass referenced by
// Instance.spec.backup.classRef, if any. Returns (nil, nil) when the Instance
// has no backup configuration.
func (c *Context) BackupClassForInstance() (*backupv1alpha1.BackupClass, error) {
	if c.in == nil || c.in.Spec.Backup == nil || c.in.Spec.Backup.ClassRef.Name == "" {
		return nil, nil
	}
	return c.BackupClass(c.in.Spec.Backup.ClassRef.Name)
}

// BackupClassLimits returns the limits declared by the BackupClass referenced
// by the Instance, or nil when the class is Job-mode, has no limits set, or
// the Instance has no backup configuration. The returned pointer aliases the
// BackupClass; callers must not mutate it.
func (c *Context) BackupClassLimits() (*backupv1alpha1.BackupClassLimits, error) {
	bc, err := c.BackupClassForInstance()
	if err != nil || bc == nil {
		return nil, err
	}
	if bc.Spec.ProviderManaged == nil {
		return nil, nil
	}
	return bc.Spec.ProviderManaged.Limits, nil
}

// PITRConfigSchema returns the free-form PITR config schema declared by the
// BackupClass referenced by the Instance, or nil when no schema is set.
// The runtime treats this payload as opaque; providers interpret it.
func (c *Context) PITRConfigSchema() (*runtime.RawExtension, error) {
	bc, err := c.BackupClassForInstance()
	if err != nil || bc == nil {
		return nil, err
	}
	if bc.Spec.ProviderManaged == nil {
		return nil, nil
	}
	return bc.Spec.ProviderManaged.PITRConfigSchema, nil
}

// BackupStorage fetches a BackupStorage by name from the instance namespace.
func (c *Context) BackupStorage(name string) (*backupv1alpha1.BackupStorage, error) {
	bs := &backupv1alpha1.BackupStorage{}
	if err := c.client.Get(c.ctx, client.ObjectKey{Namespace: c.in.Namespace, Name: name}, bs); err != nil {
		return nil, fmt.Errorf("failed to get BackupStorage %q: %w", name, err)
	}
	return bs, nil
}

// BackupStorageCredentials reads the credentials Secret referenced by an S3
// BackupStorage and returns the access key id / secret access key. Returns
// empty strings if the storage does not reference a Secret (caller can decide
// whether that is an error).
func (c *Context) BackupStorageCredentials(bs *backupv1alpha1.BackupStorage) (accessKeyID, secretAccessKey string, err error) {
	if bs == nil || bs.Spec.S3 == nil || bs.Spec.S3.CredentialsSecretName == "" {
		return "", "", nil
	}
	secret := &corev1.Secret{}
	if err := c.client.Get(c.ctx, client.ObjectKey{
		Namespace: bs.GetNamespace(),
		Name:      bs.Spec.S3.CredentialsSecretName,
	}, secret); err != nil {
		return "", "", fmt.Errorf("failed to get credentials secret %q: %w", bs.Spec.S3.CredentialsSecretName, err)
	}
	return string(secret.Data["AWS_ACCESS_KEY_ID"]), string(secret.Data["AWS_SECRET_ACCESS_KEY"]), nil
}

// BackupsForInstance lists all Backup CRs in the instance namespace whose
// .spec.instanceName matches this Instance. Requires the field index
// ".spec.instanceName" on backupv1alpha1.Backup, which the runtime registers
// automatically when the provider implements BackupProvider.
func (c *Context) BackupsForInstance() ([]backupv1alpha1.Backup, error) {
	list := &backupv1alpha1.BackupList{}
	if err := c.client.List(c.ctx, list,
		client.InNamespace(c.in.Namespace),
		client.MatchingFields{IndexBackupInstanceName: c.in.Name},
	); err != nil {
		return nil, fmt.Errorf("failed to list backups for instance: %w", err)
	}
	return list.Items, nil
}

// RestoresForInstance lists all Restore CRs in the instance namespace whose
// .spec.instanceName matches this Instance.
func (c *Context) RestoresForInstance() ([]backupv1alpha1.Restore, error) {
	list := &backupv1alpha1.RestoreList{}
	if err := c.client.List(c.ctx, list,
		client.InNamespace(c.in.Namespace),
		client.MatchingFields{IndexRestoreInstanceName: c.in.Name},
	); err != nil {
		return nil, fmt.Errorf("failed to list restores for instance: %w", err)
	}
	return list.Items, nil
}

// IndexBackupInstanceName is the field index path used for Backup.spec.instanceName.
const IndexBackupInstanceName = "spec.instanceName"

// IndexRestoreInstanceName is the field index path used for Restore.spec.instanceName.
const IndexRestoreInstanceName = "spec.instanceName"

// ShouldRetainBackupData returns true when the underlying backup data in the
// configured BackupStorage (e.g., the S3 object) must be preserved on
// deletion of the supplied Backup CR.
//
// Providers should call this from CleanupBackup to decide whether to invoke
// the engine's data-purge path or to merely strip protection finalizers and
// delete the operator-native backup CR. Centralizing the decision in the
// runtime lets us layer in additional inputs (e.g., BackupClass-level
// defaults) later without changing provider code.
func (c *Context) ShouldRetainBackupData(b *backupv1alpha1.Backup) bool {
	if b == nil {
		return false
	}
	return b.Spec.DeletionPolicy == backupv1alpha1.BackupDeletionPolicyRetain
}

// =============================================================================
// BACKUP / RESTORE EXECUTION STATUS
// =============================================================================

// BackupExecutionStatus is returned by BackupProvider.SyncBackup. The runtime
// reflects it onto the Backup CR's .status.
type BackupExecutionStatus struct {
	// State is the current state of the backup (Pending/Running/Succeeded/Failed/Error).
	State backupv1alpha1.BackupState
	// Message is a human-readable description of the current state.
	Message string
	// OperatorBackupRef points at the operator-native backup resource that was
	// created (e.g., PerconaServerMongoDBBackup). Optional but recommended.
	OperatorBackupRef *corev1.TypedLocalObjectReference
	// StartedAt is when the backup started running. Optional.
	StartedAt *metav1.Time
	// CompletedAt is when the backup completed. Optional.
	CompletedAt *metav1.Time
	// Size is the size of the backup data as reported by the engine. Optional.
	Size *string
}

// RestoreExecutionStatus is returned by BackupProvider.SyncRestore. The runtime
// reflects it onto the Restore CR's .status.
type RestoreExecutionStatus struct {
	State              backupv1alpha1.RestoreState
	Message            string
	OperatorRestoreRef *corev1.TypedLocalObjectReference
	StartedAt          *metav1.Time
	CompletedAt        *metav1.Time
}

// IsNotFound reports whether the error is a kubernetes "not found" error.
// Convenience wrapper so providers don't have to import apimachinery directly.
func IsNotFound(err error) bool {
	return apierrors.IsNotFound(err)
}

// =============================================================================
// DATASOURCE HELPER — initial seeding from .spec.dataSource
// =============================================================================

// DataSourceRestoreNameSuffix is appended to the Instance name to form the
// fixed Restore CR name used for initial seeding. Keeping it deterministic
// makes the resource easy to find and prevents duplicate Restore CRs from
// being created across reconciles.
const DataSourceRestoreNameSuffix = "-datasource"

// DataSourceState classifies the progress of an initial-seeding restore.
type DataSourceState string

const (
	// DataSourceStateNone means the Instance has no .spec.dataSource.
	DataSourceStateNone DataSourceState = ""
	// DataSourceStateWaiting means the helper has not yet created the Restore
	// because a precondition is not satisfied (e.g., source Backup missing or
	// not Succeeded, storage mismatch, BackupClass unsupported). Providers
	// typically treat this as "keep the engine in Restoring phase and let the
	// next reconcile re-evaluate".
	DataSourceStateWaiting DataSourceState = "Waiting"
	// DataSourceStateRestoring means the Restore CR exists and the
	// operator-native restore is in flight.
	DataSourceStateRestoring DataSourceState = "Restoring"
	// DataSourceStateSucceeded means the Restore reached Succeeded; the
	// Instance has been seeded.
	DataSourceStateSucceeded DataSourceState = "Succeeded"
	// DataSourceStateFailed means the Restore reached a terminal failure.
	DataSourceStateFailed DataSourceState = "Failed"
)

// DataSourceStatus is returned by Context.ReconcileDataSource and staged on
// the Context so the reconciler can flush the corresponding
// ConditionDataSourceReady onto the Instance. Providers should branch on Done
// to decide whether the engine is free to expose connection details: while
// Done is false the engine is still being seeded and should be reported as
// Restoring.
type DataSourceStatus struct {
	// Done is true when the Instance has no DataSource or when the initial
	// restore has reached a terminal state (Succeeded or Failed).
	Done bool
	// State is the current high-level state of the seeding restore.
	State DataSourceState
	// Reason is the condition reason that the reconciler should publish on
	// ConditionDataSourceReady (one of the v1alpha1.ReasonDataSource* values).
	Reason string
	// Message is a human-readable explanation suitable for the condition
	// message and for surfacing in `kubectl describe`.
	Message string
	// RestoreName is the name of the Restore CR the helper created, when one
	// was created. Empty when the helper short-circuited on a precondition.
	RestoreName string
}

// SetDataSourceStatus stages a DataSourceStatus on the Context. The runtime
// reconciler reads it after Sync completes and reflects the result onto the
// ConditionDataSourceReady condition. Providers do not normally call this
// directly — ReconcileDataSource does it for them.
func (c *Context) SetDataSourceStatus(s DataSourceStatus) {
	c.dataSourceStatus = &s
}

// GetDataSourceStatus returns the most recently staged DataSourceStatus, or
// nil when none has been set. Used by the runtime reconciler.
func (c *Context) GetDataSourceStatus() *DataSourceStatus {
	return c.dataSourceStatus
}

// ReconcileDataSource implements the initial-seeding flow for
// .spec.dataSource. It is safe to call unconditionally from Sync: when the
// Instance has no DataSource, the helper returns Done=true immediately
// without creating any resources.
//
// Behaviour when DataSource is set:
//  1. Validate the source Backup (exists, Succeeded, BackupClass is
//     ProviderManaged and supports the target provider, and the target
//     Instance has a backup storage entry that matches the source Backup's
//     storage).
//  2. Create or update a Restore CR named "{instance}-datasource", owned by
//     the Instance, with .spec.instanceName pointing at the target Instance
//     and .spec.dataSource.backup.backupName pointing at the source Backup.
//  3. Read back .status.state on the Restore and translate it into a
//     DataSourceStatus. The returned status is also staged on the Context so
//     the runtime reconciler can flush ConditionDataSourceReady.
//
// Done=true means the Instance no longer needs to be held in the Restoring
// phase — either no DataSource is configured, or the restore reached a
// terminal state (Succeeded or Failed). Providers should report
// InstancePhaseRestoring while Done is false.
func (c *Context) ReconcileDataSource() (DataSourceStatus, error) {
	ds := c.in.Spec.DataSource
	if ds == nil {
		s := DataSourceStatus{Done: true, State: DataSourceStateNone}
		c.SetDataSourceStatus(s)
		return s, nil
	}

	// 1. Source Backup must exist and be Succeeded.
	backupName := ds.Backup.BackupName
	src := &backupv1alpha1.Backup{}
	if err := c.Get(src, backupName); err != nil {
		if apierrors.IsNotFound(err) {
			s := DataSourceStatus{
				Done:    false,
				State:   DataSourceStateWaiting,
				Reason:  v1alpha1.ReasonDataSourceSourceBackupNotFound,
				Message: fmt.Sprintf("source Backup %q not found in namespace %q", backupName, c.in.Namespace),
			}
			c.SetDataSourceStatus(s)
			return s, nil
		}
		return DataSourceStatus{}, fmt.Errorf("get source Backup %q: %w", backupName, err)
	}
	if src.Status.State != backupv1alpha1.BackupStateSucceeded {
		s := DataSourceStatus{
			Done:    false,
			State:   DataSourceStateWaiting,
			Reason:  v1alpha1.ReasonDataSourceSourceBackupNotSucceeded,
			Message: fmt.Sprintf("source Backup %q is in state %q, waiting for Succeeded", backupName, src.Status.State),
		}
		c.SetDataSourceStatus(s)
		return s, nil
	}

	// 2. The source Backup's BackupClass must be ProviderManaged and support
	// this Instance's provider.
	bc, err := c.BackupClass(src.Spec.BackupClassName)
	if err != nil {
		if apierrors.IsNotFound(err) {
			s := DataSourceStatus{
				Done:    false,
				State:   DataSourceStateWaiting,
				Reason:  v1alpha1.ReasonDataSourceClassUnsupported,
				Message: fmt.Sprintf("BackupClass %q referenced by source Backup not found", src.Spec.BackupClassName),
			}
			c.SetDataSourceStatus(s)
			return s, nil
		}
		return DataSourceStatus{}, fmt.Errorf("get BackupClass %q: %w", src.Spec.BackupClassName, err)
	}
	if bc.Spec.ExecutionMode != backupv1alpha1.BackupExecutionModeProviderManaged {
		s := DataSourceStatus{
			Done:    true,
			State:   DataSourceStateFailed,
			Reason:  v1alpha1.ReasonDataSourceClassUnsupported,
			Message: fmt.Sprintf("BackupClass %q is %q; only ProviderManaged classes are supported for spec.dataSource", bc.Name, bc.Spec.ExecutionMode),
		}
		c.SetDataSourceStatus(s)
		return s, nil
	}
	if !bc.Spec.SupportedProviders.Has(c.providerName) {
		s := DataSourceStatus{
			Done:    true,
			State:   DataSourceStateFailed,
			Reason:  v1alpha1.ReasonDataSourceClassUnsupported,
			Message: fmt.Sprintf("BackupClass %q does not list provider %q in SupportedProviders", bc.Name, c.providerName),
		}
		c.SetDataSourceStatus(s)
		return s, nil
	}

	// 3. Target Instance must declare a backup storage with the same logical
	// name as the source Backup so the provider can read the data.
	if c.in.Spec.Backup == nil || !hasInstanceStorage(c.in.Spec.Backup, src.Spec.StorageName) {
		s := DataSourceStatus{
			Done:    false,
			State:   DataSourceStateWaiting,
			Reason:  v1alpha1.ReasonDataSourceStorageMismatch,
			Message: fmt.Sprintf("Instance.spec.backup.storages does not include storage %q used by source Backup %q", src.Spec.StorageName, src.Name),
		}
		c.SetDataSourceStatus(s)
		return s, nil
	}

	// 4. Create or update the Restore CR.
	restoreName := c.in.Name + DataSourceRestoreNameSuffix
	restore := &backupv1alpha1.Restore{
		ObjectMeta: metav1.ObjectMeta{
			Name:      restoreName,
			Namespace: c.in.Namespace,
		},
	}
	if _, err := controllerutil.CreateOrUpdate(c.ctx, c.client, restore, func() error {
		if restore.Labels == nil {
			restore.Labels = map[string]string{}
		}
		restore.Labels["app.kubernetes.io/managed-by"] = "everest"
		restore.Labels["app.kubernetes.io/instance"] = c.in.Name
		restore.Spec.InstanceName = c.in.Name
		restore.Spec.DataSource = *ds
		return controllerutil.SetControllerReference(c.in, restore, c.client.Scheme())
	}); err != nil {
		return DataSourceStatus{}, fmt.Errorf("create or update seeding Restore %q: %w", restoreName, err)
	}

	// 5. Translate Restore status into DataSourceStatus.
	var s DataSourceStatus
	s.RestoreName = restoreName
	switch restore.Status.State {
	case backupv1alpha1.RestoreStateSucceeded:
		s.Done = true
		s.State = DataSourceStateSucceeded
		s.Reason = v1alpha1.ReasonDataSourceSucceeded
		s.Message = fmt.Sprintf("Instance seeded from Backup %q via Restore %q", backupName, restoreName)
	case backupv1alpha1.RestoreStateFailed:
		s.Done = true
		s.State = DataSourceStateFailed
		s.Reason = v1alpha1.ReasonDataSourceFailed
		s.Message = fmt.Sprintf("Restore %q failed: %s", restoreName, restore.Status.Message)
	default:
		s.Done = false
		s.State = DataSourceStateRestoring
		s.Reason = v1alpha1.ReasonDataSourceRestoring
		if restore.Status.Message != "" {
			s.Message = restore.Status.Message
		} else {
			s.Message = fmt.Sprintf("Restore %q in progress (state=%q)", restoreName, restore.Status.State)
		}
	}
	c.SetDataSourceStatus(s)
	return s, nil
}

// hasInstanceStorage reports whether the InstanceBackupSpec declares a storage
// entry whose logical name matches the supplied name.
func hasInstanceStorage(b *v1alpha1.InstanceBackupSpec, name string) bool {
	if b == nil {
		return false
	}
	for _, s := range b.Storages {
		if s.Name == name {
			return true
		}
	}
	return false
}
