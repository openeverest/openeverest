import { getZodRulesForFieldType } from 'components/ui-generator/constants';
import {
  Component,
  FieldType,
} from 'components/ui-generator/ui-generator.types';

import { z } from 'zod';

export const buildNumberValidationSchema = (
  component: Component,
  isRequired: boolean
): z.ZodTypeAny => {
  let numberSchema = z.coerce.number();

  // Get field-type-specific validation rules
  const fieldTypeRules = getZodRulesForFieldType(FieldType.Number);

  if (component.validation) {
    Object.entries(component.validation).forEach(([rule, ruleValue]) => {
      // Handle CEL separately
      if (rule === 'celExpressions') return;

      const zodMethod = fieldTypeRules[rule];
      if (
        zodMethod &&
        typeof numberSchema[zodMethod as keyof typeof numberSchema] ===
          'function'
      ) {
        // For boolean validation methods (int, positive, etc.), call without arguments if true
        if (typeof ruleValue === 'boolean' && ruleValue) {
          numberSchema = (numberSchema as any)[zodMethod]();
        } else if (typeof ruleValue !== 'boolean') {
          // For methods that take parameters (min, max, gt, etc.)
          numberSchema = (numberSchema as any)[zodMethod](ruleValue);
        }
      }
    });
  }
  // TODO support union type from zod to be able to combine number intervals
  if (isRequired) {
    return z
      .union([z.string().min(1, { message: 'Field is required' }), z.number()])
      .pipe(numberSchema);
  } else {
    return z
      .union([z.string(), z.number()])
      .optional()
      .pipe(numberSchema)
      .catch(() => undefined as any);
  }
};

export const buildGenericValidationSchema = (
  component: Component,
  baseSchema: z.ZodTypeAny
): z.ZodTypeAny => {
  let fieldSchema = baseSchema;

  if (component.validation) {
    // Get field-type-specific validation rules
    const fieldTypeRules = getZodRulesForFieldType(component.uiType as any);

    Object.entries(component.validation).forEach(([rule, ruleValue]) => {
      if (rule === 'celExpressions') return; // Handle CEL separately

      const zodMethod = fieldTypeRules[rule];
      if (
        zodMethod &&
        typeof fieldSchema[zodMethod as keyof typeof fieldSchema] === 'function'
      ) {
        fieldSchema = (fieldSchema as any)[zodMethod](ruleValue);
      }
    });
  }

  return fieldSchema;
};
