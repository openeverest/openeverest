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

package main

import (
	"encoding/json"
	"fmt"
	"go/types"
	"reflect"
	"strings"

	"golang.org/x/tools/go/packages"
)

// loadPackages loads one or more Go packages for type inspection using go/packages.
// Each pkgPattern can be a relative path (e.g., "./types") or a full import path.
func loadPackages(pkgPatterns []string) ([]*packages.Package, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedTypes | packages.NeedImports,
	}
	var result []*packages.Package
	for _, pattern := range pkgPatterns {
		pkgs, err := packages.Load(cfg, pattern)
		if err != nil {
			return nil, fmt.Errorf("loading package %s: %w", pattern, err)
		}
		if len(pkgs) == 0 {
			return nil, fmt.Errorf("package %s not found", pattern)
		}
		for _, pkg := range pkgs {
			if len(pkg.Errors) > 0 {
				var errs []string
				for _, e := range pkg.Errors {
					errs = append(errs, e.Error())
				}
				return nil, fmt.Errorf("errors loading package %s: %s", pattern, strings.Join(errs, "; "))
			}
			result = append(result, pkg)
		}
	}
	return result, nil
}

// typeSchemaFromPackages searches the given packages in order for typeName and
// generates an OpenAPI JSON schema from the first match found.
func typeSchemaFromPackages(pkgs []*packages.Package, typeName string) (json.RawMessage, error) {
	for _, pkg := range pkgs {
		obj := pkg.Types.Scope().Lookup(typeName)
		if obj != nil {
			schema := goTypeToSchema(obj.Type())
			return json.Marshal(schema)
		}
	}
	pkgPaths := make([]string, len(pkgs))
	for i, p := range pkgs {
		pkgPaths[i] = p.PkgPath
	}
	return nil, fmt.Errorf("type %q not found in packages: %s", typeName, strings.Join(pkgPaths, ", "))
}

// openAPISchema is a minimal OpenAPI 3.x schema representation for JSON marshaling.
type openAPISchema struct {
	Type                 string                    `json:"type,omitempty"`
	Format               string                    `json:"format,omitempty"`
	Properties           map[string]*openAPISchema `json:"properties,omitempty"`
	Items                *openAPISchema            `json:"items,omitempty"`
	AdditionalProperties *openAPISchema            `json:"additionalProperties,omitempty"`
	Description          string                    `json:"description,omitempty"`
}

// goTypeToSchema converts a go/types.Type into an OpenAPI schema.
func goTypeToSchema(t types.Type) *openAPISchema {
	switch u := t.Underlying().(type) {
	case *types.Struct:
		return structToSchema(u)
	case *types.Basic:
		return basicToSchema(u)
	case *types.Slice:
		return &openAPISchema{
			Type:  "array",
			Items: goTypeToSchema(u.Elem()),
		}
	case *types.Map:
		return &openAPISchema{
			Type:                 "object",
			AdditionalProperties: goTypeToSchema(u.Elem()),
		}
	case *types.Pointer:
		return goTypeToSchema(u.Elem())
	case *types.Interface:
		// interface{} / any → no type constraint
		return &openAPISchema{}
	default:
		return &openAPISchema{Type: "object"}
	}
}

// structToSchema converts a go/types.Struct into an OpenAPI object schema.
func structToSchema(s *types.Struct) *openAPISchema {
	schema := &openAPISchema{Type: "object"}
	if s.NumFields() == 0 {
		return schema
	}

	schema.Properties = make(map[string]*openAPISchema)
	for i := range s.NumFields() {
		field := s.Field(i)
		if !field.Exported() {
			continue
		}
		tag := s.Tag(i)
		name := jsonFieldName(field.Name(), tag)
		if name == "-" {
			continue
		}
		fieldSchema := goTypeToSchema(field.Type())

		// Apply description from struct tag if present.
		st := reflect.StructTag(tag)
		if desc := st.Get("description"); desc != "" {
			fieldSchema.Description = desc
		}

		schema.Properties[name] = fieldSchema
	}
	return schema
}

// basicToSchema converts a go/types.Basic into an OpenAPI schema with type and format.
func basicToSchema(b *types.Basic) *openAPISchema {
	switch b.Kind() {
	case types.String:
		return &openAPISchema{Type: "string"}
	case types.Bool:
		return &openAPISchema{Type: "boolean"}
	case types.Int, types.Int64:
		return &openAPISchema{Type: "integer", Format: "int64"}
	case types.Int32:
		return &openAPISchema{Type: "integer", Format: "int32"}
	case types.Int16, types.Int8:
		return &openAPISchema{Type: "integer"}
	case types.Uint, types.Uint64:
		return &openAPISchema{Type: "integer", Format: "uint64"}
	case types.Uint32:
		return &openAPISchema{Type: "integer", Format: "uint32"}
	case types.Uint16, types.Uint8:
		return &openAPISchema{Type: "integer"}
	case types.Float32:
		return &openAPISchema{Type: "number", Format: "float"}
	case types.Float64:
		return &openAPISchema{Type: "number", Format: "double"}
	default:
		return &openAPISchema{Type: "string"}
	}
}

// jsonFieldName extracts the JSON field name from a struct tag.
// Falls back to lowercasing the first letter of the Go field name.
func jsonFieldName(goName, tag string) string {
	st := reflect.StructTag(tag)
	jsonTag := st.Get("json")
	if jsonTag == "" {
		return strings.ToLower(goName[:1]) + goName[1:]
	}
	parts := strings.SplitN(jsonTag, ",", 2)
	if parts[0] == "" {
		return strings.ToLower(goName[:1]) + goName[1:]
	}
	return parts[0]
}
