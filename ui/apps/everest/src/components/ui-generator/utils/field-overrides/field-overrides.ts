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
  isWidgetComponent,
  Section,
} from '../../ui-generator.types';
import { getComponentSourcePath } from '../preprocess/normalized-component';
import { FieldOverride, FieldOverrides } from './field-overrides.types';

const withOverride = <T extends { fieldParams?: FieldOverride }>(
  component: T,
  { disabled, tooltip }: FieldOverride
): T => ({
  ...component,
  fieldParams: {
    ...component.fieldParams,
    ...(disabled !== undefined && { disabled }),
    ...(tooltip !== undefined && { tooltip }),
  },
});

const overrideItem = (
  item: Component | ComponentGroup,
  overrides: FieldOverrides
): Component | ComponentGroup => {
  if ('components' in item) {
    return {
      ...item,
      components: overrideComponents(item.components, overrides),
    };
  }
  const path = getComponentSourcePath(item);
  const override = path ? overrides[path] : undefined;
  return override && !isWidgetComponent(item)
    ? withOverride(item, override)
    : item;
};

const overrideComponents = (
  components: Record<string, Component | ComponentGroup>,
  overrides: FieldOverrides
): Record<string, Component | ComponentGroup> =>
  Object.fromEntries(
    Object.entries(components).map(([key, item]) => [
      key,
      overrideItem(item, overrides),
    ])
  );

// Runtime field behaviour the consumer computes, such as a field disabled by
// the cluster's state. Applied to preprocessed sections, it wins over the
// schema's modes.
export const applyFieldOverrides = (
  sections: Record<string, Section>,
  overrides: FieldOverrides
): Record<string, Section> => {
  if (Object.keys(overrides).length === 0) {
    return sections;
  }
  return Object.fromEntries(
    Object.entries(sections).map(([sectionKey, section]) => [
      sectionKey,
      {
        ...section,
        components: overrideComponents(section.components, overrides),
      },
    ])
  );
};
