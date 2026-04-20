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

import { evaluate } from '@marcbachmann/cel-js';
import { CelExpression } from 'components/ui-generator/ui-generator.types';

export type CelValidationResult = {
  isValid: boolean;
  message?: string;
};

export const validateCelExpression = (
  celExpression: CelExpression,
  formData: Record<string, unknown>,
  originalData?: Record<string, unknown>
): CelValidationResult => {
  try {
    const context = originalData
      ? { ...formData, original: originalData }
      : formData;
    const result = evaluate(celExpression.celExpr, context);

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
    return {
      isValid: false,
      message: `CEL expression error: ${celExpression.celExpr}`,
    };
  }
};
