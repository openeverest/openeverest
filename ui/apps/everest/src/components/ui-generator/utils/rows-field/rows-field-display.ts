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

import type {
  Component,
  ComponentGroup,
  RowsComponent,
} from '../../ui-generator.types';
import { isMaskedValue, maskPasswordValue } from '../display-value';
import { getRowField } from './row-template';
import { getRow, getRowsFieldKind } from './rows-field-kinds';
import type { RowsDisplayEntry } from './rows-field.types';

// Read-only views (preview, overview) show a rows field's entries in a table
// behind an entry-count chip. What a row field typed as a password holds is
// masked, as in its input, and the entry says so: what it shows is no longer
// the entry. Reads both the form's rows and the saved value; empty when there
// is no entry.
export const getRowsFieldDisplayEntries = (
  value: unknown,
  component: RowsComponent
): RowsDisplayEntry[] => {
  const row = getRow(component);
  const keyField = getRowField(row, 'key');
  const valueField = getRowField(row, 'value');
  const show = (field: Component | ComponentGroup | undefined, text: string) =>
    field ? maskPasswordValue(field, text) : text;
  const masks = (field: Component | ComponentGroup | undefined, text: string) =>
    !!field && isMaskedValue(field, text);
  return getRowsFieldKind(component)
    .toEntries(value)
    .map(({ key, value: text }) => ({
      ...(key !== undefined && { key: show(keyField, key) }),
      value: show(valueField, text),
      ...(((key !== undefined && masks(keyField, key)) ||
        masks(valueField, text)) && { masked: true }),
    }));
};
