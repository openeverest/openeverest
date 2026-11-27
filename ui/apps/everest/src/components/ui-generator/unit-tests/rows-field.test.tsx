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
import { fireEvent, render, screen } from '@testing-library/react';
import { FormProvider, useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { parse } from 'yaml';
import { TestWrapper } from 'utils/test';
import { KEY_VALUE_ROW_YAML } from 'components/ui-generator/test-fixtures/key-value-row.fixture';
import { UIGenerator } from '../ui-generator';
import { Section, TopologyUISchemas } from '../ui-generator.types';
import { buildZodSchema } from '../utils/schema-builder';
import { getDefaultValues } from '../utils/default-values';
import { preprocessSchema } from '../utils/preprocess/preprocess-schema';
import { getSectionFormZodSchema, toSectionForm } from '../utils/section-form';

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
  return <FormProvider {...methods}>{children}</FormProvider>;
};

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
      render(
        <TestWrapper>
          <FormWrapper uiSchema={uiSchema}>
            <UIGenerator
              sections={uiSchema.replicaSet.sections}
              sectionKey="advanced"
            />
          </FormWrapper>
        </TestWrapper>
      );

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
