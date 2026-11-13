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
  type ArrayComponent,
  type Section,
  type TopologyUISchemas,
  ArrayItemType,
  ArrayOutputType,
  FieldType,
  FormMode,
} from 'components/ui-generator/ui-generator.types';
import { getDefaultValues } from '../default-values';
import { extractInstanceValues } from '../default-values/extract-instance-values';
import { deepMerge } from '../object-path';
import { postprocessSchemaData } from '../postprocess/postprocess-schema';
import { mergeSectionEdit } from '../postprocess/merge-section-edit';
import { preprocessSchema } from '../preprocess/preprocess-schema';
import { buildArrayFieldSchema } from './array-field-schema';
import { Messages } from './array-field.messages';

const PATH = 'spec.proxy.annotations';

const annotations = (
  overrides: Partial<
    Pick<ArrayComponent, 'fieldParams' | 'itemTemplate' | 'validation'>
  > = {}
): ArrayComponent => ({
  uiType: FieldType.Array,
  outputType: ArrayOutputType.Object,
  path: PATH,
  fieldParams: { label: 'Annotations' },
  itemTemplate: { uiType: ArrayItemType.KeyValue },
  ...overrides,
});

const sectionsWith = (component: ArrayComponent): Record<string, Section> => ({
  advanced: { components: { annotations: component } },
});

const schemaWith = (component: ArrayComponent) => ({
  replicaSet: { sections: sectionsWith(component) },
});

const issuesOf = (component: ArrayComponent, rows: unknown) =>
  (
    buildArrayFieldSchema(component, component.validation).safeParse(rows).error
      ?.issues ?? []
  ).map(({ path, message }) => ({ path, message }));

describe('array field rows validation', () => {
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
        uiType: ArrayItemType.KeyValue,
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

  it('counts only filled rows against required and maxItems', () => {
    expect(
      issuesOf(annotations({ validation: { required: true } }), [
        { key: '', value: '' },
      ])
    ).toEqual([{ path: [], message: Messages.required }]);

    expect(
      issuesOf(annotations({ validation: { maxItems: 1 } }), [
        { key: 'a', value: '1' },
        { key: '', value: '' },
        { key: 'b', value: '2' },
      ])
    ).toEqual([{ path: [], message: Messages.maxItems(1) }]);
  });
});

describe('array field through the pipeline', () => {
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

  it('sends an entry with an empty value, which a map still holds, to every path', () => {
    const component: ArrayComponent = {
      uiType: FieldType.Array,
      outputType: ArrayOutputType.Object,
      path: [PATH, 'spec.engine.annotations'],
      fieldParams: { label: 'Annotations' },
      itemTemplate: { uiType: ArrayItemType.KeyValue },
    };
    const infra = { 'node-role.kubernetes.io/infra': '' };

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
        { schema: schemaWith(component), selectedTopology: 'replicaSet' }
      )
    ).toEqual({
      spec: { proxy: { annotations: infra }, engine: { annotations: infra } },
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
});

describe('array field in a shape this version does not support', () => {
  // As a newer provider could ship it: a list, which this version can't edit.
  const listSchema: TopologyUISchemas = parse(`
    replicaSet:
      sections:
        advanced:
          components:
            databases:
              uiType: array
              outputType: array
              path: spec.backup.databases
              fieldParams: { label: Databases }
              itemTemplate: { uiType: text }
            replicas:
              uiType: number
              path: spec.replicas
              fieldParams: { label: Replicas }
  `);

  it.each([
    ['preprocessed', preprocessSchema(listSchema)],
    ['raw, as forms that skip preprocess get it', listSchema],
  ])('is left alone when %s', (_, schema) => {
    const options = { schema, selectedTopology: 'replicaSet' };
    expect(
      postprocessSchemaData(getDefaultValues(schema, 'replicaSet'), options)
    ).toEqual({});

    const saved = { backup: { databases: ['db1', 'db2'] }, replicas: 3 };
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
    ).toEqual({ backup: { databases: ['db1', 'db2'] }, replicas: 5 });
  });
});
