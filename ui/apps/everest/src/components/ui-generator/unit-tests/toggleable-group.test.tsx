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
import { FieldType, GroupType, TopologyUISchemas } from '../ui-generator.types';
import { buildZodSchema } from '../utils/schema-builder';
import { getDefaultValues } from '../utils/default-values';
import { postprocessSchemaData } from '../utils/postprocess/postprocess-schema';
import { preprocessSchema } from '../utils/preprocess/preprocess-schema';
import { useCelValidation } from '../hooks/use-cel-validation';

const TOPOLOGY = 'replicaSet';

const schema: TopologyUISchemas = preprocessSchema({
  [TOPOLOGY]: {
    sections: {
      advanced: {
        components: {
          monitoring: {
            uiType: 'group',
            groupType: GroupType.Toggleable,
            label: 'Monitoring',
            description: 'Collect metrics',
            components: {
              endpoint: {
                uiType: FieldType.Text,
                path: 'spec.monitoring.endpoint',
                fieldParams: { label: 'Endpoint' },
                validation: { required: true },
              },
            },
          },
          name: {
            uiType: FieldType.Text,
            path: 'spec.name',
            fieldParams: { label: 'Name', defaultValue: 'db-1' },
          },
        },
      },
    },
  },
});

const FormWrapper = ({
  onSubmit,
  defaultValues = getDefaultValues(schema, TOPOLOGY),
}: {
  onSubmit: (data: Record<string, unknown>) => void;
  defaultValues?: Record<string, unknown>;
}) => {
  const { schema: zodSchema } = buildZodSchema(schema, TOPOLOGY);
  const methods = useForm({
    resolver: zodResolver(zodSchema),
    mode: 'onChange',
    defaultValues,
  });

  return (
    <FormProvider {...methods}>
      <form
        onSubmit={methods.handleSubmit((data) =>
          onSubmit(
            postprocessSchemaData(data, {
              schema,
              selectedTopology: TOPOLOGY,
            })
          )
        )}
      >
        <UIGenerator
          sections={schema[TOPOLOGY].sections}
          sectionKey="advanced"
        />
        <Button type="submit">Submit</Button>
      </form>
    </FormProvider>
  );
};

const renderForm = (defaultValues?: Record<string, unknown>) => {
  const onSubmit = vi.fn();
  render(
    <TestWrapper>
      <FormWrapper onSubmit={onSubmit} defaultValues={defaultValues} />
    </TestWrapper>
  );
  return onSubmit;
};

const getSwitch = () =>
  screen.getByRole('switch', { name: 'Enable Monitoring' });
const submit = () =>
  fireEvent.click(screen.getByRole('button', { name: 'Submit' }));

describe('UIGenerator - toggleable group', () => {
  it('starts off: fields hidden and the section left out of the payload', async () => {
    const onSubmit = renderForm();

    expect(screen.getByText('Monitoring')).toBeInTheDocument();
    expect(getSwitch()).not.toBeChecked();
    expect(screen.queryByLabelText(/Endpoint/)).not.toBeInTheDocument();

    submit();

    // The hidden required field doesn't block submit.
    await waitFor(() => expect(onSubmit).toHaveBeenCalled());
    expect(onSubmit.mock.calls[0][0]).toEqual({ spec: { name: 'db-1' } });
  });

  it('when on, shows the fields and enforces their validation', async () => {
    const onSubmit = renderForm();

    fireEvent.click(getSwitch());
    expect(getSwitch()).toBeChecked();

    const endpoint = await screen.findByLabelText(/Endpoint/);
    submit();

    expect(await screen.findByText(/required/i)).toBeInTheDocument();
    expect(onSubmit).not.toHaveBeenCalled();

    fireEvent.change(endpoint, { target: { value: 'pmm:443' } });
    submit();

    await waitFor(() => expect(onSubmit).toHaveBeenCalled());
    expect(onSubmit.mock.calls[0][0]).toEqual({
      spec: { name: 'db-1', monitoring: { endpoint: 'pmm:443' } },
    });
  });

  it('drops values typed in before the section is switched back off', async () => {
    const onSubmit = renderForm();

    fireEvent.click(getSwitch());
    fireEvent.change(await screen.findByLabelText(/Endpoint/), {
      target: { value: 'pmm:443' },
    });
    fireEvent.click(getSwitch());
    submit();

    await waitFor(() => expect(onSubmit).toHaveBeenCalled());
    expect(onSubmit.mock.calls[0][0]).toEqual({ spec: { name: 'db-1' } });
  });

  it('works when the form defaults predate the schema (no switch value)', async () => {
    const consoleError = vi.spyOn(console, 'error');
    renderForm({ spec: { name: 'db-1' } });

    expect(getSwitch()).not.toBeChecked();
    fireEvent.click(getSwitch());

    expect(getSwitch()).toBeChecked();
    expect(await screen.findByLabelText(/Endpoint/)).toBeInTheDocument();
    expect(
      consoleError.mock.calls
        .flat()
        .some((arg) => String(arg).includes('uncontrolled'))
    ).toBe(false);
    consoleError.mockRestore();
  });

  it('turning a section on focuses its first field and shows why it is required', async () => {
    renderForm();

    fireEvent.click(getSwitch());

    const endpoint = await screen.findByLabelText(/Endpoint/);
    await waitFor(() => expect(endpoint).toHaveFocus());
    expect(await screen.findByText(/required/i)).toBeInTheDocument();
  });
});

const celSchema: TopologyUISchemas = preprocessSchema({
  [TOPOLOGY]: {
    sections: {
      advanced: {
        components: {
          monitoring: {
            uiType: 'group',
            groupType: GroupType.Toggleable,
            label: 'Monitoring',
            components: {
              interval: {
                uiType: FieldType.Number,
                path: 'spec.monitoring.interval',
                fieldParams: { label: 'Interval', defaultValue: 100 },
              },
            },
          },
          retention: {
            uiType: FieldType.Number,
            path: 'spec.backup.retention',
            fieldParams: { label: 'Retention', defaultValue: 10 },
            validation: {
              celExpressions: [
                {
                  celExpr:
                    '!has(spec.monitoring.interval) || spec.backup.retention >= spec.monitoring.interval',
                  message: 'Retention must cover the interval',
                },
              ],
            },
          },
        },
      },
    },
  },
});

const CelFormWrapper = () => {
  const { schema: zodSchema, celDependencyGroups } = buildZodSchema(
    celSchema,
    TOPOLOGY
  );
  const methods = useForm({
    resolver: zodResolver(zodSchema),
    mode: 'onChange',
    defaultValues: getDefaultValues(celSchema, TOPOLOGY),
  });
  useCelValidation(celDependencyGroups, methods.control, methods.trigger);

  return (
    <FormProvider {...methods}>
      <UIGenerator
        sections={celSchema[TOPOLOGY].sections}
        sectionKey="advanced"
      />
    </FormProvider>
  );
};

describe('UIGenerator - CEL across a toggleable group', () => {
  it('re-checks a rule outside the group when its switch flips', async () => {
    render(
      <TestWrapper>
        <CelFormWrapper />
      </TestWrapper>
    );
    const message = 'Retention must cover the interval';

    expect(screen.queryByText(message)).not.toBeInTheDocument();

    fireEvent.click(getSwitch());
    expect(await screen.findByText(message)).toBeInTheDocument();

    fireEvent.click(getSwitch());
    await waitFor(() =>
      expect(screen.queryByText(message)).not.toBeInTheDocument()
    );
  });
});
