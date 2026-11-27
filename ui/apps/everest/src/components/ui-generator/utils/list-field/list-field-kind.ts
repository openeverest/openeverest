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

import {
  type ComponentGroup,
  type ListItemTemplate,
  FieldType,
  GroupType,
} from '../../ui-generator.types';
import { toDisplayText } from '../rows-field/rows';
import type { RowsFieldKind } from '../rows-field/rows-field.types';
import { LIST_ROW_ROLES, isStringList, toList } from './list-field';
import { Messages } from './list-field.messages';

const [ENTRY] = LIST_ROW_ROLES;

// A list declares one field, the entry itself. As a row it is that field
// alone, writing the entry in the row.
export const toListRow = (itemTemplate: ListItemTemplate): ComponentGroup => ({
  uiType: 'group',
  groupType: GroupType.Line,
  components: {
    [ENTRY]: {
      ...itemTemplate,
      fieldParams: itemTemplate.fieldParams ?? {},
      path: ENTRY,
    },
  },
});

// What a list is, among the fields edited as rows: a row holds one value,
// which is the entry, and the list may hold it twice unless `unique` is set.
export const LIST_FIELD_KIND: RowsFieldKind = {
  uiType: FieldType.List,
  roles: LIST_ROW_ROLES,
  emptyRow: { [ENTRY]: '' },
  identity: ENTRY,
  isUnique: (validation) => !!validation?.unique,
  messages: {
    identityRequired: Messages.valueRequired,
    duplicate: Messages.duplicateValue,
  },
  toEntries: (value) => {
    const list = toList(value);
    return Array.isArray(list)
      ? list.map((entry) => ({ value: toDisplayText(entry) }))
      : [];
  },
  isValidDefault: isStringList,
  valueShape: 'list of strings',
  supportedItemTemplate: 'a text field with no path or id',
};
