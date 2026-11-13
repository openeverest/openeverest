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
import {
  type KeyValueWidget,
  type MapComponent,
  type Section,
  type TopologyUISchemas,
  FieldType,
  FormMode,
  TemplateWidgetType,
  WIDGET_UI_TYPE,
} from 'components/ui-generator/ui-generator.types';
import { getDefaultValues } from '../default-values';
import { extractInstanceValues } from '../default-values/extract-instance-values';
import { deepMerge, getByPath } from '../object-path';
import { postprocessSchemaData } from '../postprocess/postprocess-schema';
import { mergeSectionEdit } from '../postprocess/merge-section-edit';
import { preprocessSchema } from '../preprocess/preprocess-schema';
import { buildShapeFromComponents } from '../schema-builder/build-shape-from-components';
import { Messages } from './map-field.messages';

const PATH = 'spec.proxy.annotations';

const KEY_VALUE: KeyValueWidget = {
  uiType: WIDGET_UI_TYPE,
  widgetType: TemplateWidgetType.KeyValue,
};

const annotations = (
  overrides: Partial<
    Pick<MapComponent, 'fieldParams' | 'itemTemplate' | 'validation'>
  > = {}
): MapComponent => ({
  uiType: FieldType.Map,
  path: PATH,
  fieldParams: { label: 'Annotations' },
  itemTemplate: KEY_VALUE,
  ...overrides,
});

const sectionsWith = (component: MapComponent): Record<string, Section> => ({
  advanced: { components: { annotations: component } },
});

const schemaWith = (component: MapComponent) => ({
  replicaSet: { sections: sectionsWith(component) },
});

// The map's schema as the form gets it: rows checked against the expanded
// row's own text fields.
const issuesOf = (component: MapComponent, rows: unknown) =>
  (
    buildShapeFromComponents({ annotations: component }).schemaShape[
      PATH
    ].safeParse(rows).error?.issues ?? []
  ).map(({ path, message }) => ({ path, message }));

describe('map field rows validation', () => {
  it('reports a key set twice on the repeated row, since a map keeps one', () => {
    expect(
      issuesOf(annotations(), [
        { key: 'team', value: 'a' },
        { key: 'team', value: 'b' },
      ])
    ).toEqual([{ path: [1, 'key'], message: Messages.duplicateKey('team') }]);
  });

  it('ignores rows left empty but requires a key for a value', () => {
    expect(
      issuesOf(annotations(), [
        { key: '', value: '' },
        { key: '', value: 'orphan' },
      ])
    ).toEqual([{ path: [1, 'key'], message: Messages.keyRequired }]);
  });

  it('applies text validation to the key and value', () => {
    const component = annotations({
      itemTemplate: {
        ...KEY_VALUE,
        validation: {
          key: { regex: { pattern: '^[a-z.]+$', message: 'Lowercase only' } },
          value: { max: 3 },
        },
      },
    });

    expect(issuesOf(component, [{ key: 'Team', value: 'long' }])).toEqual([
      { path: [0, 'key'], message: 'Lowercase only' },
      { path: [0, 'value'], message: expect.any(String) },
    ]);
  });

  it('writes rows as their fields transform them, finding duplicates after it', () => {
    const component = annotations({
      itemTemplate: {
        ...KEY_VALUE,
        validation: { key: { trim: true, toLowerCase: true } },
      },
    });
    const schema = buildShapeFromComponents({ annotations: component })
      .schemaShape[PATH];

    expect(schema.parse([{ key: ' Team ', value: 'db' }])).toEqual([
      { key: 'team', value: 'db' },
    ]);
    expect(
      issuesOf(component, [
        { key: 'team', value: 'a' },
        { key: ' Team', value: 'b' },
      ])
    ).toEqual([{ path: [1, 'key'], message: Messages.duplicateKey('team') }]);
  });

  it('counts only filled rows against required and maxItems', () => {
    expect(
      issuesOf(annotations({ validation: { required: true } }), [
        { key: '', value: '' },
      ])
    ).toEqual([{ path: ['root'], message: Messages.required }]);

    expect(
      issuesOf(annotations({ validation: { maxItems: 1 } }), [
        { key: 'a', value: '1' },
        { key: '', value: '' },
        { key: 'b', value: '2' },
      ])
    ).toEqual([{ path: ['root'], message: Messages.maxItems(1, 1) }]);
  });

  it("puts the count issue beside the rows' issues, not over them", () => {
    expect(
      issuesOf(annotations({ validation: { maxItems: 1 } }), [
        { key: 'a', value: '1' },
        { key: 'a', value: '2' },
      ])
    ).toEqual([
      { path: [1, 'key'], message: Messages.duplicateKey('a') },
      { path: ['root'], message: Messages.maxItems(1, 1) },
    ]);
  });
});

describe('map field through the pipeline', () => {
  it('starts the form from the default map as rows', () => {
    const component = annotations({
      fieldParams: { defaultValue: { 'aws/type': 'nlb' } },
    });

    expect(
      getDefaultValues(schemaWith(component), 'replicaSet').spec
    ).toMatchObject({
      proxy: { annotations: [{ key: 'aws/type', value: 'nlb' }] },
    });
  });

  it('starts empty from a default that is not a map of strings', () => {
    // As a provider's YAML may hold it: read as rows, the 3 would be written
    // back as the string "3".
    const schema: TopologyUISchemas = parse(`
      replicaSet:
        sections:
          advanced:
            components:
              annotations:
                uiType: map
                path: ${PATH}
                fieldParams: { label: Annotations, defaultValue: { replicas: 3 } }
                itemTemplate: { uiType: widget, widgetType: keyValue }
    `);

    expect(
      getByPath(getDefaultValues(schema, 'replicaSet'), PATH)
    ).toBeUndefined();
  });

  it('reads a saved map into rows for edit', () => {
    expect(
      extractInstanceValues(
        sectionsWith(annotations()),
        { spec: { proxy: { annotations: { team: 'db', tier: 'gold' } } } },
        FormMode.Edit
      )
    ).toEqual({
      spec: {
        proxy: {
          annotations: [
            { key: 'team', value: 'db' },
            { key: 'tier', value: 'gold' },
          ],
        },
      },
    });
  });

  it('submits the rows as a map, and nothing when every row is empty', () => {
    const submit = (rows: unknown) =>
      postprocessSchemaData(
        { spec: { proxy: { annotations: rows, replicas: 2 } } },
        { schema: schemaWith(annotations()), selectedTopology: 'replicaSet' }
      );

    expect(
      submit([
        { key: 'team', value: 'db' },
        { key: '', value: '' },
      ])
    ).toEqual({
      spec: { proxy: { annotations: { team: 'db' }, replicas: 2 } },
    });
    expect(submit([{ key: '', value: '' }])).toEqual({
      spec: { proxy: { replicas: 2 } },
    });
  });

  it('drops an entry removed in edit instead of merging it back', () => {
    const edit = (rows: unknown) =>
      mergeSectionEdit({
        spec: { proxy: { annotations: { team: 'db', tier: 'gold' } } },
        formData: { spec: { proxy: { annotations: rows } } },
        sections: sectionsWith(annotations()),
        sectionKey: 'advanced',
        topology: 'replicaSet',
      });

    expect(edit([{ key: 'team', value: 'db' }])).toEqual({
      proxy: { annotations: { team: 'db' } },
    });
    expect(edit([])).toEqual({});
  });

  it('sends an entry with an empty value, which a map still holds', () => {
    expect(
      postprocessSchemaData(
        {
          spec: {
            proxy: {
              annotations: [
                { key: 'node-role.kubernetes.io/infra', value: '' },
              ],
            },
          },
        },
        { schema: schemaWith(annotations()), selectedTopology: 'replicaSet' }
      )
    ).toEqual({
      spec: { proxy: { annotations: { 'node-role.kubernetes.io/infra': '' } } },
    });
  });

  it('keeps a saved entry with an empty value through an edit', () => {
    const saved = {
      proxy: {
        annotations: { 'node-role.kubernetes.io/infra': '', team: 'db' },
      },
    };
    const sections = sectionsWith(annotations());

    expect(
      mergeSectionEdit({
        spec: saved,
        formData: extractInstanceValues(
          sections,
          { spec: saved },
          FormMode.Edit
        ),
        sections,
        sectionKey: 'advanced',
        topology: 'replicaSet',
      })
    ).toEqual(saved);
  });

  it('leaves a saved map the form holds as it is, instead of reading it as rows', () => {
    // As forms that load saved values unconverted, like the monitoring
    // entries editor, hand it over.
    expect(
      postprocessSchemaData(
        { spec: { proxy: { annotations: { team: 'db' } } } },
        { schema: schemaWith(annotations()), selectedTopology: 'replicaSet' }
      )
    ).toEqual({ spec: { proxy: { annotations: { team: 'db' } } } });
  });
});

describe('map field in a shape this version does not support', () => {
  // As a newer provider could ship it: a map of objects, whose rows this
  // version can't render yet.
  const usersSchema: TopologyUISchemas = parse(`
    replicaSet:
      sections:
        advanced:
          components:
            users:
              uiType: map
              path: spec.users
              fieldParams: { label: Users }
              itemTemplate:
                uiType: group
                components:
                  role: { uiType: text, path: value.role, fieldParams: { label: Role } }
            replicas:
              uiType: number
              path: spec.replicas
              fieldParams: { label: Replicas }
  `);

  it.each([
    ['preprocessed', preprocessSchema(usersSchema)],
    ['raw, as forms that skip preprocess get it', usersSchema],
  ])('is left alone when %s', (_, schema) => {
    const options = { schema, selectedTopology: 'replicaSet' };
    expect(
      postprocessSchemaData(getDefaultValues(schema, 'replicaSet'), options)
    ).toEqual({});

    const saved = { users: { alice: { role: 'admin' } }, replicas: 3 };
    const { sections } = schema.replicaSet;
    const formData = extractInstanceValues(
      sections,
      { spec: saved },
      FormMode.Edit
    );

    expect(
      mergeSectionEdit({
        spec: saved,
        formData: deepMerge(formData, { spec: { replicas: 5 } }),
        sections,
        sectionKey: 'advanced',
        topology: 'replicaSet',
      })
    ).toEqual({ users: { alice: { role: 'admin' } }, replicas: 5 });
  });
});
