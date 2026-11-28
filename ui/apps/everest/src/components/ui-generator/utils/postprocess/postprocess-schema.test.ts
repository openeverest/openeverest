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
import { postprocessSchemaData } from './postprocess-schema';
import {
  FieldType,
  TopologyUISchemas,
  WIDGET_UI_TYPE,
  WidgetType,
} from '../../ui-generator.types';
import {
  parentChildTopologySchema,
  twoTopologySchema,
} from '../topology-scope/__mocks__/topology-schemas';

describe('postprocessSchemaData', () => {
  it('removes empty values recursively and preserves meaningful falsy values', () => {
    const input = {
      provider: 'psmdb',
      dbName: 'my-db',
      resources: {
        nodes: undefined,
        cpu: '',
        memory: null,
        disk: 10,
      },
      monitoring: {
        enabled: false,
      },
      replicas: 0,
      tags: [],
    } as Record<string, unknown>;

    const result = postprocessSchemaData(input);

    expect(result).toEqual({
      provider: 'psmdb',
      dbName: 'my-db',
      resources: {
        disk: 10,
      },
      monitoring: {
        enabled: false,
      },
      replicas: 0,
      tags: [],
    });
  });

  it('maps single RHF field value into all target paths when component path is an array', () => {
    const schema: TopologyUISchemas = {
      single: {
        sections: {
          base: {
            components: {
              engineVersion: {
                uiType: FieldType.Text,
                path: ['spec.engine.version', 'spec.proxy.version'],
                fieldParams: {
                  label: 'Version',
                },
              },
            },
          },
        },
      },
    };

    const input = {
      spec: {
        engine: { version: '8.0.41' },
        notes: '',
      },
    } as Record<string, unknown>;

    const result = postprocessSchemaData(input, {
      schema,
      selectedTopology: 'single',
    });

    expect(result).toEqual({
      spec: {
        engine: { version: '8.0.41' },
        proxy: { version: '8.0.41' },
      },
    });
  });

  it('ignores malformed runtime mapping paths without throwing', () => {
    const input = {
      'g-engineVersion': '8.0.41',
    } as Record<string, unknown>;

    expect(() =>
      postprocessSchemaData(input, {
        multiPathMappings: [
          {
            sourceFieldId: 'g-engineVersion',
            targetPaths: ['spec.engine.version', null as unknown as string],
          },
        ],
      })
    ).not.toThrow();

    const result = postprocessSchemaData(input, {
      multiPathMappings: [
        {
          sourceFieldId: 'g-engineVersion',
          targetPaths: ['spec.engine.version', null as unknown as string],
        },
      ],
    });

    expect(result).toEqual({
      spec: {
        engine: { version: '8.0.41' },
      },
    });
  });

  it('appends badge suffix to field values when badgeToApi is true', () => {
    const schema: TopologyUISchemas = {
      ha: {
        sections: {
          resources: {
            components: {
              memory: {
                uiType: FieldType.Number,
                path: 'spec.resources.memory',
                fieldParams: {
                  label: 'Memory',
                  badge: 'Gi',
                  badgeToApi: true,
                },
              },
              disk: {
                uiType: FieldType.Number,
                path: 'spec.resources.disk',
                fieldParams: {
                  label: 'Disk',
                  badge: 'Gi',
                  badgeToApi: true,
                },
              },
              cpu: {
                uiType: FieldType.Number,
                path: 'spec.resources.cpu',
                fieldParams: {
                  label: 'CPU',
                },
              },
            },
          },
        },
      },
    };

    const input = {
      spec: {
        resources: { memory: '4', disk: '25', cpu: '1' },
      },
    } as Record<string, unknown>;

    const result = postprocessSchemaData(input, {
      schema,
      selectedTopology: 'ha',
    });

    expect(result).toEqual({
      spec: {
        resources: { memory: '4Gi', disk: '25Gi', cpu: 1 },
      },
    });
  });

  it('skips badge for empty values during postprocessing', () => {
    const schema: TopologyUISchemas = {
      single: {
        sections: {
          resources: {
            components: {
              memory: {
                uiType: FieldType.Number,
                path: 'spec.resources.memory',
                fieldParams: {
                  label: 'Memory',
                  badge: 'Gi',
                  badgeToApi: true,
                },
              },
            },
          },
        },
      },
    };

    const input = {
      spec: {
        resources: { memory: '' },
      },
    } as Record<string, unknown>;

    const result = postprocessSchemaData(input, {
      schema,
      selectedTopology: 'single',
    });

    // empty string → removed by removeEmptyFieldValues
    expect(result).toEqual({});
  });

  it('applies badges after multipath mapping', () => {
    const schema: TopologyUISchemas = {
      single: {
        sections: {
          resources: {
            components: {
              memory: {
                uiType: FieldType.Number,
                path: [
                  'spec.engine.resources.memory',
                  'spec.proxy.resources.memory',
                ],
                fieldParams: {
                  label: 'Memory',
                  badge: 'Gi',
                  badgeToApi: true,
                },
              },
            },
          },
        },
      },
    };

    const input = {
      spec: {
        engine: { resources: { memory: '2' } },
      },
    } as Record<string, unknown>;

    const result = postprocessSchemaData(input, {
      schema,
      selectedTopology: 'single',
    });

    expect(result).toEqual({
      spec: {
        engine: { resources: { memory: '2Gi' } },
        proxy: { resources: { memory: '2Gi' } },
      },
    });
  });

  it('drops values left over from a previously selected topology', () => {
    const input = {
      dbName: 'my-db',
      spec: {
        components: {
          standalone: { replicas: 1 },
          mixCoord: { replicas: 1 },
          proxy: { replicas: 2 },
          monitoring: { enabled: true },
        },
      },
    } as Record<string, unknown>;

    const result = postprocessSchemaData(input, {
      schema: twoTopologySchema,
      selectedTopology: 'standalone',
    });

    expect(result).toEqual({
      dbName: 'my-db',
      spec: {
        components: {
          standalone: { replicas: 1 },
          monitoring: { enabled: true },
        },
      },
    });
  });

  it('drops qualified keys registered only by another topology', () => {
    const schema: TopologyUISchemas = {
      single: {
        sections: {
          resources: {
            components: {
              cpu: {
                uiType: FieldType.Text,
                path: 'spec.resources.limits.cpu',
                fieldParams: { label: 'CPU limit' },
              },
            },
          },
        },
      },
      cluster: {
        sections: {
          resources: {
            components: {
              gpu: {
                uiType: FieldType.Text,
                path: "spec.resources.limits['nvidia.com/gpu']",
                fieldParams: { label: 'GPU limit' },
              },
            },
          },
        },
      },
    };

    expect(
      postprocessSchemaData(
        {
          spec: {
            resources: {
              limits: { cpu: '4', 'nvidia%2Ecom%2Fgpu': '2' },
            },
          },
        },
        { schema, selectedTopology: 'single' }
      )
    ).toEqual({
      spec: { resources: { limits: { cpu: '4' } } },
    });
  });

  it('scopes parent/child topology paths without rejecting either topology', () => {
    const formValues = {
      spec: { topology: { parameters: { storage: { size: 5 } } } },
    };

    expect(
      postprocessSchemaData(formValues, {
        schema: parentChildTopologySchema,
        selectedTopology: 'a',
      })
    ).toEqual(formValues);
    expect(
      postprocessSchemaData(formValues, {
        schema: parentChildTopologySchema,
        selectedTopology: 'b',
      })
    ).toEqual(formValues);
  });

  it('decodes retained qualified descendants when the selected topology registers their parent', () => {
    const schema: TopologyUISchemas = {
      a: {
        sections: {
          configuration: {
            components: {
              config: {
                uiType: FieldType.Text,
                path: 'spec.config',
                fieldParams: { label: 'Configuration' },
              },
            },
          },
        },
      },
      b: {
        sectionsOrder: [],
        sections: {
          configuration: {
            components: {
              qualifiedValue: {
                uiType: FieldType.Text,
                path: "spec.config['vendor.example/key']",
                fieldParams: { label: 'Qualified value' },
              },
            },
          },
        },
      },
    };
    const formValues = {
      spec: {
        config: {
          'vendor%2Eexample%2Fkey': 'value',
          'user%20label': 'literal',
        },
      },
    };

    expect(
      postprocessSchemaData(formValues, {
        schema,
        selectedTopology: 'a',
      })
    ).toEqual({
      spec: {
        config: {
          'vendor.example/key': 'value',
          'user%20label': 'literal',
        },
      },
    });
    expect(
      postprocessSchemaData(formValues, {
        schema,
        selectedTopology: 'b',
      })
    ).toEqual({
      spec: {
        config: {
          'vendor.example/key': 'value',
          'user%20label': 'literal',
        },
      },
    });
  });

  it('keeps arbitrary nested keys opaque when only the parent field is registered', () => {
    const schema: TopologyUISchemas = {
      single: {
        sections: {
          configuration: {
            components: {
              config: {
                uiType: FieldType.Text,
                path: 'spec.config',
                fieldParams: { label: 'Configuration' },
              },
            },
          },
        },
      },
    };
    const formValues = {
      spec: {
        config: {
          'vendor%2Eexample%2Fkey': 'literal',
          nested: { 'user%20label': 'unchanged' },
        },
      },
    };

    expect(
      postprocessSchemaData(formValues, {
        schema,
        selectedTopology: 'single',
      })
    ).toEqual(formValues);
  });

  it('restores API keys from form ids, keeping qualified names intact', () => {
    const schema: TopologyUISchemas = {
      single: {
        sections: {
          resources: {
            components: {
              gpu: {
                uiType: FieldType.Text,
                path: "spec.resources.limits['nvidia.com/gpu']",
                fieldParams: { label: 'GPU limit' },
              },
            },
          },
        },
      },
    };
    const formValues = {
      spec: {
        resources: {
          limits: {
            'nvidia%2Ecom%2Fgpu': '2',
            cpu: '1',
          },
        },
        'foo-bar': 'kept',
      },
    } as Record<string, unknown>;

    expect(
      postprocessSchemaData(formValues, {
        schema,
        selectedTopology: 'single',
      })
    ).toEqual({
      spec: {
        resources: {
          limits: {
            'nvidia.com/gpu': '2',
            cpu: '1',
          },
        },
        'foo-bar': 'kept',
      },
    });
  });

  it('decodes qualified fields in sections omitted from sectionsOrder', () => {
    const schema: TopologyUISchemas = {
      single: {
        sectionsOrder: ['listed'],
        sections: {
          listed: {
            components: {
              cpu: {
                uiType: FieldType.Text,
                path: 'spec.resources.limits.cpu',
                fieldParams: { label: 'CPU limit' },
              },
            },
          },
          omitted: {
            components: {
              gpu: {
                uiType: FieldType.Text,
                path: "spec.resources.limits['nvidia.com/gpu']",
                fieldParams: { label: 'GPU limit' },
              },
            },
          },
          unrelated: {
            components: {
              value: {
                uiType: FieldType.Text,
                path: 'spec.other.value',
                fieldParams: { label: 'Other value' },
              },
            },
          },
        },
      },
    };
    const formValues = {
      spec: {
        resources: {
          limits: {
            cpu: '4',
            'nvidia%2Ecom%2Fgpu': '2',
            custom: 'unchanged',
          },
        },
        other: { value: 'retained' },
        unregistered: { 'a%20b': 'literal' },
      },
    };

    expect(
      postprocessSchemaData(formValues, {
        schema,
        selectedTopology: 'single',
      })
    ).toEqual({
      spec: {
        resources: {
          limits: {
            cpu: '4',
            'nvidia.com/gpu': '2',
            custom: 'unchanged',
          },
        },
        other: { value: 'retained' },
        unregistered: { 'a%20b': 'literal' },
      },
    });
  });

  it('registers widget targets from sections omitted from sectionsOrder', () => {
    const schema: TopologyUISchemas = {
      single: {
        sectionsOrder: ['listed'],
        sections: {
          listed: { components: {} },
          omitted: {
            components: {
              affinity: {
                uiType: WIDGET_UI_TYPE,
                widgetType: WidgetType.Affinity,
                _widgetTargets: [
                  {
                    key: 'qualified-config',
                    path: "spec['vendor.example/config']",
                  },
                ],
              },
            },
          },
        },
      },
    };

    expect(
      postprocessSchemaData(
        {
          spec: {
            'vendor%2Eexample%2Fconfig': { affinity: 'value' },
          },
        },
        { schema, selectedTopology: 'single' }
      )
    ).toEqual({
      spec: {
        'vendor.example/config': { affinity: 'value' },
      },
    });
  });

  it('does not register multipath API destinations as RHF fields', () => {
    const schema: TopologyUISchemas = {
      single: {
        sections: {
          listed: {
            components: {
              version: {
                uiType: FieldType.Text,
                path: ['spec.engine.version', "spec['vendor.example/version']"],
                fieldParams: { label: 'Version' },
              },
            },
          },
        },
      },
    };

    expect(
      postprocessSchemaData(
        {
          spec: {
            engine: { version: '8.0' },
            'vendor%2Eexample%2Fversion': 'unregistered value',
          },
        },
        { schema, selectedTopology: 'single' }
      )
    ).toEqual({
      spec: {
        engine: { version: '8.0' },
        'vendor%2Eexample%2Fversion': 'unregistered value',
        'vendor.example/version': '8.0',
      },
    });
  });

  it('preserves literal keys inside an object-valued field', () => {
    const schema: TopologyUISchemas = {
      single: {
        sections: {
          configuration: {
            components: {
              config: {
                uiType: WIDGET_UI_TYPE,
                widgetType: WidgetType.Affinity,
                path: 'spec.config',
                _widgetTargets: [
                  {
                    key: 'qualified-config',
                    path: "spec['vendor.example/config']",
                  },
                ],
              },
            },
          },
        },
      },
    };
    const formValues = {
      spec: {
        config: {
          'a%20b': 'some-value',
          'x%2E': 'another-value',
        },
        'vendor%2Eexample%2Fconfig': {
          'a%20b': 'some-value',
          'x%2E': 'another-value',
        },
      },
    };

    expect(
      postprocessSchemaData(formValues, {
        schema,
        selectedTopology: 'single',
      })
    ).toEqual({
      spec: {
        config: {
          'a%20b': 'some-value',
          'x%2E': 'another-value',
        },
        'vendor.example/config': {
          'a%20b': 'some-value',
          'x%2E': 'another-value',
        },
      },
    });
  });

  it('converts a registered qualified parent but preserves its unrelated map value', () => {
    const schema: TopologyUISchemas = {
      single: {
        sections: {
          configuration: {
            components: {
              selected: {
                uiType: FieldType.Text,
                path: "spec['vendor.example/config'].selected",
                fieldParams: { label: 'Selected value' },
              },
            },
          },
        },
      },
    };
    const formValues = {
      spec: {
        'vendor%2Eexample%2Fconfig': {
          selected: 'enabled',
          custom: {
            'a%20b': 'some-value',
            'x%2E': 'another-value',
          },
        },
      },
    };

    expect(
      postprocessSchemaData(formValues, {
        schema,
        selectedTopology: 'single',
      })
    ).toEqual({
      spec: {
        'vendor.example/config': {
          selected: 'enabled',
          custom: {
            'a%20b': 'some-value',
            'x%2E': 'another-value',
          },
        },
      },
    });
  });

  it('rejects a registered path conversion that would overwrite an API key', () => {
    const schema: TopologyUISchemas = {
      single: {
        sections: {
          resources: {
            components: {
              gpu: {
                uiType: FieldType.Text,
                path: "spec.resources.limits['nvidia.com/gpu']",
                fieldParams: { label: 'GPU limit' },
              },
            },
          },
        },
      },
    };
    const formValues = {
      spec: {
        resources: {
          limits: {
            'nvidia%2Ecom%2Fgpu': '2',
            'nvidia.com/gpu': '3',
          },
        },
      },
    };

    expect(() =>
      postprocessSchemaData(formValues, {
        schema,
        selectedTopology: 'single',
      })
    ).toThrow(/API key "nvidia\.com\/gpu" already exists/);
    expect(formValues).toEqual({
      spec: {
        resources: {
          limits: {
            'nvidia%2Ecom%2Fgpu': '2',
            'nvidia.com/gpu': '3',
          },
        },
      },
    });
  });

  it('rejects a retained descendant collision without mutating the parent value', () => {
    const schema: TopologyUISchemas = {
      a: {
        sections: {
          configuration: {
            components: {
              config: {
                uiType: FieldType.Text,
                path: 'spec.config',
                fieldParams: { label: 'Configuration' },
              },
            },
          },
        },
      },
      b: {
        sections: {
          configuration: {
            components: {
              qualifiedValue: {
                uiType: FieldType.Text,
                path: "spec.config['vendor.example/key']",
                fieldParams: { label: 'Qualified value' },
              },
            },
          },
        },
      },
    };
    const formValues = {
      spec: {
        config: {
          'vendor%2Eexample%2Fkey': 'encoded',
          'vendor.example/key': 'existing',
          'user%20label': 'literal',
        },
      },
    };

    expect(() =>
      postprocessSchemaData(formValues, {
        schema,
        selectedTopology: 'a',
      })
    ).toThrow(/API key "vendor\.example\/key" already exists/);
    expect(formValues.spec.config).toEqual({
      'vendor%2Eexample%2Fkey': 'encoded',
      'vendor.example/key': 'existing',
      'user%20label': 'literal',
    });
  });

  it('does not decode arbitrary keys when schema metadata is missing', () => {
    const formValues = {
      config: {
        'a%20b': 'some-value',
        'x%2E': 'another-value',
      },
    };

    expect(postprocessSchemaData(formValues)).toEqual(formValues);
  });

  describe('quantity fields', () => {
    const quantitySchema: TopologyUISchemas = {
      ha: {
        sections: {
          resources: {
            components: {
              cpu: {
                uiType: FieldType.Quantity,
                path: 'spec.resources.cpu',
                fieldParams: { label: 'CPU', unitLabel: 'cores', step: 0.1 },
              },
              memory: {
                uiType: FieldType.Quantity,
                path: 'spec.resources.memory',
                fieldParams: { label: 'Memory', unit: 'Gi' },
              },
            },
          },
        },
      },
    };

    it('writes fractional CPU as a string, never a JSON float', () => {
      const result = postprocessSchemaData(
        { spec: { resources: { cpu: 0.5, memory: 10 } } },
        { schema: quantitySchema, selectedTopology: 'ha' }
      );

      expect(result).toEqual({
        spec: { resources: { cpu: '0.5', memory: '10Gi' } },
      });
      expect(
        typeof (result as { spec: { resources: { cpu: unknown } } }).spec
          .resources.cpu
      ).toBe('string');
    });

    it('keeps an untouched quantity as stored when originalData matches', () => {
      const result = postprocessSchemaData(
        { spec: { resources: { cpu: 0.5, memory: 10 } } },
        {
          schema: quantitySchema,
          selectedTopology: 'ha',
          originalData: {
            spec: { resources: { cpu: '500m', memory: '10240Mi' } },
          },
        }
      );

      expect(result).toEqual({
        spec: { resources: { cpu: '500m', memory: '10240Mi' } },
      });
    });

    it('rewrites a changed quantity in the field unit', () => {
      const result = postprocessSchemaData(
        { spec: { resources: { cpu: 1, memory: 12 } } },
        {
          schema: quantitySchema,
          selectedTopology: 'ha',
          originalData: {
            spec: { resources: { cpu: '500m', memory: '10240Mi' } },
          },
        }
      );

      expect(result).toEqual({
        spec: { resources: { cpu: '1', memory: '12Gi' } },
      });
    });

    it('leaves number + badgeToApi behavior unchanged', () => {
      const schema: TopologyUISchemas = {
        ha: {
          sections: {
            resources: {
              components: {
                disk: {
                  uiType: FieldType.Number,
                  path: 'spec.resources.disk',
                  fieldParams: {
                    label: 'Disk',
                    badge: 'Gi',
                    badgeToApi: true,
                  },
                },
                cpu: {
                  uiType: FieldType.Quantity,
                  path: 'spec.resources.cpu',
                  fieldParams: { label: 'CPU' },
                },
              },
            },
          },
        },
      };

      const result = postprocessSchemaData(
        { spec: { resources: { disk: 25, cpu: 0.5 } } },
        { schema, selectedTopology: 'ha' }
      );

      expect(result).toEqual({
        spec: { resources: { disk: '25Gi', cpu: '0.5' } },
      });
    });
  });
});
