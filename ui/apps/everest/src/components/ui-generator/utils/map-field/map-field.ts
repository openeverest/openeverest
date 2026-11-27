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

// A map field keeps rows ({ key, value }) in the form and writes the map to the
// API. What turns one into the other lives here.

import { ROW_ROLES } from '../../ui-generator.types';
import { isPlainObject } from '../object-path';
import { isRow } from '../rows-field/rows';
import type { Row } from '../rows-field/rows-field.types';

// The value a map field edits; anything else is left as it is.
export const isStringMap = (value: unknown): value is Record<string, string> =>
  isPlainObject(value) &&
  Object.values(value).every((entry) => typeof entry === 'string');

export const objectToRows = (value: unknown): Row[] =>
  isPlainObject(value)
    ? Object.entries(value).map(([key, entryValue]) => ({
        key,
        value: String(entryValue),
      }))
    : [];

const rowsToObject = (rows: unknown): Record<string, string> | undefined => {
  const entries = (Array.isArray(rows) ? rows : [])
    .filter((row): row is Row => isRow(row, { roles: ROW_ROLES }))
    .flatMap(({ key, value }) => (key ? [[key, value]] : []));
  return entries.length > 0 ? Object.fromEntries(entries) : undefined;
};

// The map the form's rows hold: none when no row is filled. A value that
// isn't rows, like a saved map a form loaded unconverted, is in the API shape
// already.
export const toMap = (formValue: unknown): unknown =>
  formValue !== undefined && !Array.isArray(formValue)
    ? formValue
    : rowsToObject(formValue);
