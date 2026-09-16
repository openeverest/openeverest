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

import { z } from 'zod';
import {
  Component,
  CelExpression,
} from 'components/ui-generator/ui-generator.types';

import { extractCelFieldPaths } from './cel-validation';
import {
  buildGenericValidationSchema,
  buildNumberValidationSchema,
  buildSelectValidationSchema,
  buildTextValidationSchema,
  buildToggleValidationSchema,
} from './zod-validation/validation-schema-by-type';

export type CelValidationData = {
  celExpValidation?: { path: string[]; celExpressions: CelExpression[] };
  celDependencyGroup?: string[];
};

const applyCommonValidations = (
  schema: z.ZodTypeAny,
  component: Component,
  isRequired: boolean
): z.ZodTypeAny => {
  let result = schema;

  // For select fields, regex is handled in buildSelectValidationSchema
  // For text fields, regex is handled in buildTextValidationSchema
  // Skip regex here to avoid double-validation
  const shouldApplyRegex =
    component.uiType !== 'select' &&
    component.uiType !== 'text' &&
    component.uiType !== 'toggle' &&
    component.validation !== undefined &&
    'regex' in component.validation &&
    component.validation.regex !== undefined;

  if (shouldApplyRegex && component.validation) {
    const regexValidation: { pattern: string; message?: string } =
      component.validation.regex!;
    const pattern = new RegExp(regexValidation.pattern);
    const message = regexValidation.message || 'Invalid format';

    // Zod's regex works on strings, so we need to handle string conversion
    result = z
      .union([z.string(), z.number()])
      .refine(
        (val) => {
          if (val === '' || val === undefined || val === null) {
            return !isRequired; // Empty is ok if not required
          }
          return pattern.test(String(val));
        },
        { message }
      )
      .transform((val) => val);
  }

  if (!isRequired) {
    result = result.optional();
  }

  return result;
};

export const applyValidationFromSchema = (
  component: Component,
  baseSchema: z.ZodTypeAny,
  fieldId: string
): { fieldSchema: z.ZodTypeAny; celData: CelValidationData } => {
  let fieldSchema: z.ZodTypeAny;
  const isRequired = !!component.validation?.required;

  switch (component.uiType) {
    case 'number':
      fieldSchema = buildNumberValidationSchema(component);
      break;

    case 'select':
      fieldSchema = buildSelectValidationSchema(component);
      break;

    case 'text':
      fieldSchema = buildTextValidationSchema(component);
      break;

    case 'toggle':
      fieldSchema = buildToggleValidationSchema();
      break;

    default:
      fieldSchema = buildGenericValidationSchema(component, baseSchema);
      break;
  }

  const effectiveRequired = component.uiType === 'toggle' ? false : isRequired;

  // Apply common validations (regex, required/optional) for all field types
  fieldSchema = applyCommonValidations(
    fieldSchema,
    component,
    effectiveRequired
  );

  // Handle CEL expressions for cross-field validation
  let celData: CelValidationData = {};
  if (
    component.validation &&
    'celExpressions' in component.validation &&
    component.validation.celExpressions
  ) {
    celData = extractCelValidationData(
      component.validation.celExpressions,
      fieldId
    );
  }

  return { fieldSchema, celData };
};

const extractCelValidationData = (
  celExpressions: CelExpression[],
  fieldId: string
): CelValidationData => {
  // Extract all field dependencies from all CEL expressions
  const allDeps = new Set<string>();
  celExpressions.forEach((celExpr) => {
    const deps = extractCelFieldPaths(celExpr.celExpr);
    deps.forEach((dep) => allDeps.add(dep.join('.')));
  });

  const celData: CelValidationData = {
    celExpValidation: {
      path: [fieldId],
      celExpressions,
    },
  };

  // Add dependency group if there are dependencies
  if (allDeps.size > 0) {
    celData.celDependencyGroup = [fieldId, ...Array.from(allDeps)];
  }

  return celData;
};
