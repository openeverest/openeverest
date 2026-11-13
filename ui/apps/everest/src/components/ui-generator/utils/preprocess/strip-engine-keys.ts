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
  Section,
} from '../../ui-generator.types';
import { isPlainObject } from '../object-path';

export interface StrippedEngineKey {
  // Section key followed by group keys and the component key, e.g.
  // `advanced.labels`.
  keyPath: string;
  key: string;
}

// Keys starting with `_` are the engine's: preprocess derives `_normalized`,
// `_row`, `_toggleable`, `_widgetTargets`, and the later stages trust them.
const isEngineKey = (key: string): boolean => key.startsWith('_');

const withoutEngineKeys = <T extends object>(
  item: T,
  keyPath: string,
  stripped: StrippedEngineKey[]
): T => {
  const copy = { ...item };
  Object.keys(copy)
    .filter(isEngineKey)
    .forEach((key) => {
      Reflect.deleteProperty(copy, key);
      stripped.push({ keyPath, key });
    });
  return copy;
};

const stripComponents = (
  components: Record<string, Component | ComponentGroup>,
  parentKeyPath: string,
  stripped: StrippedEngineKey[]
): Record<string, Component | ComponentGroup> =>
  Object.fromEntries(
    Object.entries(components).map(([key, item]) => {
      if (!isPlainObject(item)) return [key, item];
      const keyPath = `${parentKeyPath}.${key}`;
      const kept = withoutEngineKeys(item, keyPath, stripped);
      return [
        key,
        'components' in kept && kept.components
          ? {
              ...kept,
              components: stripComponents(kept.components, keyPath, stripped),
            }
          : kept,
      ];
    })
  );

// A schema can't set the engine's keys: one would pass for the engine's own,
// e.g. redirect where a widget writes. They are dropped before anything reads
// the schema.
export const stripEngineKeys = (
  sections: Record<string, Section>
): { sections: Record<string, Section>; stripped: StrippedEngineKey[] } => {
  const stripped: StrippedEngineKey[] = [];
  const kept = Object.fromEntries(
    Object.entries(sections).map(([sectionKey, section]) => [
      sectionKey,
      section?.components
        ? {
            ...section,
            components: stripComponents(
              section.components,
              sectionKey,
              stripped
            ),
          }
        : section,
    ])
  );
  return { sections: kept, stripped };
};
