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
  type ListComponent,
  type TopologyUISchemas,
  FieldType,
  FormMode,
} from 'components/ui-generator/ui-generator.types';
import { getCelValueReaders, readValuesForCel } from '../field-codecs';
import { postprocessSchemaData } from '../postprocess/postprocess-schema';
import { preprocessSchema } from '../preprocess/preprocess-schema';
import { buildZodSchema } from '../schema-builder';
import { createAddedRow, getRowsFieldDisplayEntries } from '../rows-field';
import type { Row } from '../rows-field';
import { LIST_FIELD_KIND } from './list-field-kind';
import { Messages } from './list-field.messages';

const PATH = ['spec', 'args'];
const FLAG = { pattern: '^--', message: 'Start with --' };

const args = (
  overrides: Partial<
    Pick<ListComponent, 'fieldParams' | 'itemTemplate' | 'validation'>
  > = {}
): ListComponent => ({
  uiType: FieldType.List,
  path: PATH.join('.'),
  fieldParams: { label: 'Arguments' },
  itemTemplate: { uiType: FieldType.Text },
  ...overrides,
});

// The schema as a form of the given mode gets it, with `fieldParams.modes`
// applied to the list and to its item.
const schemaWith = (
  component: ListComponent,
  formMode?: FormMode
): TopologyUISchemas =>
  preprocessSchema(
    {
      replicaSet: {
        sections: { advanced: { components: { args: component } } },
      },
    },
    undefined,
    formMode
  );

const issuesOf = (
  component: ListComponent,
  rows: Row[],
  formMode?: FormMode
) => {
  const { schema } = buildZodSchema(
    schemaWith(component, formMode),
    'replicaSet',
    { formMode }
  );
  return (schema.safeParse({ spec: { args: rows } }).error?.issues ?? []).map(
    ({ path, message }) => ({ path: path.slice(PATH.length), message })
  );
};

// A row the form opened with: a saved entry, or one of the list's default.
const opened = (value: string): Row => ({ value });
// A row the user added with the Add button and then filled.
const added = (value: string): Row => ({
  ...createAddedRow(LIST_FIELD_KIND),
  value,
});

describe('an item disabled in a list', () => {
  const fixedInEdit = args({
    itemTemplate: {
      uiType: FieldType.Text,
      fieldParams: { modes: { edit: { disabled: true } } },
      validation: { regex: FLAG },
    },
  });
  const rows = [opened('verbose'), added('quiet')];
  const wrongFlag = (row: number) => ({
    path: [row, 'value'],
    message: FLAG.message,
  });

  it("skips the item's rules in a row the form opened with, and applies them to an added row", () => {
    expect(issuesOf(fixedInEdit, rows, FormMode.Edit)).toEqual([wrongFlag(1)]);
  });

  it('checks every row in a mode that does not fix the item', () => {
    expect(issuesOf(fixedInEdit, rows, FormMode.New)).toEqual([
      wrongFlag(0),
      wrongFlag(1),
    ]);
  });

  it('finds a value an added row repeats, also when the other row is fixed', () => {
    const unique = args({
      validation: { unique: true },
      itemTemplate: {
        uiType: FieldType.Text,
        fieldParams: { disabled: true },
        validation: { trim: true },
      },
    });

    expect(
      issuesOf(unique, [opened('--verbose'), added(' --verbose ')])
    ).toEqual([
      { path: [1, 'value'], message: Messages.duplicateValue('--verbose') },
    ]);
  });
});

describe('the mark of a row added to a list', () => {
  const rows = [opened('--verbose'), added('--quiet'), added('')];

  it('keeps a row the form opened with from being emptied away once removing is off', () => {
    const locked = args({
      fieldParams: { label: 'Arguments', disableRemove: true },
    });
    const emptied = [opened(''), added('')];

    // An emptied row is dropped like any empty row, which removes its entry.
    expect(issuesOf(args(), emptied)).toEqual([]);
    // The added row is still the user's to leave empty.
    expect(issuesOf(locked, emptied)).toEqual([
      { path: [0, 'value'], message: Messages.valueRequired },
    ]);
  });

  it('is not written to the API, and a row added but left empty is dropped', () => {
    expect(
      postprocessSchemaData(
        { spec: { args: rows } },
        { schema: schemaWith(args()), selectedTopology: 'replicaSet' }
      )
    ).toEqual({ spec: { args: ['--verbose', '--quiet'] } });
  });

  it('is not in the list a CEL rule reads', () => {
    const { sections } = schemaWith(args()).replicaSet;

    expect(
      readValuesForCel({ spec: { args: rows } }, getCelValueReaders(sections))
    ).toEqual({ spec: { args: ['--verbose', '--quiet'] } });
  });

  it('is not shown by a read-only view', () => {
    expect(getRowsFieldDisplayEntries(rows, args())).toEqual([
      { value: '--verbose' },
      { value: '--quiet' },
    ]);
  });
});
