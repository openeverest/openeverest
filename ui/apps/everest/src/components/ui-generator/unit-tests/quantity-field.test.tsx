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

import { render, screen, waitFor, fireEvent } from '@testing-library/react';
import { FormProvider, useForm } from 'react-hook-form';
import { TestWrapper } from 'utils/test';
import { UIGenerator } from '../ui-generator';
import { Component, FieldType, TopologyUISchemas } from '../ui-generator.types';
import { zodResolver } from '@hookform/resolvers/zod';
import { buildZodSchema } from '../utils/schema-builder';
import { getDefaultValues } from '../utils/default-values';
import { Button } from '@mui/material';
import { postprocessSchemaData } from '../utils/postprocess/postprocess-schema';

vi.mock('../utils/cel-validation', () => ({
  extractCelFieldPaths: vi.fn(() => []),
  validateCelExpression: vi.fn(() => true),
}));

vi.mock('../utils/schema-builder/cel-validation', () => ({
  extractCelFieldPaths: vi.fn(() => []),
  validateCelExpression: vi.fn(() => true),
}));

const createQuantitySchema = (
  config: {
    fieldParams?: Record<string, unknown>;
    validation?: Record<string, unknown>;
    path?: string;
    label?: string;
  } = {}
): TopologyUISchemas => {
  const testQuantity: Extract<Component, { uiType: FieldType.Quantity }> = {
    uiType: FieldType.Quantity,
    path: config.path ?? 'spec.resources.cpu',
    fieldParams: {
      label: config.label ?? 'CPU',
      step: 0.1,
      ...config.fieldParams,
    },
  };

  if (config.validation) {
    testQuantity.validation = config.validation;
  }

  return {
    testTopology: {
      sections: {
        resources: {
          label: 'Resources',
          components: {
            quantity: testQuantity,
          },
        },
      },
      sectionsOrder: ['resources'],
    },
  };
};

interface FormWrapperProps {
  children: React.ReactNode;
  schema: TopologyUISchemas;
  onSubmit: (data: Record<string, unknown>) => void;
  defaultValues?: Record<string, unknown>;
}

const FormWrapper = ({
  children,
  schema,
  onSubmit,
  defaultValues: defaultsOverride,
}: FormWrapperProps) => {
  const { schema: zodSchema } = buildZodSchema(schema, 'testTopology');
  const defaultValues =
    defaultsOverride ?? getDefaultValues(schema, 'testTopology');

  const methods = useForm({
    resolver: zodResolver(zodSchema),
    mode: 'onChange',
    defaultValues,
    reValidateMode: 'onChange',
  });

  return (
    <FormProvider {...methods}>
      <form onSubmit={methods.handleSubmit(onSubmit)}>
        {children}
        <Button
          type="submit"
          disabled={!methods.formState.isValid}
          data-testid="submit-button"
        >
          Submit
        </Button>
      </form>
    </FormProvider>
  );
};

describe('UIGenerator - Quantity Field', () => {
  it('renders unitLabel as an end adornment', () => {
    const schema = createQuantitySchema({
      fieldParams: { unitLabel: 'cores' },
    });

    render(
      <TestWrapper>
        <FormWrapper schema={schema} onSubmit={vi.fn()}>
          <UIGenerator
            sections={schema.testTopology!.sections}
            sectionKey="resources"
          />
        </FormWrapper>
      </TestWrapper>
    );

    expect(screen.getByLabelText('CPU')).toBeInTheDocument();
    expect(screen.getByText('cores')).toBeInTheDocument();
  });

  it('falls back to unit for the adornment when unitLabel is omitted', () => {
    const schema = createQuantitySchema({
      path: 'spec.resources.memory',
      label: 'Memory',
      fieldParams: { unit: 'Gi', step: 1 },
    });

    render(
      <TestWrapper>
        <FormWrapper schema={schema} onSubmit={vi.fn()}>
          <UIGenerator
            sections={schema.testTopology!.sections}
            sectionKey="resources"
          />
        </FormWrapper>
      </TestWrapper>
    );

    expect(screen.getByLabelText('Memory')).toBeInTheDocument();
    expect(screen.getByText('Gi')).toBeInTheDocument();
  });

  it('submits fractional CPU as the string "0.5" after postprocess', async () => {
    const mockSubmit = vi.fn();
    const schema = createQuantitySchema({
      fieldParams: { unitLabel: 'cores' },
      validation: { required: true },
    });

    render(
      <TestWrapper>
        <FormWrapper schema={schema} onSubmit={mockSubmit}>
          <UIGenerator
            sections={schema.testTopology!.sections}
            sectionKey="resources"
          />
        </FormWrapper>
      </TestWrapper>
    );

    const input = screen.getByLabelText('CPU');
    const submitButton = screen.getByTestId('submit-button');

    fireEvent.change(input, { target: { value: '0.5' } });

    await waitFor(() => {
      expect(submitButton).not.toBeDisabled();
    });

    fireEvent.click(submitButton);

    await waitFor(() => {
      expect(mockSubmit).toHaveBeenCalledTimes(1);
    });

    const submittedData = mockSubmit.mock.calls[0][0] as Record<
      string,
      unknown
    >;
    // Form holds a number; API write is a quantity string.
    expect(submittedData).toMatchObject({
      spec: { resources: { cpu: 0.5 } },
    });

    const postProcessed = postprocessSchemaData(submittedData, {
      schema,
      selectedTopology: 'testTopology',
    });
    expect(postProcessed).toEqual({
      spec: { resources: { cpu: '0.5' } },
    });
    expect(
      typeof (postProcessed as { spec: { resources: { cpu: unknown } } }).spec
        .resources.cpu
    ).toBe('string');
  });

  it('applies min validation to the number in unit', async () => {
    const mockSubmit = vi.fn();
    const schema = createQuantitySchema({
      fieldParams: { unitLabel: 'cores' },
      validation: { required: true, min: 1 },
    });

    render(
      <TestWrapper>
        <FormWrapper schema={schema} onSubmit={mockSubmit}>
          <UIGenerator
            sections={schema.testTopology!.sections}
            sectionKey="resources"
          />
        </FormWrapper>
      </TestWrapper>
    );

    const input = screen.getByLabelText('CPU') as HTMLInputElement;
    const submitButton = screen.getByTestId('submit-button');

    expect(input.min).toBe('1');

    fireEvent.change(input, { target: { value: '0.5' } });

    await waitFor(() => {
      expect(submitButton).toBeDisabled();
    });
    expect(mockSubmit).not.toHaveBeenCalled();

    fireEvent.change(input, { target: { value: '1' } });

    await waitFor(() => {
      expect(submitButton).not.toBeDisabled();
    });
  });

  it('exposes step so the browser accepts fractional values', () => {
    const schema = createQuantitySchema({
      fieldParams: { step: 0.1 },
    });

    render(
      <TestWrapper>
        <FormWrapper schema={schema} onSubmit={vi.fn()}>
          <UIGenerator
            sections={schema.testTopology!.sections}
            sectionKey="resources"
          />
        </FormWrapper>
      </TestWrapper>
    );

    const input = screen.getByLabelText('CPU') as HTMLInputElement;
    expect(input.step).toBe('0.1');
  });
});
