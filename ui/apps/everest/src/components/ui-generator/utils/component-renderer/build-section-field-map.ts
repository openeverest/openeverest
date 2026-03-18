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
  Section,
  Component,
  ComponentGroup,
} from '../../ui-generator.types';
import { generateFieldId } from './generate-field-id';
import { getComponentTargetPaths } from '../preprocess/normalized-component';

// TODO probably may be improved and be a part of some other function that walks throught
// all section
export const buildSectionFieldMap = (
  sections: { [key: string]: Section },
  sectionsOrder: string[] | undefined
): Record<string, string> => {
  const map: Record<string, string> = {};

  const walkComponents = (
    components: { [key: string]: Component | ComponentGroup },
    sectionKey: string,
    basePath = ''
  ) => {
    Object.entries(components).forEach(([componentKey, comp]) => {
      if (!comp) return;

      const generatedName = basePath
        ? `${basePath}.${componentKey}`
        : componentKey;

      if (comp.uiType === 'group' || comp.uiType === 'hidden') {
        // Recurse into group children
        walkComponents(
          (comp as ComponentGroup).components,
          sectionKey,
          generatedName
        );
        return;
      }

      const leaf = comp as Component;
      const targetPaths = getComponentTargetPaths(leaf);

      if (targetPaths.length > 0) {
        const registerPath = (path: string) => {
          if (!path || typeof path !== 'string') {
            return;
          }

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
        };

        targetPaths.forEach(registerPath);

        // For multipath fields RHF stores value under generated ID, so errors can
        // be reported using this source field name as well.
        map[generateFieldId(leaf, generatedName)] = sectionKey;
      }
    });
  };

  const orderedKeys = sectionsOrder || Object.keys(sections);
  orderedKeys.forEach((sectionKey) => {
    if (sections[sectionKey]) {
      walkComponents(sections[sectionKey].components, sectionKey);
    }
  });

  return map;
};
