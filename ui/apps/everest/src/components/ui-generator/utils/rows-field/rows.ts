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

// A rows field keeps rows in the form and writes its entries to the API. What
// every kind of row shares lives here.

import { isPlainObject } from '../object-path';
import { ADDED_ROW_FLAG } from './rows-field.constants';
import type { Row, RowsFieldKind } from './rows-field.types';

// What a row holds is all these checks need of a field's kind.
type RowRoles = Pick<RowsFieldKind, 'roles'>;

export const isRow = (value: unknown, { roles }: RowRoles): value is Row =>
  isPlainObject(value) &&
  roles.every((role) => typeof value[role] === 'string');

// A row added but never filled is dropped, not reported.
export const isEmptyRow = (row: Row, { roles }: RowRoles): boolean =>
  roles.every((role) => (row[role] ?? '') === '');

// Whether a field holds its one empty row and nothing else: the row that
// stands in while the field has no entries.
export const isOneEmptyRow = (rows: unknown, kind: RowRoles): boolean => {
  if (!Array.isArray(rows) || rows.length !== 1) return false;
  const [row] = rows;
  return isRow(row, kind) && isEmptyRow(row, kind);
};

// The row the Add button appends: empty, and marked as added.
export const createAddedRow = ({ emptyRow }: RowsFieldKind): Row => ({
  ...emptyRow,
  [ADDED_ROW_FLAG]: true,
});

// Whether the row is one the user may fill and remove freely: one added since
// the form opened, or the empty row of a field with no entries. The entries a
// form opens with, a saved value's or a default's, are built by the codec,
// unmarked.
export const isAddedRow = (row: unknown): boolean =>
  isPlainObject(row) && row[ADDED_ROW_FLAG] === true;

// A saved value may hold what its field can't write, e.g. a number set with
// kubectl; it is still shown as stored.
export const toDisplayText = (value: unknown): string =>
  typeof value === 'string' ? value : JSON.stringify(value);
