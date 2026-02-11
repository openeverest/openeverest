import { evaluate } from '@marcbachmann/cel-js';
import { CelExpression } from 'components/ui-generator/ui-generator.types';

export type CelValidationResult = {
  isValid: boolean;
  message?: string;
};

export const validateCelExpression = (
  celExpression: CelExpression,
  formData: Record<string, any>
): CelValidationResult => {
  try {
    const result = evaluate(celExpression.celExpr, formData);

    // CEL expression should return true for valid, false for invalid
    if (result === false || !result) {
      return {
        isValid: false,
        message:
          celExpression.message ||
          `Validation failed: ${celExpression.celExpr}`,
      };
    }

    return { isValid: true };
  } catch (error) {
    console.error('CEL expression evaluation error:', error);
    return {
      isValid: false,
      message: `CEL expression error: ${celExpression.celExpr}`,
    };
  }
};
