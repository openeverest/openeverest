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
import { UIGenerator } from '../ui-generator';
import { FieldType, GroupType, TopologyUISchemas } from '../ui-generator.types';
import { buildSectionZodSchema } from '../utils/schema-builder';
import { getDefaultValues } from '../utils/default-values';
import { postprocessSchemaData } from '../utils/postprocess/postprocess-schema';
import { preprocessSchema } from '../utils/preprocess/preprocess-schema';

const ROOT = 'stringData';
const SECTION = 'credentials';

// Paths are relative to the object the section describes; the host mounts it at ROOT.
const schema: TopologyUISchemas = preprocessSchema({
  [SECTION]: {
    sections: {
      [SECTION]: {
        components: {
          user: {
            uiType: FieldType.Text,
            path: 'USER',
            fieldParams: { label: 'User' },
            validation: { required: true },
          },
          pmm: {
            uiType: 'group',
            groupType: GroupType.Toggleable,
            label: 'PMM',
            components: {
              token: {
                uiType: FieldType.Text,
                path: 'PMM_TOKEN',
                fieldParams: { label: 'Token' },
              },
            },
          },
        },
      },
    },
  },
});
const sections = schema[SECTION].sections;

const Harness = ({
  onSubmit,
}: {
  onSubmit: (data: Record<string, unknown>) => void;
}) => {
  const methods = useForm<Record<string, Record<string, unknown>>>({
    mode: 'onChange',
    resolver: zodResolver(
      z.object({ [ROOT]: buildSectionZodSchema(SECTION, sections).schema })
    ),
    defaultValues: { [ROOT]: getDefaultValues(schema, SECTION) },
  });

  return (
    <FormProvider {...methods}>
      <form
        onSubmit={methods.handleSubmit((data) =>
          onSubmit(
            postprocessSchemaData(data[ROOT], {
              schema,
              selectedTopology: SECTION,
            })
          )
        )}
      >
        <UIGenerator sectionKey={SECTION} sections={sections} root={ROOT} />
        <Button type="submit">Submit</Button>
      </form>
    </FormProvider>
  );
};

const renderForm = () => {
  const onSubmit = vi.fn();
  render(
    <TestWrapper>
      <Harness onSubmit={onSubmit} />
    </TestWrapper>
  );
  return onSubmit;
};

const submit = () =>
  fireEvent.click(screen.getByRole('button', { name: 'Submit' }));

describe('UIGenerator - section mounted at a root', () => {
  it('validates and writes fields under the root', async () => {
    const onSubmit = renderForm();

    submit();
    expect(await screen.findByText(/required/i)).toBeInTheDocument();
    expect(onSubmit).not.toHaveBeenCalled();

    fireEvent.change(screen.getByLabelText(/User/), {
      target: { value: 'admin' },
    });
    submit();

    await waitFor(() => expect(onSubmit).toHaveBeenCalled());
    expect(onSubmit.mock.calls[0][0]).toEqual({ USER: 'admin' });
  });

  it('keeps a toggleable switch under the root', async () => {
    const onSubmit = renderForm();

    fireEvent.change(screen.getByLabelText(/User/), {
      target: { value: 'admin' },
    });
    fireEvent.click(screen.getByRole('switch', { name: 'Enable PMM' }));
    fireEvent.change(await screen.findByLabelText(/Token/), {
      target: { value: 't0ken' },
    });
    submit();

    await waitFor(() => expect(onSubmit).toHaveBeenCalled());
    expect(onSubmit.mock.calls[0][0]).toEqual({
      USER: 'admin',
      PMM_TOKEN: 't0ken',
    });
  });
});
