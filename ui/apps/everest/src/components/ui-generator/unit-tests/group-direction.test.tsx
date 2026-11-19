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
import {
  FieldType,
  GroupDirection,
  GroupType,
  TopologyUISchemas,
} from '../ui-generator.types';
import { getDefaultValues } from '../utils/default-values';
import { preprocessSchema } from '../utils/preprocess/preprocess-schema';

const TOPOLOGY = 'replicaSet';

const FormWrapper = ({ schema }: { schema: TopologyUISchemas }) => {
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

describe('UIGenerator - groupParams.direction', () => {
  it('defaults to a column stack when direction is omitted', () => {
    const schema = preprocessSchema({
      [TOPOLOGY]: {
        sections: {
          resources: {
            components: {
              resources: {
                uiType: 'group',
                components: {
                  cpu: {
                    uiType: FieldType.Text,
                    path: 'spec.cpu',
                    fieldParams: { label: 'CPU' },
                  },
                  memory: {
                    uiType: FieldType.Text,
                    path: 'spec.memory',
                    fieldParams: { label: 'Memory' },
                  },
                },
                componentsOrder: ['cpu', 'memory'],
              },
            },
          },
        },
      },
    });

    render(
      <TestWrapper>
        <FormWrapper schema={schema} />
      </TestWrapper>
    );

    expect(
      screen
        .getByTestId('text-input-spec.cpu')
        .closest('[data-group-direction]')
    ).toHaveAttribute('data-group-direction', GroupDirection.Column);
  });

  it('lays out direct children in a row when direction is row', () => {
    const schema = preprocessSchema({
      [TOPOLOGY]: {
        sections: {
          resources: {
            components: {
              resources: {
                uiType: 'group',
                groupParams: { direction: GroupDirection.Row },
                components: {
                  cpu: {
                    uiType: FieldType.Text,
                    path: 'spec.cpu',
                    fieldParams: { label: 'CPU' },
                  },
                  memory: {
                    uiType: FieldType.Text,
                    path: 'spec.memory',
                    fieldParams: { label: 'Memory' },
                  },
                },
                componentsOrder: ['cpu', 'memory'],
              },
            },
          },
        },
      },
    });

    render(
      <TestWrapper>
        <FormWrapper schema={schema} />
      </TestWrapper>
    );

    expect(
      screen
        .getByTestId('text-input-spec.cpu')
        .closest('[data-group-direction]')
    ).toHaveAttribute('data-group-direction', GroupDirection.Row);
  });

  it('composes bordered chrome with a row body', () => {
    const schema = preprocessSchema({
      [TOPOLOGY]: {
        sections: {
          resources: {
            components: {
              resources: {
                uiType: 'group',
                groupType: GroupType.Bordered,
                label: 'Resources',
                groupParams: { direction: GroupDirection.Row },
                components: {
                  cpu: {
                    uiType: FieldType.Text,
                    path: 'spec.cpu',
                    fieldParams: { label: 'CPU' },
                  },
                  memory: {
                    uiType: FieldType.Text,
                    path: 'spec.memory',
                    fieldParams: { label: 'Memory' },
                  },
                },
                componentsOrder: ['cpu', 'memory'],
              },
            },
          },
        },
      },
    });

    render(
      <TestWrapper>
        <FormWrapper schema={schema} />
      </TestWrapper>
    );

    expect(screen.getByText('Resources')).toBeInTheDocument();
    expect(
      screen
        .getByTestId('text-input-spec.cpu')
        .closest('[data-group-direction]')
    ).toHaveAttribute('data-group-direction', GroupDirection.Row);
  });

  it('nests a row group inside a column group', () => {
    const schema = preprocessSchema({
      [TOPOLOGY]: {
        sections: {
          resources: {
            components: {
              resources: {
                uiType: 'group',
                components: {
                  outerField: {
                    uiType: FieldType.Text,
                    path: 'spec.name',
                    fieldParams: { label: 'Name' },
                  },
                  cpuMem: {
                    uiType: 'group',
                    groupParams: { direction: GroupDirection.Row },
                    components: {
                      cpu: {
                        uiType: FieldType.Text,
                        path: 'spec.cpu',
                        fieldParams: { label: 'CPU' },
                      },
                      memory: {
                        uiType: FieldType.Text,
                        path: 'spec.memory',
                        fieldParams: { label: 'Memory' },
                      },
                    },
                    componentsOrder: ['cpu', 'memory'],
                  },
                },
                componentsOrder: ['outerField', 'cpuMem'],
              },
            },
          },
        },
      },
    });

    render(
      <TestWrapper>
        <FormWrapper schema={schema} />
      </TestWrapper>
    );

    expect(
      screen
        .getByTestId('text-input-spec.name')
        .closest('[data-group-direction]')
    ).toHaveAttribute('data-group-direction', GroupDirection.Column);
    expect(
      screen
        .getByTestId('text-input-spec.cpu')
        .closest('[data-group-direction]')
    ).toHaveAttribute('data-group-direction', GroupDirection.Row);
  });
});
