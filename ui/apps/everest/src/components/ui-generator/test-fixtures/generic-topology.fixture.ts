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
  FieldType,
  FormMode,
} from 'components/ui-generator/ui-generator.types';
import type {
  Component,
  NumberValidation,
  Section,
  TopologyUISchemas,
} from 'components/ui-generator/ui-generator.types';

const makeNumber = (
  path: string,
  overrides: Partial<Component> = {}
): Component =>
  ({
    uiType: FieldType.Number,
    path,
    fieldParams: { label: path },
    ...overrides,
  }) as Component;

export const genericResourceSections: Record<string, Section> = {
  resources: {
    label: 'Resources',
    components: {
      numberOfNodes: makeNumber('spec.components.engine.replicas', {
        validation: {
          required: true,
          min: 1,
          int: true,
          celExpressions: [
            {
              celExpr: 'spec.components.engine.replicas % 2 == 1',
              message: 'The number of nodes must be odd',
            },
          ],
          modes: {
            [FormMode.Edit]: {
              celExpressions: [
                {
                  celExpr:
                    '!(spec.components.engine.replicas == 1 && original.spec.components.engine.replicas > 1)',
                  message: 'Cannot scale down to a single node',
                },
              ],
            },
          },
        } as NumberValidation,
      }),
      shards: makeNumber('spec.sharding.shards', {
        validation: {
          required: true,
          min: 1,
          int: true,
          modes: {
            [FormMode.Edit]: {
              celExpressions: [
                {
                  celExpr:
                    'spec.sharding.shards >= original.spec.sharding.shards',
                  message: 'Number of shards cannot be decreased',
                },
              ],
            },
          },
        } as NumberValidation,
      }),
      disk: makeNumber('spec.components.engine.storage.size', {
        validation: {
          required: true,
          min: 1,
          int: true,
          modes: {
            [FormMode.Edit]: {
              celExpressions: [
                {
                  celExpr:
                    'spec.components.engine.storage.size >= original.spec.components.engine.storage.size',
                  message: 'Disk size cannot be decreased',
                },
              ],
            },
          },
        } as NumberValidation,
      }),
      configServers: makeNumber('spec.components.configServer.replicas', {
        validation: {
          celExpressions: [
            {
              celExpr:
                '!(spec.components.engine.replicas > 1 && spec.components.configServer.replicas == 1)',
              message:
                'Config servers cannot be 1 if the number of nodes is greater than 1',
            },
          ],
        } as NumberValidation,
      }),
    },
  },
};

export const genericOriginalInstance = {
  spec: {
    components: {
      engine: {
        replicas: 3,
        storage: { size: 25 },
      },
      configServer: { replicas: 3 },
    },
    sharding: { shards: 2 },
  },
};

export const genericTopologySchemaFixture: TopologyUISchemas = {
  baseline: {
    sections: {
      ...genericResourceSections,
      advanced: {
        label: 'Advanced',
        components: {
          profileLevel: {
            uiType: FieldType.Text,
            path: 'spec.advanced.profileLevel',
            fieldParams: { label: 'Profile level', defaultValue: 'normal' },
          },
        },
      },
    },
    sectionsOrder: ['resources', 'advanced'],
  },
};

export const cloneFixture = <T>(data: T): T =>
  typeof structuredClone === 'function'
    ? structuredClone(data)
    : (JSON.parse(JSON.stringify(data)) as T);

export const getFixtureSections = (
  topologyKey: keyof TopologyUISchemas = 'baseline'
): Record<string, Section> => {
  const topology = genericTopologySchemaFixture[topologyKey as string];
  return cloneFixture(topology.sections);
};

export const pickFixtureSections = (
  sectionKeys: string[],
  topologyKey: keyof TopologyUISchemas = 'baseline'
): Record<string, Section> => {
  const sections = getFixtureSections(topologyKey);
  return Object.fromEntries(
    sectionKeys
      .filter((sectionKey) => Boolean(sections[sectionKey]))
      .map((sectionKey) => [sectionKey, sections[sectionKey]])
  );
};
