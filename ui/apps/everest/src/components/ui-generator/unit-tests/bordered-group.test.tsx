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

import { render, screen } from '@testing-library/react';
import { FormProvider, useForm } from 'react-hook-form';
import { TestWrapper } from 'utils/test';
import { UIGenerator } from '../ui-generator';
import { FieldType, GroupType, TopologyUISchemas } from '../ui-generator.types';
import { getDefaultValues } from '../utils/default-values';
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
                fieldParams: { label: 'Size' },
              },
            },
          },
        },
      },
    },
  },
});

const FormWrapper = () => {
  const methods = useForm({
    defaultValues: getDefaultValues(schema, TOPOLOGY),
  });

  return (
    <FormProvider {...methods}>
      <UIGenerator
        sections={schema[TOPOLOGY].sections}
        sectionKey="resources"
      />
    </FormProvider>
  );
};

describe('UIGenerator - bordered group', () => {
  it('shows the group label and description', () => {
    render(
      <TestWrapper>
        <FormWrapper />
      </TestWrapper>
    );

    expect(screen.getByText('Storage')).toBeInTheDocument();
    expect(
      screen.getByText('Type and performance of storage')
    ).toBeInTheDocument();
  });
});
