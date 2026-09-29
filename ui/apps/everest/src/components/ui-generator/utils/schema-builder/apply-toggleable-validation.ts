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
import { getByPath } from '../object-path/object-path';
import type { ToggleableFieldRule } from './schema-builder.types';

// Validation of a toggleable group's fields is paused while its switch is off;
// when on, each field's real schema applies and errors land on the field itself.
export const applyToggleableValidation = (
  schema: z.ZodTypeAny,
  rules: ToggleableFieldRule[]
): z.ZodTypeAny => {
  if (rules.length === 0) {
    return schema;
  }

  return schema.superRefine((data, ctx) => {
    rules.forEach(({ switchName, fieldId, schema: fieldSchema }) => {
      if (getByPath(data, switchName) !== true) return;

      const result = fieldSchema.safeParse(getByPath(data, fieldId));
      if (result.success) return;

      result.error.issues.forEach((issue) => {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          message: issue.message,
          path: [...fieldId.split('.'), ...issue.path],
        });
      });
    });
  });
};
