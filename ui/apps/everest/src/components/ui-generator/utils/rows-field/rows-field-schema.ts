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
import type { RowsValidation } from '../../ui-generator.types';
import { isAddedRow, isEmptyRow, isRow } from './rows';
import { Messages } from './rows-field.messages';
import type { Row, RowShapes, RowsFieldKind } from './rows-field.types';

const getCountIssue = (
  filledRows: number,
  validation: RowsValidation | undefined
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
// issue on the field's path itself would replace them all.
export const ROWS_ROOT_ERROR_PATH = ['root'];

// Validates the rows the form keeps, each against the rules of the row's own
// fields, built like any section's. A row the form opened with and a row added
// since have a shape each: a disabled field skips its rules, and a field
// disabled in the first is enabled in the second. The issues land on the row's
// input, or on the field itself for counts. The output is each filled row as
// its fields parse it (trim, toLowerCase, …), like a text field's value.
export const buildRowsFieldSchema = (
  kind: RowsFieldKind,
  rowShapes: RowShapes,
  validation: RowsValidation | undefined,
  // With `disableRemove`, clearing a row the form opened with must not remove
  // its entry the way an empty row is otherwise dropped.
  keepsOpenedWithRows = false
): z.ZodTypeAny => {
  const { identity, messages } = kind;
  const unique = kind.isUnique(validation);
  const openedWithRowSchema = z.object(rowShapes.openedWith);
  const addedRowSchema = z.object(rowShapes.added);
  const parseRow = (row: unknown) =>
    isRow(row, kind) && !isEmptyRow(row, kind)
      ? (isAddedRow(row) ? addedRowSchema : openedWithRowSchema).safeParse(row)
      : undefined;

  return z
    .array(z.unknown())
    .optional()
    .superRefine((held = [], ctx) => {
      const rows = held.filter((row): row is Row => isRow(row, kind));
      // What isn't rows is not the form's to check, or to save.
      if (rows.length !== held.length) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          path: ROWS_ROOT_ERROR_PATH,
          message: Messages.notRows,
        });
        return;
      }

      const seen = new Set<string>();
      let filledRows = 0;
      const reportOnIdentity = (index: number, message: string) =>
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          path: [index, identity],
          message,
        });

      rows.forEach((row, index) => {
        if (keepsOpenedWithRows && isEmptyRow(row, kind) && !isAddedRow(row)) {
          reportOnIdentity(index, messages.identityRequired);
          return;
        }
        const parsed = parseRow(row);
        if (!parsed) return;
        filledRows += 1;

        // An entry goes by its identity as the field writes it, e.g. trimmed.
        const written =
          parsed.success && isRow(parsed.data, kind) ? parsed.data : row;
        const entry = written[identity] ?? '';
        // A filled row needs one: the field can't hold the entry without it.
        const missing = entry === '';
        if (missing) reportOnIdentity(index, messages.identityRequired);
        parsed.error?.issues
          .filter(({ path: [field] }) => !missing || field !== identity)
          .forEach((issue) =>
            ctx.addIssue({ ...issue, path: [index, ...issue.path] })
          );

        if (missing || !unique) return;
        if (seen.has(entry)) reportOnIdentity(index, messages.duplicate(entry));
        seen.add(entry);
      });

      const countIssue = getCountIssue(filledRows, validation);
      if (countIssue) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          path: ROWS_ROOT_ERROR_PATH,
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
