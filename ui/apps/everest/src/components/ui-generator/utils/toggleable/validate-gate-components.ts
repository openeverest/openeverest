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
  FieldType,
  GroupType,
  TopologyUISchemas,
} from 'components/ui-generator/ui-generator.types';

export const validateGateComponents = (schema: TopologyUISchemas): void => {
  if (!schema || typeof schema !== 'object') return;

  Object.values(schema).forEach((topology) => {
    if (
      !topology ||
      typeof topology !== 'object' ||
      !('sections' in topology)
    ) {
      return;
    }

    Object.values(topology.sections).forEach((section) => {
      if (!section?.components) return;
      validateGroupComponents(section.components, null);
    });
  });
};

const validateGroupComponents = (
  components: Record<string, Component | ComponentGroup>,
  parentGroup: ComponentGroup | null
): void => {
  let gateCount = 0;

  Object.entries(components).forEach(([key, item]) => {
    if (
      (item.uiType === 'group' || item.uiType === 'hidden') &&
      'components' in item
    ) {
      const group = item as ComponentGroup;
      validateGroupComponents(group.components, group);
    } else {
      const comp = item as Component;
      if (comp.gate === true) {
        if (
          comp.uiType !== FieldType.Toggle &&
          (comp.uiType as string) !== 'toggle'
        ) {
          throw new Error(
            `Component "${key}" has gate: true but uiType is "${comp.uiType}". Only toggle components can be gate switches.`
          );
        }

        if (!parentGroup || parentGroup.groupType !== GroupType.Toggleable) {
          throw new Error(
            `Component "${key}" has gate: true but is placed outside a toggleable group.`
          );
        }

        gateCount++;
        if (gateCount > 1) {
          throw new Error(
            `Toggleable group "${parentGroup.label ?? 'unknown'}" has more than one gate component.`
          );
        }
      }
    }
  });
};
