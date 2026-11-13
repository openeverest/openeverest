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

import type { ComponentGroup } from '../ui-generator.types';
import { Messages } from './map-field.messages';
import type { RowState } from './map-field.types';

// One row's fields: each named for the row ("Key 2"), disabled with the map,
// and the first one focused when the row was just added.
export const toRowAtIndex = (
  row: ComponentGroup,
  index: number,
  { disabled, focus }: RowState
): ComponentGroup => ({
  ...row,
  components: Object.fromEntries(
    Object.entries(row.components).map(([key, item], position) => [
      key,
      'components' in item
        ? item
        : {
            ...item,
            _renderOverrides: {
              accessibleName: `${item.fieldParams?.label || Messages.fieldName(key)} ${index + 1}`,
              disabled,
              autoFocus: focus && position === 0,
            },
          },
    ])
  ),
});
