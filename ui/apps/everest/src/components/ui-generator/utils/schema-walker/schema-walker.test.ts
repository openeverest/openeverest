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
import type {
  Component,
  ComponentGroup,
  Section,
} from '../../ui-generator.types';
import {
  collectAllSchemaPaths,
  walkLeafComponents,
  walkTopologyComponents,
} from './schema-walker';

describe('schema-walker utils', () => {
  describe('walkLeafComponents', () => {
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

    it('walks hidden groups the same as regular groups', () => {
      const components: Record<string, Component | ComponentGroup> = {
        hidden: {
          uiType: 'hidden' as const,
          components: {
            secret: {
              uiType: FieldType.Text,
              path: 'spec.secret',
              fieldParams: { label: 'Secret' },
            },
          },
        },
      };

      const collected: string[] = [];
      walkLeafComponents(components, ({ generatedName }) => {
        collected.push(generatedName);
      });

      expect(collected).toEqual(['hidden.secret']);
    });

    it('stops early when visitor returns false', () => {
      const components: Record<string, Component | ComponentGroup> = {
        fieldA: {
          uiType: FieldType.Text,
          path: 'spec.a',
          fieldParams: { label: 'A' },
        },
        fieldB: {
          uiType: FieldType.Text,
          path: 'spec.b',
          fieldParams: { label: 'B' },
        },
      };

      const collected: string[] = [];
      walkLeafComponents(components, ({ key }) => {
        collected.push(key);
        if (key === 'fieldA') return false;
      });

      expect(collected).toEqual(['fieldA']);
    });

    it('handles empty components object', () => {
      const collected: string[] = [];
      walkLeafComponents({}, ({ key }) => {
        collected.push(key);
      });

      expect(collected).toEqual([]);
    });
  });

  describe('walkTopologyComponents', () => {
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

    it('is a no-op for a non-existent topology', () => {
      const schema: TopologyUISchemas = {};
      const collected: string[] = [];

      walkTopologyComponents(schema, 'missing', ({ key }) => {
        collected.push(key);
      });

      expect(collected).toEqual([]);
    });

    it('falls back to Object.keys order when sectionsOrder is absent', () => {
      const schema: TopologyUISchemas = {
        ha: {
          sections: {
            only: {
              components: {
                field: {
                  uiType: FieldType.Text,
                  path: 'spec.x',
                  fieldParams: { label: 'X' },
                },
              },
            },
          },
        },
      };

      const collected: string[] = [];
      walkTopologyComponents(schema, 'ha', ({ sectionKey, key }) => {
        collected.push(`${sectionKey}.${key}`);
      });

      expect(collected).toEqual(['only.field']);
    });
  });

  describe('collectAllSchemaPaths', () => {
    it('collects target paths from schema components', () => {
      const schema: TopologyUISchemas = {
        standalone: {
          sections: {
            base: {
              components: {
                version: {
                  uiType: FieldType.Text,
                  path: [
                    'spec.components.psmdb.version',
                    'spec.components.proxy.version',
                  ],
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

    it('returns an empty set for empty sections', () => {
      expect(collectAllSchemaPaths({})).toEqual(new Set());
    });

    it('skips components without paths', () => {
      const sections: Record<string, Section> = {
        s: {
          components: {
            noop: {
              uiType: FieldType.Text,
              id: 'noop-field',
              fieldParams: { label: 'No path' },
            },
          },
        },
      };

      expect(collectAllSchemaPaths(sections)).toEqual(new Set());
    });
  });
});
