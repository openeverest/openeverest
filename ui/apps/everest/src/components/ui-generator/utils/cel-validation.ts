import { evaluate } from '@marcbachmann/cel-js';
import { CelExpression } from '../ui-generator.types';

export const extractCelFieldPaths = (celExpr: string): string[][] => {
  // Regex matches patterns like: word.word.word (at least 2 segments)
  const regex = /([a-zA-Z0-9_]+(?:\.[a-zA-Z0-9_]+)+)/g;
  const matches = celExpr.match(regex) || [];
  // Remove duplicates and split each path into an array
  return Array.from(new Set(matches)).map((f) => f.split('.'));
};

export const validateCelExpression = (
  celExpression: CelExpression,
  formData: Record<string, any>
): { isValid: boolean; message?: string } => {
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
