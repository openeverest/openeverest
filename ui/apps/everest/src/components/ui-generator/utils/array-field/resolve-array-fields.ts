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
  isArrayComponent,
} from '../../ui-generator.types';
import { isSameOrNestedPath } from '../object-path';
import { getComponentTargetPaths } from '../preprocess/normalized-component';
import { walkLeafComponents } from '../schema-walker';
import { getWidgetTargetPaths } from '../widget-targets';

export type IgnoredArrayReason = 'unsupported' | 'overlap';

export interface IgnoredArrayField {
  // Section key followed by group keys and the field key, e.g. `advanced.labels`.
  keyPath: string;
  reason: IgnoredArrayReason;
  // For an 'overlap': the path of the other field the array's value clashes with.
  overlappingPath?: string;
}

interface WritingField {
  keyPath: string;
  component: Component;
  paths: string[];
}

const findOverlappingPath = (
  array: WritingField,
  fields: WritingField[]
): string | undefined =>
  fields
    .filter(({ keyPath }) => keyPath !== array.keyPath)
    .flatMap(({ paths }) => paths)
    .find((path) => array.paths.some((own) => isSameOrNestedPath(path, own)));

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

// An array field owns the whole value at its paths. So the UI renders only a
// shape it implements, and only when no other field writes those paths, a path
// inside them or one they sit in. Any other array field is dropped, like an
// unknown uiType: not rendered or sent, and an edit keeps its saved value.
// Runs before the toggleable rules, which then don't see a dropped array.
export const resolveArrayFields = (
  sections: Record<string, Section>
): { sections: Record<string, Section>; ignored: IgnoredArrayField[] } => {
  const fields: WritingField[] = [];
  const ignored: IgnoredArrayField[] = [];

  Object.entries(sections).forEach(([sectionKey, section]) => {
    if (!section?.components) return;
    walkLeafComponents(
      section.components,
      ({ component, generatedName }) => {
        if (
          component.uiType === FieldType.Array &&
          !isArrayComponent(component)
        ) {
          ignored.push({ keyPath: generatedName, reason: 'unsupported' });
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
    .filter(({ component }) => isArrayComponent(component))
    .forEach((array) => {
      const overlappingPath = findOverlappingPath(array, fields);
      if (overlappingPath) {
        ignored.push({
          keyPath: array.keyPath,
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
