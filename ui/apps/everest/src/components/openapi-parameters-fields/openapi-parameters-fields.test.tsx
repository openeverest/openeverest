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
import { z } from 'zod';
import { Button } from '@mui/material';
import { TestWrapper } from 'utils/test';
import { OpenApiParametersFields } from './openapi-parameters-fields';
import {
  buildOpenApiParametersSchema,
  fromOpenApiParameterFormValues,
  getOpenApiParameterFields,
  toOpenApiParameterFormValues,
} from './openapi-parameters-fields.utils';

// The Prometheus class's destinationParametersSchema, plus shapes the form
// can't edit.
const prometheusSchema = {
  type: 'object',
  required: ['interval'],
  properties: {
    podMonitorLabels: {
      type: 'object',
      additionalProperties: { type: 'string' },
    },
    interval: { type: 'string', description: 'Scrape interval' },
    replicas: { type: 'integer' },
    tls: { type: 'object', properties: { ca: { type: 'string' } } },
  },
};
const parameterFields = getOpenApiParameterFields(prometheusSchema);

describe('getOpenApiParameterFields', () => {
  it('keeps strings and string maps and lists the rest as skipped', () => {
    expect(parameterFields).toEqual({
      fields: [
        {
          name: 'podMonitorLabels',
          kind: 'stringMap',
          required: false,
          description: undefined,
        },
        {
          name: 'interval',
          kind: 'text',
          required: true,
          description: 'Scrape interval',
        },
      ],
      skipped: ['replicas', 'tls'],
    });
  });

  it('has no fields without an object schema', () => {
    expect(getOpenApiParameterFields(undefined)).toEqual({
      fields: [],
      skipped: [],
    });
  });

  it('round-trips parameters through the form values', () => {
    const parameters = {
      podMonitorLabels: { release: 'prometheus' },
      interval: '30s',
    };
    const values = toOpenApiParameterFormValues(parameters, parameterFields);

    expect(values).toEqual({
      podMonitorLabels: [{ key: 'release', value: 'prometheus' }],
      interval: '30s',
    });
    expect(fromOpenApiParameterFormValues(values, parameterFields)).toEqual(
      parameters
    );
    expect(
      fromOpenApiParameterFormValues(
        { podMonitorLabels: [], interval: '' },
        parameterFields
      )
    ).toEqual({});
  });
});

const Harness = ({ onSubmit }: { onSubmit: (v: unknown) => void }) => {
  const methods = useForm({
    mode: 'onChange',
    resolver: zodResolver(
      z.object({ parameters: buildOpenApiParametersSchema(parameterFields) })
    ),
    defaultValues: {
      parameters: toOpenApiParameterFormValues(undefined, parameterFields),
    },
  });
  return (
    <FormProvider {...methods}>
      <OpenApiParametersFields
        name="parameters"
        parameterFields={parameterFields}
      />
      <Button
        onClick={methods.handleSubmit((data) =>
          onSubmit(
            fromOpenApiParameterFormValues(data.parameters, parameterFields)
          )
        )}
      >
        Submit
      </Button>
    </FormProvider>
  );
};

describe('OpenApiParametersFields', () => {
  it('fills text and map parameters and enforces required ones', async () => {
    const onSubmit = vi.fn();
    render(
      <TestWrapper>
        <Harness onSubmit={onSubmit} />
      </TestWrapper>
    );

    expect(
      screen.getByText(
        'Not editable here: replicas, tls. Set them through the API.'
      )
    ).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Submit' }));
    expect(await screen.findByText('interval is required')).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText(/interval/), {
      target: { value: '15s' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Add' }));
    fireEvent.change(screen.getByRole('textbox', { name: 'Key 1' }), {
      target: { value: 'release' },
    });
    fireEvent.change(screen.getByRole('textbox', { name: 'Value 1' }), {
      target: { value: 'prometheus' },
    });
    fireEvent.click(screen.getByRole('button', { name: 'Submit' }));

    await waitFor(() =>
      expect(onSubmit).toHaveBeenCalledWith({
        interval: '15s',
        podMonitorLabels: { release: 'prometheus' },
      })
    );
  });
});
