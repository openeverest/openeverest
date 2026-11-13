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
import { Messages } from './map-field.messages';

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
      fieldParams: { placeholder: Messages.key, ...fieldParams?.key },
      ...(validation?.key && { validation: validation.key }),
    },
    value: {
      uiType: FieldType.Text,
      path: 'value',
      fieldParams: { placeholder: Messages.value, ...fieldParams?.value },
      ...(validation?.value && { validation: validation.value }),
    },
  },
});

// Preprocess sets `_row`; a form that skips it gets the row expanded here.
export const getMapRow = (component: MapComponent): ComponentGroup =>
  component._row ?? expandKeyValueWidget(component.itemTemplate);
