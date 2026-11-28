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
import { FormProvider, useForm, useFormContext } from 'react-hook-form';
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
import { TOGGLEABLE_SWITCHES_KEY } from '../utils/toggleable/toggleable';

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

  it('turning a section on focuses its first field without flagging it', async () => {
    renderForm();

    fireEvent.click(getSwitch());

    const endpoint = await screen.findByLabelText(/Endpoint/);
    await waitFor(() => expect(endpoint).toHaveFocus());
    expect(screen.queryByText(/required/i)).not.toBeInTheDocument();
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

const qualifiedCelSchema: TopologyUISchemas = preprocessSchema({
  [TOPOLOGY]: {
    sections: {
      advanced: {
        components: {
          monitoring: {
            uiType: 'group',
            groupType: GroupType.Toggleable,
            label: 'Monitoring',
            components: {
              gpuLimit: {
                uiType: FieldType.Number,
                path: "spec.monitoring['nvidia.com/gpu']",
                fieldParams: { label: 'GPU limit', defaultValue: 4 },
                validation: {
                  celExpressions: [
                    {
                      celExpr: 'spec.backup.retention >= spec.backup.minimum',
                      message: 'Retention must cover the minimum',
                    },
                  ],
                },
              },
            },
          },
          retention: {
            uiType: FieldType.Number,
            path: 'spec.backup.retention',
            fieldParams: { label: 'Retention', defaultValue: 1 },
          },
          minimum: {
            uiType: FieldType.Number,
            path: 'spec.backup.minimum',
            fieldParams: { label: 'Minimum', defaultValue: 3 },
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

const QualifiedCelErrorProbe = () => {
  const { formState, getFieldState } = useFormContext();
  const { error } = getFieldState(
    'spec.monitoring.nvidia%2Ecom%2Fgpu',
    formState
  );

  return (
    <output
      data-testid="qualified-cel-error-state"
      data-error-active={error ? 'true' : 'false'}
    />
  );
};

const QualifiedCelFormWrapper = ({
  onTrigger,
}: {
  onTrigger: (fields?: string | string[]) => void;
}) => {
  const { schema: zodSchema, celDependencyGroups } = buildZodSchema(
    qualifiedCelSchema,
    TOPOLOGY
  );
  const methods = useForm({
    resolver: zodResolver(zodSchema),
    mode: 'onSubmit',
    defaultValues: getDefaultValues(qualifiedCelSchema, TOPOLOGY),
  });
  useCelValidation(celDependencyGroups, methods.control, (fields) => {
    onTrigger(fields);
    return methods.trigger(fields);
  });

  return (
    <FormProvider {...methods}>
      <UIGenerator
        sections={qualifiedCelSchema[TOPOLOGY].sections}
        sectionKey="advanced"
      />
      <QualifiedCelErrorProbe />
    </FormProvider>
  );
};

describe('UIGenerator - CEL across a toggleable group', () => {
  it('uses the qualified form ID and adds its toggleable switch to CEL dependencies', () => {
    const { celDependencyGroups } = buildZodSchema(
      qualifiedCelSchema,
      TOPOLOGY
    );

    expect(celDependencyGroups).toContainEqual([
      'spec.monitoring.nvidia%2Ecom%2Fgpu',
      'spec.backup.retention',
      'spec.backup.minimum',
      'toggleable-switches.advanced~monitoring',
    ]);
  });

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

  it('clears the CEL error when a qualified child field switch turns off', async () => {
    const onTrigger = vi.fn();
    const { schema: zodSchema } = buildZodSchema(qualifiedCelSchema, TOPOLOGY);
    const enabledValues = {
      ...getDefaultValues(qualifiedCelSchema, TOPOLOGY),
      [TOGGLEABLE_SWITCHES_KEY]: { 'advanced~monitoring': true },
    };
    expect(zodSchema.safeParse(enabledValues).success).toBe(false);

    render(
      <TestWrapper>
        <QualifiedCelFormWrapper onTrigger={onTrigger} />
      </TestWrapper>
    );
    const message = 'Retention must cover the minimum';

    expect(screen.queryByText(message)).not.toBeInTheDocument();

    fireEvent.click(getSwitch());
    fireEvent.change(await screen.findByLabelText(/Retention/), {
      target: { value: '2' },
    });
    expect(await screen.findByText(message)).toBeInTheDocument();

    const triggerCountBeforeSwitchOff = onTrigger.mock.calls.length;
    fireEvent.click(getSwitch());
    await waitFor(() =>
      expect(screen.queryByText(message)).not.toBeInTheDocument()
    );
    await waitFor(() =>
      expect(onTrigger).toHaveBeenCalledTimes(triggerCountBeforeSwitchOff + 1)
    );
    expect(onTrigger).toHaveBeenLastCalledWith([
      'spec.monitoring.nvidia%2Ecom%2Fgpu',
      'spec.backup.retention',
      'spec.backup.minimum',
      'toggleable-switches.advanced~monitoring',
    ]);
    await waitFor(() =>
      expect(screen.getByTestId('qualified-cel-error-state')).toHaveAttribute(
        'data-error-active',
        'false'
      )
    );
  });
});
