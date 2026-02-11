import { z } from 'zod';
import { CelExpression } from 'components/ui-generator/ui-generator.types';
import { validateCelExpression } from './cel-validation';

export const applyCelValidation = (
  schema: z.ZodTypeAny,
  celExpValidations: { path: string[]; celExpressions: CelExpression[] }[]
): z.ZodTypeAny => {
  if (celExpValidations.length === 0) {
    return schema;
  }

  return schema.superRefine((data, ctx) => {
    celExpValidations.forEach(({ path, celExpressions }) => {
      // Evaluate each CEL expression for this field
      celExpressions.forEach((celExpr) => {
        const validationResult = validateCelExpression(celExpr, data);

        if (!validationResult.isValid) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            message: validationResult.message || 'Validation failed',
            path: path,
          });
        }
      });
    });
  });
};
