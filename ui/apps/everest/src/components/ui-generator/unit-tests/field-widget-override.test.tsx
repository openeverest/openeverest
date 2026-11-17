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
import { zodResolver } from '@hookform/resolvers/zod';
import { UIGenerator } from '../ui-generator';
import {
  TopologyUISchemas,
  WidgetComponent,
  WidgetRegistry,
  WidgetRendererProps,
  WIDGET_UI_TYPE,
  WidgetType,
} from '../ui-generator.types';
import { buildZodSchema } from '../utils/schema-builder';
import { getDefaultValues } from '../utils/default-values';

vi.mock('../utils/schema-builder/cel-validation', () => ({
  extractCelFieldPaths: vi.fn(() => []),
  validateCelExpression: vi.fn(() => true),
}));

const affinity: WidgetComponent = {
  uiType: WIDGET_UI_TYPE,
  widgetType: WidgetType.Affinity,
  path: 'spec.affinity',
};

const schema: TopologyUISchemas = {
  testTopology: {
    sections: {
      basicInfo: {
        label: 'Basic Information',
        components: { affinity },
      },
    },
    sectionsOrder: ['basicInfo'],
  },
};

const FormWrapper = ({ children }: { children: React.ReactNode }) => {
  const { schema: zodSchema } = buildZodSchema(schema, 'testTopology');
  const defaultValues = getDefaultValues(schema, 'testTopology');
  const methods = useForm({
    resolver: zodResolver(zodSchema),
    mode: 'onChange',
    defaultValues,
  });

  return <FormProvider {...methods}>{children}</FormProvider>;
};

describe('UI generator field-widget override', () => {
  it('renders a consumer-registered widget for its widgetType, passing the field name and item', () => {
    const receivedProps: WidgetRendererProps[] = [];
    function RuleWidget({ name, item }: WidgetRendererProps) {
      receivedProps.push({ name, item });
      return <div data-testid="rule-widget">{name}</div>;
    }
    const widgetRegistry: WidgetRegistry = {
      [WidgetType.Affinity]: RuleWidget,
    };

    render(
      <FormWrapper>
        <UIGenerator
          sections={schema.testTopology!.sections}
          sectionKey="basicInfo"
          widgetRegistry={widgetRegistry}
        />
      </FormWrapper>
    );

    expect(screen.getByTestId('rule-widget')).toHaveTextContent(
      'spec.affinity'
    );
    expect(receivedProps).toHaveLength(1);
    expect(receivedProps[0].name).toBe('spec.affinity');
    expect(receivedProps[0].item.widgetType).toBe(WidgetType.Affinity);
  });

  it('renders nothing for the field when no widget is registered for its widgetType', () => {
    render(
      <FormWrapper>
        <UIGenerator
          sections={schema.testTopology!.sections}
          sectionKey="basicInfo"
        />
      </FormWrapper>
    );

    expect(screen.queryByTestId('rule-widget')).not.toBeInTheDocument();
  });

  it('does not make the schema required for a marker widget without a bound path', () => {
    const markerSchema: TopologyUISchemas = {
      testTopology: {
        sections: {
          podScheduling: {
            label: 'Pod scheduling policy',
            components: {
              podSchedulingPolicy: {
                uiType: WIDGET_UI_TYPE,
                widgetType: WidgetType.PodSchedulingPolicy,
              },
            },
          },
        },
        sectionsOrder: ['podScheduling'],
      },
    };

    const { schema: zodSchema } = buildZodSchema(markerSchema, 'testTopology');

    expect(zodSchema.safeParse({}).success).toBe(true);
  });
});
