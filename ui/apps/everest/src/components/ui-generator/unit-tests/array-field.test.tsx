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
import { act, fireEvent, render, screen } from '@testing-library/react';
import { FormProvider, useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { TestWrapper } from 'utils/test';
import { UIGenerator } from '../ui-generator';
import {
  ArrayItemType,
  ArrayOutputType,
  FieldType,
  TopologyUISchemas,
} from '../ui-generator.types';
import { buildZodSchema } from '../utils/schema-builder';
import { getDefaultValues } from '../utils/default-values';

const schema: TopologyUISchemas = {
  replicaSet: {
    sections: {
      advanced: {
        components: {
          annotations: {
            uiType: FieldType.Array,
            outputType: ArrayOutputType.Object,
            path: 'spec.proxy.annotations',
            fieldParams: {
              label: 'Annotations',
              addButtonLabel: 'Add annotation',
              defaultValue: { team: 'db' },
            },
            itemTemplate: { uiType: ArrayItemType.KeyValue },
          },
        },
      },
    },
  },
};

const FormWrapper = ({ children }: { children: ReactNode }) => {
  const methods = useForm({
    resolver: zodResolver(buildZodSchema(schema, 'replicaSet').schema),
    mode: 'onChange',
    defaultValues: getDefaultValues(schema, 'replicaSet'),
  });
  return <FormProvider {...methods}>{children}</FormProvider>;
};

const type = async (name: string, value: string) =>
  act(async () => {
    fireEvent.change(screen.getByRole('textbox', { name }), {
      target: { value },
    });
  });

describe('UIGenerator - array field', () => {
  it('flags a key set twice until either row changes it', async () => {
    render(
      <TestWrapper>
        <FormWrapper>
          <UIGenerator
            sections={schema.replicaSet.sections}
            sectionKey="advanced"
          />
        </FormWrapper>
      </TestWrapper>
    );

    expect(screen.getByRole('textbox', { name: 'Key 1' })).toHaveValue('team');

    fireEvent.click(screen.getByRole('button', { name: 'Add annotation' }));
    expect(screen.getByRole('textbox', { name: 'Key 2' })).toHaveFocus();
    await type('Key 2', 'team');

    expect(
      await screen.findByText('team is set more than once')
    ).toBeInTheDocument();

    // The error sits on row 2, so clearing it needs both rows re-validated.
    await type('Key 1', 'tier');

    expect(
      screen.queryByText('team is set more than once')
    ).not.toBeInTheDocument();
  });
});
