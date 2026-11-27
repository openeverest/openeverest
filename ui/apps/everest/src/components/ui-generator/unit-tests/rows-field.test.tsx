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

import { ReactNode } from 'react';
import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { FormProvider, useForm, useFormState } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { parse } from 'yaml';
import { TestWrapper } from 'utils/test';
import { KEY_VALUE_ROW_YAML } from 'components/ui-generator/test-fixtures/key-value-row.fixture';
import { UIGenerator } from '../ui-generator';
import { FormMode, Section, TopologyUISchemas } from '../ui-generator.types';
import { buildZodSchema } from '../utils/schema-builder';
import { getDefaultValues } from '../utils/default-values';
import { preprocessSchema } from '../utils/preprocess/preprocess-schema';
import { getSectionFormZodSchema, toSectionForm } from '../utils/section-form';

// Says whether the form differs from the values it opened with.
const ChangeProbe = () => {
  const { isDirty } = useFormState();
  return <output>{isDirty ? 'changed' : 'unchanged'}</output>;
};

const FormWrapper = ({
  children,
  uiSchema,
}: {
  children: ReactNode;
  uiSchema: TopologyUISchemas;
}) => {
  const methods = useForm<Record<string, unknown>>({
    resolver: zodResolver(buildZodSchema(uiSchema, 'replicaSet').schema),
    mode: 'onChange',
    defaultValues: getDefaultValues(uiSchema, 'replicaSet'),
  });
  return (
    <FormProvider {...methods}>
      {children}
      <ChangeProbe />
    </FormProvider>
  );
};

const rowsForm = (uiSchema: TopologyUISchemas) => (
  <TestWrapper>
    <FormWrapper uiSchema={uiSchema}>
      <UIGenerator
        sections={uiSchema.replicaSet.sections}
        sectionKey="advanced"
      />
    </FormWrapper>
  </TestWrapper>
);

// Both fields the form edits as rows, each with a row of its own.
const ROWS_FIELDS: [uiType: string, itemTemplate: string, input: string][] = [
  ['map', KEY_VALUE_ROW_YAML, 'Key'],
  ['list', '{ uiType: text }', 'Value'],
];

describe.each(ROWS_FIELDS)(
  'UIGenerator - a %s the schema gives no fieldParams',
  (uiType, itemTemplate, input) => {
    // As a provider's YAML may hold it: only `uiType`, `path` and
    // `itemTemplate` are required.
    const field = `{ uiType: ${uiType}, path: parameters.entries, itemTemplate: ${itemTemplate} }`;
    const declared: TopologyUISchemas = parse(`
      replicaSet:
        sections:
          advanced:
            components:
              entries: ${field}
    `);

    it.each([
      ['preprocessed', preprocessSchema(declared)],
      ['raw, as forms that skip preprocess get it', declared],
    ])('builds its rules and draws its rows when %s', (_, uiSchema) => {
      render(rowsForm(uiSchema));

      fireEvent.click(screen.getByRole('button', { name: 'Add' }));

      expect(screen.getByRole('textbox', { name: `${input} 1` })).toBeEnabled();
    });

    it('is checked against the contract of a section form', () => {
      const section: Section = parse(`components: { entries: ${field} }`);
      const form = toSectionForm('destination', section, {
        contract: {
          path: 'parameters',
          schema: {
            type: 'object',
            required: ['entries'],
            properties: { entries: {} },
          },
        },
      });

      expect(
        getSectionFormZodSchema(form)
          .safeParse({ parameters: {} })
          .error?.issues.map(({ path }) => path.join('.'))
      ).toEqual(['parameters.entries']);
    });
  }
);

// The row of each field, with `rules` set on the input the tests type in: a
// map's key, a list's item.
const ROWS_FIELDS_WITH_RULES: [
  uiType: string,
  itemTemplate: (rules?: string) => string,
  input: string,
  // One entry, as a schema's `defaultValue` writes it.
  entry: string,
][] = [
  [
    'map',
    (rules = '') =>
      `{ uiType: group, groupType: line, components: { key: { uiType: text, path: key, ${rules} }, value: { uiType: text, path: value } } }`,
    'Key',
    '{ team: db }',
  ],
  ['list', (rules = '') => `{ uiType: text, ${rules} }`, 'Value', '[db]'],
];

describe.each(ROWS_FIELDS_WITH_RULES)(
  'UIGenerator - a %s with no entries',
  (uiType, itemTemplate, input, entry) => {
    const schemaWith = (
      { fieldParams = '', rules = '' } = {},
      formMode = FormMode.New
    ): TopologyUISchemas =>
      preprocessSchema(
        parse(`
          replicaSet:
            sections:
              advanced:
                components:
                  entries:
                    uiType: ${uiType}
                    path: spec.entries
                    fieldParams: { label: Entries, ${fieldParams} }
                    itemTemplate: ${itemTemplate(rules)}
        `),
        undefined,
        formMode
      );
    const firstInput = () =>
      screen.getByRole('textbox', { name: `${input} 1` });
    const removeFirst = () => screen.getByRole('button', { name: 'Remove 1' });
    const type = async (value: string) =>
      act(async () => {
        fireEvent.change(firstInput(), { target: { value } });
      });

    it('opens with one empty row to fill, which changes nothing until the user types', async () => {
      render(rowsForm(schemaWith()));

      expect(
        screen.queryByRole('textbox', { name: `${input} 2` })
      ).not.toBeInTheDocument();
      expect(firstInput()).toBeEnabled();
      expect(firstInput()).toHaveValue('');
      expect(screen.getByRole('button', { name: 'Add' })).toBeEnabled();
      expect(screen.getByText('unchanged')).toBeInTheDocument();

      await type('team');
      expect(screen.getByText('changed')).toBeInTheDocument();

      // Emptied again, the form holds what it opened with.
      await type('');
      expect(screen.getByText('unchanged')).toBeInTheDocument();
    });

    it("lets the user fill the row though the row's field is disabled, and checks what they type", async () => {
      render(
        rowsForm(
          schemaWith({
            rules:
              "fieldParams: { disabled: true }, validation: { regex: { pattern: '^[a-z]+$', message: Lowercase only } }",
          })
        )
      );

      expect(firstInput()).toBeEnabled();

      await type('Team');
      expect(await screen.findByText('Lowercase only')).toBeInTheDocument();
    });

    it.each([
      ['the field is disabled', 'disabled: true'],
      ['no entry can be added', 'disableAdd: true'],
    ])('shows the row disabled when %s', (_, fieldParams) => {
      render(rowsForm(schemaWith({ fieldParams })));

      expect(firstInput()).toBeDisabled();
      expect(removeFirst()).toBeDisabled();
      expect(screen.getByRole('button', { name: 'Add' })).toBeDisabled();
    });

    it('has nothing to remove until something is typed, and leaves an empty row when the last row is removed', async () => {
      render(rowsForm(schemaWith()));

      expect(removeFirst()).toBeDisabled();

      // With a second row, either can go, empty or not.
      fireEvent.click(screen.getByRole('button', { name: 'Add' }));
      expect(removeFirst()).toBeEnabled();
      fireEvent.click(screen.getByRole('button', { name: 'Remove 2' }));
      await waitFor(() => expect(removeFirst()).toBeDisabled());

      await type('team');
      expect(removeFirst()).toBeEnabled();

      fireEvent.click(removeFirst());
      await waitFor(() => expect(firstInput()).toHaveValue(''));
      expect(firstInput()).toBeEnabled();
      expect(removeFirst()).toBeDisabled();
      expect(screen.getByText('unchanged')).toBeInTheDocument();
    });

    it('leaves an empty row to fill when the last entry is removed', async () => {
      render(rowsForm(schemaWith({ fieldParams: `defaultValue: ${entry}` })));

      expect(firstInput()).not.toHaveValue('');
      fireEvent.click(removeFirst());

      await waitFor(() => expect(firstInput()).toHaveValue(''));
      expect(firstInput()).toBeEnabled();
      expect(screen.getByText('changed')).toBeInTheDocument();
    });

    it('leaves a disabled row when the last entry is removed and none can be added', async () => {
      render(
        rowsForm(
          schemaWith({
            fieldParams: `defaultValue: ${entry}, disableAdd: true`,
          })
        )
      );

      // The entry the form opened with can be changed and removed.
      expect(firstInput()).toBeEnabled();
      fireEvent.click(removeFirst());

      await waitFor(() => expect(firstInput()).toHaveValue(''));
      expect(firstInput()).toBeDisabled();
      expect(removeFirst()).toBeDisabled();
    });

    it('keeps the text under the rows clear of the next field, and adds no gap without it', () => {
      const { unmount } = render(
        rowsForm(
          schemaWith({ fieldParams: 'helperText: Added to the Service' })
        )
      );
      expect(screen.getByText('Added to the Service')).toHaveStyle({
        marginBottom: '8px',
      });
      unmount();

      const { container } = render(rowsForm(schemaWith()));
      expect(
        container.querySelector('.MuiFormHelperText-root')
      ).not.toBeInTheDocument();
    });
  }
);
