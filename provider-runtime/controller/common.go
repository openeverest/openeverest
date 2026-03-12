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
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

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

// Status helper functions

// Creating returns a status indicating the cluster is being created.
func Creating(message string) Status {
	return Status{Phase: v1alpha1.InstancePhaseCreating, Message: message}
}

// Running returns a status indicating the cluster is running
// without connection details. Use RunningWithConnectionDetails
// when connection information is available.
func Running() Status {
	return Status{Phase: v1alpha1.InstancePhaseRunning}
}

// RunningWithConnectionDetails returns a running status with connection details.
// The reconciler writes these details to an auto-generated Secret.
//
// Providers should populate the well-known fields so the API server can expose
// them generically without any provider-specific logic.
//
// Example:
//
//	return controller.RunningWithConnectionDetails(controller.ConnectionDetails{
//		Type:     "mongodb",
//		Provider: "percona-server-mongodb",
//		Host:     host,
//		Port:     "27017",
//		Username: user,
//		Password: pass,
//		URI:      uri,
//	})
func RunningWithConnectionDetails(details ConnectionDetails) Status {
	return Status{
		Phase:             v1alpha1.InstancePhaseRunning,
		ConnectionDetails: details,
	}
}

// Failed returns a status indicating the cluster has failed.
func Failed(message string) Status {
	return Status{Phase: v1alpha1.InstancePhaseFailed, Message: message}
}

// =============================================================================
// WAIT HELPERS
// =============================================================================

// WaitError signals that a step is waiting for something.
type WaitError struct {
	Reason   string
	Duration time.Duration
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
