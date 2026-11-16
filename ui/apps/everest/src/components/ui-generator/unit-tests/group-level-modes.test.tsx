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
  FormMode,
  GroupType,
  TopologyUISchemas,
  WidgetRegistry,
  WidgetRendererProps,
  WidgetType,
  WIDGET_UI_TYPE,
} from '../ui-generator.types';
import { getDefaultValues } from '../utils/default-values';
import { preprocessSchema } from '../utils/preprocess/preprocess-schema';
import { Messages as ToggleableMessages } from '../ui-group-wrappers/toggleable-wrapper/toggleable-wrapper.messages';

const TOPOLOGY = 'replicaSet';

const FormWrapper = ({
  schema,
  sectionKey,
  widgetRegistry,
}: {
  schema: TopologyUISchemas;
  sectionKey: string;
  widgetRegistry?: WidgetRegistry;
}) => {
  const methods = useForm({
    defaultValues: getDefaultValues(schema, TOPOLOGY),
  });

  return (
    <FormProvider {...methods}>
      <UIGenerator
        sections={schema[TOPOLOGY].sections}
        sectionKey={sectionKey}
        widgetRegistry={widgetRegistry}
      />
    </FormProvider>
  );
};

describe('UIGenerator - group-level modes', () => {
  it('does not render a group hidden via modes', () => {
    const schema = preprocessSchema(
      {
        [TOPOLOGY]: {
          sections: {
            advanced: {
              components: {
                monitoring: {
                  uiType: 'group',
                  groupType: GroupType.Bordered,
                  label: 'Monitoring',
                  modes: { [FormMode.Edit]: { hidden: true } },
                  components: {
                    endpoint: {
                      uiType: FieldType.Text,
                      path: 'spec.monitoring.endpoint',
                      fieldParams: { label: 'Endpoint' },
                    },
                  },
                },
                name: {
                  uiType: FieldType.Text,
                  path: 'spec.name',
                  fieldParams: { label: 'Name' },
                },
              },
            },
          },
        },
      },
      undefined,
      FormMode.Edit
    );

    render(
      <TestWrapper>
        <FormWrapper schema={schema} sectionKey="advanced" />
      </TestWrapper>
    );

    expect(screen.queryByText('Monitoring')).not.toBeInTheDocument();
    expect(screen.queryByLabelText('Endpoint')).not.toBeInTheDocument();
    expect(screen.getByLabelText('Name')).toBeInTheDocument();
  });

  it('disables the toggleable switch when the group is mode-disabled', () => {
    const schema = preprocessSchema(
      {
        [TOPOLOGY]: {
          sections: {
            advanced: {
              components: {
                monitoring: {
                  uiType: 'group',
                  groupType: GroupType.Toggleable,
                  label: 'Monitoring',
                  modes: { [FormMode.Edit]: { disabled: true } },
                  components: {
                    endpoint: {
                      uiType: FieldType.Text,
                      path: 'spec.monitoring.endpoint',
                      fieldParams: { label: 'Endpoint' },
                    },
                  },
                },
              },
            },
          },
        },
      },
      undefined,
      FormMode.Edit
    );

    render(
      <TestWrapper>
        <FormWrapper schema={schema} sectionKey="advanced" />
      </TestWrapper>
    );

    expect(
      screen.getByRole('switch', {
        name: ToggleableMessages.switchAriaLabel('Monitoring'),
      })
    ).toBeDisabled();
  });

  it('cascades disabled onto fields inside a bordered group', () => {
    const schema = preprocessSchema(
      {
        [TOPOLOGY]: {
          sections: {
            advanced: {
              components: {
                monitoring: {
                  uiType: 'group',
                  groupType: GroupType.Bordered,
                  label: 'Monitoring',
                  modes: { [FormMode.Edit]: { disabled: true } },
                  components: {
                    endpoint: {
                      uiType: FieldType.Text,
                      path: 'spec.monitoring.endpoint',
                      fieldParams: { label: 'Endpoint' },
                    },
                  },
                },
              },
            },
          },
        },
      },
      undefined,
      FormMode.Edit
    );

    render(
      <TestWrapper>
        <FormWrapper schema={schema} sectionKey="advanced" />
      </TestWrapper>
    );

    expect(screen.getByLabelText(/Endpoint/)).toBeDisabled();
  });

  it('passes disabled to a nested widget when the group is mode-disabled', () => {
    const received: WidgetRendererProps[] = [];
    function ProbeWidget(props: WidgetRendererProps) {
      received.push(props);
      return <div data-testid="probe-widget" />;
    }

    const schema = preprocessSchema(
      {
        [TOPOLOGY]: {
          sections: {
            advanced: {
              components: {
                scheduling: {
                  uiType: 'group',
                  groupType: GroupType.Bordered,
                  label: 'Scheduling',
                  modes: { [FormMode.Edit]: { disabled: true } },
                  components: {
                    policy: {
                      uiType: WIDGET_UI_TYPE,
                      widgetType: WidgetType.Affinity,
                      path: 'spec.affinity',
                    },
                  },
                },
              },
            },
          },
        },
      },
      undefined,
      FormMode.Edit
    );

    render(
      <TestWrapper>
        <FormWrapper
          schema={schema}
          sectionKey="advanced"
          widgetRegistry={{ [WidgetType.Affinity]: ProbeWidget }}
        />
      </TestWrapper>
    );

    expect(screen.getByTestId('probe-widget')).toBeInTheDocument();
    expect(received).toHaveLength(1);
    expect(received[0].disabled).toBe(true);
    expect(received[0].item.disabled).toBe(true);
  });
});
