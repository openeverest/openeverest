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

import { describe, expect, it } from 'vitest';
import { FieldType, TopologyUISchemas } from '../../ui-generator.types';
import {
  collectAllSchemaPaths,
  walkLeafComponents,
  walkTopologyComponents,
} from './schema-walker';

describe('schema-walker utils', () => {
  it('walks leaf components recursively and generates nested names', () => {
    const schema: TopologyUISchemas = {
      standalone: {
        sections: {
          base: {
            components: {
              psmdb: {
                uiType: 'group',
                components: {
                  replicas: {
                    uiType: FieldType.Number,
                    path: 'spec.components.psmdb.replicas',
                    fieldParams: {
                      label: 'Replicas',
                    },
                  },
                },
              },
            },
          },
        },
      },
    };

    const collected: string[] = [];

    walkLeafComponents(
      schema.standalone!.sections.base.components,
      ({ generatedName }) => {
        collected.push(generatedName);
      }
    );

    expect(collected).toEqual(['psmdb.replicas']);
  });

  it('walks topology sections in sectionsOrder', () => {
    const schema: TopologyUISchemas = {
      standalone: {
        sections: {
          second: {
            components: {
              fieldB: {
                uiType: FieldType.Text,
                path: 'spec.b',
                fieldParams: {
                  label: 'B',
                },
              },
            },
          },
          first: {
            components: {
              fieldA: {
                uiType: FieldType.Text,
                path: 'spec.a',
                fieldParams: {
                  label: 'A',
                },
              },
            },
          },
        },
        sectionsOrder: ['first', 'second'],
      },
    };

    const order: string[] = [];

    walkTopologyComponents(schema, 'standalone', ({ sectionKey, key }) => {
      order.push(`${sectionKey}.${key}`);
    });

    expect(order).toEqual(['first.fieldA', 'second.fieldB']);
  });

  it('collects target paths from schema components', () => {
    const schema: TopologyUISchemas = {
      standalone: {
        sections: {
          base: {
            components: {
              version: {
                uiType: FieldType.Text,
                path: ['spec.components.psmdb.version', 'spec.components.proxy.version'],
                fieldParams: {
                  label: 'Version',
                },
              },
            },
          },
        },
      },
    };

    const paths = collectAllSchemaPaths(schema.standalone!.sections);

    expect(paths).toEqual(
      new Set([
        'spec.components.psmdb.version',
        'spec.components.proxy.version',
      ])
    );
  });
});
