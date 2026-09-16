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

import {
  getZodRulesForFieldType,
  applyZodValidation,
} from 'components/ui-generator/constants';
import {
  Component,
  FieldType,
} from 'components/ui-generator/ui-generator.types';

import { z } from 'zod';

export const buildNumberValidationSchema = (
  component: Component
): z.ZodTypeAny => {
  const isRequired = !!component.validation?.required;
  let numberSchema: z.ZodTypeAny = z.coerce.number();

  // Get field-type-specific validation rules
  const fieldTypeRules = getZodRulesForFieldType(FieldType.Number);

  if (component.validation) {
    Object.entries(component.validation).forEach(([rule, ruleValue]) => {
      // Handle CEL separately
      if (rule === 'celExpressions') return;

      const zodMethod = fieldTypeRules[rule];
      if (!zodMethod) return;
      numberSchema = applyZodValidation(numberSchema, zodMethod, ruleValue);
    });
  }
  // TODO support union type from zod to be able to combine number intervals
  if (isRequired) {
    return z
      .union([z.string().min(1, { message: 'Field is required' }), z.number()])
      .pipe(numberSchema);
  } else {
    // For optional fields: convert empty strings to undefined before validation
    return z
      .union([z.string(), z.number(), z.undefined()])
      .transform((val) => {
        // Treat empty string and undefined as undefined (skip validation)
        if (val === '' || val === undefined) {
          return undefined;
        }
        return val;
      })
      .pipe(z.union([z.undefined(), numberSchema]))
      .optional();
  }
};

export const buildSelectValidationSchema = (
  component: Component
): z.ZodTypeAny => {
  const isRequired = !!component.validation?.required;
  if (
    component.uiType !== FieldType.Select ||
    !('options' in component.fieldParams)
  ) {
    // Fallback to basic string validation for no options case
    return z.string();
  }

  const optionValues =
    component.fieldParams.options?.map((opt) => opt.value) || [];
  const hasDisplayEmpty = !!component.fieldParams.displayEmpty;

  // This matches the auto-injected empty option in the UI
  const allowedValues =
    !isRequired && hasDisplayEmpty && !optionValues.includes('')
      ? ['', ...optionValues]
      : optionValues;

  if (allowedValues.length === 0) {
    // Fallback to basic string validation for no options case
    return z.string();
  }

  const enumSchema = z.enum(allowedValues as [string, ...string[]]);

  // For select fields, we handle regex validation in the enum refinement
  // to ensure it applies to the selected value(s)
  const regexValidation = component.validation?.regex as
    | { pattern: string; message?: string }
    | undefined;
  const pattern = regexValidation ? new RegExp(regexValidation.pattern) : null;
  const regexMessage = regexValidation?.message || 'Invalid format';

  let baseSchema: z.ZodTypeAny;

  if (isRequired) {
    baseSchema = z.string().min(1, { message: 'Field is required' });

    if (pattern) {
      // Apply regex, then enum
      baseSchema = baseSchema
        .refine((val) => pattern.test(val), { message: regexMessage })
        .refine((val) => allowedValues.includes(val), {
          message: 'Invalid selection',
        });
    } else {
      baseSchema = baseSchema.pipe(enumSchema);
    }
  } else {
    baseSchema = z.union([z.string(), z.undefined()]);

    if (pattern) {
      // Apply regex and enum for non-empty values
      baseSchema = baseSchema
        .refine(
          (val) => {
            if (val === undefined || val === '') return true;
            return pattern.test(val) && allowedValues.includes(val);
          },
          { message: regexMessage }
        )
        .optional();
    } else {
      baseSchema = baseSchema
        .transform((val) => (val === '' ? undefined : val))
        .pipe(z.union([z.undefined(), enumSchema]))
        .optional();
    }
  }

  return baseSchema;
};

/**
 * Builds a Zod schema for a Text field.
 *
 * Validation order matters because transforms (trim/toLowerCase/toUpperCase)
 * return ZodEffects and can no longer chain ZodString methods afterwards.
 * Order applied:
 *   1. min / max / length  (length checks)
 *   2. email / url / uuid  (format checks — still on ZodString)
 *   3. trim / toLowerCase / toUpperCase  (transforms — return ZodEffects)
 *   4. required / optional wrapper
 */
export const buildTextValidationSchema = (
  component: Component
): z.ZodTypeAny => {
  const isRequired = !!component.validation?.required;
  const validation = component.validation as
    | import('components/ui-generator/ui-generator.types').TextValidation
    | undefined;

  // plain ZodString so we can chain all methods
  let schema: z.ZodString = z.string();

  // value-based length validators (still ZodString)
  const minVal = validation?.min;
  const maxVal = validation?.max;
  const lengthVal = validation?.length;

  if (minVal !== undefined) {
    schema = schema.min(minVal);
  }
  if (maxVal !== undefined) {
    schema = schema.max(maxVal);
  }
  if (lengthVal !== undefined) {
    schema = schema.length(lengthVal);
  }

  // regex pattern (still ZodString, before format checkers)

  const regexValidation = validation?.regex as
    | { pattern: string; message?: string }
    | undefined;
  if (regexValidation) {
    schema = schema.regex(new RegExp(regexValidation.pattern), {
      message: regexValidation.message || 'Invalid format',
    });
  }

  // boolean format validators (still ZodString)
  if (validation?.email === true) {
    schema = schema.email();
  }
  if (validation?.url === true) {
    schema = schema.url();
  }
  if (validation?.uuid === true) {
    schema = schema.uuid();
  }

  // transforms return ZodEffects so they must be last
  let effectsSchema: z.ZodTypeAny = schema;

  if (validation?.trim === true) {
    effectsSchema = (effectsSchema as z.ZodString).trim();
  }
  if (validation?.toLowerCase === true) {
    effectsSchema = (effectsSchema as z.ZodString).toLowerCase();
  }
  if (validation?.toUpperCase === true) {
    effectsSchema = (effectsSchema as z.ZodString).toUpperCase();
  }

  // required / optional
  if (isRequired) {
    if (minVal === undefined && lengthVal === undefined) {
      // We can only call .min() on ZodString, not on ZodEffects.
      // If transforms were applied we refine instead.
      if (effectsSchema instanceof z.ZodString) {
        effectsSchema = effectsSchema.min(1, { message: 'Field is required' });
      } else {
        effectsSchema = effectsSchema.refine(
          (val) => typeof val === 'string' && val.length > 0,
          { message: 'Field is required' }
        );
      }
    }
    return effectsSchema;
  }

  // For optional text fields we treat an empty string the same as
  // "no value" so that format validators (email, url, uuid, regex, …)
  // do NOT complain when the user hasn't typed anything yet.
  // z.preprocess runs before Zod validation, so '' → undefined and the
  // .optional() wrapper then accepts undefined without further checks.
  return z.preprocess(
    (val) => (val === '' ? undefined : val),
    effectsSchema.optional()
  );
};

/**
 * Toggles validate as optional booleans. `.optional()` is required, not
 * cosmetic: in Edit mode a value missing from the instance is left `undefined`
 * (see extract-instance-values), which must still pass validation. `required`
 * is intentionally unsupported for toggles.
 */
export const buildToggleValidationSchema = (): z.ZodTypeAny => {
  return z.boolean().optional();
};

export const buildGenericValidationSchema = (
  component: Component,
  baseSchema: z.ZodTypeAny
): z.ZodTypeAny => {
  let fieldSchema = baseSchema;

  if (component.validation) {
    // Get field-type-specific validation rules
    const fieldTypeRules = getZodRulesForFieldType(
      component.uiType as FieldType
    );

    Object.entries(component.validation).forEach(([rule, ruleValue]) => {
      // Skip CEL and regex (handled separately)
      if (rule === 'celExpressions' || rule === 'regex') return;

      const zodMethod = fieldTypeRules[rule];
      if (zodMethod) {
        fieldSchema = applyZodValidation(fieldSchema, zodMethod, ruleValue);
      }
    });
  }

  return fieldSchema;
};
