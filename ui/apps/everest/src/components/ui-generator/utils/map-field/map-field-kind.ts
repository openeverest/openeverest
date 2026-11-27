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

import { FieldType, ROW_ROLES } from '../../ui-generator.types';
import { isPlainObject } from '../object-path';
import { toDisplayText } from '../rows-field/rows';
import type { RowsFieldKind } from '../rows-field/rows-field.types';
import { isStringMap, toMap } from './map-field';
import { Messages } from './map-field.messages';

// What a map is, among the fields edited as rows: a row holds a key and a
// value, and the entry goes by its key, which a map holds once.
export const MAP_FIELD_KIND: RowsFieldKind = {
  uiType: FieldType.Map,
  roles: ROW_ROLES,
  emptyRow: { key: '', value: '' },
  identity: 'key',
  isUnique: () => true,
  messages: {
    identityRequired: Messages.keyRequired,
    duplicate: Messages.duplicateKey,
  },
  toEntries: (value) => {
    const map = toMap(value);
    return isPlainObject(map)
      ? Object.entries(map).map(([key, entryValue]) => ({
          key,
          value: toDisplayText(entryValue),
        }))
      : [];
  },
  isValidDefault: isStringMap,
  valueShape: 'map of strings',
  supportedItemTemplate:
    'a group, in a line or stacked, of a text field with path "key" and a text field with path "value"',
};
