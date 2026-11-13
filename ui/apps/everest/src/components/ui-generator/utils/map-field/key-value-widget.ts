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
  type KeyValueWidget,
  type MapComponent,
  FieldType,
  GroupType,
} from '../../ui-generator.types';
import { IGNORED_ROW_FIELD_PARAMS } from './map-field.constants';
import { Messages } from './map-field.messages';

// Drops what a row's field can't act on: params it has no use for, and CEL,
// which a row can't evaluate yet.
const without = <T extends object>(options: T, ignored: string[]): T => {
  const copy = { ...options };
  ignored.forEach((option) => Reflect.deleteProperty(copy, option));
  return copy;
};

// The row a keyValue widget stands for: a text `key` and a text `value` side
// by side, with the widget's params and rules over theirs.
export const expandKeyValueWidget = ({
  fieldParams,
  validation,
}: KeyValueWidget): ComponentGroup => ({
  uiType: 'group',
  groupType: GroupType.Line,
  components: {
    key: {
      uiType: FieldType.Text,
      path: 'key',
      fieldParams: {
        placeholder: Messages.key,
        ...without(fieldParams?.key ?? {}, IGNORED_ROW_FIELD_PARAMS),
      },
      ...(validation?.key && {
        validation: without(validation.key, ['celExpressions']),
      }),
    },
    value: {
      uiType: FieldType.Text,
      path: 'value',
      fieldParams: {
        placeholder: Messages.value,
        ...without(fieldParams?.value ?? {}, IGNORED_ROW_FIELD_PARAMS),
      },
      ...(validation?.value && {
        validation: without(validation.value, ['celExpressions']),
      }),
    },
  },
});

// Preprocess sets `_row`; a form that skips it gets the row expanded here.
export const getMapRow = (component: MapComponent): ComponentGroup =>
  component._row ?? expandKeyValueWidget(component.itemTemplate);
