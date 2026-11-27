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

import type { FieldCodec } from '../field-codecs/field-codecs.types';
import { createAddedRow } from './rows';
import type { Row, RowsFieldKind } from './rows-field.types';

// What a kind converts in its own way: the form's rows into the value the API
// takes, and into the one a CEL rule reads.
interface RowsConversions {
  toApi: (formValue: unknown) => unknown;
  toCel: (formValue: unknown) => unknown;
}

// A rows field always has a row to fill, so a value with no entries is one
// empty row in the form. The row is marked as added: no entry stands behind
// it, so nothing a schema locks holds for it, and it is no entry itself.
const toFormRows = (apiValue: unknown, kind: RowsFieldKind): Row[] => {
  const rows = kind.toRows(apiValue);
  return rows.length > 0 ? rows : [createAddedRow(kind)];
};

// The form keeps a rows field as rows; the API takes what the kind writes.
// The empty row comes with the form's values rather than from the component,
// so a form holds it from the start and is not changed by being opened.
export const createRowsFieldCodec = (
  kind: RowsFieldKind,
  { toApi, toCel }: RowsConversions
): FieldCodec => ({
  toForm: (apiValue) => toFormRows(apiValue, kind),
  toApi,
  // A default is written in the API shape. One in another shape is ignored
  // rather than written back as text.
  defaultToForm: (defaultValue) =>
    kind.isValidDefault(defaultValue)
      ? toFormRows(defaultValue, kind)
      : undefined,
  emptyToForm: () => toFormRows(undefined, kind),
  // A rule reads what the field writes, so it can be written against the
  // API: the value, and nothing at all while it has no entries.
  toCel,
  ownsWholeValue: true,
});
