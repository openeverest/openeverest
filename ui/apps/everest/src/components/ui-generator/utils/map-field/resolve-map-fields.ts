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
  type Component,
  type ComponentGroup,
  type Section,
  FieldType,
  isMapComponent,
} from '../../ui-generator.types';
import { isSameOrNestedPath } from '../object-path';
import { getComponentTargetPaths } from '../preprocess/normalized-component';
import { walkLeafComponents } from '../schema-walker';
import { getWidgetTargetPaths } from '../widget-targets';

export type IgnoredMapReason = 'unsupported' | 'multi-path' | 'overlap';

export interface IgnoredMapField {
  // Section key followed by group keys and the field key, e.g. `advanced.labels`.
  keyPath: string;
  reason: IgnoredMapReason;
  // For an 'overlap': the path of the other field the map's value clashes with.
  overlappingPath?: string;
}

interface WritingField {
  keyPath: string;
  component: Component;
  paths: string[];
}

const findOverlappingPath = (
  map: WritingField,
  fields: WritingField[]
): string | undefined =>
  fields
    .filter(({ keyPath }) => keyPath !== map.keyPath)
    .flatMap(({ paths }) => paths)
    .find((path) => map.paths.some((own) => isSameOrNestedPath(path, own)));

const getIgnoredReason = (
  component: Component
): IgnoredMapReason | undefined => {
  if (component.uiType !== FieldType.Map) return undefined;
  if (!isMapComponent(component)) return 'unsupported';
  // An edit reads one path, so the others would drift from it.
  if (getComponentTargetPaths(component).length > 1) return 'multi-path';
  return undefined;
};

const withoutFields = (
  components: Record<string, Component | ComponentGroup>,
  parentKeyPath: string,
  ignoredKeyPaths: Set<string>
): Record<string, Component | ComponentGroup> =>
  Object.fromEntries(
    Object.entries(components)
      .filter(([key]) => !ignoredKeyPaths.has(`${parentKeyPath}.${key}`))
      .map(([key, item]) => [
        key,
        'components' in item && item.components
          ? {
              ...item,
              components: withoutFields(
                item.components,
                `${parentKeyPath}.${key}`,
                ignoredKeyPaths
              ),
            }
          : item,
      ])
  );

// A map field owns the whole value at its path. So the UI renders only a
// shape it implements, with one path, and only when no other field writes that
// path, a path inside it or one it sits in. Any other map field is dropped,
// like an unknown uiType: not rendered or sent, and an edit keeps its saved
// value. Runs before the toggleable rules, which then don't see a dropped map.
export const resolveMapFields = (
  sections: Record<string, Section>
): { sections: Record<string, Section>; ignored: IgnoredMapField[] } => {
  const fields: WritingField[] = [];
  const ignored: IgnoredMapField[] = [];

  Object.entries(sections).forEach(([sectionKey, section]) => {
    if (!section?.components) return;
    walkLeafComponents(
      section.components,
      ({ component, generatedName }) => {
        const reason = getIgnoredReason(component);
        if (reason) {
          ignored.push({ keyPath: generatedName, reason });
          return;
        }
        fields.push({
          keyPath: generatedName,
          component,
          paths: [
            ...getComponentTargetPaths(component),
            ...getWidgetTargetPaths(component),
          ],
        });
      },
      sectionKey
    );
  });

  fields
    .filter(({ component }) => isMapComponent(component))
    .forEach((map) => {
      const overlappingPath = findOverlappingPath(map, fields);
      if (overlappingPath) {
        ignored.push({
          keyPath: map.keyPath,
          reason: 'overlap',
          overlappingPath,
        });
      }
    });

  if (ignored.length === 0) return { sections, ignored };

  const ignoredKeyPaths = new Set(ignored.map(({ keyPath }) => keyPath));
  return {
    sections: Object.fromEntries(
      Object.entries(sections).map(([sectionKey, section]) => [
        sectionKey,
        section?.components
          ? {
              ...section,
              components: withoutFields(
                section.components,
                sectionKey,
                ignoredKeyPaths
              ),
            }
          : section,
      ])
    ),
    ignored,
  };
};
