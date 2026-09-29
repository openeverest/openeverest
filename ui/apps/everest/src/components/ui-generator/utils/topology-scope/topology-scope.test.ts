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
  ComponentGroup,
  FieldType,
  GroupType,
  TopologyUISchemas,
} from '../../ui-generator.types';
import { getByPath } from '../object-path';
import { preprocessSchema } from '../preprocess/preprocess-schema';
import { TOGGLEABLE_SWITCHES_KEY } from '../toggleable/toggleable';
import { numberField, twoTopologySchema } from './__mocks__/topology-schemas';
import { dropOtherTopologyValues } from './topology-scope';

const toggleableGroup = (path: string): ComponentGroup => ({
  uiType: 'group',
  groupType: GroupType.Toggleable,
  label: path,
  components: { field: numberField(path) },
});

describe('dropOtherTopologyValues', () => {
  it('drops values only other topologies bind and keeps shared ones', () => {
    const input = {
      spec: {
        components: {
          mixCoord: { replicas: 1 },
          monitoring: { enabled: false },
        },
      },
    };

    const result = dropOtherTopologyValues(
      input,
      twoTopologySchema,
      'standalone'
    );

    expect(
      getByPath(result, 'spec.components.mixCoord.replicas')
    ).toBeUndefined();
    expect(result).toMatchObject({
      spec: { components: { monitoring: { enabled: false } } },
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

  it('keeps values bound by a selected-topology section missing from sectionsOrder', () => {
    const schema: TopologyUISchemas = {
      a: {
        sectionsOrder: [],
        sections: {
          s: { components: { replicas: numberField('spec.replicas') } },
        },
      },
      b: {
        sections: {
          s: { components: { replicas: numberField('spec.replicas') } },
        },
      },
    };
    const input = { spec: { replicas: 3 } };

    expect(dropOtherTopologyValues(input, schema, 'a')).toEqual(input);
  });

  it('resets switches of toggleable groups the selected topology lacks', () => {
    const schema = preprocessSchema({
      cluster: {
        sections: {
          advanced: {
            components: {
              monitoring: toggleableGroup('spec.monitoring.interval'),
              sharding: toggleableGroup('spec.sharding.shards'),
            },
          },
        },
      },
      standalone: {
        sections: {
          advanced: {
            components: {
              monitoring: toggleableGroup('spec.monitoring.interval'),
            },
          },
        },
      },
    });
    const input = {
      [TOGGLEABLE_SWITCHES_KEY]: {
        'advanced~monitoring': true,
        'advanced~sharding': true,
      },
    };

    expect(dropOtherTopologyValues(input, schema, 'standalone')).toEqual({
      [TOGGLEABLE_SWITCHES_KEY]: { 'advanced~monitoring': true },
    });
  });

  it('does not mutate its input', () => {
    const input = { spec: { components: { mixCoord: { replicas: 1 } } } };

    dropOtherTopologyValues(input, twoTopologySchema, 'standalone');

    expect(input).toEqual({
      spec: { components: { mixCoord: { replicas: 1 } } },
    });
  });
});
