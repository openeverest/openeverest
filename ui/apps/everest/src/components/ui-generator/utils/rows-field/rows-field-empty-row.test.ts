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
  type Section,
  type TopologyUISchemas,
  FormMode,
  isRowsComponent,
} from 'components/ui-generator/ui-generator.types';
import { KEY_VALUE_ROW_YAML } from 'components/ui-generator/test-fixtures/key-value-row.fixture';
import { getDefaultValues } from '../default-values';
import { buildDefaultsFromComponents } from '../default-values/build-defaults-from-components';
import { extractInstanceValues } from '../default-values/extract-instance-values';
import {
  withEmptyFormValues,
  getCelValueReaders,
  readValuesForCel,
} from '../field-codecs';
import { LIST_FIELD_KIND } from '../list-field/list-field-kind';
import { MAP_FIELD_KIND } from '../map-field/map-field-kind';
import { deepMerge, getByPath } from '../object-path';
import { mergeSectionEdit } from '../postprocess/merge-section-edit';
import { postprocessSchemaData } from '../postprocess/postprocess-schema';
import { preprocessSchema } from '../preprocess/preprocess-schema';
import { buildZodSchema } from '../schema-builder';
import { switchTopologyValues } from '../topology-scope';
import {
  getSectionFormDefaults,
  getSectionFormUpdate,
  getSectionFormValues,
  getSectionFormZodSchema,
  postprocessSectionForm,
  toSectionForm,
} from '../section-form';
import { Messages } from './rows-field.messages';
import type { Row, RowsFieldKind } from './rows-field.types';
import { createAddedRow } from '.';
import { getRowsFieldDisplayEntries } from './rows-field-display';

const PATH = 'spec.proxy.entries';

interface Kind {
  kind: RowsFieldKind;
  itemTemplate: string;
  // A value with no entries, a row holding one entry and that entry as the
  // API takes it, each in YAML.
  noEntries: string;
  row: Row;
  saved: unknown;
  // A row holding another entry, as a user types it.
  typed: Row;
}

const KINDS: [string, Kind][] = [
  [
    'map',
    {
      kind: MAP_FIELD_KIND,
      itemTemplate: KEY_VALUE_ROW_YAML,
      noEntries: '{}',
      row: { key: 'team', value: 'db' },
      saved: { team: 'db' },
      typed: { key: 'tier', value: 'gold' },
    },
  ],
  [
    'list',
    {
      kind: LIST_FIELD_KIND,
      itemTemplate: '{ uiType: text }',
      noEntries: '[]',
      row: { value: 'db' },
      saved: ['db'],
      typed: { value: 'gold' },
    },
  ],
];

describe.each(KINDS)(
  'the one empty row of a %s with no entries',
  (uiType, { kind, itemTemplate, noEntries, row, saved, typed }) => {
    // The field as a YAML schema writes it; `options` are more of its keys.
    const schemaWith = (options = '', formMode?: FormMode): TopologyUISchemas =>
      preprocessSchema(
        parse(`
          replicaSet:
            sections:
              advanced:
                components:
                  entries:
                    uiType: ${uiType}
                    path: ${PATH}
                    itemTemplate: ${itemTemplate}
                    ${options}
                  owner: { uiType: text, path: spec.owner }
        `),
        undefined,
        formMode
      );
    // The row is one the user may fill, not one the form opened with.
    const EMPTY_ROW = [createAddedRow(kind)];
    const rowsOf = (values: Record<string, unknown>) => getByPath(values, PATH);
    const fromInstance = (spec: Record<string, unknown>, formMode: FormMode) =>
      extractInstanceValues(
        schemaWith('', formMode).replicaSet.sections,
        { spec },
        formMode
      );
    // The values of a form that shows an instance and no defaults, as the
    // section edit dialog builds them.
    const editValues = (spec: Record<string, unknown>, options = '') => {
      const { sections } = schemaWith(options, FormMode.Edit).replicaSet;
      return withEmptyFormValues(
        extractInstanceValues(sections, { spec }, FormMode.Edit),
        sections
      );
    };
    // One entry, as a schema writes it in \`defaultValue\`.
    const ONE_ENTRY = JSON.stringify(saved);

    it.each([
      ['no default', ''],
      [
        'a default with no entries',
        `fieldParams: { defaultValue: ${noEntries} }`,
      ],
      ['a default it cannot use', 'fieldParams: { defaultValue: 3 }'],
    ])('opens a new form that has %s', (_, options) => {
      expect(
        rowsOf(getDefaultValues(schemaWith(options), 'replicaSet'))
      ).toEqual(EMPTY_ROW);
    });

    it.each([
      ['holds no value for it', {}],
      [
        'holds a value with no entries',
        { proxy: { entries: parse(noEntries) } },
      ],
    ])('opens the edit of an instance that %s', (_, spec) => {
      expect(rowsOf(editValues(spec))).toEqual(EMPTY_ROW);
    });

    it('opens that edit in place of a default too, which an edit does not apply', () => {
      expect(
        rowsOf(editValues({}, `fieldParams: { defaultValue: ${ONE_ENTRY} }`))
      ).toEqual(EMPTY_ROW);
    });

    it('is no value of the instance, so the values read from one can be laid over defaults', () => {
      expect(rowsOf(fromInstance({}, FormMode.Edit))).toBeUndefined();
    });

    it('opens a restore from an instance that holds no value for it', () => {
      expect(rowsOf(fromInstance({}, FormMode.Restore))).toEqual(EMPTY_ROW);
    });

    it.each([
      ['restore', FormMode.Restore],
      ['preset, which a new form reads', FormMode.New],
    ])(
      'gives way to the default where a %s has no value for the field',
      (_, formMode) => {
        const { sections } = schemaWith(
          `fieldParams: { defaultValue: ${ONE_ENTRY} }`,
          formMode
        ).replicaSet;

        expect(
          rowsOf(extractInstanceValues(sections, { spec: {} }, formMode))
        ).toEqual([row]);
      }
    );

    it('is not among the defaults a form applies over the values it holds', () => {
      // As a backup class's defaults are applied on a class change: only what
      // the schema declares, so the rows the form holds are not replaced.
      const { components } = schemaWith().replicaSet.sections.advanced;

      expect(buildDefaultsFromComponents(components, 'advanced', true)).toEqual(
        {}
      );
    });

    it('is not put beside the entries a form opens with', () => {
      expect(rowsOf(editValues({ proxy: { entries: saved } }))).toEqual([row]);
    });

    it('writes nothing, so an edit that changes another field leaves the instance without the field', () => {
      const schema = schemaWith('', FormMode.Edit);
      const { sections } = schema.replicaSet;
      const spec = { owner: 'team-a' };
      const formData = editValues(spec);

      expect(
        postprocessSchemaData(getDefaultValues(schemaWith(), 'replicaSet'), {
          schema: schemaWith(),
          selectedTopology: 'replicaSet',
        })
      ).toEqual({});
      expect(
        mergeSectionEdit({
          spec,
          formData: deepMerge(formData, { spec: { owner: 'team-b' } }),
          sections,
          sectionKey: 'advanced',
          topology: 'replicaSet',
        })
      ).toEqual({ owner: 'team-b' });
    });

    it('is no entry for the count, nor for a rule, nor for a read-only view', () => {
      const schema = schemaWith(`validation:
                      required: true
                      celExpressions:
                        - { celExpr: "!has(${PATH})", message: Set }`);
      const values = getDefaultValues(schema, 'replicaSet');
      const { sections } = schema.replicaSet;
      const { entries } = sections.advanced.components;

      // Only the count fails: the rule finds the field unset, as the API does.
      expect(
        buildZodSchema(schema, 'replicaSet')
          .schema.safeParse(values)
          .error?.issues.map(({ path, message }) => [path.join('.'), message])
      ).toEqual([[`${PATH}.root`, Messages.required]]);
      expect(
        rowsOf(readValuesForCel(values, getCelValueReaders(sections)))
      ).toBeUndefined();
      if ('components' in entries || !isRowsComponent(entries)) {
        throw new Error('The field under test is not rendered as rows');
      }
      expect(getRowsFieldDisplayEntries(rowsOf(values), entries)).toEqual([]);
    });

    it('is not taken for an entry the user emptied, once removing is off', () => {
      const schema = schemaWith(
        'fieldParams: { disableRemove: true }',
        FormMode.Edit
      );
      const parse = (rows: Row[]) =>
        buildZodSchema(schema, 'replicaSet', {
          formMode: FormMode.Edit,
        }).schema.safeParse({ spec: { proxy: { entries: rows } } });

      expect(parse(EMPTY_ROW).success).toBe(true);
      // An emptied row the form opened with still is.
      expect(parse([kind.emptyRow]).success).toBe(false);
    });

    describe('on a switch of topology', () => {
      // `standalone` and `cluster` both have `entries`, with a default in
      // `cluster` alone, which also has `more`.
      const field = (path: string, fieldParams = '{}') =>
        `{ uiType: ${uiType}, path: ${path}, fieldParams: ${fieldParams}, itemTemplate: ${itemTemplate} }`;
      const schema: TopologyUISchemas = preprocessSchema(
        parse(`
          standalone:
            sections:
              advanced: { components: { entries: ${field('spec.entries')} } }
          cluster:
            sections:
              advanced:
                components:
                  entries: ${field('spec.entries', `{ defaultValue: ${ONE_ENTRY} }`)}
                  more: ${field('spec.more')}
        `)
      );
      const switchFrom = (entries: Row[], topology = 'cluster') =>
        switchTopologyValues({ spec: { entries } }, schema, topology);

      it('gives way to the default the new topology has for the field', () => {
        expect(switchFrom(EMPTY_ROW)).toEqual({
          spec: { entries: [row], more: EMPTY_ROW },
        });
        // Rows added and left empty hold no entry either.
        expect(
          switchFrom([createAddedRow(kind), createAddedRow(kind)])
        ).toMatchObject({ spec: { entries: [row] } });
      });

      it('is what a field holds again when the new topology has no default for it', () => {
        expect(
          switchTopologyValues(
            getDefaultValues(schema, 'cluster'),
            schema,
            'standalone'
          )
        ).toEqual({ spec: { entries: [row] } });
        expect(switchFrom(EMPTY_ROW, 'standalone')).toEqual({
          spec: { entries: EMPTY_ROW },
        });
      });

      it('leaves the rows the user filled as they are', () => {
        expect(switchFrom([typed, createAddedRow(kind)])).toMatchObject({
          spec: { entries: [typed, createAddedRow(kind)] },
        });
      });
    });

    describe('in a section form', () => {
      const section: Section = parse(`
        components:
          entries:
            uiType: ${uiType}
            path: parameters.entries
            itemTemplate: ${itemTemplate}
      `);
      const SAVED = { parameters: { url: 'http://pmm' } };

      it.each([FormMode.New, FormMode.Edit])(
        'opens the %s form with the row, and sends nothing for it',
        (formMode) => {
          const form = toSectionForm('destination', section, { formMode });
          const values =
            formMode === FormMode.New
              ? getSectionFormDefaults(form)
              : getSectionFormValues(form, SAVED);

          expect(getByPath(values, 'parameters.entries')).toEqual(EMPTY_ROW);
          expect(postprocessSectionForm(form, values)).toEqual({});
          expect(getSectionFormUpdate(form, values, SAVED)).toEqual(SAVED);
        }
      );

      describe('with a default', () => {
        // The field as a class may lock it: it needs an entry and takes no
        // new one, so a form that lost the default could never be saved.
        const withDefault: Section = parse(`
          components:
            entries:
              uiType: ${uiType}
              path: parameters.entries
              fieldParams: { defaultValue: ${ONE_ENTRY}, disableAdd: true }
              validation: { required: true }
              itemTemplate: ${itemTemplate}
        `);
        const form = toSectionForm('destination', withDefault, {
          formMode: FormMode.Edit,
        });
        const entriesOf = (savedObject: Record<string, unknown>) =>
          getByPath(
            getSectionFormValues(form, savedObject),
            'parameters.entries'
          );

        it('gives way to the default in the edit of an object saved without the field', () => {
          const values = getSectionFormValues(form, SAVED);

          expect(getByPath(values, 'parameters.entries')).toEqual([row]);
          expect(getSectionFormZodSchema(form).safeParse(values).success).toBe(
            true
          );
        });

        it('stands for a saved value with no entries, which the default does not replace', () => {
          expect(
            entriesOf({ parameters: { entries: parse(noEntries) } })
          ).toEqual(EMPTY_ROW);
        });

        it('leaves the saved entries to the form', () => {
          const savedObject = postprocessSectionForm(form, {
            parameters: { entries: [typed] },
          });

          expect(entriesOf(savedObject)).toEqual([typed]);
        });
      });
    });
  }
);
