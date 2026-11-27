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
  type ListComponent,
  type Section,
  type TopologyUISchemas,
  FieldType,
  FormMode,
} from 'components/ui-generator/ui-generator.types';
import { getDefaultValues } from '../default-values';
import { extractInstanceValues } from '../default-values/extract-instance-values';
import { deepMerge, getByPath } from '../object-path';
import { postprocessSchemaData } from '../postprocess/postprocess-schema';
import { mergeSectionEdit } from '../postprocess/merge-section-edit';
import { preprocessSchema } from '../preprocess/preprocess-schema';
import { buildZodSchema } from '../schema-builder';
import { buildShapeFromComponents } from '../schema-builder/build-shape-from-components';
import {
  getRow,
  getRowsFieldDisplayEntries,
  lintRowsFields,
} from '../rows-field';
import { Messages as RowsMessages } from '../rows-field/rows-field.messages';
import { Messages } from './list-field.messages';

const PATH = 'spec.proxy.sourceRanges';

const sourceRanges = (
  overrides: Partial<
    Pick<ListComponent, 'fieldParams' | 'itemTemplate' | 'validation'>
  > = {}
): ListComponent => ({
  uiType: FieldType.List,
  path: PATH,
  fieldParams: { label: 'Source ranges' },
  itemTemplate: { uiType: FieldType.Text },
  ...overrides,
});

const sectionsWith = (component: ListComponent): Record<string, Section> => ({
  advanced: { components: { sourceRanges: component } },
});

const schemaWith = (component: ListComponent) => ({
  replicaSet: { sections: sectionsWith(component) },
});

const rowsOf = (...values: string[]) => values.map((value) => ({ value }));

const schemaOf = (component: ListComponent) =>
  buildShapeFromComponents({ sourceRanges: component }).schemaShape[PATH];

// The list's schema as the form gets it: rows checked against the item field.
const issuesOf = (component: ListComponent, rows: unknown) =>
  (schemaOf(component).safeParse(rows).error?.issues ?? []).map(
    ({ path, message }) => ({ path, message })
  );

describe('list field rows validation', () => {
  it('lets a value repeat, as an argument list does', () => {
    expect(issuesOf(sourceRanges(), rowsOf('--verbose', '--verbose'))).toEqual(
      []
    );
  });

  it('reports a value set twice on the later row once the list is unique', () => {
    const unique = sourceRanges({ validation: { unique: true } });

    expect(
      issuesOf(unique, rowsOf('10.0.0.0/8', '192.168.0.0/16', '10.0.0.0/8'))
    ).toEqual([
      { path: [2, 'value'], message: Messages.duplicateValue('10.0.0.0/8') },
    ]);
    expect(
      issuesOf(
        sourceRanges({ validation: { unique: false } }),
        rowsOf('10.0.0.0/8', '10.0.0.0/8')
      )
    ).toEqual([]);
  });

  it('writes a value as the item transforms it, finding a repeated one after it', () => {
    const component = sourceRanges({
      validation: { unique: true },
      itemTemplate: {
        uiType: FieldType.Text,
        validation: { trim: true, toLowerCase: true },
      },
    });

    expect(schemaOf(component).parse(rowsOf(' Example.COM '))).toEqual(
      rowsOf('example.com')
    );
    expect(issuesOf(component, rowsOf('example.com', ' Example.COM'))).toEqual([
      { path: [1, 'value'], message: Messages.duplicateValue('example.com') },
    ]);
  });

  it("reports the item's rules on the row that breaks them", () => {
    const component = sourceRanges({
      itemTemplate: {
        uiType: FieldType.Text,
        validation: {
          max: 14,
          regex: { pattern: '^[0-9./]+$', message: 'Use CIDR notation' },
        },
      },
    });

    expect(
      issuesOf(component, rowsOf('10.0.0.0/8', 'office', '192.168.100.0/24'))
    ).toEqual([
      { path: [1, 'value'], message: 'Use CIDR notation' },
      { path: [2, 'value'], message: expect.any(String) },
    ]);
  });

  it('asks for a value in a row the item trims to nothing', () => {
    const component = sourceRanges({
      itemTemplate: { uiType: FieldType.Text, validation: { trim: true } },
    });

    expect(issuesOf(component, rowsOf('10.0.0.0/8', '   '))).toEqual([
      { path: [1, 'value'], message: Messages.valueRequired },
    ]);
  });

  it('counts only filled rows against required, min and max', () => {
    expect(
      issuesOf(sourceRanges({ validation: { required: true } }), rowsOf(''))
    ).toEqual([{ path: ['root'], message: RowsMessages.required }]);

    expect(
      issuesOf(sourceRanges({ validation: { min: 2 } }), rowsOf('a', ''))
    ).toEqual([{ path: ['root'], message: RowsMessages.tooFewEntries(2, 1) }]);

    expect(
      issuesOf(sourceRanges({ validation: { max: 1 } }), rowsOf('a', '', 'a'))
    ).toEqual([{ path: ['root'], message: RowsMessages.tooManyEntries(1, 1) }]);
  });

  it("puts the count issue beside the rows' issues, not over them", () => {
    expect(
      issuesOf(
        sourceRanges({ validation: { max: 1, unique: true } }),
        rowsOf('a', 'a')
      )
    ).toEqual([
      { path: [1, 'value'], message: Messages.duplicateValue('a') },
      { path: ['root'], message: RowsMessages.tooManyEntries(1, 1) },
    ]);
  });

  it('refuses what is not rows, as a saved list becomes in a form that loads it unconverted', () => {
    // The form's array turns each string into an object with nothing in it,
    // which would be saved in place of the entry.
    expect(issuesOf(sourceRanges(), [{}, {}])).toEqual([
      { path: ['root'], message: RowsMessages.notRows },
    ]);
    expect(
      issuesOf(sourceRanges({ validation: { max: 1 } }), [
        '10.0.0.0/8',
        '192.168.0.0/16',
      ])
    ).toEqual([{ path: ['root'], message: RowsMessages.notRows }]);
  });

  it('applies the rules of the form mode, `unique` among them', () => {
    const schema = preprocessSchema(
      schemaWith(
        sourceRanges({
          validation: { max: 3, modes: { edit: { unique: true, max: 1 } } },
        })
      ),
      undefined,
      FormMode.Edit
    );
    const issuesIn = (formMode: FormMode) =>
      (
        buildZodSchema(schema, 'replicaSet', { formMode }).schema.safeParse({
          spec: { proxy: { sourceRanges: rowsOf('a', 'a') } },
        }).error?.issues ?? []
      ).map(({ message }) => message);

    expect(issuesIn(FormMode.New)).toEqual([]);
    expect(issuesIn(FormMode.Edit)).toEqual([
      Messages.duplicateValue('a'),
      RowsMessages.tooManyEntries(1, 1),
    ]);
  });
});

describe('list field through the pipeline', () => {
  it('starts the form from the default list as rows, in its order', () => {
    const component = sourceRanges({
      fieldParams: { defaultValue: ['10.0.0.0/8', '192.168.0.0/16'] },
    });

    expect(
      getByPath(getDefaultValues(schemaWith(component), 'replicaSet'), PATH)
    ).toEqual(rowsOf('10.0.0.0/8', '192.168.0.0/16'));
  });

  it.each([
    ['a list holding a number', '[10, 20]'],
    ['a map', '{ office: 10.0.0.0/8 }'],
    ['a string', '10.0.0.0/8'],
  ])('starts empty from a default that is %s', (_, defaultValue) => {
    // As a provider's YAML may hold it: read as rows, the 10 would be written
    // back as the string "10".
    const schema: TopologyUISchemas = parse(`
      replicaSet:
        sections:
          advanced:
            components:
              sourceRanges:
                uiType: list
                path: ${PATH}
                fieldParams: { label: Source ranges, defaultValue: ${defaultValue} }
                itemTemplate: { uiType: text }
    `);

    expect(
      getByPath(getDefaultValues(schema, 'replicaSet'), PATH)
    ).toBeUndefined();
  });

  it('reads a saved list into rows for edit, in its order', () => {
    expect(
      extractInstanceValues(
        sectionsWith(sourceRanges()),
        { spec: { proxy: { sourceRanges: ['192.168.0.0/16', '10.0.0.0/8'] } } },
        FormMode.Edit
      )
    ).toEqual({
      spec: { proxy: { sourceRanges: rowsOf('192.168.0.0/16', '10.0.0.0/8') } },
    });
  });

  it('submits the filled rows as a list in their order, and nothing when every row is empty', () => {
    const submit = (rows: unknown) =>
      postprocessSchemaData(
        { spec: { proxy: { sourceRanges: rows, replicas: 2 } } },
        { schema: schemaWith(sourceRanges()), selectedTopology: 'replicaSet' }
      );

    expect(submit(rowsOf('b', '', 'a', 'b'))).toEqual({
      spec: { proxy: { sourceRanges: ['b', 'a', 'b'], replicas: 2 } },
    });
    expect(submit(rowsOf('', ''))).toEqual({
      spec: { proxy: { replicas: 2 } },
    });
    expect(submit([])).toEqual({ spec: { proxy: { replicas: 2 } } });
  });

  it('replaces the saved list on edit, and removes it once no entry is left', () => {
    const edit = (rows: unknown) =>
      mergeSectionEdit({
        spec: { proxy: { sourceRanges: ['10.0.0.0/8', '192.168.0.0/16'] } },
        formData: { spec: { proxy: { sourceRanges: rows } } },
        sections: sectionsWith(sourceRanges()),
        sectionKey: 'advanced',
        topology: 'replicaSet',
      });

    expect(edit(rowsOf('192.168.0.0/16'))).toEqual({
      proxy: { sourceRanges: ['192.168.0.0/16'] },
    });
    expect(edit([])).toEqual({});
  });

  it('keeps a saved list as it is through an edit that changes nothing, repeated values included', () => {
    const saved = { proxy: { sourceRanges: ['-v', '--name', 'db', '-v'] } };
    const sections = sectionsWith(sourceRanges());

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

  // What an edit that changes nothing saves for a list the form did not
  // write. Only free-form \`parameters\` can hold one, e.g. set with kubectl.
  it.each<[string, unknown, unknown]>([
    [
      'reads every entry that is not a string as text',
      [3, true, null, { a: 1 }, 'x'],
      ['3', 'true', 'null', '[object Object]', 'x'],
    ],
    ['drops an entry that is an empty string', ['a', '', 'b'], ['a', 'b']],
    ['removes a string', '10.0.0.0/8', undefined],
    ['removes a map', { office: '10.0.0.0/8' }, undefined],
  ])('%s when the saved value is not a list of strings', (_, saved, sent) => {
    const sections = sectionsWith(sourceRanges());
    const spec = { proxy: { sourceRanges: saved, replicas: 2 } };

    expect(
      mergeSectionEdit({
        spec,
        formData: extractInstanceValues(sections, { spec }, FormMode.Edit),
        sections,
        sectionKey: 'advanced',
        topology: 'replicaSet',
      })
    ).toEqual({
      proxy: { ...(sent !== undefined && { sourceRanges: sent }), replicas: 2 },
    });
  });

  it('leaves a saved list the form holds as it is, instead of reading it as rows', () => {
    // As forms that load saved values unconverted, like the monitoring
    // entries editor, hand it over.
    expect(
      postprocessSchemaData(
        { spec: { proxy: { sourceRanges: ['10.0.0.0/8', ''] } } },
        { schema: schemaWith(sourceRanges()), selectedTopology: 'replicaSet' }
      )
    ).toEqual({ spec: { proxy: { sourceRanges: ['10.0.0.0/8', ''] } } });
    // Objects that are not rows are no rows either.
    expect(
      postprocessSchemaData(
        { spec: { proxy: { sourceRanges: [{ name: 'office' }] } } },
        { schema: schemaWith(sourceRanges()), selectedTopology: 'replicaSet' }
      )
    ).toEqual({ spec: { proxy: { sourceRanges: [{ name: 'office' }] } } });
  });
});

describe('list field in a read-only view', () => {
  const tokens = sourceRanges({
    itemTemplate: { uiType: FieldType.Text, fieldParams: { type: 'password' } },
  });

  it('shows the entries of the form rows and of the saved list alike, without a key', () => {
    const entries = [{ value: '10.0.0.0/8' }, { value: '192.168.0.0/16' }];

    expect(
      getRowsFieldDisplayEntries(
        rowsOf('10.0.0.0/8', '', '192.168.0.0/16'),
        sourceRanges()
      )
    ).toEqual(entries);
    expect(
      getRowsFieldDisplayEntries(
        ['10.0.0.0/8', '192.168.0.0/16'],
        sourceRanges()
      )
    ).toEqual(entries);
    expect(getRowsFieldDisplayEntries(undefined, sourceRanges())).toEqual([]);
  });

  it('masks every entry of a password item and says so', () => {
    expect(getRowsFieldDisplayEntries(['s3cr3t'], tokens)).toEqual([
      { value: expect.not.stringContaining('s3cr3t'), masked: true },
    ]);
  });

  it('shows a saved entry that is not a string as stored', () => {
    expect(getRowsFieldDisplayEntries([8080, 'http'], sourceRanges())).toEqual([
      { value: '8080' },
      { value: 'http' },
    ]);
  });
});

describe('the row a list draws', () => {
  const component: ListComponent = parse(`
    uiType: list
    path: ${PATH}
    fieldParams: { label: Tokens }
    itemTemplate:
      uiType: text
      fieldParams: { type: password, defaultValue: s3, badge: Gi, badgeToApi: true }
      dataSource: { provider: secrets }
      modes: { edit: { uiType: hidden } }
      validation:
        max: 3
        celExpressions: [{ celExpr: "1 == 2", message: Never }]
  `);

  it('is the item field alone, named Value, without what a row cannot act on yet', () => {
    expect(getRow(component)).toEqual({
      uiType: 'group',
      groupType: 'line',
      components: {
        value: {
          uiType: 'text',
          path: 'value',
          fieldParams: { placeholder: 'Value', type: 'password' },
          validation: { max: 3 },
        },
      },
    });
  });

  it('names what it ignores where the schema writes it, for a list and for a map', () => {
    const labels = parse(`
      uiType: map
      path: spec.labels
      fieldParams: { label: Labels }
      itemTemplate:
        uiType: group
        components:
          key: { uiType: text, path: key }
          value: { uiType: text, path: value, fieldParams: { badge: Gi } }
    `);
    const [list, map] = lintRowsFields({
      advanced: { components: { tokens: component, labels } },
    }).map(({ rule, message }) => ({ rule, message }));

    expect(list).toEqual({
      rule: 'list-ignored-option',
      message: expect.stringContaining(
        'sets itemTemplate.dataSource, itemTemplate.modes, itemTemplate.fieldParams.defaultValue, itemTemplate.fieldParams.badge, itemTemplate.fieldParams.badgeToApi, itemTemplate.validation.celExpressions,'
      ),
    });
    expect(map).toEqual({
      rule: 'map-ignored-option',
      message: expect.stringContaining(
        'sets itemTemplate.components.value.fieldParams.badge,'
      ),
    });
  });
});

describe('list field in a shape this version does not support', () => {
  // As a newer provider could ship them. `replicas` is a field next to the
  // list that the form does edit.
  const schemaWithList = (list: string): TopologyUISchemas =>
    parse(`
      replicaSet:
        sections:
          advanced:
            components:
              users:
                uiType: list
                path: spec.users
                fieldParams: { label: Users }
                validation:
                  required: true
                  max: 0
                  celExpressions: [{ celExpr: "1 == 2", message: Never }]
                ${list}
              replicas:
                uiType: number
                path: spec.replicas
                fieldParams: { label: Replicas }
    `);

  const UNSUPPORTED: [string, string][] = [
    [
      'a list of objects',
      'itemTemplate: { uiType: group, components: { name: { uiType: text, path: name } } }',
    ],
    ['an item that is not text', 'itemTemplate: { uiType: number }'],
    ['an item with a path', 'itemTemplate: { uiType: text, path: value }'],
    ['an item with an id', 'itemTemplate: { uiType: text, id: entry }'],
    [
      'a widget as the item',
      'itemTemplate: { uiType: widget, widgetType: chips }',
    ],
    ['no item template', 'techPreview: true'],
  ];

  describe.each(UNSUPPORTED)('%s', (_, list) => {
    const declared = schemaWithList(list);

    it.each([
      ['preprocessed', preprocessSchema(declared)],
      ['raw, as forms that skip preprocess get it', declared],
    ])('is left alone when %s', (__, schema) => {
      const options = { schema, selectedTopology: 'replicaSet' };
      const saved = { users: [{ name: 'alice' }], replicas: 3 };
      const { sections } = schema.replicaSet;
      const formData = extractInstanceValues(
        sections,
        { spec: saved },
        FormMode.Edit
      );

      // Its rules don't run: the value is not the form's to judge.
      expect(
        buildZodSchema(schema, 'replicaSet').schema.safeParse(formData).success
      ).toBe(true);
      // An edit of the field next to it keeps the saved list.
      expect(
        mergeSectionEdit({
          spec: saved,
          formData: deepMerge(formData, { spec: { replicas: 5 } }),
          sections,
          sectionKey: 'advanced',
          topology: 'replicaSet',
        })
      ).toEqual({ users: [{ name: 'alice' }], replicas: 5 });
      // A new form sends nothing in its place.
      expect(
        postprocessSchemaData(getDefaultValues(schema, 'replicaSet'), options)
      ).toEqual({});
    });
  });
});
