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
  type ModeAwareValidation,
  type TextFieldParams,
  type TextValidation,
  FieldType,
  GroupType,
} from 'components/ui-generator/ui-generator.types';

interface RowField {
  fieldParams?: TextFieldParams;
  validation?: ModeAwareValidation<TextValidation>;
}

// A key/value map's row as a provider writes it in `itemTemplate`.
export const keyValueRow = ({
  key,
  value,
}: { key?: RowField; value?: RowField } = {}): ComponentGroup => ({
  uiType: 'group',
  groupType: GroupType.Line,
  components: {
    key: { uiType: FieldType.Text, path: 'key', fieldParams: {}, ...key },
    value: { uiType: FieldType.Text, path: 'value', fieldParams: {}, ...value },
  },
});

// The same row in a YAML schema, after `itemTemplate:`.
export const KEY_VALUE_ROW_YAML =
  '{ uiType: group, groupType: line, components: { key: { uiType: text, path: key }, value: { uiType: text, path: value } } }';
