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
  Component,
  ComponentGroup,
} from 'components/ui-generator/ui-generator.types';
import { generateFieldId } from '../component-renderer/generate-field-id';
import { UI_TYPE_DEFAULT_VALUE } from 'components/ui-generator/constants';

export const buildDefaultsFromComponents = (
  components: { [key: string]: Component | ComponentGroup },
  basePath: string = '',
  /**
   * When true, only includes defaults explicitly set in the schema's
   * `fieldParams.defaultValue`. Skips the fallback to UI_TYPE_DEFAULT_VALUE
   * (which assigns empty string / false / etc. based on uiType).
   * This is useful when applying provider-driven defaults on top of an
   * already-initialized form — we want to set only what the schema declares,
   * without resetting other fields to generic empty values.
   */
  buildOnlySchemaDefaults: boolean = false
): Record<string, unknown> => {
  const result: Record<string, unknown> = {};

  Object.entries(components).forEach(([key, item]) => {
    const generatedName = basePath ? `${basePath}.${key}` : key;
    const fieldId = generateFieldId(item, generatedName);

    if (item.uiType === 'group' && 'components' in item) {
      // Recursively process nested components
      const nestedDefaults = buildDefaultsFromComponents(
        (item as ComponentGroup).components,
        generatedName,
        buildOnlySchemaDefaults
      );
      Object.assign(result, nestedDefaults);
    } else {
      const component = item as Component;
      if (
        'fieldParams' in component &&
        component.fieldParams?.defaultValue !== undefined
      ) {
        result[fieldId] = component.fieldParams.defaultValue;
      } else if (!buildOnlySchemaDefaults) {
        result[fieldId] = UI_TYPE_DEFAULT_VALUE[component.uiType];
      }
    }
  });

  return result;
};
