package controller

// Provider SDK
//
// Implement the Provider interface to create a provider.
// Embed BaseProvider for default implementations.
//
// See examples/psmdb for a complete example.

import (
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ProviderInterface defines the interface for a database provider.
type ProviderInterface interface {
	// Name returns the unique identifier for this provider (e.g., "psmdb", "postgresql").
	Name() string

	// Types returns the scheme builder for registering provider-specific CRDs.
	Types() func(*runtime.Scheme) error

	// OwnedTypes returns types this provider creates (triggers reconciliation on changes).
	OwnedTypes() []client.Object

	// Validate checks if the Workload spec is valid.
	Validate(c *Context) error

	// Sync ensures all required resources exist and are configured.
	Sync(c *Context) error

	// Status computes the current status of the database.
	Status(c *Context) (Status, error)

	// Cleanup handles deletion (called when deletion timestamp is set).
	Cleanup(c *Context) error
}

// MetadataProvider is an optional interface for exposing provider metadata.
// Providers that embed BaseProvider with Metadata set automatically satisfy this.
type MetadataProvider interface {
	GetMetadata() *ProviderMetadata
}

// BaseProvider provides default implementations for common Provider methods.
// Embed this in your provider struct to inherit defaults.
type BaseProvider struct {
	ProviderName string
	SchemeFuncs  []func(*runtime.Scheme) error
	Owned        []client.Object
	Metadata     *ProviderMetadata
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

func (b *BaseProvider) OwnedTypes() []client.Object {
	return b.Owned
}

// GetMetadata returns the provider metadata.
// Returns nil if no metadata is configured.
func (b *BaseProvider) GetMetadata() *ProviderMetadata {
	return b.Metadata
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
	// Example: map[string]interface{}{"replicas": 3}
	Defaults map[string]interface{}
}

// TopologyDefinition combines a topology's configuration schema with its supported components.
type TopologyDefinition struct {
	// Schema is the Go type that defines the topology-specific configuration.
	// This will be converted to an OpenAPI schema.
	Schema interface{}

	// Components maps component names to their definitions within this topology.
	// Example: map[string]TopologyComponentDefinition{
	//   "engine":       {Optional: false, Defaults: map[string]interface{}{"replicas": 3}},
	//   "backupAgent":  {Optional: true},
	//   "monitoring":   {Optional: true},
	// }
	Components map[string]TopologyComponentDefinition
}

// SchemaProvider is an optional interface that providers can implement
// to expose OpenAPI schemas for their components, topologies, and global config.
// This enables the HTTP server to serve schema information for documentation
// and client-side validation.
//
// Example implementation:
//
//	func (p *PSMDBProvider) ComponentSchemas() map[string]interface{} {
//	    return map[string]interface{}{
//	        "engine":       &MongodCustomSpec{},
//	        "configServer": &MongodCustomSpec{},
//	        "proxy":        &MongosCustomSpec{},
//	    }
//	}
//
//	func (p *PSMDBProvider) Topologies() map[string]TopologyDefinition {
//	    return map[string]TopologyDefinition{
//	        "replicaset": {
//	            Schema:     &ReplicaSetTopologyConfig{},
//	            Components: map[string]TopologyComponentDefinition{
//	                "engine":      {Optional: false, Defaults: map[string]interface{}{"replicas": 3}},
//	                "backupAgent": {Optional: true},
//	                "monitoring":  {Optional: true},
//	            },
//	        },
//	        "sharded": {
//	            Schema:     &ShardedTopologyConfig{},
//	            Components: map[string]TopologyComponentDefinition{
//	                "engine":       {Optional: false},
//	                "proxy":        {Optional: false},
//	                "configServer": {Optional: false},
//	                "backupAgent":  {Optional: true},
//	                "monitoring":   {Optional: true},
//	            },
//	        },
//	    }
//	}
//
//	func (p *PSMDBProvider) GlobalSchema() interface{} {
//	    return &GlobalConfig{}
//	}
type SchemaProvider interface {
	// ComponentSchemas returns a map of component names to their custom spec types.
	// The Go types are converted to OpenAPI v3 JSON schemas.
	ComponentSchemas() map[string]interface{}

	// Topologies returns a map of topology names to their definitions.
	// Each definition includes both the configuration schema and the list of supported components.
	Topologies() map[string]TopologyDefinition

	// GlobalSchema returns the type for global configuration.
	// Returns nil if no global schema is needed.
	GlobalSchema() interface{}
}
