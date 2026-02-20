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

package openapi

import (
	"encoding/json"
	"sync"

	"k8s.io/kube-openapi/pkg/common"
	"k8s.io/kube-openapi/pkg/validation/spec"
)

// SchemaRegistry holds registered schemas for components, topologies, and global config.
// It uses kube-openapi for pre-generated OpenAPI schema serving.
//
// There are two ways to use SchemaRegistry:
//
// 1. With pre-generated definitions (recommended for production):
//
//	defs := generated.GetOpenAPIDefinitions(openapi.DefaultReferenceCallback)
//	registry := openapi.NewSchemaRegistryFromDefinitions(defs)
//	registry.MapComponent("mongod", "github.com/myorg/provider/types.MongodCustomSpec")
//
// 2. With direct schema registration (for testing or dynamic schemas):
//
//	registry := openapi.NewSchemaRegistry()
//	registry.RegisterComponentSchema("mongod", mySchema)
type SchemaRegistry struct {
	mu sync.RWMutex

	// definitions holds all pre-generated OpenAPI definitions keyed by canonical type name
	definitions map[string]common.OpenAPIDefinition

	// Components maps component name to its schema (e.g., "mongod", "mongos")
	Components map[string]*spec.Schema

	// Topologies maps topology name to its schema (e.g., "replicaset", "sharded")
	Topologies map[string]*spec.Schema

	// Global is the schema for global/cluster-wide configuration
	Global *spec.Schema

	// typeMapping maps short names to canonical type names
	componentTypeMap map[string]string
	topologyTypeMap  map[string]string
	globalTypeName   string
}

// NewSchemaRegistry creates a new empty SchemaRegistry.
func NewSchemaRegistry() *SchemaRegistry {
	return &SchemaRegistry{
		definitions:      make(map[string]common.OpenAPIDefinition),
		Components:       make(map[string]*spec.Schema),
		Topologies:       make(map[string]*spec.Schema),
		componentTypeMap: make(map[string]string),
		topologyTypeMap:  make(map[string]string),
	}
}

// NewSchemaRegistryFromDefinitions creates a SchemaRegistry pre-populated with
// OpenAPI definitions from a GetOpenAPIDefinitions function.
//
// Example:
//
//	import "github.com/myorg/provider/pkg/generated/openapi"
//
//	defs := openapi.GetOpenAPIDefinitions(openapi.DefaultReferenceCallback)
//	registry := NewSchemaRegistryFromDefinitions(defs)
func NewSchemaRegistryFromDefinitions(defs map[string]common.OpenAPIDefinition) *SchemaRegistry {
	r := NewSchemaRegistry()
	r.definitions = defs
	return r
}

// NewSchemaRegistryWithFunc creates a SchemaRegistry using a GetOpenAPIDefinitions function.
// This is a convenience wrapper that calls the function with DefaultReferenceCallback.
//
// Example:
//
//	import "github.com/myorg/provider/pkg/generated/openapi"
//
//	registry := NewSchemaRegistryWithFunc(openapi.GetOpenAPIDefinitions)
func NewSchemaRegistryWithFunc(fn GetOpenAPIDefinitionsFunc) *SchemaRegistry {
	defs := fn(DefaultReferenceCallback)
	return NewSchemaRegistryFromDefinitions(defs)
}

// =============================================================================
// TYPE MAPPING (for pre-generated schemas)
// =============================================================================

// MapComponent maps a component name to a pre-generated schema by type name.
// The typeName should be the canonical Go type name used in GetOpenAPIDefinitions.
//
// Example:
//
//	registry.MapComponent("mongod", "github.com/myorg/provider/types.MongodCustomSpec")
func (r *SchemaRegistry) MapComponent(componentName, typeName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	def, ok := r.definitions[typeName]
	if !ok {
		return &SchemaNotFoundError{TypeName: typeName}
	}

	schema := def.Schema
	r.Components[componentName] = &schema
	r.componentTypeMap[componentName] = typeName
	return nil
}

// MapTopology maps a topology name to a pre-generated schema by type name.
func (r *SchemaRegistry) MapTopology(topologyName, typeName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	def, ok := r.definitions[typeName]
	if !ok {
		return &SchemaNotFoundError{TypeName: typeName}
	}

	schema := def.Schema
	r.Topologies[topologyName] = &schema
	r.topologyTypeMap[topologyName] = typeName
	return nil
}

// MapGlobal maps the global schema to a pre-generated schema by type name.
func (r *SchemaRegistry) MapGlobal(typeName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	def, ok := r.definitions[typeName]
	if !ok {
		return &SchemaNotFoundError{TypeName: typeName}
	}

	schema := def.Schema
	r.Global = &schema
	r.globalTypeName = typeName
	return nil
}

// MustMapComponent is like MapComponent but panics on error.
func (r *SchemaRegistry) MustMapComponent(componentName, typeName string) {
	if err := r.MapComponent(componentName, typeName); err != nil {
		panic(err)
	}
}

// MustMapTopology is like MapTopology but panics on error.
func (r *SchemaRegistry) MustMapTopology(topologyName, typeName string) {
	if err := r.MapTopology(topologyName, typeName); err != nil {
		panic(err)
	}
}

// MustMapGlobal is like MapGlobal but panics on error.
func (r *SchemaRegistry) MustMapGlobal(typeName string) {
	if err := r.MapGlobal(typeName); err != nil {
		panic(err)
	}
}

// =============================================================================
// DIRECT SCHEMA REGISTRATION (for testing or dynamic schemas)
// =============================================================================

// RegisterComponentSchema directly registers a schema for a component.
// Use this for testing or when you have dynamically created schemas.
func (r *SchemaRegistry) RegisterComponentSchema(name string, schema *spec.Schema) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Components[name] = schema
}

// RegisterTopologySchema directly registers a schema for a topology.
func (r *SchemaRegistry) RegisterTopologySchema(name string, schema *spec.Schema) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Topologies[name] = schema
}

// RegisterGlobalSchema directly registers the global schema.
func (r *SchemaRegistry) RegisterGlobalSchema(schema *spec.Schema) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Global = schema
}

// =============================================================================
// SCHEMA RETRIEVAL
// =============================================================================

// GetComponentSchema returns the schema for a component by name.
func (r *SchemaRegistry) GetComponentSchema(name string) (*spec.Schema, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	schema, ok := r.Components[name]
	return schema, ok
}

// GetTopologySchema returns the schema for a topology by name.
func (r *SchemaRegistry) GetTopologySchema(name string) (*spec.Schema, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	schema, ok := r.Topologies[name]
	return schema, ok
}

// GetGlobalSchema returns the global configuration schema.
func (r *SchemaRegistry) GetGlobalSchema() (*spec.Schema, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.Global, r.Global != nil
}

// GetDefinition returns the full OpenAPI definition for a type name.
func (r *SchemaRegistry) GetDefinition(typeName string) (common.OpenAPIDefinition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	def, ok := r.definitions[typeName]
	return def, ok
}

// =============================================================================
// LISTING
// =============================================================================

// ListComponents returns all registered component names.
func (r *SchemaRegistry) ListComponents() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.Components))
	for name := range r.Components {
		names = append(names, name)
	}
	return names
}

// ListTopologies returns all registered topology names.
func (r *SchemaRegistry) ListTopologies() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.Topologies))
	for name := range r.Topologies {
		names = append(names, name)
	}
	return names
}

// ListDefinitions returns all available pre-generated type names.
func (r *SchemaRegistry) ListDefinitions() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.definitions))
	for name := range r.definitions {
		names = append(names, name)
	}
	return names
}

// =============================================================================
// DOCUMENT GENERATION
// =============================================================================

// AllSchemas returns a complete OpenAPI 3.0 document containing all registered schemas.
// This is useful for documentation and can be served at an endpoint.
func (r *SchemaRegistry) AllSchemas() *OpenAPIDocument {
	r.mu.RLock()
	defer r.mu.RUnlock()

	schemas := make(map[string]*spec.Schema)

	// Add component schemas
	for name, schema := range r.Components {
		schemas["component."+name] = schema
	}

	// Add topology schemas
	for name, schema := range r.Topologies {
		schemas["topology."+name] = schema
	}

	// Add global schema
	if r.Global != nil {
		schemas["global"] = r.Global
	}

	return &OpenAPIDocument{
		OpenAPI: "3.0.3",
		Info: OpenAPIInfo{
			Title:       "Provider Configuration Schemas",
			Version:     "1.0.0",
			Description: "OpenAPI schemas for provider component, topology, and global configurations",
		},
		Components: &OpenAPIComponents{
			Schemas: schemas,
		},
		Paths: make(map[string]interface{}),
	}
}

// AllSchemasJSON returns the complete OpenAPI document as JSON bytes.
func (r *SchemaRegistry) AllSchemasJSON() ([]byte, error) {
	doc := r.AllSchemas()
	return json.MarshalIndent(doc, "", "  ")
}

// =============================================================================
// ERROR TYPES
// =============================================================================

// SchemaNotFoundError is returned when a schema type is not found in definitions.
type SchemaNotFoundError struct {
	TypeName string
}

func (e *SchemaNotFoundError) Error() string {
	return "schema not found for type: " + e.TypeName
}
