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

import type { Section } from '../../ui-generator.types';
import { generateFieldId } from './generate-field-id';
import { getComponentTargetPaths } from '../preprocess/normalized-component';
import { walkLeafComponents } from '../schema-walker';

export const buildSectionFieldMap = (
  sections: { [key: string]: Section },
  sectionsOrder: string[] | undefined
): Record<string, string> => {
  const map: Record<string, string> = {};

  const orderedKeys = sectionsOrder || Object.keys(sections);
  orderedKeys.forEach((sectionKey) => {
    const section = sections[sectionKey];
    if (!section?.components) return;

    walkLeafComponents(section.components, ({ component, generatedName }) => {
      const targetPaths = getComponentTargetPaths(component);

      if (targetPaths.length > 0) {
        targetPaths.forEach((path) => {
          if (!path || typeof path !== 'string') return;

          map[path] = sectionKey;
          // Register ALL intermediate path prefixes so that Zod errors at parent
          // nodes (e.g. when a nested object is undefined on topology switch) still
          // map to the correct step, rather than falling back to the top-level key
          // which may belong to a completely different step.
          const parts = path.split('.');
          for (let i = 1; i < parts.length; i++) {
            const prefix = parts.slice(0, i).join('.');
            if (!(prefix in map)) {
              map[prefix] = sectionKey;
            }
          }
        });

        // For multipath fields RHF stores value under generated ID, so errors can
        // be reported using this source field name as well.
        map[generateFieldId(component, generatedName)] = sectionKey;
      }
    });
  });

  return map;
};
