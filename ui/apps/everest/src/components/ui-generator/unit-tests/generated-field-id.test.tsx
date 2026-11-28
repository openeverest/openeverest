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

import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { zodResolver } from '@hookform/resolvers/zod';
import { Button } from '@mui/material';
import { FormProvider, useForm, useFormContext } from 'react-hook-form';
import { useState } from 'react';
import { TestWrapper } from 'utils/test';
import { UIGenerator } from '../ui-generator';
import {
  FieldType,
  type Section,
  type TopologyUISchemas,
} from '../ui-generator.types';
import { getDefaultValues } from '../utils/default-values';
import { buildZodSchema } from '../utils/schema-builder';

const TOPOLOGY = 'pathless';
const FIELD_ID = 'g-basic.resources.volume%5B0%5D';
const NUMERIC_FIELD_ID = 'g-basic.resources.%30';

const sections: Record<string, Section> = {
  basic: {
    components: {
      resources: {
        uiType: 'group',
        components: {
          'volume[0]': {
            uiType: FieldType.Number,
            id: 'volume[0]',
            fieldParams: { label: 'Volume', defaultValue: 4 },
            validation: { required: true },
          },
          '0': {
            uiType: FieldType.Number,
            id: '0',
            fieldParams: { label: 'Numeric key', defaultValue: 7 },
            validation: { required: true },
          },
        },
      },
    },
  },
};

const schema: TopologyUISchemas = {
  [TOPOLOGY]: { sections, sectionsOrder: ['basic'] },
};

const FormStateProbe = () => {
  const methods = useFormContext();
  const [validationResult, setValidationResult] = useState('not-run');

  return (
    <>
      <output data-testid="generated-value">
        {String(methods.getValues(FIELD_ID) ?? '')}
      </output>
      <output data-testid="numeric-generated-value">
        {String(methods.getValues(NUMERIC_FIELD_ID) ?? '')}
      </output>
      <Button
        type="button"
        onClick={async () => {
          const results = await Promise.all([
            methods.trigger(FIELD_ID),
            methods.trigger(NUMERIC_FIELD_ID),
          ]);
          setValidationResult(String(results.every(Boolean)));
        }}
      >
        Validate generated fields
      </Button>
      <output data-testid="validation-result">{validationResult}</output>
    </>
  );
};

const Form = () => {
  const defaults = getDefaultValues(schema, TOPOLOGY);
  const { schema: validationSchema } = buildZodSchema(schema, TOPOLOGY);
  const methods = useForm({
    defaultValues: defaults,
    resolver: zodResolver(validationSchema),
  });

  return (
    <FormProvider {...methods}>
      <form>
        <UIGenerator sections={sections} sectionKey="basic" />
        <FormStateProbe />
      </form>
    </FormProvider>
  );
};

describe('pathless generated field IDs', () => {
  it('uses the same encoded group hierarchy for defaults, Zod, rendering, and RHF', async () => {
    const defaults = getDefaultValues(schema, TOPOLOGY);
    const { schema: validationSchema } = buildZodSchema(schema, TOPOLOGY);

    expect(defaults).toEqual({
      'g-basic': { resources: { 'volume%5B0%5D': 4, '%30': 7 } },
    });
    expect(validationSchema.safeParse(defaults).success).toBe(true);

    render(
      <TestWrapper>
        <Form />
      </TestWrapper>
    );

    const input = screen.getByLabelText('Volume');
    expect(input).toHaveAttribute('name', FIELD_ID);
    expect(input).toHaveValue(4);
    const numericInput = screen.getByLabelText('Numeric key');
    expect(numericInput).toHaveAttribute('name', NUMERIC_FIELD_ID);
    expect(numericInput).toHaveValue(7);
    expect(screen.getByTestId('generated-value')).toHaveTextContent('4');
    expect(screen.getByTestId('numeric-generated-value')).toHaveTextContent(
      '7'
    );

    await act(async () => {
      fireEvent.click(
        screen.getByRole('button', { name: 'Validate generated fields' })
      );
    });

    await waitFor(() =>
      expect(screen.getByTestId('validation-result')).toHaveTextContent('true')
    );
  });
});
