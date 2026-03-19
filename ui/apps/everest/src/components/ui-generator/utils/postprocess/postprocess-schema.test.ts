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
import { FieldType, TopologyUISchemas } from '../../ui-generator.types';

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
});
