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
import {
  dropOtherTopologyValues,
  postprocessSchemaData,
} from './postprocess-schema';
import {
  Component,
  FieldType,
  TopologyUISchemas,
} from '../../ui-generator.types';

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
});

const numberField = (path: string): Component => ({
  uiType: FieldType.Number,
  path,
  fieldParams: { label: path },
});

const monitoringField: Component = {
  uiType: FieldType.Toggle,
  path: 'spec.components.monitoring.enabled',
  fieldParams: { label: 'Monitoring' },
};

// Milvus-shaped: each topology owns its own components; monitoring is shared.
const twoTopologySchema: TopologyUISchemas = {
  standalone: {
    sections: {
      resources: {
        components: {
          replicas: numberField('spec.components.standalone.replicas'),
          monitoring: monitoringField,
        },
      },
    },
  },
  cluster: {
    sections: {
      resources: {
        components: {
          coordinator: numberField('spec.components.mixCoord.replicas'),
          proxy: numberField('spec.components.proxy.replicas'),
          monitoring: monitoringField,
        },
      },
    },
  },
};

describe('dropOtherTopologyValues', () => {
  it('keeps values of paths shared with the selected topology', () => {
    const input = {
      spec: {
        components: {
          mixCoord: { replicas: 1 },
          monitoring: { enabled: false },
        },
      },
    };

    expect(
      dropOtherTopologyValues(input, twoTopologySchema, 'standalone')
    ).toEqual({
      spec: { components: { mixCoord: {}, monitoring: { enabled: false } } },
    });
  });

  it('keeps values the selected topology owns through a parent or child path', () => {
    const schema: TopologyUISchemas = {
      a: {
        sections: {
          s: {
            components: {
              storage: {
                uiType: FieldType.Text,
                path: 'spec.topology.parameters.storage',
                fieldParams: { label: 'Storage' },
              },
            },
          },
        },
      },
      b: {
        sections: {
          s: {
            components: {
              size: numberField('spec.topology.parameters.storage.size'),
            },
          },
        },
      },
    };
    const input = {
      spec: { topology: { parameters: { storage: { size: 5 } } } },
    };

    expect(dropOtherTopologyValues(input, schema, 'a')).toEqual(input);
  });

  it('does not mutate its input', () => {
    const input = { spec: { components: { mixCoord: { replicas: 1 } } } };

    dropOtherTopologyValues(input, twoTopologySchema, 'standalone');

    expect(input).toEqual({
      spec: { components: { mixCoord: { replicas: 1 } } },
    });
  });
});
