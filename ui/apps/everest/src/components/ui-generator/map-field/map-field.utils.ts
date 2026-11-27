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

import { orderComponents } from 'components/ui-generator/utils/component-renderer/order-components';
import { getRowFieldRole } from 'components/ui-generator/utils/map-field';
import type { ComponentGroup } from '../ui-generator.types';
import { Messages } from './map-field.messages';
import type { RowState } from './map-field.types';

// One row's fields: each named for the row ("Key 2"), disabled with the map,
// showing a cut-off value on hover, and the first one focused when the row was
// just added.
export const toRowAtIndex = (
  row: ComponentGroup,
  index: number,
  { disabled, focus }: RowState
): ComponentGroup => {
  // The first as the row draws them, which `componentsOrder` may change.
  const [firstDrawn] = orderComponents(row.components, row.componentsOrder);
  return {
    ...row,
    components: Object.fromEntries(
      Object.entries(row.components).map(([key, item]) => [
        key,
        'components' in item
          ? item
          : {
              ...item,
              _renderOverrides: {
                accessibleName: `${item.fieldParams?.label || Messages.fieldName(getRowFieldRole(item) ?? key)} ${index + 1}`,
                disabled,
                autoFocus: focus && key === firstDrawn?.[0],
                showFullValueOnHover: true,
              },
            },
      ])
    ),
  };
};
