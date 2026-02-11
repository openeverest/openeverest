import { z } from 'zod';
import {
  Component,
  CelExpression,
} from 'components/ui-generator/ui-generator.types';
import {
  buildNumberValidationSchema,
  buildGenericValidationSchema,
} from '../validation/validation-schema-by-type';
import { extractCelFieldPaths } from './cel-validation';

export type CelValidationData = {
  celExpValidation?: { path: string[]; celExpressions: CelExpression[] };
  celDependencyGroup?: string[];
};

export const applyValidationFromSchema = (
  component: Component,
  baseSchema: z.ZodTypeAny,
  fieldId: string
): { fieldSchema: z.ZodTypeAny; celData: CelValidationData } => {
  let fieldSchema: z.ZodTypeAny;
  const isRequired = component.fieldParams?.required !== false;

  switch (component.uiType) {
    case 'number':
      fieldSchema = buildNumberValidationSchema(component, isRequired);
      break;

    case 'select':
    default:
      fieldSchema = buildGenericValidationSchema(component, baseSchema);
      break;
  }

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
