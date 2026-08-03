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

package v1alpha1

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/xeipuuv/gojsonschema"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ParametersSchema declares the OpenAPI v3 schema that a parameters payload
// (a schema-validated, inline RawExtension field such as
// Instance.spec.parameters or Backup.spec.parameters) is validated against.
//
// Providers and BackupClasses declare these schemas; the runtime and the
// admission webhooks validate user-supplied parameters with Validate(); the
// UI renders forms from them.
type ParametersSchema struct {
	// OpenAPIV3Schema is the OpenAPI v3 schema describing the accepted
	// parameters payload.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +kubebuilder:validation:Schemaless
	// +optional
	OpenAPIV3Schema *apiextensionsv1.JSONSchemaProps `json:"openAPIV3Schema,omitempty"` //nolint:tagliatelle
}

// ErrSchemaValidationFailure is returned when a parameters payload does not
// conform to the declared ParametersSchema.
var ErrSchemaValidationFailure = errors.New("schema validation failed")

// Validate checks the parameters payload against the declared schema.
// A payload without a declared schema is rejected; an absent payload is
// always valid.
func (s *ParametersSchema) Validate(params *runtime.RawExtension) error {
	var schema *apiextensionsv1.JSONSchemaProps
	if s != nil {
		schema = s.OpenAPIV3Schema
	}
	if schema == nil && params != nil {
		return ErrSchemaValidationFailure
	}
	if schema == nil && params == nil {
		return nil
	}
	if params == nil {
		return nil
	}

	// Additional properties are implicitly disallowed.
	schema = schema.DeepCopy()
	schema.AdditionalProperties = &apiextensionsv1.JSONSchemaPropsOrBool{
		Allows: false,
	}

	// Unmarshal the parameters into a generic map.
	var paramsMap map[string]any
	if err := json.Unmarshal(params.Raw, &paramsMap); err != nil {
		return fmt.Errorf("failed to unmarshal parameters: %w", err)
	}

	// Convert the OpenAPI v3 schema to a JSON schema validator.
	schemaJSON, err := json.Marshal(schema)
	if err != nil {
		return fmt.Errorf("failed to marshal OpenAPI v3 schema: %w", err)
	}

	schemaLoader := gojsonschema.NewStringLoader(string(schemaJSON))
	paramsLoader := gojsonschema.NewGoLoader(paramsMap)

	result, err := gojsonschema.Validate(schemaLoader, paramsLoader)
	if err != nil {
		return fmt.Errorf("failed to validate parameters: %w", err)
	}

	if !result.Valid() {
		validationErrors := make([]string, 0, len(result.Errors()))
		for _, err := range result.Errors() {
			validationErrors = append(validationErrors, err.String())
		}
		return errors.Join(ErrSchemaValidationFailure, fmt.Errorf("validation errors: %s", strings.Join(validationErrors, "; ")))
	}
	return nil
}
