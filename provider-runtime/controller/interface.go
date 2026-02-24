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

// Implement the Provider interface to create a provider.
// Embed BaseProvider for default implementations.

import (
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// ProviderInterface defines the interface for a database provider.
type ProviderInterface interface {
	// Name returns the unique identifier for this provider (e.g., "psmdb", "postgresql").
	Name() string

	// Types returns the scheme builder for registering provider-specific CRDs.
	Types() func(*runtime.Scheme) error

	// Validate checks if the Instance spec is valid.
	Validate(c *Context) error

	// Sync ensures all required resources exist and are configured.
	Sync(c *Context) error

	// Status computes the current status of the database.
	Status(c *Context) (Status, error)

	// Cleanup handles deletion (called when deletion timestamp is set).
	Cleanup(c *Context) error
}

// =============================================================================
// WATCH CONFIGURATION (Optional interface for advanced watch configuration)
// =============================================================================

// WatchConfig defines configuration for watching a Kubernetes resource type.
// It provides control over which resources trigger reconciliation and how events are handled.
type WatchConfig struct {
	// Object is the resource type to watch (e.g., &corev1.Secret{}, &v2alpha1.BackupJob{})
	Object client.Object

	// Owned indicates if this resource is owned by the DataStore (has owner reference).
	// If true, uses Owns() which sets up automatic cleanup and owner-reference-based filtering.
	// If false, uses Watches() which requires a custom Handler.
	Owned bool

	// Handler defines how events on this resource map to DataStore reconciliation requests.
	// Only used when Owned=false. Common patterns:
	//   - handler.EnqueueRequestForObject{} - reconcile the object itself
	//   - handler.EnqueueRequestsFromMapFunc - custom mapping logic
	// If nil and Owned=false, uses EnqueueRequestForObject (not typically useful for external resources).
	Handler handler.EventHandler

	// Predicates filter which events trigger reconciliation.
	// Common predicates:
	//   - predicate.GenerationChangedPredicate{} - only spec changes
	//   - predicate.ResourceVersionChangedPredicate{} - any changes including status
	//   - Custom predicates for fine-grained control
	// If empty, all events trigger reconciliation.
	Predicates []predicate.Predicate

	// WatchOptions provides additional watch configuration (rarely needed).
	// Can be used for things like OnlyMetadata watches for performance.
	WatchOptions []builder.WatchesOption
}

// WatchProvider is an optional interface that providers can implement
// to configure watched resources.
//
// This enables:
//   - Watching owned resources with predicates
//   - Watching external resources (e.g., BackupStorage, Secrets)
//   - Custom event handlers for mapping resource changes to DataStore reconciliations
//   - Fine-grained predicate control for filtering events
//
// Example implementation:
//
//	func (p *PSMDBProvider) Watches() []controller.WatchConfig {
//	    return []controller.WatchConfig{
//	        // Watch owned PSMDB resources with generation-changed predicate
//	        controller.WatchOwned(&psmdbv1.PerconaServerMongoDB{},
//	            predicate.GenerationChangedPredicate{}),
//
//	        // Watch BackupJob resources and trigger reconciliation for referenced DataStores
//	        controller.WatchExternal(&v2alpha1.BackupJob{},
//	            handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
//	                backupJob := obj.(*v2alpha1.BackupJob)
//	                if backupJob.Spec.SourceClusterName == nil {
//	                    return nil
//	                }
//	                return []reconcile.Request{{
//	                    NamespacedName: types.NamespacedName{
//	                        Name:      *backupJob.Spec.SourceClusterName,
//	                        Namespace: obj.GetNamespace(),
//	                    },
//	                }}
//	            }),
//	            predicate.ResourceVersionChangedPredicate{}),
//	    }
//	}
type WatchProvider interface {
	// Watches returns the list of watch configurations for this provider.
	// This replaces OwnedTypes() when implemented - the controller will not call OwnedTypes().
	Watches() []WatchConfig
}

// WatchOwned creates a WatchConfig for an owned resource type.
// Owned resources have owner references set by the provider and are automatically cleaned up
// when the DataStore is deleted.
//
// Example:
//
//	WatchOwned(&psmdbv1.PerconaServerMongoDB{}, predicate.GenerationChangedPredicate{})
func WatchOwned(obj client.Object, predicates ...predicate.Predicate) WatchConfig {
	return WatchConfig{
		Object:     obj,
		Owned:      true,
		Predicates: predicates,
	}
}

// WatchExternal creates a WatchConfig for an external (non-owned) resource type.
// External resources require a custom handler to map events to DataStore reconciliations.
//
// Example:
//
//	WatchExternal(&corev1.Secret{},
//	    handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, obj client.Object) []reconcile.Request {
//	        // Map secret changes to DataStore reconciliations
//	        return []reconcile.Request{...}
//	    }),
//	    predicate.GenerationChangedPredicate{})
func WatchExternal(obj client.Object, handler handler.EventHandler, predicates ...predicate.Predicate) WatchConfig {
	return WatchConfig{
		Object:     obj,
		Owned:      false,
		Handler:    handler,
		Predicates: predicates,
	}
}

// WatchWithHandler creates a WatchConfig with full control over the handler and predicates.
// Use this for advanced scenarios where you need complete control.
func WatchWithHandler(obj client.Object, owned bool, handler handler.EventHandler, predicates ...predicate.Predicate) WatchConfig {
	return WatchConfig{
		Object:     obj,
		Owned:      owned,
		Handler:    handler,
		Predicates: predicates,
	}
}

// Re-exported predicate types for convenience
var (
	// GenerationChangedPredicate only triggers on spec changes (not status updates)
	GenerationChangedPredicate = predicate.GenerationChangedPredicate{}

	// ResourceVersionChangedPredicate triggers on any changes including status
	ResourceVersionChangedPredicate = predicate.ResourceVersionChangedPredicate{}
)

// =============================================================================
// FIELD INDEX CONFIGURATION (Optional interface for efficient querying)
// =============================================================================

// FieldIndex defines a field index for efficient querying of Kubernetes resources.
// Field indexes allow you to quickly find objects based on specific field values,
// which is necessary when using FieldSelector in List operations.
type FieldIndex struct {
	// Object is the resource type to index (e.g., &v2alpha1.BackupJob{})
	Object client.Object

	// FieldPath is the field to index using dot notation.
	// Examples: ".spec.sourceClusterName", ".spec.backupStorageName"
	FieldPath string

	// Extractor returns the values to index for a given object.
	// It should return nil or empty slice if the field is not set.
	// Example:
	//   func(obj client.Object) []string {
	//       if bj, ok := obj.(*v2alpha1.BackupJob); ok && bj.Spec.SourceClusterName != "" {
	//           return []string{bj.Spec.SourceClusterName}
	//       }
	//       return nil
	//   }
	Extractor func(client.Object) []string
}

// FieldIndexProvider is an optional interface that providers can implement
// to declare field indexes for efficient querying of related resources.
//
// Field indexes are required when you need to find objects based on field values
// using FieldSelector in List operations. This is commonly needed when watching
// external resources and querying for objects that reference the DataStore.
//
// Example implementation:
//
//	func (p *PSMDBProvider) FieldIndexes() []controller.FieldIndex {
//	    return []controller.FieldIndex{
//	        {
//	            Object:    &v2alpha1.BackupJob{},
//	            FieldPath: ".spec.sourceClusterName",
//	            Extractor: func(obj client.Object) []string {
//	                if bj, ok := obj.(*v2alpha1.BackupJob); ok && bj.Spec.SourceClusterName != "" {
//	                    return []string{bj.Spec.SourceClusterName}
//	                }
//	                return nil
//	            },
//	        },
//	    }
//	}
type FieldIndexProvider interface {
	// FieldIndexes returns the list of field indexes to create for this provider.
	FieldIndexes() []FieldIndex
}

// =============================================================================
// BASE PROVIDER
// =============================================================================

// BaseProvider provides default implementations for common Provider methods.
// Embed this in your provider struct to inherit defaults.
type BaseProvider struct {
	ProviderName string
	SchemeFuncs  []func(*runtime.Scheme) error
	WatchConfigs []WatchConfig
}

func (b *BaseProvider) Name() string {
	return b.ProviderName
}

func (b *BaseProvider) Types() func(*runtime.Scheme) error {
	if len(b.SchemeFuncs) == 0 {
		return nil
	}
	return func(s *runtime.Scheme) error {
		for _, fn := range b.SchemeFuncs {
			if err := fn(s); err != nil {
				return err
			}
		}
		return nil
	}
}

// Watches returns the configured watch configurations.
func (b *BaseProvider) Watches() []WatchConfig {
	return b.WatchConfigs
}

// =============================================================================
// SCHEMA PROVIDER (Optional interface for HTTP server)
// =============================================================================

// TopologyComponentDefinition defines a component within a topology with its metadata.
type TopologyComponentDefinition struct {
	// Optional indicates if this component is optional in the topology.
	// If false, the component is required.
	Optional bool

	// Defaults provides default values for this component in this topology.
	Defaults map[string]interface{}
}
