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
  FormMode,
  Section,
} from '../../ui-generator.types';

const hasEditableComponent = (
  components: Record<string, Component | ComponentGroup>,
  mode: FormMode
): boolean => {
  for (const item of Object.values(components)) {
    if (
      (item.uiType === 'group' || item.uiType === 'hidden') &&
      'components' in item
    ) {
      if (hasEditableComponent((item as ComponentGroup).components, mode)) {
        return true;
      }
      continue;
    }

    const component = item as Component;

    // Component-level modes: check if uiType is overridden to 'hidden'
    const componentOverrides = component.modes?.[mode];
    if (componentOverrides?.uiType === 'hidden') {
      continue;
    }

    // FieldParams-level modes: check if disabled or readOnly
    const fpOverrides = component.fieldParams?.modes?.[mode];
    if (fpOverrides?.disabled || fpOverrides?.readOnly) {
      continue;
    }

    // Also check base fieldParams disabled/readOnly (without mode overrides)
    const fp = component.fieldParams as Record<string, unknown> | undefined;
    if (fp?.disabled || fp?.readOnly) {
      continue;
    }

    // A field without blocking overrides is editable
    return true;
  }

  return false;
};

export const isSectionEditable = (
  section: Section,
  mode: FormMode
): boolean => {
  if (!section?.components) return false;
  return hasEditableComponent(section.components, mode);
};
