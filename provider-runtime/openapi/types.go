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

// +k8s:openapi-gen=true
package openapi

import (
	"encoding/json"

	"k8s.io/kube-openapi/pkg/common"
	"k8s.io/kube-openapi/pkg/validation/spec"
)

// =============================================================================
// SCHEMA DEFINITION TYPES
// =============================================================================

// OpenAPISchemaDefinition holds a pre-generated OpenAPI schema and its metadata.
type OpenAPISchemaDefinition struct {
	// Schema is the OpenAPI v3 schema
	Schema *spec.Schema
	// TypeName is the Go type name (e.g., "MongodCustomSpec")
	TypeName string
	// PackagePath is the full Go package path
	PackagePath string
}

// GetOpenAPIDefinitionsFunc is the signature for generated GetOpenAPIDefinitions functions.
type GetOpenAPIDefinitionsFunc func(ref common.ReferenceCallback) map[string]common.OpenAPIDefinition

// =============================================================================
// SCHEMA UTILITIES
// =============================================================================

// SchemaToJSON converts a spec.Schema to JSON bytes.
func SchemaToJSON(schema *spec.Schema) ([]byte, error) {
	return json.Marshal(schema)
}

// DefinitionsToSchemaMap extracts schemas from OpenAPI definitions into a simple map.
// This is useful for serving schemas without the full OpenAPI document structure.
func DefinitionsToSchemaMap(defs map[string]common.OpenAPIDefinition) map[string]*spec.Schema {
	result := make(map[string]*spec.Schema, len(defs))
	for name, def := range defs {
		schema := def.Schema
		result[name] = &schema
	}
	return result
}

// SchemaForType extracts a single schema from definitions by type name.
// The typeName should be the canonical Go type name (e.g., "github.com/pkg/types.MyType").
func SchemaForType(defs map[string]common.OpenAPIDefinition, typeName string) (*spec.Schema, bool) {
	def, ok := defs[typeName]
	if !ok {
		return nil, false
	}
	schema := def.Schema
	return &schema, true
}

// =============================================================================
// REFERENCE CALLBACK HELPERS
// =============================================================================

// DefaultReferenceCallback creates a reference callback that generates standard OpenAPI refs.
// References are formatted as "#/components/schemas/{name}".
func DefaultReferenceCallback(path string) spec.Ref {
	return spec.MustCreateRef("#/components/schemas/" + path)
}

// DefinitionsReferenceCallback creates a reference callback for OpenAPI 2.0 style refs.
// References are formatted as "#/definitions/{name}".
func DefinitionsReferenceCallback(path string) spec.Ref {
	return spec.MustCreateRef("#/definitions/" + path)
}

// =============================================================================
// OPENAPI DOCUMENT BUILDER
// =============================================================================

// OpenAPIDocument represents a complete OpenAPI 3.0 document.
type OpenAPIDocument struct {
	OpenAPI    string                 `json:"openapi"`
	Info       OpenAPIInfo            `json:"info"`
	Components *OpenAPIComponents     `json:"components,omitempty"`
	Paths      map[string]interface{} `json:"paths,omitempty"`
}

// OpenAPIInfo contains API metadata.
type OpenAPIInfo struct {
	Title       string `json:"title"`
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
}

// OpenAPIComponents contains reusable components.
type OpenAPIComponents struct {
	Schemas map[string]*spec.Schema `json:"schemas,omitempty"`
}

// BuildOpenAPIDocument creates an OpenAPI 3.0 document from definitions.
func BuildOpenAPIDocument(title, version, description string, defs map[string]common.OpenAPIDefinition) *OpenAPIDocument {
	schemas := make(map[string]*spec.Schema, len(defs))
	for name, def := range defs {
		schema := def.Schema
		schemas[name] = &schema
	}

	return &OpenAPIDocument{
		OpenAPI: "3.0.3",
		Info: OpenAPIInfo{
			Title:       title,
			Version:     version,
			Description: description,
		},
		Components: &OpenAPIComponents{
			Schemas: schemas,
		},
		Paths: make(map[string]interface{}),
	}
}

// ToJSON serializes the document to JSON.
func (d *OpenAPIDocument) ToJSON() ([]byte, error) {
	return json.MarshalIndent(d, "", "  ")
}
