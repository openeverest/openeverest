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

// Package schemavalidation provides utilities for validating data against OpenAPI v3 schemas.
package schemavalidation

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/xeipuuv/gojsonschema"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// Options configures how schema validation behaves.
type Options struct {
	// AllowAdditionalProperties allows properties not defined in the schema.
	// Default is false (additional properties are disallowed).
	AllowAdditionalProperties bool
}

// Validate validates data against an OpenAPI v3 schema.
// By default, additional properties not in the schema are disallowed.
func Validate(schema *apiextensionsv1.JSONSchemaProps, data any, opts ...Options) error {
	if schema == nil {
		return nil
	}

	opt := Options{}
	if len(opts) > 0 {
		opt = opts[0]
	}

	// Make a copy to avoid modifying the original schema.
	schemaCopy := schema.DeepCopy()

	// By default, additional properties are disallowed.
	if !opt.AllowAdditionalProperties {
		schemaCopy.AdditionalProperties = &apiextensionsv1.JSONSchemaPropsOrBool{
			Allows: false,
		}
	}

	// Convert the OpenAPI v3 schema to JSON for the validator.
	schemaJSON, err := json.Marshal(schemaCopy)
	if err != nil {
		return fmt.Errorf("failed to marshal OpenAPI v3 schema: %w", err)
	}

	schemaLoader := gojsonschema.NewStringLoader(string(schemaJSON))
	dataLoader := gojsonschema.NewGoLoader(data)

	// Validate the data against the schema.
	result, err := gojsonschema.Validate(schemaLoader, dataLoader)
	if err != nil {
		return fmt.Errorf("failed to validate data against schema: %w", err)
	}

	if !result.Valid() {
		var validationErrors []string
		for _, verr := range result.Errors() {
			validationErrors = append(validationErrors, verr.String())
		}
		return fmt.Errorf("validation errors: %s", strings.Join(validationErrors, "; "))
	}

	return nil
}
