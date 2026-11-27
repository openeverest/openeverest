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

// A list field keeps rows ({ value }) in the form, since a form array holds
// objects, and writes the list of their values to the API. What turns one
// into the other lives here.

import type { RowRole } from '../../ui-generator.types';
import { isRow } from '../rows-field/rows';
import type { Row } from '../rows-field/rows-field.types';

// A list's row holds the entry alone, where a map's row holds its value.
export const LIST_ROW_ROLES = ['value'] as const satisfies readonly RowRole[];

const isListRow = (value: unknown): value is Row =>
  isRow(value, { roles: LIST_ROW_ROLES });

// The value a list field edits; anything else is left as it is.
export const isStringList = (value: unknown): value is string[] =>
  Array.isArray(value) && value.every((entry) => typeof entry === 'string');

export const listToRows = (value: unknown): Row[] =>
  Array.isArray(value) ? value.map((entry) => ({ value: String(entry) })) : [];

// A row left empty is not an entry, and no entry leaves no list to write.
const rowsToList = (rows: Row[]): string[] | undefined => {
  const entries = rows
    .map(({ value }) => value)
    .filter((value) => value !== '');
  return entries.length > 0 ? entries : undefined;
};

// The list the form's rows hold. A value that isn't rows, like a saved list a
// form loaded unconverted, is in the API shape already. An empty array is no
// rows, so a saved `[]` is no list either.
export const toList = (formValue: unknown): unknown =>
  Array.isArray(formValue) && formValue.every(isListRow)
    ? rowsToList(formValue)
    : formValue;
