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

const TOPOLOGY = 'replicaSet';

const schema: TopologyUISchemas = preprocessSchema({
  [TOPOLOGY]: {
    sections: {
      resources: {
        components: {
          storage: {
            uiType: 'group',
            groupType: GroupType.Bordered,
            label: 'Storage',
            description: 'Type and performance of storage',
            components: {
              size: {
                uiType: FieldType.Text,
                path: 'spec.storage.size',
                fieldParams: { label: 'Size', defaultValue: '10Gi' },
              },
            },
          },
          limits: {
            uiType: 'group',
            groupType: GroupType.Bordered,
            components: {
              row: {
                uiType: 'group',
                groupType: GroupType.Line,
                components: {
                  cpu: {
                    uiType: FieldType.Text,
                    path: 'spec.resources.cpu',
                    fieldParams: { label: 'CPU', defaultValue: '1' },
                  },
                  memory: {
                    uiType: FieldType.Text,
                    path: 'spec.resources.memory',
                    fieldParams: { label: 'Memory', defaultValue: '2Gi' },
                  },
                },
              },
            },
          },
        },
      },
    },
  },
});

const FormWrapper = ({
  onSubmit,
}: {
  onSubmit: (data: Record<string, unknown>) => void;
}) => {
  const { schema: zodSchema } = buildZodSchema(schema, TOPOLOGY);
  const methods = useForm({
    resolver: zodResolver(zodSchema),
    defaultValues: getDefaultValues(schema, TOPOLOGY),
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
          sectionKey="resources"
        />
        <Button type="submit">Submit</Button>
      </form>
    </FormProvider>
  );
};

const renderForm = () => {
  const onSubmit = vi.fn();
  render(
    <TestWrapper>
      <FormWrapper onSubmit={onSubmit} />
    </TestWrapper>
  );
  return onSubmit;
};

describe('UIGenerator - bordered group', () => {
  it('shows the group label and description', () => {
    renderForm();

    expect(screen.getByText('Storage')).toBeInTheDocument();
    expect(
      screen.getByText('Type and performance of storage')
    ).toBeInTheDocument();
  });

  it('is visual only: nested fields render and submit to their own paths', async () => {
    const onSubmit = renderForm();

    expect(screen.queryByRole('switch')).not.toBeInTheDocument();
    expect(screen.getByLabelText('CPU')).toBeInTheDocument();
    expect(screen.getByLabelText('Memory')).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', { name: 'Submit' }));

    await waitFor(() => expect(onSubmit).toHaveBeenCalled());
    expect(onSubmit.mock.calls[0][0]).toEqual({
      spec: {
        storage: { size: '10Gi' },
        resources: { cpu: '1', memory: '2Gi' },
      },
    });
  });
});
