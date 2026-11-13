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
import { parse } from 'yaml';
import { collectSectionFields } from './cluster-overview.helpers';
import type {
  Component,
  ComponentGroup,
  TopologyUISchemas,
} from 'components/ui-generator/ui-generator.types';
import {
  FieldType,
  GroupType,
  TemplateWidgetType,
  WIDGET_UI_TYPE,
} from 'components/ui-generator/ui-generator.types';
import { preprocessSchema } from 'components/ui-generator/utils/preprocess/preprocess-schema';

const numberField = (
  path: string,
  label: string,
  badge?: string
): Component => ({
  uiType: FieldType.Number,
  path,
  fieldParams: { label, ...(badge ? { badge, badgeToApi: true } : {}) },
});

const quantityField = (
  path: string,
  label: string,
  fieldParams: {
    unit?: Extract<
      Component,
      { uiType: FieldType.Quantity }
    >['fieldParams']['unit'];
    unitLabel?: string;
  } = {}
): Component => ({
  uiType: FieldType.Quantity,
  path,
  fieldParams: { label, ...fieldParams },
});

const components: Record<string, Component | ComponentGroup> = {
  cpu: numberField('spec.components.engine.resources.limits.cpu', 'CPU'),
  memory: numberField(
    'spec.components.engine.resources.limits.memory',
    'Memory',
    'Gi'
  ),
  disk: numberField('spec.components.engine.storage.size', 'Disk', 'Gi'),
};

describe('collectSectionFields', () => {
  it('renders a badged field in its badge unit, converting Kubernetes-normalised quantities', () => {
    const instance = {
      spec: {
        components: {
          engine: {
            resources: { limits: { cpu: '1', memory: '644245094400m' } },
            storage: { size: '25Gi' },
          },
        },
      },
    };

    const fields = collectSectionFields(components, instance, [
      'cpu',
      'memory',
      'disk',
    ]);

    expect(fields).toEqual([
      {
        label: 'CPU',
        path: 'spec.components.engine.resources.limits.cpu',
        value: '1',
      },
      {
        label: 'Memory',
        path: 'spec.components.engine.resources.limits.memory',
        value: '0.6Gi',
      },
      {
        label: 'Disk',
        path: 'spec.components.engine.storage.size',
        value: '25Gi',
      },
    ]);
  });

  it('leaves non-standard units untouched', () => {
    const instance = {
      spec: {
        components: {
          engine: {
            resources: { limits: { cpu: '1', memory: '16kg' } },
            storage: { size: '25Gi' },
          },
        },
      },
    };

    const fields = collectSectionFields(components, instance, ['memory']);

    expect(fields[0].value).toBe('16kg');
  });

  it('renders quantity fields with unitLabel / unit (not raw milli values)', () => {
    const quantityComponents: Record<string, Component | ComponentGroup> = {
      cpu: quantityField('spec.components.engine.resources.limits.cpu', 'CPU', {
        unitLabel: 'cores',
      }),
      memory: quantityField(
        'spec.components.engine.resources.limits.memory',
        'Memory',
        { unit: 'Gi' }
      ),
    };

    const fields = collectSectionFields(
      quantityComponents,
      {
        spec: {
          components: {
            engine: {
              resources: {
                limits: { cpu: '500m', memory: '10240Mi' },
              },
            },
          },
        },
      },
      ['cpu', 'memory']
    );

    expect(fields).toEqual([
      {
        label: 'CPU',
        path: 'spec.components.engine.resources.limits.cpu',
        value: '0.5 cores',
      },
      {
        label: 'Memory',
        path: 'spec.components.engine.resources.limits.memory',
        value: '10 Gi',
      },
    ]);
  });

  it('converts Kubernetes-normalised memory quantities for quantity fields', () => {
    const fields = collectSectionFields(
      {
        memory: quantityField(
          'spec.components.engine.resources.limits.memory',
          'Memory',
          { unit: 'Gi' }
        ),
      },
      {
        spec: {
          components: {
            engine: {
              resources: { limits: { memory: '644245094400m' } },
            },
          },
        },
      },
      ['memory']
    );

    expect(fields[0].value).toBe('0.6 Gi');
  });

  describe('toggleable group', () => {
    const { advanced } = preprocessSchema({
      replicaSet: {
        sections: {
          advanced: {
            components: {
              monitoring: {
                uiType: 'group',
                groupType: GroupType.Toggleable,
                label: 'Monitoring',
                components: {
                  url: {
                    uiType: FieldType.Text,
                    path: 'spec.monitoring.url',
                    fieldParams: { label: 'URL' },
                  },
                },
              },
            },
          },
        },
      },
    }).replicaSet.sections;

    it('shows a switched-off group as a single Disabled row', () => {
      expect(
        collectSectionFields(advanced.components, { spec: {} }).map(
          ({ label, value }) => ({ label, value })
        )
      ).toEqual([{ label: 'Monitoring', value: 'Disabled' }]);
    });

    it('lists the fields of a switched-on group', () => {
      expect(
        collectSectionFields(advanced.components, {
          spec: { monitoring: { url: 'http://pmm' } },
        }).map(({ label, value }) => ({ label, value }))
      ).toEqual([{ label: 'URL', value: 'http://pmm' }]);
    });
  });

  describe('values a read-only view must not show as typed', () => {
    const { advanced } = preprocessSchema({
      replicaSet: {
        sections: {
          advanced: {
            components: {
              tags: {
                uiType: FieldType.Map,
                path: 'spec.tags',
                fieldParams: { label: 'Tags' },
                itemTemplate: {
                  uiType: WIDGET_UI_TYPE,
                  widgetType: TemplateWidgetType.KeyValue,
                },
              },
              env: {
                uiType: FieldType.Map,
                path: 'spec.env',
                fieldParams: { label: 'Environment' },
                itemTemplate: {
                  uiType: WIDGET_UI_TYPE,
                  widgetType: TemplateWidgetType.KeyValue,
                  fieldParams: { value: { type: 'password' } },
                },
              },
              token: {
                uiType: FieldType.Text,
                path: 'spec.token',
                fieldParams: { label: 'Token', type: 'password' },
              },
            },
          },
        },
      },
    }).replicaSet.sections;

    it('lists a map one entry per line, since a value may hold commas', () => {
      const [tags] = collectSectionFields(advanced.components, {
        spec: { tags: { 'aws-tags': 'Env=dev,Team=db', team: 'db' } },
      });

      expect(tags).toMatchObject({
        value: 'aws-tags = Env=dev,Team=db\nteam = db',
        multiline: true,
      });
    });

    it('masks password values, in a text field or a map row', () => {
      const [, env, token] = collectSectionFields(advanced.components, {
        spec: { env: { API_KEY: 'secret' }, token: 's3cr3t' },
      });

      expect(env.value).toBe('API_KEY = ***************');
      expect(token.value).toBe('***************');
    });

    it('shows a field the schema gives no fieldParams', () => {
      const schema: TopologyUISchemas = parse(`
        replicaSet:
          sections:
            advanced:
              components:
                region: { uiType: text, path: spec.region }
      `);

      expect(
        collectSectionFields(schema.replicaSet.sections.advanced.components, {
          spec: { region: 'eu' },
        }).map(({ label, value }) => ({ label, value }))
      ).toEqual([{ label: 'region', value: 'eu' }]);
    });
  });
});
