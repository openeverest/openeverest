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
  CommonValidation,
  FormMode,
  ModeAwareValidation,
  Section,
} from '../../ui-generator.types';

const stripCelFromValidation = (
  validation: ModeAwareValidation<CommonValidation>
): ModeAwareValidation<CommonValidation> => {
  const { modes, ...baseWithCel } = validation;
  const { celExpressions: baseCelExpressions, ...baseRest } = baseWithCel;
  void baseCelExpressions;
  const result: ModeAwareValidation<CommonValidation> = {
    ...baseRest,
  } as ModeAwareValidation<CommonValidation>;

  if (modes) {
    result.modes = {};
    for (const [key, branch] of Object.entries(modes)) {
      if (branch) {
        const { celExpressions: branchCelExpressions, ...rest } = branch;
        void branchCelExpressions;
        result.modes[key as FormMode] = rest;
      }
    }
  }

  return result;
};

const applyModeToComponent = (
  item: Component | ComponentGroup,
  mode: FormMode
): Component | ComponentGroup => {
  if (
    (item.uiType === 'group' || item.uiType === 'hidden') &&
    'components' in item
  ) {
    const group = item as ComponentGroup;
    return {
      ...group,
      components: applyModeToComponents(group.components, mode),
    };
  }

  const component = item as Component;

  // 1. Apply component-level mode overrides (uiType)
  const componentOverrides = component.modes?.[mode];
  if (componentOverrides?.uiType === 'hidden') {
    return {
      ...component,
      uiType: 'hidden',
      // Strip CEL expressions so hidden field's own validation is excluded
      validation: component.validation
        ? stripCelFromValidation(
            component.validation as ModeAwareValidation<CommonValidation>
          )
        : undefined,
    } as Component;
  }

  // 2. Apply fieldParams-level mode overrides (disabled, readOnly)
  const fieldParamsOverrides = component.fieldParams?.modes?.[mode];
  if (fieldParamsOverrides) {
    const { modes: _fpModes, ...restFieldParams } =
      component.fieldParams as Record<string, unknown>;
    void _fpModes;
    return {
      ...component,
      fieldParams: { ...restFieldParams, ...fieldParamsOverrides },
    } as Component;
  }

  return component;
};

const applyModeToComponents = (
  components: Record<string, Component | ComponentGroup>,
  mode: FormMode
): Record<string, Component | ComponentGroup> =>
  Object.fromEntries(
    Object.entries(components).map(([key, item]) => [
      key,
      applyModeToComponent(item, mode),
    ])
  );

export const applyModeOverrides = (
  sections: Record<string, Section>,
  mode: FormMode
): Record<string, Section> =>
  Object.fromEntries(
    Object.entries(sections).map(([sectionKey, section]) => [
      sectionKey,
      {
        ...section,
        components: applyModeToComponents(section.components, mode),
      },
    ])
  );
