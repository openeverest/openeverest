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

import type {
  Component,
  ComponentGroup,
} from 'components/ui-generator/ui-generator.types';
import {
  getByPath,
  formatDisplayValue,
} from 'components/ui-generator/utils/object-path';
import { getComponentTargetPaths } from 'components/ui-generator/utils/preprocess/normalized-component';

export type SectionField = {
  label: string;
  path: string;
  value: string;
};

export const collectSectionFields = (
  components: Record<string, Component | ComponentGroup>,
  instance: Record<string, unknown>,
  componentsOrder?: string[]
): SectionField[] => {
  const fields: SectionField[] = [];
  const keys = componentsOrder ?? Object.keys(components);

  for (const key of keys) {
    const comp = components[key];
    if (!comp) continue;

    if (comp.uiType === 'group' || comp.uiType === 'hidden') {
      const group = comp as ComponentGroup;
      if (group.components) {
        fields.push(
          ...collectSectionFields(
            group.components,
            instance,
            group.componentsOrder
          )
        );
      }
      continue;
    }

    const component = comp as Component;
    const paths = getComponentTargetPaths(component);
    const path = paths[0];
    if (!path) continue;

    fields.push({
      label: component.fieldParams?.label ?? key,
      path,
      value: formatDisplayValue(getByPath(instance, path)),
    });
  }

  return fields;
};
