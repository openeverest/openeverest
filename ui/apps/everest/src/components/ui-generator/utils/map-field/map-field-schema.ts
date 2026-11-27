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
import { isAddedRow, isEmptyRow, isMapRow } from './map-field';
import { Messages } from './map-field.messages';
import type { MapRowShapes } from './map-field.types';

const getCountIssue = (
  filledRows: number,
  validation: MapValidation | undefined
): string | undefined => {
  const { required, min, max } = validation ?? {};
  if (required && filledRows === 0) return Messages.required;
  if (min !== undefined && filledRows < min) {
    return Messages.tooFewEntries(min, min - filledRows);
  }
  if (max !== undefined && filledRows > max) {
    return Messages.tooManyEntries(max, filledRows - max);
  }
  return undefined;
};

// RHF keeps an array's own error under `root`, beside its rows' errors. An
// issue on the map's path itself would replace them all.
export const MAP_ROOT_ERROR_PATH = ['root'];

// Validates the rows the form keeps, each against the rules of the row's own
// fields, built like any section's. A row the form opened with and a row added
// since have a shape each: a disabled field skips its rules, and a field
// disabled in the first is enabled in the second. The issues land on the row's
// input, or on the map itself for counts. The output is each filled row as its
// fields parse it (trim, toLowerCase, …), like a text field's value.
export const buildMapFieldSchema = (
  rowShapes: MapRowShapes,
  validation: MapValidation | undefined,
  // With `disableRemove`, clearing a row the form opened with must not remove
  // its entry the way an empty row is otherwise dropped.
  keepsOpenedWithRows = false
): z.ZodTypeAny => {
  const openedWithRowSchema = z.object(rowShapes.openedWith);
  const addedRowSchema = z.object(rowShapes.added);
  const parseRow = (row: unknown) =>
    isMapRow(row) && !isEmptyRow(row)
      ? (isAddedRow(row) ? addedRowSchema : openedWithRowSchema).safeParse(row)
      : undefined;

  return z
    .array(z.unknown())
    .optional()
    .superRefine((rows = [], ctx) => {
      const seenKeys = new Set<string>();
      let filledRows = 0;

      rows.forEach((row, index) => {
        if (
          keepsOpenedWithRows &&
          isMapRow(row) &&
          isEmptyRow(row) &&
          !isAddedRow(row)
        ) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            path: [index, 'key'],
            message: Messages.keyRequired,
          });
          return;
        }
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
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          path: MAP_ROOT_ERROR_PATH,
          message: countIssue,
        });
      }
    })
    .transform((rows) =>
      rows?.map((row) => {
        const parsed = parseRow(row);
        return parsed?.success ? parsed.data : row;
      })
    );
};
