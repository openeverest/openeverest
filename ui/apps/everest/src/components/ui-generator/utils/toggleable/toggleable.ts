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

import { z } from 'zod';
import {
  Component,
  ComponentGroup,
  GroupType,
  Section,
  ToggleableMeta,
} from 'components/ui-generator/ui-generator.types';
import {
  getByPath,
  isEmptyFieldValue,
  isSameOrNestedPath,
} from '../object-path/object-path';
import { walkLeafComponents } from '../schema-walker/schema-walker';
import { getComponentTargetPaths } from '../preprocess/normalized-component';
import { getWidgetTargetPaths } from '../widget-targets';

// Reserved root form key holding every toggleable group's form-only switch.
// Postprocess strips it so the switches never reach the API payload.
export const TOGGLEABLE_SWITCHES_KEY = 'toggleable-switches';

const isToggleableGroup = (
  item: Component | ComponentGroup
): item is ComponentGroup =>
  item.uiType === 'group' &&
  'groupType' in item &&
  item.groupType === GroupType.Toggleable;

// Resolved once by preprocess; unset on degraded groups and on schemas that
// were not preprocessed, which then render as plain bordered cards.
export const getToggleableMeta = (
  item: Component | ComponentGroup
): ToggleableMeta | undefined =>
  'components' in item ? item._toggleable : undefined;

// Named after the group's position (section and group keys), which YAML keeps
// unique and form modes never change.
const getToggleableSwitchName = (groupKeyPath: string[]): string =>
  `${TOGGLEABLE_SWITCHES_KEY}.${groupKeyPath.join('~')}`;

// Keys become one form field name; `.`, `~`, brackets or quotes would collide
// or be split by React Hook Form.
const SWITCH_KEY_PATTERN = /^[A-Za-z0-9_-]+$/;

const getGroupPaths = (group: ComponentGroup): string[] => {
  const paths: string[] = [];
  walkLeafComponents(group.components, ({ component }) => {
    paths.push(
      ...getComponentTargetPaths(component),
      ...getWidgetTargetPaths(component)
    );
  });
  return paths;
};

const countPaths = (paths: string[]): Map<string, number> => {
  const counts = new Map<string, number>();
  paths.forEach((path) => counts.set(path, (counts.get(path) ?? 0) + 1));
  return counts;
};

export interface ToggleableScope {
  insideToggleable: boolean;
  // How many fields across the topology write each API path.
  pathUsage: Map<string, number>;
}

export const createToggleableScope = (
  sections: Record<string, Section>
): ToggleableScope => {
  const paths: string[] = [];
  Object.values(sections).forEach((section) => {
    if (!section?.components) return;
    walkLeafComponents(section.components, ({ component }) => {
      paths.push(
        ...getComponentTargetPaths(component),
        ...getWidgetTargetPaths(component)
      );
    });
  });
  return { insideToggleable: false, pathUsage: countPaths(paths) };
};

export type ToggleableDegradeReason =
  | 'unsafe-key'
  | 'no-fields'
  | 'nested'
  | 'overlap';

export interface ToggleableResolution {
  // Set only for a group that acts as a toggleable.
  meta?: ToggleableMeta;
  degradeReason?: ToggleableDegradeReason;
  // For an 'overlap': the path another field outside the group also writes.
  overlappingPath?: string;
  childScope: ToggleableScope;
}

// Decides whether a group acts as a toggleable. Switching one off deletes its
// paths, so it must own them exclusively: a group sharing a path with any field
// outside it, or nested in another toggleable, degrades to a bordered card.
export const resolveToggleable = (
  item: Component | ComponentGroup,
  groupKeyPath: string[],
  scope: ToggleableScope
): ToggleableResolution => {
  if (!isToggleableGroup(item)) return { childScope: scope };
  if (!groupKeyPath.every((key) => SWITCH_KEY_PATTERN.test(key))) {
    return { degradeReason: 'unsafe-key', childScope: scope };
  }

  const childPaths = getGroupPaths(item);
  if (childPaths.length === 0) {
    return { degradeReason: 'no-fields', childScope: scope };
  }
  if (scope.insideToggleable) {
    return { degradeReason: 'nested', childScope: scope };
  }
  const ownUsage = countPaths(childPaths);
  const overlappingPath = childPaths.find(
    (path) => (scope.pathUsage.get(path) ?? 0) > (ownUsage.get(path) ?? 0)
  );
  if (overlappingPath) {
    return { degradeReason: 'overlap', overlappingPath, childScope: scope };
  }

  return {
    meta: {
      switchName: getToggleableSwitchName(groupKeyPath),
      childPaths,
    },
    childScope: { ...scope, insideToggleable: true },
  };
};

const collectFromComponents = (
  components: Record<string, Component | ComponentGroup>,
  metas: ToggleableMeta[]
) => {
  Object.values(components).forEach((item) => {
    if (!('components' in item)) return;
    const meta = getToggleableMeta(item);
    if (meta) metas.push(meta);
    collectFromComponents(item.components, metas);
  });
};

export const collectToggleableMetas = (
  sections: Record<string, Section>
): ToggleableMeta[] => {
  const metas: ToggleableMeta[] = [];
  Object.values(sections).forEach((section) => {
    if (section?.components) {
      collectFromComponents(section.components, metas);
    }
  });
  return metas;
};

// On load a section is on when the saved instance has any of its fields set.
// Schema defaults alone never count (they're not user intent).
export const isToggleableOnInInstance = (
  meta: ToggleableMeta,
  instance: Record<string, unknown>
): boolean =>
  meta.childPaths.some((path) => !isEmptyFieldValue(getByPath(instance, path)));

export const isToggleableOn = (
  meta: ToggleableMeta,
  formValues: Record<string, unknown>
): boolean => getByPath(formValues, meta.switchName) === true;

// API paths owned by switched-off groups: they must be removed from the payload
// (and explicitly deleted on edit, where a deep merge would otherwise keep them).
export const getSwitchedOffPaths = (
  metas: ToggleableMeta[],
  formValues: Record<string, unknown>
): string[] =>
  metas
    .filter((meta) => !isToggleableOn(meta, formValues))
    .flatMap((meta) => meta.childPaths);

// CEL sees a switched-off group's paths as absent, so a rule reading them must
// re-run when that group's switch flips.
export const withToggleableSwitchDependencies = (
  celDependencyGroups: string[][],
  metas: ToggleableMeta[]
): string[][] =>
  celDependencyGroups.map((group) => {
    const switches = metas
      .filter((meta) =>
        meta.childPaths.some((path) =>
          group.some((dep) => isSameOrNestedPath(path, dep))
        )
      )
      .map((meta) => meta.switchName);
    return switches.length > 0 ? [...group, ...switches] : group;
  });

export const getInactiveToggleablePaths = (
  sections: Record<string, Section>,
  formValues: Record<string, unknown>
): string[] =>
  getSwitchedOffPaths(collectToggleableMetas(sections), formValues);

// One optional root entry for all switches: a nested path would make zod require
// the `toggleable-switches` parent object, invalidating forms whose values lack it.
export const toggleableSwitchesSchema = z
  .record(z.string(), z.boolean())
  .optional();
