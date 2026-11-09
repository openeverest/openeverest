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
import {
  type ArrayComponent,
  type ArrayValidation,
  type KeyValueItemTemplate,
  FieldType,
} from '../../ui-generator.types';
import { buildTextValidationSchema } from '../schema-builder/zod-validation/validation-schema-by-type';
import { isEmptyRow, isKeyValuePair } from './array-field';
import { Messages } from './array-field.messages';

type RowPart = keyof NonNullable<KeyValueItemTemplate['validation']>;

const buildRowPartSchema = (
  template: KeyValueItemTemplate,
  part: RowPart
): z.ZodTypeAny =>
  buildTextValidationSchema({
    uiType: FieldType.Text,
    path: part,
    fieldParams: {},
    validation: template.validation?.[part],
  });

const getCountIssue = (
  filledRows: number,
  validation: ArrayValidation | undefined
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

// Validates the rows the form keeps. The issues land on the row's own input,
// or on the array itself for counts.
export const buildArrayFieldSchema = (
  { itemTemplate }: ArrayComponent,
  validation: ArrayValidation | undefined
): z.ZodTypeAny => {
  const rowSchema = z.object({
    key: z
      .string()
      .min(1, { message: Messages.keyRequired })
      .pipe(buildRowPartSchema(itemTemplate, 'key')),
    value: buildRowPartSchema(itemTemplate, 'value'),
  });

  return z
    .array(z.unknown())
    .optional()
    .superRefine((rows = [], ctx) => {
      const seenKeys = new Set<string>();
      let filledRows = 0;

      rows.forEach((row, index) => {
        if (!isKeyValuePair(row) || isEmptyRow(row)) return;
        filledRows += 1;

        rowSchema
          .safeParse(row)
          .error?.issues.forEach((issue) =>
            ctx.addIssue({ ...issue, path: [index, ...issue.path] })
          );

        if (row.key === '') return;
        if (seenKeys.has(row.key)) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            path: [index, 'key'],
            message: Messages.duplicateKey(row.key),
          });
        }
        seenKeys.add(row.key);
      });

      const countIssue = getCountIssue(filledRows, validation);
      if (countIssue) {
        ctx.addIssue({ code: z.ZodIssueCode.custom, message: countIssue });
      }
    });
};
