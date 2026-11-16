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

package classgen

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

// Validation is a CEL rule of a class, evaluated against the spec of each
// object that references the class (self).
type Validation struct {
	// Expression is the CEL expression, true when the object is valid.
	Expression string `json:"expression"`
	// Message describes the violation to the user.
	Message string `json:"message"`
	// Reason is the machine-readable reason of the violation.
	Reason string `json:"reason"`
}

// Validations returns the validations a view declares: the
// x-kubernetes-validations at the root of view, the schema of a view type
// (see [Packages.View]), in order.
//
// Each rule sets a message and no messageExpression, reason or
// optionalOldSelf, and names in fieldPath the field of the view it judges,
// such as .location.bucket. reasonFor returns the reason of a rule from that
// path, or an error for a field the class's rules must not judge. Rules below
// the root of the view are an error, as self is always the whole spec.
func Validations(
	view *apiextensionsv1.JSONSchemaProps,
	reasonFor func(fieldPath string) (string, error),
) ([]Validation, error) {
	if view == nil {
		return nil, errors.New("no view schema")
	}
	var errs []error
	for name, prop := range view.Properties {
		errs = append(errs, rejectCEL("."+name, &prop))
	}
	out := make([]Validation, 0, len(view.XValidations))
	for i, r := range view.XValidations {
		at := fmt.Sprintf("rule %d (%s)", i, r.Rule)
		switch {
		case r.Message == "":
			errs = append(errs, fmt.Errorf("%s: message is required", at))
			continue
		case r.MessageExpression != "" || r.Reason != nil || r.OptionalOldSelf != nil:
			errs = append(errs, fmt.Errorf("%s: messageExpression, reason and optionalOldSelf are not supported", at))
			continue
		}
		if err := fieldExists(view, r.FieldPath); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", at, err))
			continue
		}
		reason, err := reasonFor(r.FieldPath)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: fieldPath %q: %w", at, r.FieldPath, err))
			continue
		}
		out = append(out, Validation{Expression: r.Rule, Message: r.Message, Reason: reason})
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	return out, nil
}

// fieldExists checks that path, such as .location.bucket, names a field of
// view.
func fieldExists(view *apiextensionsv1.JSONSchemaProps, path string) error {
	segments := strings.Split(strings.TrimPrefix(path, "."), ".")
	if !strings.HasPrefix(path, ".") || slices.Contains(segments, "") {
		return fmt.Errorf("fieldPath %q must name the field the rule judges, such as .location.bucket", path)
	}
	s := view
	for _, seg := range segments {
		prop, ok := s.Properties[seg]
		if !ok {
			return fmt.Errorf("fieldPath %q: no field %q", path, seg)
		}
		s = &prop
	}
	return nil
}
