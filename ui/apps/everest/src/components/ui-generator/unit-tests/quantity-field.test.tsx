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

import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { FormProvider, useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { Button } from '@mui/material';
import { TestWrapper } from 'utils/test';
import { UIGenerator } from '../ui-generator';
import { FieldType, TopologyUISchemas } from '../ui-generator.types';
import { buildZodSchema } from '../utils/schema-builder';
import { getDefaultValues } from '../utils/default-values';
import { postprocessSchemaData } from '../utils/postprocess/postprocess-schema';

const TOPOLOGY = 'replicaSet';

const schema: TopologyUISchemas = {
  [TOPOLOGY]: {
    sections: {
      resources: {
        components: {
          cpu: {
            uiType: FieldType.Quantity,
            path: 'spec.cpu',
            fieldParams: { label: 'CPU', unitLabel: 'cores', defaultValue: 1 },
            validation: { min: 0.1 },
          },
        },
      },
    },
  },
};

const FormWrapper = ({
  onSubmit,
}: {
  onSubmit: (data: Record<string, unknown>) => void;
}) => {
  const methods = useForm({
    resolver: zodResolver(buildZodSchema(schema, TOPOLOGY).schema),
    mode: 'onChange',
    defaultValues: getDefaultValues(schema, TOPOLOGY),
  });

  return (
    <FormProvider {...methods}>
      <form
        // The schema validates; jsdom would also apply the input's native step.
        noValidate
        onSubmit={methods.handleSubmit((data) =>
          onSubmit(
            postprocessSchemaData(data, { schema, selectedTopology: TOPOLOGY })
          )
        )}
      >
        <UIGenerator
          sections={schema[TOPOLOGY].sections}
          sectionKey="resources"
        />
        <Button type="submit">Submit</Button>
      </form>
    </FormProvider>
  );
};

describe('UIGenerator - quantity field', () => {
  it('shows the unit and sends a fractional value as a quantity string', async () => {
    const onSubmit = vi.fn();
    render(
      <TestWrapper>
        <FormWrapper onSubmit={onSubmit} />
      </TestWrapper>
    );

    expect(screen.getByText('cores')).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText('CPU'), {
      target: { value: '0.5' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Submit' }));

    await waitFor(() => expect(onSubmit).toHaveBeenCalled());
    expect(onSubmit.mock.calls[0][0]).toEqual({ spec: { cpu: '0.5' } });
  });

  it('validates the value as a number in the field unit', async () => {
    const onSubmit = vi.fn();
    render(
      <TestWrapper>
        <FormWrapper onSubmit={onSubmit} />
      </TestWrapper>
    );

    fireEvent.change(screen.getByLabelText('CPU'), {
      target: { value: '0.05' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Submit' }));

    expect(await screen.findByText(/0\.1/)).toBeInTheDocument();
    expect(onSubmit).not.toHaveBeenCalled();
  });
});
