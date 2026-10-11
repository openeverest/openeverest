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

package validation

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"

	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	structuralschema "k8s.io/apiextensions-apiserver/pkg/apiserver/schema"
	"k8s.io/apiextensions-apiserver/pkg/apiserver/schema/cel"
	"k8s.io/apiextensions-apiserver/pkg/apiserver/schema/pruning"
	apiextensionsvalidation "k8s.io/apiextensions-apiserver/pkg/apiserver/validation"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	celconfig "k8s.io/apiserver/pkg/apis/cel"

	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
	"github.com/openeverest/openeverest/v2/api/core/v1alpha1"
)

// checkSchemas checks every parameters payload against the schema the Provider
// declares for it, and each component's core fields against the topology's
// supportedFields.
func checkSchemas(ctx context.Context, in *v1alpha1.Instance, spec *v1alpha1.ProviderSpec) field.ErrorList {
	specPath := field.NewPath("spec")
	errs := checkParameters(ctx, specPath.Child("parameters"), in.Spec.Parameters, spec.ParametersSchema)

	topologyName := in.GetTopologyType()
	topology, topologyKnown := spec.Topologies[topologyName]
	if topologyKnown {
		errs = append(errs, checkParameters(
			ctx, specPath.Child("topology", "parameters"), in.GetTopologyParameters(), topology.ParametersSchema,
		)...)
	}

	for _, name := range slices.Sorted(maps.Keys(in.Spec.Components)) {
		declared, ok := spec.Components[name]
		if !ok {
			continue
		}
		path := specPath.Child("components").Key(name)
		component := in.Spec.Components[name]
		errs = append(errs, checkParameters(ctx, path.Child("parameters"), component.Parameters, declared.ParametersSchema)...)
		if member, ok := topology.Components[name]; topologyKnown && ok {
			errs = append(errs, checkComponentFields(ctx, path, component, member.SupportedFields, topologyName)...)
		}
	}
	return errs
}

// checkParameters checks a parameters payload against its declared schema. A
// payload with no declared schema accepts no fields.
func checkParameters(
	ctx context.Context,
	path *field.Path,
	raw *runtime.RawExtension,
	declared *common.ParametersSchema,
) field.ErrorList {
	if raw == nil {
		return nil
	}
	data, err := raw.MarshalJSON()
	if err != nil {
		return field.ErrorList{field.InternalError(path, err)}
	}
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return field.ErrorList{field.Invalid(path, string(data), err.Error())}
	}
	if value == nil {
		return nil
	}

	schema := &apiextensionsv1.JSONSchemaProps{Type: "object"}
	if declared != nil && declared.OpenAPIV3Schema != nil {
		schema = declared.OpenAPIV3Schema
	}
	s, err := compile(schema)
	if err != nil {
		return field.ErrorList{field.InternalError(path, err)}
	}

	errs := undeclaredMetaFields(path, value, s.structural)
	unknown := pruning.PruneWithOptions(value, s.structural, false, structuralschema.UnknownFieldPathOptions{
		TrackUnknownFieldPaths: true,
	})
	for _, p := range unknown {
		errs = append(errs, undeclared(path.Child(p), "the provider declares no such parameter"))
	}
	return append(errs, s.validate(ctx, path, value)...)
}

// undeclaredMetaFields reports apiVersion, kind and metadata at the top of a
// payload that does not declare them: pruning drops those without a record.
func undeclaredMetaFields(path *field.Path, value any, s *structuralschema.Structural) field.ErrorList {
	m, ok := value.(map[string]any)
	if !ok || s.XPreserveUnknownFields || s.AdditionalProperties != nil {
		return nil
	}
	var errs field.ErrorList
	for _, name := range []string{"apiVersion", "kind", "metadata"} {
		if _, set := m[name]; !set {
			continue
		}
		if _, declared := s.Properties[name]; !declared {
			errs = append(errs, undeclared(path.Child(name), "the provider declares no such parameter"))
		}
	}
	return errs
}

// undeclared is the error for a field the schema does not declare.
func undeclared(path *field.Path, detail string) *field.Error {
	return &field.Error{
		Type:     field.ErrorTypeNotSupported,
		Field:    path.String(),
		BadValue: field.OmitValueType{},
		Detail:   detail,
	}
}

// compiledSchema holds the validators the API server builds for a CRD schema.
type compiledSchema struct {
	structural *structuralschema.Structural
	openAPI    apiextensionsvalidation.SchemaValidator
	cel        *cel.Validator
}

func compile(schema *apiextensionsv1.JSONSchemaProps) (*compiledSchema, error) {
	internal := &apiextensions.JSONSchemaProps{}
	if err := apiextensionsv1.Convert_v1_JSONSchemaProps_To_apiextensions_JSONSchemaProps(schema, internal, nil); err != nil {
		return nil, fmt.Errorf("converting the declared schema: %w", err)
	}
	structural, err := structuralschema.NewStructural(internal)
	if err != nil {
		return nil, fmt.Errorf("reading the declared schema: %w", err)
	}
	openAPI, _, err := apiextensionsvalidation.NewSchemaValidator(internal)
	if err != nil {
		return nil, fmt.Errorf("building a validator for the declared schema: %w", err)
	}
	return &compiledSchema{
		structural: structural,
		openAPI:    openAPI,
		cel:        cel.NewValidator(structural, false, celconfig.PerCallLimit),
	}, nil
}

// validate applies the schema's value validations and its CEL rules.
func (s *compiledSchema) validate(ctx context.Context, path *field.Path, value any) field.ErrorList {
	errs := apiextensionsvalidation.ValidateCustomResource(path, value, s.openAPI)
	if s.cel != nil {
		celErrs, _ := s.cel.Validate(ctx, path, s.structural, value, nil, celconfig.RuntimeCELCostBudget)
		errs = append(errs, celErrs...)
	}
	return errs
}
