import { z } from 'zod';
import {
  Component,
  CelExpression,
} from 'components/ui-generator/ui-generator.types';

import { extractCelFieldPaths } from './cel-validation';
import {
  buildGenericValidationSchema,
  buildNumberValidationSchema,
} from './zod-validation/validation-schema-by-type';

export type CelValidationData = {
  celExpValidation?: { path: string[]; celExpressions: CelExpression[] };
  celDependencyGroup?: string[];
};

const isFieldRequired = (component: Component): boolean =>
  component.fieldParams?.required === true;

const applyCommonValidations = (
  schema: z.ZodTypeAny,
  component: Component,
  isRequired: boolean
): z.ZodTypeAny => {
  let result = schema;

  if (
    component.validation &&
    'regex' in component.validation &&
    component.validation.regex
  ) {
    const regexValidation = component.validation.regex as any;
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

  // Apply required/optional based on isRequired flag
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
  const isRequired = isFieldRequired(component);

  switch (component.uiType) {
    case 'number':
      fieldSchema = buildNumberValidationSchema(component, isRequired);
      break;

    case 'select':
    default:
      fieldSchema = buildGenericValidationSchema(component, baseSchema);
      break;
  }

  // Apply common validations (regex, required/optional) for all field types
  fieldSchema = applyCommonValidations(fieldSchema, component, isRequired);

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
