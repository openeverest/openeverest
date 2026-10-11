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

// Package validation checks an Instance against what its Provider declares
// and then against the provider's own rules.
package validation

import (
	"context"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/openeverest/openeverest/v2/provider-runtime/controller"
)

// maxSummarized is how many errors Summary spells out.
const maxSummarized = 10

// Validator is the part of a provider that Run calls.
type Validator interface {
	Validate(in controller.ValidationInput) controller.ValidationResult
}

// Run checks in.Instance against the parameter schemas, the supportedFields
// and the catalog that in.Provider declares, then calls v.Validate only if
// those found no errors, so Validate can rely on them.
func Run(ctx context.Context, v Validator, in controller.ValidationInput) controller.ValidationResult {
	spec := &in.Provider.Spec
	errs := checkSchemas(ctx, in.Instance, spec)
	catalogErrs, warnings := checkCatalog(in.Instance, spec)
	errs = append(errs, catalogErrs...)
	if len(errs) > 0 {
		return controller.ValidationResult{Errors: errs, Warnings: warnings}
	}

	res := v.Validate(in)
	return controller.ValidationResult{
		Errors:   res.Errors,
		Warnings: append(warnings, res.Warnings...),
	}
}

// Summary renders errs on one line: the first ten as "field: detail", then
// the number left out.
func Summary(errs field.ErrorList) string {
	shown := min(len(errs), maxSummarized)
	parts := make([]string, 0, shown+1)
	for _, err := range errs[:shown] {
		parts = append(parts, err.Error())
	}
	if rest := len(errs) - shown; rest > 0 {
		parts = append(parts, fmt.Sprintf("and %d more", rest))
	}
	return strings.Join(parts, "; ")
}
