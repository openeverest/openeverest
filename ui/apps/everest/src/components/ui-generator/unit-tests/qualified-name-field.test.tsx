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
  render,
  screen,
  fireEvent,
  act,
  waitFor,
} from '@testing-library/react';
import { FormProvider, useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { Button } from '@mui/material';
import { TestWrapper } from 'utils/test';
import { UIGenerator } from '../ui-generator';
import { FieldType, Section, TopologyUISchemas } from '../ui-generator.types';
import { buildZodSchema } from '../utils/schema-builder';
import { getDefaultValues } from '../utils/default-values';
import { postprocessSchemaData } from '../utils/postprocess/postprocess-schema';

vi.mock('../utils/cel-validation', () => ({
  extractCelFieldPaths: vi.fn(() => []),
  validateCelExpression: vi.fn(() => true),
}));

vi.mock('../utils/schema-builder/cel-validation', () => ({
  extractCelFieldPaths: vi.fn(() => []),
  validateCelExpression: vi.fn(() => true),
}));

const TOPOLOGY = 'testTopology';
const GPU_PATH = "spec.resources.limits['nvidia.com/gpu']";

const sections: Record<string, Section> = {
  basicInfo: {
    label: 'Basic Information',
    components: {
      gpuLimit: {
        uiType: FieldType.Text,
        path: GPU_PATH,
        fieldParams: { label: 'GPU limit', defaultValue: '1' },
      },
    },
  },
};

const gpuSchema: TopologyUISchemas = {
  [TOPOLOGY]: { sections, sectionsOrder: ['basicInfo'] },
};

const FormWrapper = ({
  onSubmit,
}: {
  onSubmit: (data: Record<string, unknown>) => void;
}) => {
  const { schema: zodSchema } = buildZodSchema(gpuSchema, TOPOLOGY);
  const methods = useForm({
    resolver: zodResolver(zodSchema),
    mode: 'onChange',
    defaultValues: getDefaultValues(gpuSchema, TOPOLOGY),
    reValidateMode: 'onChange',
  });

  return (
    <FormProvider {...methods}>
      <form
        onSubmit={methods.handleSubmit((values) =>
          onSubmit(
            postprocessSchemaData(values, {
              schema: gpuSchema,
              selectedTopology: TOPOLOGY,
            })
          )
        )}
      >
        <UIGenerator sections={sections} sectionKey="basicInfo" />
        <Button type="submit" disabled={!methods.formState.isValid}>
          Submit
        </Button>
      </form>
    </FormProvider>
  );
};

describe('UIGenerator - Kubernetes qualified-name path', () => {
  it('shows the default value of a bracket-quoted path', async () => {
    render(
      <TestWrapper>
        <FormWrapper onSubmit={vi.fn()} />
      </TestWrapper>
    );

    await waitFor(() =>
      expect(screen.getByLabelText('GPU limit')).toHaveValue('1')
    );
  });

  it('submits the value under the qualified key, not split on dots', async () => {
    const onSubmit = vi.fn();

    render(
      <TestWrapper>
        <FormWrapper onSubmit={onSubmit} />
      </TestWrapper>
    );

    const input = await screen.findByLabelText('GPU limit');
    await act(async () => {
      fireEvent.change(input, { target: { value: '2' } });
      fireEvent.blur(input);
    });

    await act(async () => {
      fireEvent.click(screen.getByText('Submit'));
    });

    await waitFor(() => expect(onSubmit).toHaveBeenCalledTimes(1));
    expect(onSubmit.mock.calls[0][0]).toEqual({
      spec: { resources: { limits: { 'nvidia.com/gpu': '2' } } },
    });
  });
});
