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

import type { z } from 'zod';
import type {
  RowRole,
  RowsFieldType,
  RowsValidation,
} from '../../ui-generator.types';
import type { ADDED_ROW_FLAG } from './rows-field.constants';

// An entry of a rows field: a map's has a key, a list's is the value alone.
export interface RowsEntry {
  key?: string;
  value: string;
}

// One row of a rows field in the form, holding what its fields write.
export interface Row extends RowsEntry {
  // Set on a row added since the form opened, see ADDED_ROW_FLAG.
  [ADDED_ROW_FLAG]?: true;
}

// An entry as a read-only view shows it.
export interface RowsDisplayEntry extends RowsEntry {
  // A part of the entry is shown as a mask, not as stored.
  masked?: boolean;
}

// The zod shape of a row's fields, for each kind of row.
export interface RowShapes {
  // A row the form was (re)initialised with: a saved entry, a default one.
  openedWith: Record<string, z.ZodTypeAny>;
  // A row the user added since.
  added: Record<string, z.ZodTypeAny>;
}

// What tells a map from a list, once both are rows.
export interface RowsFieldKind {
  // Names the field for a schema author, and starts the names of its rules,
  // e.g. `list-overlap`.
  uiType: RowsFieldType;
  // What a row holds, by the `path` of the row's field that writes it.
  roles: readonly RowRole[];
  // A row with nothing in it.
  emptyRow: Row;
  // The part of a row its entry goes by: a map's key, a list's value. A
  // filled row has one.
  identity: RowRole;
  // Whether no two entries may go by the same identity.
  isUnique: (validation: RowsValidation | undefined) => boolean;
  messages: {
    identityRequired: string;
    duplicate: (identity: string) => string;
  };
  // The entries of the form's rows or of a saved value, for a read-only view.
  toEntries: (value: unknown) => RowsEntry[];
  // Whether a schema's `defaultValue` is in the shape the field writes.
  isValidDefault: (defaultValue: unknown) => boolean;
  // The two below word a warning to a schema author: the shape the field
  // writes, and the item template this version implements.
  valueShape: string;
  supportedItemTemplate: string;
}
