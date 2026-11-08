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

// Package classvalidation evaluates the validations of a class, CEL rules on
// the spec of the objects that reference it, with the CEL library, cost
// limits and semantics of CRD validation rules.
package classvalidation

import (
	"context"
	"errors"
	"fmt"

	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextensionsvalidation "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/validation"
	structuralschema "k8s.io/apiextensions-apiserver/pkg/apiserver/schema"
	"k8s.io/apiextensions-apiserver/pkg/apiserver/schema/cel"
	"k8s.io/apiextensions-apiserver/pkg/apiserver/schema/cel/model"
	"k8s.io/apimachinery/pkg/util/validation/field"
	celconfig "k8s.io/apiserver/pkg/apis/cel"
	"k8s.io/apiserver/pkg/cel/environment"

	common "github.com/openeverest/openeverest/v2/api/common/v1alpha1"
)

// Violation is a validation an object breaks.
type Violation struct {
	// Reason is the validation's reason.
	Reason string
	// Message is the validation's message, or why it could not be
	// evaluated.
	Message string
}

// Rules are compiled validations.
type Rules struct {
	rules []rule
}

type rule struct {
	validation common.ClassValidation
	schema     *structuralschema.Structural
	validator  *cel.Validator
}

// Compile compiles validations with self typed by schema, the OpenAPI schema
// of the referencing objects' spec. The schema's own x-kubernetes-validations
// are ignored: the API server enforces them. Compile fails on a schema that is
// not structural and on a validation that does not compile, uses oldSelf, or
// whose estimated cost exceeds the limit of a CRD validation rule.
func Compile(schema *apiextensionsv1.JSONSchemaProps, validations []common.ClassValidation) (*Rules, error) {
	if len(validations) == 0 {
		return &Rules{}, nil
	}
	if schema == nil {
		return nil, errors.New("no schema to type self with")
	}
	internal := &apiextensions.JSONSchemaProps{}
	if err := apiextensionsv1.Convert_v1_JSONSchemaProps_To_apiextensions_JSONSchemaProps(withoutValidations(schema), internal, nil); err != nil {
		return nil, err
	}
	base, err := structuralschema.NewStructural(internal)
	if err != nil {
		return nil, fmt.Errorf("schema is not structural: %w", err)
	}
	if errs := structuralschema.ValidateStructural(nil, base); len(errs) > 0 {
		return nil, fmt.Errorf("schema is not structural: %w", errs.ToAggregate())
	}
	declType := model.SchemaDeclType(base, false)
	envSet := environment.MustBaseEnvSet(environment.DefaultCompatibilityVersion())

	out := &Rules{rules: make([]rule, 0, len(validations))}
	var errs []error
	for i, v := range validations {
		s := base.DeepCopy()
		s.XValidations = apiextensionsv1.ValidationRules{{Rule: v.Expression, Message: v.Message}}
		results, err := cel.Compile(s, declType, celconfig.PerCallLimit, envSet, cel.NewExpressionsEnvLoader())
		if err != nil {
			return nil, err
		}
		switch r := results[0]; {
		case r.Error != nil:
			errs = append(errs, fmt.Errorf("validations[%d]: %w", i, r.Error))
		case r.UsesOldSelf:
			errs = append(errs, fmt.Errorf("validations[%d]: oldSelf is not available", i))
		case r.MaxCost > apiextensionsvalidation.StaticEstimatedCostLimit:
			errs = append(errs, fmt.Errorf("validations[%d]: estimated cost %d exceeds the limit of %d",
				i, r.MaxCost, apiextensionsvalidation.StaticEstimatedCostLimit,
			))
		default:
			out.rules = append(out.rules, rule{validation: v, schema: s, validator: cel.NewValidator(s, false, celconfig.PerCallLimit)})
		}
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	return out, nil
}

// Evaluate evaluates the validations against spec, the unstructured spec of
// a referencing object, and returns those it breaks, in order. A validation
// that cannot be evaluated, or is not reached because the cost budget of the
// object is spent, is broken.
func (r *Rules) Evaluate(ctx context.Context, spec map[string]any) []Violation {
	var out []Violation
	budget := int64(celconfig.RuntimeCELCostBudget)
	for _, rl := range r.rules {
		if budget <= 0 {
			out = append(out, Violation{Reason: rl.validation.Reason, Message: "not evaluated: the cost budget is spent"})
			continue
		}
		var errs field.ErrorList
		errs, budget = rl.validator.Validate(ctx, field.NewPath("spec"), rl.schema, spec, nil, budget)
		if len(errs) > 0 {
			out = append(out, Violation{Reason: rl.validation.Reason, Message: errs[0].Detail})
		}
	}
	return out
}

// withoutValidations returns a copy of s without x-kubernetes-validations.
func withoutValidations(s *apiextensionsv1.JSONSchemaProps) *apiextensionsv1.JSONSchemaProps {
	out := s.DeepCopy()
	stripValidations(out)
	return out
}

func stripValidations(s *apiextensionsv1.JSONSchemaProps) {
	s.XValidations = nil
	for name, p := range s.Properties {
		stripValidations(&p)
		s.Properties[name] = p
	}
	if s.Items != nil && s.Items.Schema != nil {
		stripValidations(s.Items.Schema)
	}
	if s.AdditionalProperties != nil && s.AdditionalProperties.Schema != nil {
		stripValidations(s.AdditionalProperties.Schema)
	}
	for _, list := range [][]apiextensionsv1.JSONSchemaProps{s.AllOf, s.AnyOf, s.OneOf} {
		for i := range list {
			stripValidations(&list[i])
		}
	}
	if s.Not != nil {
		stripValidations(s.Not)
	}
}
