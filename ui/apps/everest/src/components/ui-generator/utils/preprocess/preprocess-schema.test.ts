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

import { describe, expect, it, vi } from 'vitest';
import {
  Component,
  DataSource,
  FieldType,
  TopologyUISchemas,
} from '../../ui-generator.types';
import { preprocessSchema } from './preprocess-schema';

// Suppress console.error from dev-time validation when using test providers
vi.mock('../../api-providers/registry', async () => {
  const actual = await vi.importActual<
    typeof import('../../api-providers/registry')
  >('../../api-providers/registry');
  return {
    ...actual,
    providerRegistry: {
      ...actual.providerRegistry,
      has: () => true,
      getAvailableKeys: () => ['monitoringConfigs', 'storageClasses'],
    },
  };
});

describe('preprocessSchema', () => {
  it('normalizes path metadata even without provider object', () => {
    const schema: TopologyUISchemas = {
      single: {
        sections: {
          base: {
            components: {
              version: {
                uiType: FieldType.Text,
                path: ['spec.engine.version', 'spec.proxy.version'],
                fieldParams: { label: 'Version' },
              },
            },
          },
        },
      },
    };

    const result = preprocessSchema(schema);
    const component = result.single.sections.base.components.version;

    if ('uiType' in component && !('components' in component)) {
      const leaf = component as Component;
      expect(leaf._normalized?.sourcePath).toBe('spec.engine.version');
      expect(leaf._normalized?.targetPaths).toEqual([
        'spec.engine.version',
        'spec.proxy.version',
      ]);
    }
  });

  it('preserves dataSource through preprocessing', () => {
    const dataSource: DataSource = {
      provider: 'monitoringConfigs',
      config: { refetchInterval: 15000 },
    };
    const schema: TopologyUISchemas = {
      single: {
        sections: {
          monitoring: {
            components: {
              monitoringEndpoint: {
                uiType: FieldType.Select,
                dataSource,
                path: 'spec.components.monitoring.customSpec.monitoringConfigName',
                fieldParams: { label: 'Monitoring endpoint' },
              },
            },
          },
        },
      },
    };

    const result = preprocessSchema(schema);
    const comp = result.single.sections.monitoring.components
      .monitoringEndpoint as Component;

    expect(comp.uiType).toBe(FieldType.Select);
    expect(comp.dataSource).toEqual(dataSource);
  });
});
