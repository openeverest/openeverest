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
import type { ToggleableMeta } from 'components/ui-generator/ui-generator.types';
import { validateCelExpression } from './cel-validation';
import {
  deepClone,
  deleteByPathAndEmptyParents,
  isPlainObject,
} from '../object-path/object-path';
import { getSwitchedOffPaths, isToggleableOn } from '../toggleable/toggleable';
import type { CelExpValidation } from './schema-builder.types';

// A switched-off group's fields are not sent, so CEL sees them as absent (as
// the API would), not as the values the form still holds.
const withoutSwitchedOffFields = (
  data: Record<string, unknown>,
  toggleables: ToggleableMeta[]
): Record<string, unknown> => {
  const offPaths = getSwitchedOffPaths(toggleables, data);
  if (offPaths.length === 0) return data;
  const visible = deepClone(data);
  offPaths.forEach((path) => {
    const [root, ...rest] = path.split('.');
    const scope = visible[root];
    // Keep the root (e.g. `spec`): CEL fails on an unknown variable, not on an absent field.
    if (isPlainObject(scope) && rest.length > 0) {
      deleteByPathAndEmptyParents(scope, rest.join('.'));
    }
  });
  return visible;
};

export const applyCelValidation = (
  schema: z.ZodTypeAny,
  celExpValidations: CelExpValidation[],
  originalData?: Record<string, unknown>,
  toggleables: ToggleableMeta[] = []
): z.ZodTypeAny => {
  if (celExpValidations.length === 0) {
    return schema;
  }

  return schema.superRefine((data, ctx) => {
    const celData = withoutSwitchedOffFields(data, toggleables);

    celExpValidations.forEach(({ path, celExpressions, activeWhen }) => {
      if (activeWhen && !isToggleableOn(data, activeWhen)) return;

      // Evaluate each CEL expression for this field
      celExpressions.forEach((celExpr) => {
        const validationResult = validateCelExpression(
          celExpr,
          celData,
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
