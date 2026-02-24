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

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3gen"
)

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
