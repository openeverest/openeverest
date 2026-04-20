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
  FormMode,
  Section,
} from 'components/ui-generator/ui-generator.types';
import { UI_TYPE_DEFAULT_VALUE } from 'components/ui-generator/constants';
import { stripBadgeFromValue } from '../badge-to-api/badge-to-api';
import { generateFieldId } from '../component-renderer/generate-field-id';
import { getComponentSourcePath } from '../preprocess/normalized-component';
import { getByPath } from '../object-path/object-path';
import { convertToNestedObject } from './convert-to-nested-object';

/*
 Walks schema components across all sections and extracts current values
 from an existing instance, producing form-compatible default values.

 In Edit mode: only instance values are used. Missing values are left undefined.
 In New/Restore/Import modes: falls back to schema defaultValue, then type default.
*/
const extractFlat = (
  components: Record<string, Component | ComponentGroup>,
  instance: Record<string, unknown>,
  basePath: string,
  formMode?: FormMode
): Record<string, unknown> => {
  const result: Record<string, unknown> = {};

  for (const [key, item] of Object.entries(components)) {
    const generatedName = basePath ? `${basePath}.${key}` : key;

    if (
      (item.uiType === 'group' || item.uiType === 'hidden') &&
      'components' in item
    ) {
      Object.assign(
        result,
        extractFlat(
          (item as ComponentGroup).components,
          instance,
          generatedName,
          formMode
        )
      );
      continue;
    }

    const component = item as Component;
    const fieldId = generateFieldId(component, generatedName);
    const sourcePath = getComponentSourcePath(component);

    // Try reading from instance first
    if (sourcePath) {
      const instanceValue = getByPath(instance, sourcePath);
      if (instanceValue !== undefined) {
        result[fieldId] = stripBadgeFromValue(
          instanceValue,
          component.fieldParams?.badgeToApi
            ? component.fieldParams.badge
            : undefined
        );
        continue;
      }
    }

    // In Edit mode, only use instance values — no schema/type defaults
    if (formMode === FormMode.Edit) {
      continue;
    }

    // Fallback to schema default, then type default
    if (component.fieldParams?.defaultValue !== undefined) {
      result[fieldId] = component.fieldParams.defaultValue;
    } else {
      result[fieldId] = UI_TYPE_DEFAULT_VALUE[component.uiType];
    }
  }

  return result;
};

export const extractInstanceValues = (
  sections: Record<string, Section>,
  instance: Record<string, unknown>,
  formMode?: FormMode
): Record<string, unknown> => {
  const flatValues: Record<string, unknown> = {};

  for (const [sectionKey, section] of Object.entries(sections)) {
    if (section?.components) {
      Object.assign(
        flatValues,
        extractFlat(section.components, instance, sectionKey, formMode)
      );
    }
  }

  return convertToNestedObject(flatValues);
};
