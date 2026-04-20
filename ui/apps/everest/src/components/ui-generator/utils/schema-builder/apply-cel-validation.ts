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
import { CelExpression } from 'components/ui-generator/ui-generator.types';
import { validateCelExpression } from './cel-validation';

export const applyCelValidation = (
  schema: z.ZodTypeAny,
  celExpValidations: { path: string[]; celExpressions: CelExpression[] }[],
  originalData?: Record<string, unknown>
): z.ZodTypeAny => {
  if (celExpValidations.length === 0) {
    return schema;
  }

  return schema.superRefine((data, ctx) => {
    celExpValidations.forEach(({ path, celExpressions }) => {
      // Evaluate each CEL expression for this field
      celExpressions.forEach((celExpr) => {
        const validationResult = validateCelExpression(
          celExpr,
          data,
          originalData
        );

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
