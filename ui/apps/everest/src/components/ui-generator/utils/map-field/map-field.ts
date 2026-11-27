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

import type {
  Component,
  ComponentGroup,
  MapComponent,
} from '../../ui-generator.types';
import { isMaskedValue, maskPasswordValue } from '../display-value';
import { isPlainObject } from '../object-path';
import { ADDED_ROW_FLAG } from './map-field.constants';
import { getMapRow, getRowField } from './map-row';
import type { MapDisplayRow, MapRow } from './map-field.types';

export const isMapRow = (value: unknown): value is MapRow =>
  isPlainObject(value) &&
  typeof value.key === 'string' &&
  typeof value.value === 'string';

// The row the Add button appends: empty, and marked as added.
export const createAddedRow = (): MapRow => ({
  key: '',
  value: '',
  [ADDED_ROW_FLAG]: true,
});

// Whether the user added the row since the form opened. The rows a form opens
// with, a saved map's or a default's, are built by `objectToRows`, unmarked.
export const isAddedRow = (row: unknown): boolean =>
  isPlainObject(row) && row[ADDED_ROW_FLAG] === true;

// The value a map field edits; anything else is left as it is.
export const isStringMap = (value: unknown): value is Record<string, string> =>
  isPlainObject(value) &&
  Object.values(value).every((entry) => typeof entry === 'string');

// A row added but never filled is dropped, not reported.
export const isEmptyRow = ({ key, value }: MapRow): boolean =>
  key === '' && value === '';

export const objectToRows = (value: unknown): MapRow[] =>
  isPlainObject(value)
    ? Object.entries(value).map(([key, entryValue]) => ({
        key,
        value: String(entryValue),
      }))
    : [];

export const rowsToObject = (
  rows: unknown
): Record<string, string> | undefined => {
  const entries = (Array.isArray(rows) ? rows : [])
    .filter(isMapRow)
    .filter(({ key }) => key !== '')
    .map(({ key, value }) => [key, value]);
  return entries.length > 0 ? Object.fromEntries(entries) : undefined;
};

// A saved map may hold what its field can't write, e.g. a number set with
// kubectl; it is still shown as stored.
const toDisplayText = (value: unknown): string =>
  typeof value === 'string' ? value : JSON.stringify(value);

// Read-only views (preview, overview) show a map's entries in a key/value
// table behind an entry-count chip. A row field typed as a password is
// masked, as in its input, and the row says so: what it shows is no longer the
// entry. Reads both the form's rows and the saved map; empty when there is no
// entry.
export const getMapFieldDisplayRows = (
  value: unknown,
  component: MapComponent
): MapDisplayRow[] => {
  const map = Array.isArray(value) ? rowsToObject(value) : value;
  if (!isPlainObject(map)) return [];
  const row = getMapRow(component);
  const keyField = getRowField(row, 'key');
  const valueField = getRowField(row, 'value');
  const show = (field: Component | ComponentGroup | undefined, text: string) =>
    field ? maskPasswordValue(field, text) : text;
  const masks = (field: Component | ComponentGroup | undefined, text: string) =>
    !!field && isMaskedValue(field, text);
  return Object.entries(map).map(([key, entryValue]) => {
    const text = toDisplayText(entryValue);
    return {
      key: show(keyField, key),
      value: show(valueField, text),
      ...((masks(keyField, key) || masks(valueField, text)) && {
        masked: true,
      }),
    };
  });
};
