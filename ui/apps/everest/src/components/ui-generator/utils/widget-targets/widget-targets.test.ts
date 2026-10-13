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
import { Provider } from 'shared-types/api.types';
import {
  ComponentGroup,
  GroupType,
  TopologyUISchemas,
  WIDGET_UI_TYPE,
  WidgetType,
} from '../../ui-generator.types';
import { preprocessSchema } from '../preprocess/preprocess-schema';
import {
  getToggleableMeta,
  TOGGLEABLE_SWITCHES_KEY,
} from '../toggleable/toggleable';
import { extractInstanceValues } from '../default-values/extract-instance-values';
import { mergeSectionEdit } from '../postprocess/merge-section-edit';

const ENGINE_AFFINITY = 'spec.components.engine.schedulingPolicy.affinity';

const provider: Provider = {
  spec: {
    topologies: {
      replicaSet: {
        components: {
          engine: {
            supportedFields: {
              openAPIV3Schema: {
                properties: {
                  schedulingPolicy: { properties: { affinity: {} } },
                },
              },
            },
          },
          monitoring: {
            supportedFields: { openAPIV3Schema: { properties: {} } },
          },
        },
      },
    },
  },
};

const rawSchema: TopologyUISchemas = {
  replicaSet: {
    sections: {
      advanced: {
        components: {
          scheduling: {
            uiType: 'group',
            groupType: GroupType.Toggleable,
            label: 'Pod scheduling policy',
            components: {
              policy: {
                uiType: WIDGET_UI_TYPE,
                widgetType: WidgetType.PodSchedulingPolicy,
                id: 'podSchedulingPolicy',
              },
            },
          },
        },
      },
    },
  },
};

const sectionsFor = (withProvider?: Provider) =>
  preprocessSchema(rawSchema, withProvider).replicaSet.sections;

const schedulingGroup = (withProvider?: Provider): ComponentGroup => {
  const group = sectionsFor(withProvider).advanced.components.scheduling;
  if (!('components' in group)) throw new Error('expected a group');
  return group;
};

const affinity = { nodeAffinity: { preferred: [{ weight: 1 }] } };

describe('podSchedulingPolicy marker targets', () => {
  it('lets a toggleable group own the paths its marker writes', () => {
    expect(getToggleableMeta(schedulingGroup(provider))).toMatchObject({
      childPaths: [ENGINE_AFFINITY],
    });
  });

  it('degrades the group when the provider is unknown', () => {
    const group = schedulingGroup();
    expect(getToggleableMeta(group)).toBeUndefined();
    expect(group.groupType).toBe(GroupType.Bordered);
  });

  it('loads saved affinity and turns the group on in edit', () => {
    const values = extractInstanceValues(sectionsFor(provider), {
      spec: { components: { engine: { schedulingPolicy: { affinity } } } },
    });

    expect(values).toMatchObject({
      [TOGGLEABLE_SWITCHES_KEY]: { 'advanced~scheduling': true },
      spec: { components: { engine: { schedulingPolicy: { affinity } } } },
    });
  });

  it('removes saved affinity when the group is switched off on edit', () => {
    const spec = {
      replicas: 3,
      components: { engine: { schedulingPolicy: { affinity } } },
    };

    expect(
      mergeSectionEdit({
        spec,
        sections: sectionsFor(provider),
        sectionKey: 'advanced',
        topology: 'replicaSet',
        formData: {
          [TOGGLEABLE_SWITCHES_KEY]: { 'advanced~scheduling': false },
          spec: { components: { engine: { schedulingPolicy: { affinity } } } },
        },
      })
    ).toEqual({ replicas: 3 });
  });
});
