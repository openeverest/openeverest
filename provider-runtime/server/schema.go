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

package server

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3gen"
)

// SchemaRegistry holds the registered schemas for components, topologies, and global config.
// It uses kin-openapi for OpenAPI 3.0 schema generation from Go types.
type SchemaRegistry struct {
	mu sync.RWMutex

	// Components maps component name to its schema (e.g., "mongod", "mongos", "cfg")
	Components map[string]*openapi3.SchemaRef

	// Topologies maps topology name to its schema (e.g., "replicaset", "sharded")
	Topologies map[string]*openapi3.SchemaRef

	// TopologyComponents maps topology name to list of supported component names
	TopologyComponents map[string][]string

	// Global is the schema for global/cluster-wide configuration
	Global *openapi3.SchemaRef
}

// NewSchemaRegistry creates a new SchemaRegistry.
func NewSchemaRegistry() *SchemaRegistry {
	return &SchemaRegistry{
		Components:         make(map[string]*openapi3.SchemaRef),
		Topologies:         make(map[string]*openapi3.SchemaRef),
		TopologyComponents: make(map[string][]string),
	}
}

// defaultSchemaCustomizer parses common OpenAPI-related struct tags and applies them to the schema.
// Supported tags:
//   - description: Schema description
//   - enum: Comma-separated list of allowed values (e.g., `enum:"a,b,c"`)
//   - default: Default value
//   - example: Example value
//   - minimum: Minimum value for numbers
//   - maximum: Maximum value for numbers
//   - minLength: Minimum length for strings
//   - maxLength: Maximum length for strings
func defaultSchemaCustomizer(name string, t reflect.Type, tag reflect.StructTag, schema *openapi3.Schema) error {
	// Description
	if desc := tag.Get("description"); desc != "" {
		schema.Description = desc
	}

	// Enum values
	if enumStr := tag.Get("enum"); enumStr != "" {
		values := strings.Split(enumStr, ",")
		schema.Enum = make([]any, len(values))
		for i, v := range values {
			schema.Enum[i] = strings.TrimSpace(v)
		}
	}

	// Default value (parse based on schema type)
	if defaultStr := tag.Get("default"); defaultStr != "" {
		schema.Default = parseValue(defaultStr, schema)
	}

	// Example value
	if exampleStr := tag.Get("example"); exampleStr != "" {
		schema.Example = parseValue(exampleStr, schema)
	}

	// Minimum (for numbers)
	if minStr := tag.Get("minimum"); minStr != "" {
		val, err := strconv.ParseFloat(minStr, 64)
		if err != nil {
			return fmt.Errorf("invalid minimum value %q for field %q: %w", minStr, name, err)
		}
		schema.Min = &val
	}

	// Maximum (for numbers)
	if maxStr := tag.Get("maximum"); maxStr != "" {
		val, err := strconv.ParseFloat(maxStr, 64)
		if err != nil {
			return fmt.Errorf("invalid maximum value %q for field %q: %w", maxStr, name, err)
		}
		schema.Max = &val
	}

	// MinLength (for strings)
	if minLenStr := tag.Get("minLength"); minLenStr != "" {
		val, err := strconv.ParseUint(minLenStr, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid minLength value %q for field %q: %w", minLenStr, name, err)
		}
		schema.MinLength = val
	}

	// MaxLength (for strings)
	if maxLenStr := tag.Get("maxLength"); maxLenStr != "" {
		val, err := strconv.ParseUint(maxLenStr, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid maxLength value %q for field %q: %w", maxLenStr, name, err)
		}
		maxLen := val
		schema.MaxLength = &maxLen
	}

	return nil
}

// parseValue attempts to parse a string value based on the schema type.
func parseValue(s string, schema *openapi3.Schema) any {
	if schema.Type == nil {
		return s
	}

	switch {
	case schema.Type.Is("boolean"):
		if v, err := strconv.ParseBool(s); err == nil {
			return v
		}
	case schema.Type.Is("integer"):
		if v, err := strconv.ParseInt(s, 10, 64); err == nil {
			return v
		}
	case schema.Type.Is("number"):
		if v, err := strconv.ParseFloat(s, 64); err == nil {
			return v
		}
	}
	return s
}

// generateSchema generates an OpenAPI schema for a Go type with custom tag support.
func generateSchema(typ any) (*openapi3.SchemaRef, error) {
	return openapi3gen.NewSchemaRefForValue(
		typ,
		nil,
		openapi3gen.SchemaCustomizer(defaultSchemaCustomizer),
	)
}

// RegisterComponent registers a component type schema.
// The type should be the Go struct that represents the component's custom spec.
// Example: registry.RegisterComponent("mongod", MongodCustomSpec{})
func (r *SchemaRegistry) RegisterComponent(name string, typ any) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	schemaRef, err := generateSchema(typ)
	if err != nil {
		return fmt.Errorf("failed to generate schema for component %q: %w", name, err)
	}
	r.Components[name] = schemaRef
	return nil
}

// RegisterTopology registers a topology type schema.
// Example: registry.RegisterTopology("replicaset", ReplicaSetTopologyConfig{})
func (r *SchemaRegistry) RegisterTopology(name string, typ any) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	schemaRef, err := generateSchema(typ)
	if err != nil {
		return fmt.Errorf("failed to generate schema for topology %q: %w", name, err)
	}
	r.Topologies[name] = schemaRef
	return nil
}

// RegisterGlobal registers the global configuration schema.
// Example: registry.RegisterGlobal(GlobalConfig{})
func (r *SchemaRegistry) RegisterGlobal(typ any) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	schemaRef, err := generateSchema(typ)
	if err != nil {
		return fmt.Errorf("failed to generate schema for global config: %w", err)
	}
	r.Global = schemaRef
	return nil
}

// RegisterTopologyComponents registers which components are supported by a topology.
// This is used for validation and documentation purposes.
// Example: registry.RegisterTopologyComponents("replicaset", []string{"engine", "backupAgent", "monitoring"})
func (r *SchemaRegistry) RegisterTopologyComponents(topologyName string, components []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.TopologyComponents[topologyName] = components
}

// MustRegisterComponent is like RegisterComponent but panics on error.
func (r *SchemaRegistry) MustRegisterComponent(name string, typ any) {
	if err := r.RegisterComponent(name, typ); err != nil {
		panic(err)
	}
}

// MustRegisterTopology is like RegisterTopology but panics on error.
func (r *SchemaRegistry) MustRegisterTopology(name string, typ any) {
	if err := r.RegisterTopology(name, typ); err != nil {
		panic(err)
	}
}

// MustRegisterGlobal is like RegisterGlobal but panics on error.
func (r *SchemaRegistry) MustRegisterGlobal(typ any) {
	if err := r.RegisterGlobal(typ); err != nil {
		panic(err)
	}
}

// GetComponentSchema returns the schema for a component by name.
func (r *SchemaRegistry) GetComponentSchema(name string) (*openapi3.SchemaRef, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	schema, ok := r.Components[name]
	return schema, ok
}

// GetTopologySchema returns the schema for a topology by name.
func (r *SchemaRegistry) GetTopologySchema(name string) (*openapi3.SchemaRef, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	schema, ok := r.Topologies[name]
	return schema, ok
}

// GetGlobalSchema returns the global configuration schema.
func (r *SchemaRegistry) GetGlobalSchema() (*openapi3.SchemaRef, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.Global, r.Global != nil
}

// GetTopologyComponents returns the list of components supported by a topology.
func (r *SchemaRegistry) GetTopologyComponents(topologyName string) ([]string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	components, ok := r.TopologyComponents[topologyName]
	return components, ok
}

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

// AllSchemas returns a complete OpenAPI 3.0 document containing all registered schemas.
// This is useful for documentation and can be served at an endpoint.
func (r *SchemaRegistry) AllSchemas() *openapi3.T {
	r.mu.RLock()
	defer r.mu.RUnlock()

	doc := &openapi3.T{
		OpenAPI: "3.0.3",
		Info: &openapi3.Info{
			Title:       "Provider Configuration Schemas",
			Version:     "1.0.0",
			Description: "OpenAPI schemas for provider component, topology, and global configurations",
		},
		Components: &openapi3.Components{
			Schemas: make(openapi3.Schemas),
		},
	}

	// Add component schemas
	for name, schema := range r.Components {
		doc.Components.Schemas["component."+name] = schema
	}

	// Add topology schemas
	for name, schema := range r.Topologies {
		doc.Components.Schemas["topology."+name] = schema
	}

	// Add global schema
	if r.Global != nil {
		doc.Components.Schemas["global"] = r.Global
	}

	// Add topology components information as extension
	if len(r.TopologyComponents) > 0 {
		doc.Extensions = make(map[string]interface{})
		doc.Extensions["x-topology-components"] = r.TopologyComponents
	}

	return doc
}

// GenerateSchema generates an OpenAPI schema for any Go type using kin-openapi.
// This is a convenience function for one-off schema generation.
// It supports the following struct tags for schema customization:
//   - description: Schema description
//   - enum: Comma-separated list of allowed values
//   - default: Default value
//   - example: Example value
//   - minimum/maximum: Numeric bounds
//   - minLength/maxLength: String length bounds
func GenerateSchema(typ any) (*openapi3.SchemaRef, error) {
	return generateSchema(typ)
}

// MustGenerateSchema is like GenerateSchema but panics on error.
func MustGenerateSchema(typ any) *openapi3.SchemaRef {
	schema, err := GenerateSchema(typ)
	if err != nil {
		panic(err)
	}
	return schema
}
