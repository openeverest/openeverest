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
  TopologyUISchemas,
} from '../../ui-generator.types';
import { getComponentTargetPaths } from '../preprocess/normalized-component';

export interface LeafComponentInfo {
  component: Component;
  key: string;
  generatedName: string;
}

export const walkLeafComponents = (
  components: Record<string, Component | ComponentGroup>,
  visitor: (info: LeafComponentInfo) => void | false,
  basePath = ''
): void => {
  for (const [key, item] of Object.entries(components)) {
    const generatedName = basePath ? `${basePath}.${key}` : key;

    if (item.uiType === 'group' || item.uiType === 'hidden') {
      const group = item as ComponentGroup;
      if (group.components) {
        walkLeafComponents(group.components, visitor, generatedName);
      }
      continue;
    }

    const stop = visitor({
      component: item as Component,
      key,
      generatedName,
    });

    if (stop === false) return;
  }
};

export const walkTopologyComponents = (
  schema: TopologyUISchemas,
  selectedTopology: string,
  visitor: (info: LeafComponentInfo & { sectionKey: string }) => void | false
): void => {
  const topology = schema[selectedTopology];
  if (!topology?.sections) return;

  const order = topology.sectionsOrder ?? Object.keys(topology.sections);

  for (const sectionKey of order) {
    const section = topology.sections[sectionKey];
    if (!section?.components) continue;

    walkLeafComponents(section.components, (info) =>
      visitor({ ...info, sectionKey })
    );
  }
};

export const collectAllSchemaPaths = (
  sections: Record<string, Section>
): Set<string> => {
  const paths = new Set<string>();

  for (const section of Object.values(sections)) {
    if (!section?.components) continue;

    walkLeafComponents(section.components, ({ component }) => {
      getComponentTargetPaths(component).forEach((path) => {
        if (path) paths.add(path);
      });
    });
  }

  return paths;
};
