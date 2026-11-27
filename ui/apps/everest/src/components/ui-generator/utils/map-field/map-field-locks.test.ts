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
  type MapComponent,
  type TopologyUISchemas,
  FieldType,
  FormMode,
} from 'components/ui-generator/ui-generator.types';
import { keyValueRow } from 'components/ui-generator/test-fixtures/key-value-row.fixture';
import { getCelValueReaders, readValuesForCel } from '../field-codecs';
import { postprocessSchemaData } from '../postprocess/postprocess-schema';
import { preprocessSchema } from '../preprocess/preprocess-schema';
import { buildZodSchema } from '../schema-builder';
import { createAddedRow, getRowsFieldDisplayEntries } from '../rows-field';
import type { Row } from '../rows-field';
import { Messages as RowsMessages } from '../rows-field/rows-field.messages';
import { MAP_FIELD_KIND } from './map-field-kind';
import { Messages } from './map-field.messages';

const PATH = ['spec', 'labels'];
const LOWERCASE = { pattern: '^[a-z]+$', message: 'Lowercase only' };

const labels = (
  overrides: Partial<
    Pick<MapComponent, 'fieldParams' | 'itemTemplate' | 'validation'>
  > = {}
): MapComponent => ({
  uiType: FieldType.Map,
  path: PATH.join('.'),
  fieldParams: { label: 'Labels' },
  itemTemplate: keyValueRow(),
  ...overrides,
});

// The schema as a form of the given mode gets it, with `fieldParams.modes`
// applied to the map and to its row's fields.
const schemaWith = (
  component: MapComponent,
  formMode?: FormMode
): TopologyUISchemas =>
  preprocessSchema(
    {
      replicaSet: {
        sections: { advanced: { components: { labels: component } } },
      },
    },
    undefined,
    formMode
  );

const issuesOf = (
  component: MapComponent,
  rows: Row[],
  formMode?: FormMode
) => {
  const { schema } = buildZodSchema(
    schemaWith(component, formMode),
    'replicaSet',
    { formMode }
  );
  return (schema.safeParse({ spec: { labels: rows } }).error?.issues ?? []).map(
    ({ path, message }) => ({ path: path.slice(PATH.length), message })
  );
};

// A row the form opened with: a saved entry, or one of the map's default.
const opened = (key: string, value: string): Row => ({ key, value });
// A row the user added with the Add button and then filled.
const added = (key: string, value: string): Row => ({
  ...createAddedRow(MAP_FIELD_KIND),
  key,
  value,
});

describe('a map with no fieldParams', () => {
  it('still builds its rules, since a schema may leave fieldParams out', () => {
    // As a provider's YAML may hold it; the types ask for fieldParams.
    const schema: TopologyUISchemas = parse(`
      replicaSet:
        sections:
          advanced:
            components:
              labels:
                uiType: map
                path: spec.labels
                itemTemplate:
                  uiType: group
                  components:
                    key: { uiType: text, path: key }
                    value: { uiType: text, path: value }
    `);
    const { schema: zodSchema } = buildZodSchema(
      preprocessSchema(schema),
      'replicaSet'
    );

    expect(
      zodSchema.safeParse({ spec: { labels: [{ key: 'team', value: 'db' }] } })
        .success
    ).toBe(true);
  });
});

describe('a row field disabled in a map', () => {
  it("skips the key's rules in a row the form opened with, and applies them to an added row", () => {
    const component = labels({
      itemTemplate: keyValueRow({
        key: {
          fieldParams: { disabled: true },
          validation: { regex: LOWERCASE },
        },
      }),
    });

    expect(
      issuesOf(component, [opened('Team', 'db'), added('Tier', 'gold')])
    ).toEqual([{ path: [1, 'key'], message: LOWERCASE.message }]);
  });

  it('requires a value in an added row only, when the value is the fixed field', () => {
    const component = labels({
      itemTemplate: keyValueRow({
        value: {
          fieldParams: { disabled: true },
          validation: { required: true },
        },
      }),
    });

    expect(
      issuesOf(component, [opened('team', ''), added('tier', '')])
    ).toEqual([{ path: [1, 'value'], message: expect.any(String) }]);
  });

  it('fixes the rows the form opened with in the mode that says so, and in no other', () => {
    const component = labels({
      itemTemplate: keyValueRow({
        key: {
          fieldParams: { modes: { edit: { disabled: true } } },
          validation: { regex: LOWERCASE },
        },
      }),
    });
    const rows = [opened('Team', 'db'), added('Tier', 'gold')];
    const wrongKey = (row: number) => ({
      path: [row, 'key'],
      message: LOWERCASE.message,
    });

    expect(issuesOf(component, rows, FormMode.Edit)).toEqual([wrongKey(1)]);
    // In the wizard the key is enabled in every row, a default's too.
    expect(issuesOf(component, rows, FormMode.New)).toEqual([
      wrongKey(0),
      wrongKey(1),
    ]);
  });

  it('finds a key an added row repeats, as the added row writes it', () => {
    const component = labels({
      itemTemplate: keyValueRow({
        key: { fieldParams: { disabled: true }, validation: { trim: true } },
      }),
    });

    expect(
      issuesOf(component, [opened('team', 'db'), added(' team ', 'core')])
    ).toEqual([{ path: [1, 'key'], message: Messages.duplicateKey('team') }]);
  });
});

describe('the mark of a row added to a map', () => {
  const rows = [opened('team', 'db'), added('tier', 'gold'), added('', '')];

  it('keeps a row the form opened with from being emptied away once removing is off', () => {
    const locked = labels({
      fieldParams: { label: 'Labels', disableRemove: true },
    });
    const emptied = [opened('', ''), added('', '')];

    // An emptied row is dropped like any empty row, which removes its entry.
    expect(issuesOf(labels(), emptied)).toEqual([]);
    // The added row is still the user's to leave empty.
    expect(issuesOf(locked, emptied)).toEqual([
      { path: [0, 'key'], message: Messages.keyRequired },
    ]);
  });

  it('is not written to the API, and a row added but left empty is dropped', () => {
    expect(
      postprocessSchemaData(
        { spec: { labels: rows } },
        { schema: schemaWith(labels()), selectedTopology: 'replicaSet' }
      )
    ).toEqual({ spec: { labels: { team: 'db', tier: 'gold' } } });
  });

  it('is not in the map a CEL rule reads', () => {
    const { sections } = schemaWith(labels()).replicaSet;

    expect(
      readValuesForCel({ spec: { labels: rows } }, getCelValueReaders(sections))
    ).toEqual({ spec: { labels: { team: 'db', tier: 'gold' } } });
  });

  it('is not shown by a read-only view', () => {
    expect(getRowsFieldDisplayEntries(rows, labels())).toEqual([
      { key: 'team', value: 'db' },
      { key: 'tier', value: 'gold' },
    ]);
  });

  it('leaves the entry count to the filled rows, added or not', () => {
    expect(issuesOf(labels({ validation: { max: 2 } }), rows)).toEqual([]);
    expect(issuesOf(labels({ validation: { max: 1 } }), rows)).toEqual([
      { path: ['root'], message: RowsMessages.tooManyEntries(1, 1) },
    ]);
    expect(
      issuesOf(labels({ validation: { required: true } }), [added('', '')])
    ).toEqual([{ path: ['root'], message: RowsMessages.required }]);
  });
});
