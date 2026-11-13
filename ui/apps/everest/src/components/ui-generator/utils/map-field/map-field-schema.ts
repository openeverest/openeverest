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
import type { MapValidation } from '../../ui-generator.types';
import { isEmptyRow, isMapRow } from './map-field';
import { Messages } from './map-field.messages';

const getCountIssue = (
  filledRows: number,
  validation: MapValidation | undefined
): string | undefined => {
  const { required, minItems, maxItems } = validation ?? {};
  if (required && filledRows === 0) return Messages.required;
  if (minItems !== undefined && filledRows < minItems) {
    return Messages.minItems(minItems);
  }
  if (maxItems !== undefined && filledRows > maxItems) {
    return Messages.maxItems(maxItems);
  }
  return undefined;
};

// Validates the rows the form keeps, each against `rowShape`: the rules of the
// row's own fields, built like any section's. The issues land on the row's
// input, or on the map itself for counts. The output is each filled row as its
// fields parse it (trim, toLowerCase, …), like a text field's value.
export const buildMapFieldSchema = (
  rowShape: Record<string, z.ZodTypeAny>,
  validation: MapValidation | undefined
): z.ZodTypeAny => {
  const rowSchema = z.object(rowShape);
  const parseRow = (row: unknown) =>
    isMapRow(row) && !isEmptyRow(row) ? rowSchema.safeParse(row) : undefined;

  return z
    .array(z.unknown())
    .optional()
    .superRefine((rows = [], ctx) => {
      const seenKeys = new Set<string>();
      let filledRows = 0;

      rows.forEach((row, index) => {
        const parsed = parseRow(row);
        if (!parsed || !isMapRow(row)) return;
        filledRows += 1;

        // The map is keyed by the key as its field writes it, e.g. trimmed.
        const key =
          parsed.success && isMapRow(parsed.data) ? parsed.data.key : row.key;
        // A filled row needs a key: the map can't hold the entry without one.
        const missingKey = key === '';
        if (missingKey) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            path: [index, 'key'],
            message: Messages.keyRequired,
          });
        }
        parsed.error?.issues
          .filter(({ path: [field] }) => !missingKey || field !== 'key')
          .forEach((issue) =>
            ctx.addIssue({ ...issue, path: [index, ...issue.path] })
          );

        if (missingKey) return;
        if (seenKeys.has(key)) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            path: [index, 'key'],
            message: Messages.duplicateKey(key),
          });
        }
        seenKeys.add(key);
      });

      const countIssue = getCountIssue(filledRows, validation);
      if (countIssue) {
        ctx.addIssue({ code: z.ZodIssueCode.custom, message: countIssue });
      }
    })
    .transform((rows) =>
      rows?.map((row) => {
        const parsed = parseRow(row);
        return parsed?.success ? parsed.data : row;
      })
    );
};
